# Guide d'utilisation des Middlewares d'Authentification

## 🎯 Vue d'ensemble

Les middlewares d'authentification de Soins Suite offrent une approche modulaire et type-safe pour gérer :
- **SessionMiddleware** : Validation des tokens Bearer et enrichissement du contexte
- **PermissionMiddleware** : Contrôle d'accès granulaire par modules et rubriques

## 🚀 Configuration de base

### Injection Fx

```go
// Dans votre module principal
var AppModule = fx.Options(
    // Infrastructure
    database.Module,
    
    // Middlewares
    middleware.Module,               // Middlewares existants
    auth.AuthMiddlewareModule,      // Nouveaux middlewares auth
    
    // Modules métier
    establishment.Module,
    auth.Module,
)
```

### Initialisation des middlewares

```go
// Le stack de middlewares est automatiquement injecté par Fx
func RegisterRoutes(
    r *gin.Engine,
    authStack *auth.AuthMiddlewareStack,
) {
    // Utilisation des middlewares...
}
```

## 📋 Utilisation dans les Routes

### 1. Authentification de base (Session uniquement)

```go
// Routes nécessitant uniquement une session valide
userAPI := r.Group("/api/v1/user")
userAPI.Use(auth.Protected(authStack)...)
{
    userAPI.GET("/profile", controller.GetProfile)
    userAPI.PUT("/profile", controller.UpdateProfile)
}
```

### 2. Contrôle d'accès par module

```go
// Accès complet au module USERS
usersAPI := r.Group("/api/v1/users")
usersAPI.Use(auth.RequireModule(authStack, "USERS")...)
{
    usersAPI.GET("", controller.GetUsers)
    usersAPI.POST("", controller.CreateUser)
}

// Accès complet au module CAISSE
caisseAPI := r.Group("/api/v1/caisse")
caisseAPI.Use(auth.RequireModule(authStack, "CAISSE")...)
{
    caisseAPI.GET("/operations", controller.GetOperations)
    caisseAPI.POST("/operations", controller.CreateOperation)
}
```

### 3. Contrôle d'accès par rubrique

```go
// Accès spécifique à la rubrique CREATE_USER du module USERS
r.POST("/api/v1/users/create", 
    append(auth.RequireRubrique(authStack, "USERS", "CREATE_USER"), 
           controller.CreateUser)...)

// Accès spécifique à la rubrique ENCAISSEMENT du module CAISSE
r.POST("/api/v1/caisse/encaissement",
    append(auth.RequireRubrique(authStack, "CAISSE", "ENCAISSEMENT"),
           controller.ProcessPayment)...)
```

### 4. Contrôle d'accès administrateur

```go
// Routes réservées aux administrateurs
adminAPI := r.Group("/api/v1/admin")
adminAPI.Use(auth.RequireAdmin(authStack)...)
{
    adminAPI.GET("/system-info", controller.GetSystemInfo)
    adminAPI.POST("/maintenance", controller.EnableMaintenance)
}

// Ou spécifiquement pour le back-office
backOfficeAPI := r.Group("/api/v1/back-office")
backOfficeAPI.Use(auth.RequireBackOffice(authStack)...)
{
    backOfficeAPI.GET("/dashboard", controller.GetDashboard)
}
```

### 5. Contrôle par type de client

```go
// Routes spécifiques au front-office
frontOfficeAPI := r.Group("/api/v1/front-office")
frontOfficeAPI.Use(auth.RequireFrontOffice(authStack)...)
{
    frontOfficeAPI.GET("/consultations", controller.GetConsultations)
}
```

## 🔧 Utilisation avancée

### Middlewares multiples et conditionnels

```go
// Combinaison de middlewares
r.GET("/api/v1/reports/financial",
    // Base : establishment + session
    auth.Protected(authStack)[0],
    auth.Protected(authStack)[1],
    // Permission : module REPORTS
    authStack.PermissionMiddleware.RequireModule("REPORTS"),
    // Permission : admin requis
    authStack.PermissionMiddleware.RequireAdmin(),
    controller.GetFinancialReports,
)

// Middleware conditionnel dans le controller
func (c *Controller) GetSensitiveData(ctx *gin.Context) {
    // Vérification additionnelle dans le controller
    if !authStack.PermissionMiddleware.CheckRubriqueAccess(ctx, "ADMIN", "SENSITIVE_DATA") {
        ctx.JSON(465, gin.H{"error": "Permissions insuffisantes"})
        return
    }
    
    // Logique métier...
}
```

### Permissions dynamiques

