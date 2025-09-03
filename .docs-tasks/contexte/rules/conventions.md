# Conventions Soins Suite - Go/Gin/Fx

## 📋 Architecture MVP Simplifiée

**Philosophie :** Services → Queries → DB (pas de repositories pour MVP)

```
module-name/
├── module-name.module.go    # Module Fx
├── controllers/             # Endpoints HTTP
├── services/                # Logique métier → queries directement
├── dto/                     # Types requête/réponse
└── queries/                 # SQL natif PostgreSQL
```

---

## 🏷️ Conventions de Nommage

### Packages & Fichiers

```go
// ✅ Packages : lowercase, sans underscore
package auth, establishment, license

// ✅ Fichiers : notation par points
auth.controller.go, license.service.go, user.queries.go

// ✅ Modules Fx : nom-module.module.go
auth.module.go, establishment.module.go
```

### Variables & Types

```go
// ✅ Variables : camelCase
var establishmentID, licenseInfo, setupCompleted

// ✅ Fonctions : Verbes d'action
func ValidateLicense(), CreateEstablishment(), GetSetupState()

// ✅ Types : PascalCase
type LicenseInfo struct { Type string `json:"type"` }

// ✅ Constantes : SCREAMING_SNAKE_CASE
const DEFAULT_SESSION_TTL = 3600
```

---

## 🔧 Patterns Obligatoires

### Module Fx Standard

```go
var Module = fx.Options(
    fx.Provide(NewAuthService),
    fx.Provide(NewAuthController),
    fx.Invoke(RegisterAuthRoutes),
)
```

### Controller Standard

```go
type AuthController struct {
    service *AuthService
}

func (c *AuthController) Login(ctx *gin.Context) {
    var req dto.LoginRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": "Données invalides"})
        return
    }

    result, err := c.service.Authenticate(ctx, req)
    if err != nil {
        ctx.JSON(401, gin.H{"error": "Authentification échouée"})
        return
    }

    ctx.JSON(200, gin.H{"success": true, "data": result})
}
```

### Service Standard (MVP)

```go
type AuthService struct {
    db          *postgres.Client  // ✅ Client centralisé
    redisClient *redis.Client
}

func (s *AuthService) Authenticate(ctx context.Context, req dto.LoginRequest) (*dto.AuthResult, error) {
    // Utilise queries directement
    var user User
    err := s.db.QueryRow(ctx, queries.UserQueries.GetByEmail, req.Email).Scan(&user.ID, &user.Email)
    if err != nil {
        return nil, err
    }

    // Cache intelligent Redis
    sessionKey := utils.SessionKey(user.EstablishmentCode, sessionID)
    s.redisClient.Set(ctx, sessionKey, sessionData, time.Hour)

    return &dto.AuthResult{Token: sessionID}, nil
}
```

### Queries SQL Natives

```go
// queries/user.postgres.go
var UserQueries = struct {
    GetByEmail    string
    Create        string
    UpdateStatus  string
}{
    GetByEmail: `
        SELECT id, email, password_hash, establishment_code
        FROM users
        WHERE email = $1 AND deleted_at IS NULL
    `,
    Create: `
        INSERT INTO users (email, password_hash, establishment_code)
        VALUES ($1, $2, $3)
        RETURNING id, email, establishment_code
    `,
}
```

---

## 🌐 Réponses API Standard

### Succès (200, 201)

```json
{
  "success": true,
  "data": "...", // Objet, liste ou null
  "message": "Optionnel"
}
```

### Erreur (4xx, 5xx)

```json
{
  "error": "Message utilisateur clair", // Obligatoire - Frontend
  "details": {} // Obligatoire - Développeurs
}
```

---

## 🔴 Conventions Redis

### Pattern Obligatoire

`soins_suite_{code_etablissement}_{domain}_{context}:{identifier}`

### Clés Standards

```go
const (
    // Auth
    RedisKeySession = "soins_suite_%s_auth_session:%s"          // etablissement:sessionId
    RedisKeyPermissions = "soins_suite_%s_auth_permissions:%s"  // etablissement:userID

    // Setup
    RedisKeySetupState = "soins_suite_%s_setup_state:%s"        // etablissement:step
    RedisKeySetupLicense = "soins_suite_%s_setup_license:%s"    // etablissement:licenseKey

    // Cache
    RedisKeyLicenseCache = "soins_suite_%s_cache_license:%s"    // etablissement:licenseKey
    RedisKeyFormSchema = "soins_suite_%s_cache_form:%s_%s"      // etablissement:module:formType
)
```

### Helpers Type-Safe

```go
func SessionKey(establishmentCode, sessionID string) string {
    return fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, sessionID)
}

func LicenseCacheKey(establishmentCode, licenseKey string) string {
    return fmt.Sprintf("soins_suite_%s_cache_license:%s", establishmentCode, licenseKey)
}
```

### TTL Standards

```go
const (
    TTLSession         = 3600 * time.Second   // 1 heure
    TTLLicenseCache    = 1800 * time.Second   // 30 minutes
    TTLSetupTemp       = 7200 * time.Second   // 2 heures
)
```

---

## 🖨️ Logging Simple

**✅ OBLIGATOIRE : Logger Gin par défaut uniquement**

```go
// Configuration Gin
func NewGinEngine() *gin.Engine {
    gin.SetMode(gin.DebugMode)
    return gin.Default() // Logger + Recovery automatiques
}

// Log manuel pour erreurs critiques seulement
func (s *Service) CriticalOperation(ctx context.Context) error {
    if err := s.operation(); err != nil {
        fmt.Printf("[CRITICAL] %s Operation failed: %v\n", time.Now().Format("15:04:05"), err)
        return err
    }
    return nil
}
```

---

## 🗄️ Base de Données

### Client PostgreSQL Centralisé

```go
// ✅ CORRECT - Injection du client centralisé
type Service struct {
    db *postgres.Client  // ✅ Utiliser cette abstraction
}

// ❌ INTERDIT - Accès direct au pool
// r.db.Pool().QueryRow() ❌
```

### Gestion d'Erreurs MVP

```go
type ServiceError struct {
    Type    string                 // "validation", "not_found", "conflict"
    Message string
    Details map[string]interface{}
}

func (e *ServiceError) Error() string { return e.Message }
```

---

## ✅ Checklist Validation

- [ ] **Modules Fx** : `.module.go`, providers corrects
- [ ] **Architecture MVP** : Services → Queries → Client PostgreSQL centralisé
- [ ] **Redis** : Pattern `soins_suite_{etablissement}_{domain}_{context}:{id}` + helpers
- [ ] **API** : Format success/error respecté
- [ ] **Logging** : Gin par défaut uniquement
- [ ] **SQL** : Requêtes natives dans `queries/`
- [ ] **Injection Fx** : Constructeurs compatibles