```go
// Middleware avec permission calculée
func DynamicPermissionMiddleware(authStack *auth.AuthMiddlewareStack, getPermission func(*gin.Context) string) gin.HandlerFunc {
    return func(c *gin.Context) {
        permission := getPermission(c)
        authStack.PermissionMiddleware.RequirePermission(permission)(c)
    }
}

// Utilisation
r.GET("/api/v1/modules/:module/data",
    append(auth.Protected(authStack),
        DynamicPermissionMiddleware(authStack, func(c *gin.Context) string {
            moduleCode := c.Param("module")
            return "module:" + strings.ToUpper(moduleCode)
        }),
        controller.GetModuleData,
    )...,
)
```

## 📊 Utilisation dans les Controllers

### Récupération du contexte

```go
func (c *Controller) GetUserData(ctx *gin.Context) {
    // Récupérer les informations de session
    session, exists := ctx.Get("session")
    if !exists {
        ctx.JSON(401, gin.H{"error": "Session manquante"})
        return
    }
    
    sessionCtx := session.(auth.SessionContext)
    
    // Utiliser les données de session
    userID := sessionCtx.UserID
    establishmentCode := sessionCtx.EtablissementCode
    clientType := sessionCtx.ClientType
    
    // Ou utiliser les raccourcis
    userIDShortcut := ctx.GetString("user_id")
    establishmentID := ctx.GetString("establishment_id")
    
    // Logique métier...
}
```

### Vérifications de permissions dans les controllers

```go
func (c *Controller) UpdateUser(ctx *gin.Context, authStack *auth.AuthMiddlewareStack) {
    userID := ctx.Param("id")
    
    // Vérification conditionnelle
    if !authStack.PermissionMiddleware.CheckRubriqueAccess(ctx, "USERS", "EDIT_USER") {
        ctx.JSON(465, gin.H{"error": "Permission refusée"})
        return
    }
    
    // Vérification de propriété (utilisateur peut modifier ses propres données)
    sessionUserID := ctx.GetString("user_id")
    if userID != sessionUserID && !authStack.PermissionMiddleware.CheckModuleAccess(ctx, "USERS") {
        ctx.JSON(465, gin.H{"error": "Vous ne pouvez modifier que vos propres données"})
        return
    }
    
    // Logique métier...
}
```

## 🛡️ Codes d'erreur

### SessionMiddleware

- **460** : `TOKEN_REQUIRED` - Token Bearer manquant
- **460** : `INVALID_TOKEN` - Token invalide ou expiré
- **460** : `TOKEN_REVOKED` - Token révoqué (blacklist)
- **403** : `ESTABLISHMENT_MISMATCH` - Token valide mais mauvais établissement

### PermissionMiddleware

- **465** : `INSUFFICIENT_PERMISSIONS` - Permissions insuffisantes
- **465** : `ADMIN_ACCESS_REQUIRED` - Accès administrateur requis
- **465** : `CLIENT_TYPE_MISMATCH` - Type de client incorrect

## 🔗 Exemple complet

```go
func RegisterUserRoutes(
    r *gin.Engine,
    userController *controllers.UserController,
    authStack *auth.AuthMiddlewareStack,
) {
    // API utilisateurs
    userAPI := r.Group("/api/v1/users")
    
    // Routes publiques (pas d'authentification)
    userAPI.POST("/forgot-password", userController.ForgotPassword)
    
    // Routes authentifiées (session uniquement)
    authenticated := userAPI.Group("")
    authenticated.Use(auth.Protected(authStack)...)
    {
        authenticated.GET("/me", userController.GetMe)
        authenticated.PUT("/me", userController.UpdateMe)
    }
    
    // Routes avec permissions module
    moduleProtected := userAPI.Group("")
    moduleProtected.Use(auth.RequireModule(authStack, "USERS")...)
    {
        moduleProtected.GET("", userController.GetUsers)
        moduleProtected.GET("/:id", userController.GetUser)
    }
    
    // Routes avec permissions granulaires
    userAPI.POST("/create",
        append(auth.RequireRubrique(authStack, "USERS", "CREATE_USER"),
               userController.CreateUser)...)
               
    userAPI.DELETE("/:id",
        append(auth.RequireRubrique(authStack, "USERS", "DELETE_USER"),
               userController.DeleteUser)...)
    
    // Routes admin uniquement
    userAPI.POST("/bulk-import",
        append(auth.RequireAdmin(authStack),
               userController.BulkImport)...)
}
```

## 🎯 Bonnes pratiques

1. **Ordre des middlewares** : Toujours `EstablishmentMiddleware` → `SessionMiddleware` → `PermissionMiddleware`

2. **Performance** : Utiliser les groupes de routes pour éviter la duplication

3. **Sécurité** : Préférer les permissions granulaires aux permissions larges

4. **Lisibilité** : Utiliser les helpers `auth.RequireModule()` plutôt que les middlewares directement

5. **Debuggage** : Les erreurs incluent tous les détails nécessaires pour diagnostiquer les problèmes