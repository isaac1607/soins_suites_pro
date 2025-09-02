# Conventions de Développement - Soins Suite (Go)

## Standards de Code, Nommage et Bonnes Pratiques

---

## 📋 Table des Matières

1. [**Conventions de Nommage**](#1-conventions-de-nommage)
2. [**Structure de Code**](#2-structure-de-code)
3. [**Conventions Uber Fx**](#3-conventions-uber-fx)
4. [**Standards de Réponse API**](#4-standards-de-réponse-api)
5. [**Gestion des Erreurs**](#5-gestion-des-erreurs)
6. [**Conventions Redis**](#6-conventions-redis)
7. [**Conventions Logging**](#7-conventions-logging)
8. [**Documentation et Commentaires**](#8-documentation-et-commentaires)
9. [**Conventions base de données**](#9-conventions-base-de-données)

---

## 🏷️ 1. CONVENTIONS DE NOMMAGE

### 1.1. Packages et Modules

```go
// ✅ CORRECT - Noms courts, descriptifs, sans underscore
package auth
package establishment
package license
package user

// ❌ INCORRECT
package auth_service
package establishmentAPI
```

### 1.2. Fichiers et Dossiers

```go
// ✅ CORRECT - Notation par points pour fichiers standards
license.handler.go
establishment.service.go
auth.middleware.go
user.repository.go

// ✅ CORRECT - Convention modules Fx
auth.module.go              // Module Fx authentification
patient.module.go           // Module Fx patient
database.module.go          // Module Fx database global
postgres.module.go          // Module Fx PostgreSQL
setup.module.go             // Module Fx setup

// Organisation logique par domaine avec modules Fx
modules/
├── auth/
│   ├── auth.module.go      # Module Fx
│   ├── controllers/
│   ├── services/
│   └── repositories/
├── patient/
│   ├── patient.module.go   # Module Fx
│   ├── controllers/
│   ├── services/
│   └── repositories/
└── setup/
    ├── setup.module.go     # Module Fx
    ├── controllers/
    ├── services/
    └── repositories/
```

### 1.3. Variables et Fonctions

```go
// ✅ CORRECT - camelCase descriptif
var establishmentID string
var licenseInfo *types.LicenseInfo
var setupCompleted bool

// Fonctions - Verbes d'action clairs
func ValidateLicense(key string) (*types.LicenseInfo, error)
func CreateEstablishment(data *dto.EstablishmentRequest) error
func GetSetupState(establishmentID string) (*types.SetupState, error)

// Constants - SCREAMING_SNAKE_CASE
const DEFAULT_SESSION_TTL = 3600
const MAX_LOGIN_ATTEMPTS = 5
const REDIS_KEY_PREFIX = "soins_suite"
```

### 1.4. Types et Structures

```go
// ✅ CORRECT - PascalCase pour types exportés
type LicenseInfo struct {
    Type      string     `json:"type"`
    Modules   []string   `json:"modules"`
    ExpiresAt *time.Time `json:"expires_at"`
}

type EstablishmentConfig struct {
    ID        string `json:"id" db:"id"`
    Name      string `json:"name" db:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Interfaces - Suffixe descriptif
type LicenseValidator interface {
    ValidateAndDecode(licenseKey string) (*LicenseInfo, error)
    CheckExpiration(license *LicenseInfo) bool
}
```

### 1.5. Constantes par Domaine

```go
// ✅ CORRECT - Groupement par domaine
const (
    // Auth & Sessions
    SessionTTLDefault = 3600 * time.Second
    TokenLength       = 32
    TokenPrefix       = "soins_suite_session_"

    // License Types
    LicenseTypePremium = "premium"
    LicenseTypeStandard = "standard"

    // User Roles
    RoleSuperAdmin  = "super_admin"
    RoleAdminSimple = "admin_simple"
    RoleUser        = "user"
)
```

---

## 🏗️ 2. STRUCTURE DE CODE

### 2.1. Handler Standard

```go
// Structure handler cohérente
type LicenseHandler struct {
    licenseService   *services.LicenseService
}

func NewLicenseHandler(
    licenseService *services.LicenseService,
) *LicenseHandler {
    return &LicenseHandler{
        licenseService: licenseService,
    }
}

func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
    // 1. Décoder et valider requête
    // 2. Traiter via service
    // 3. Retourner réponse standardisée
    // Note: Logging automatique via middleware Gin
}
```

### 2.2. Service Standard

```go
// Service avec dépendances claires
type LicenseService struct {
    encryptionKey []byte
    cacheRepo     *repositories.LicenseCacheRepository
}

func (s *LicenseService) ValidateAndDecode(licenseKey string) (*types.LicenseInfo, error) {
    // 1. Vérifier cache
    // 2. Valider et décoder
    // 3. Mettre en cache
    // 4. Retourner résultat
}
```

### 2.3. Repository Standard

```go
// Repository avec interface claire
type StateRepository struct {
    client *redis.Client
}

func (r *StateRepository) SaveSetupState(establishmentID string, state *types.SetupState) error {
    key := fmt.Sprintf("soins_suite_setup_state:%s", establishmentID)
    // Sérialisation, sauvegarde avec TTL
}
```

### 2.4. Module Fx Standard

```go
// ✅ CORRECT - Structure module Fx avec injection claire
package auth

import (
    "go.uber.org/fx"
    "github.com/gin-gonic/gin"
)

// Module regroupe tous les providers du domaine Auth
var Module = fx.Options(
    // Services
    fx.Provide(NewAuthService),
    fx.Provide(NewTokenService),
    fx.Provide(NewSessionService),

    // Repositories
    fx.Provide(NewUserRepository),
    fx.Provide(NewSessionRepository),

    // Controllers
    fx.Provide(NewWebAuthController),
    fx.Provide(NewMobileAuthController),

    // Configuration des routes
    fx.Invoke(RegisterAuthRoutes),
)

// RegisterAuthRoutes configure les routes Gin pour Auth
func RegisterAuthRoutes(
    r *gin.Engine,
    webCtrl *WebAuthController,
    mobileCtrl *MobileAuthController,
    authMiddleware *middleware.AuthMiddleware,
) {
    authGroup := r.Group("/api/v1/auth")
    {
        // Routes publiques
        authGroup.POST("/login", webCtrl.Login)
        authGroup.POST("/mobile/login", mobileCtrl.Login)

        // Routes protégées
        protected := authGroup.Group("")
        protected.Use(authMiddleware.RequireAuth())
        {
            protected.POST("/logout", webCtrl.Logout)
            protected.GET("/profile", webCtrl.GetProfile)
        }
    }
}
```

### 2.5. Constructeur avec Injection Fx

```go
// ✅ CORRECT - Constructeur compatible Fx
type AuthService struct {
    userRepo      UserRepository
    sessionRepo   SessionRepository
    tokenService  *TokenService
}

func NewAuthService(
    userRepo UserRepository,
    sessionRepo SessionRepository,
    tokenService *TokenService,
) *AuthService {
    return &AuthService{
        userRepo:     userRepo,
        sessionRepo:  sessionRepo,
        tokenService: tokenService,
    }
}

// ❌ INCORRECT - Constructeur non compatible Fx
func NewAuthService() *AuthService {
    // Dépendances hardcodées
    userRepo := postgres.NewUserRepository(db)
    return &AuthService{userRepo: userRepo}
}
```

---

## 🔧 3. CONVENTIONS UBER FX

### 3.1. Organisation des Modules

```go
// ✅ CORRECT - Module par domaine métier
internal/modules/
├── auth/auth.module.go                    # Authentification
├── back-office/
│   ├── setup/setup.module.go              # Configuration initiale
│   ├── users/users.module.go              # Gestion utilisateurs
│   └── establishment/establishment.module.go # Gestion établissements
└── front-office/
    ├── patient/patient.module.go          # Gestion patients
    └── medical/medical.module.go          # Modules médicaux
```

### 3.2. Assemblage Principal

```go
// internal/app/modules.go - Point central d'assemblage
package app

import (
    "go.uber.org/fx"
    "soins-suite-core/internal/infrastructure/database"
    "soins-suite-core/internal/modules/auth"
    "soins-suite-core/internal/modules/back-office/setup"
    // ... autres imports
)

var AppModule = fx.Options(
    // Infrastructure (consolidée)
    database.Module,        // PostgreSQL + Redis + MongoDB

    // Modules métier
    auth.Module,
    setup.Module,
    patient.Module,

    // Bootstrap application
    fx.Provide(NewGinEngine),
    fx.Provide(NewConfig),
    fx.Invoke(RegisterRoutes),
    fx.Invoke(StartServer),
)
```

### 3.3. Patterns d'Injection

```go
// ✅ CORRECT - Interface injection
type UserService struct {
    repo   UserRepository    // Interface, pas implémentation
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

// ✅ CORRECT - Provider avec validation
func NewUserService(repo UserRepository) (*UserService, error) {
    if repo == nil {
        return nil, errors.New("user repository is required")
    }
    return &UserService{repo: repo}, nil
}
```

### 3.4. Lifecycle Hooks

```go
// ✅ CORRECT - Hooks de démarrage/arrêt
type DatabaseService struct {
    client *postgres.Client
}

func NewDatabaseService(lc fx.Lifecycle, config *Config) *DatabaseService {
    client := postgres.NewClient(config.Database)

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return client.Connect(ctx)
        },
        OnStop: func(ctx context.Context) error {
            return client.Close(ctx)
        },
    })

    return &DatabaseService{client: client}
}
```

---

## 🌐 4. STANDARDS DE RÉPONSE API

### 4.1. Format Succès (200, 201)

```json
{
  "success": true,
  "data": "...", // Objet, liste ou null
  "message": "Opération réussie." // Optionnel
}
```

**Exemples :**

```json
// GET /api/setup/status
{
  "success": true,
  "data": {
    "establishment_id": "est_123abc456def789ghi",
    "current_step": "establishment_configured",
    "progress_percentage": 80
  }
}

// POST /api/setup/validate-license
{
  "success": true,
  "data": {
    "license_type": "standard",
    "expires_at": "2026-07-23T23:59:59Z",
    "modules_available": ["consultation", "facturation"]
  },
  "message": "Licence validée avec succès."
}
```

### 4.2. Format Erreur (4xx, 5xx)

```json
{
  "error": "Message d'erreur clair et concis.",
  "details": {} // Optionnel - détails supplémentaires
}
```

**Exemples :**

```json
// 400 Bad Request
{
  "error": "Les données fournies sont invalides.",
  "details": {
    "license_key": "Format de clé invalide (attendu: XXXX-XXXX-XXXX-XXXX).",
    "establishment_name": "Doit contenir entre 3 et 100 caractères."
  }
}

// 401 Unauthorized
{
  "error": "Identifiants incorrects."
}

// 409 Conflict
{
  "error": "Un conflit empêche l'opération.",
  "details": {
    "reason": "Setup déjà en cours",
    "current_step": "establishment_config"
  }
}
```

### 4.3. Codes Personnalisés Soins Suite

```go
const (
    StatusTokenInvalidOrExpired    = 460
    StatusInsufficientPermissions = 465
)

// 460 Token Invalid/Expired
{
  "error": "Token invalide ou expiré."
}

// 465 Insufficient Permissions
{
  "error": "Permissions insuffisantes pour accéder à cette ressource.",
  "details": {
    "required_module": "establishment",
    "required_rubrique": "modules_services"
  }
}
```

### 4.4. Helpers de Réponse

```go
// Réponses standardisées
func WriteSuccessResponse(w http.ResponseWriter, data interface{}, message ...string)
func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string, details ...interface{})
func WriteTokenErrorResponse(w http.ResponseWriter)
func WritePermissionErrorResponse(w http.ResponseWriter, requiredModule, requiredRubrique string)
```

---

## ⚠️ 5. GESTION DES ERREURS

### 5.1. Types d'Erreurs Métier

```go
// Erreurs Setup
type SetupError struct {
    Step    string
    Reason  string
    Details map[string]interface{}
}

// Erreurs Licence
type LicenseError struct {
    Type    string // "invalid", "expired", "corrupted"
    Message string
    Key     string
}

// Erreurs Auth
type AuthError struct {
    Type    string // "invalid_credentials", "token_expired", "insufficient_permissions"
    Message string
}
```

### 5.2. Patterns de Gestion

```go
// Wrapping avec contexte
func (s *LicenseService) ValidateAndDecode(licenseKey string) (*types.LicenseInfo, error) {
    if err := s.validateFormat(licenseKey); err != nil {
        return nil, fmt.Errorf("validation format licence échouée: %w", err)
    }

    info, err := s.decodeLicense(licenseKey)
    if err != nil {
        return nil, &LicenseError{
            Type:    "corrupted",
            Message: "Impossible de décoder la licence",
            Key:     licenseKey,
        }
    }

    return info, nil
}
```

### 5.3. Middleware Recovery

```go
// Capture des paniques avec logging détaillé
func RecoveryMiddleware() gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        // Le logging est géré automatiquement par Gin
        c.JSON(500, gin.H{"error": "Une erreur interne est survenue."})
    })
}
```

---

## 🔴 6. CONVENTIONS REDIS

### 6.1. Nomenclature des Clés

**Pattern :** `soins_suite_{domain}_{context}:{identifier}`

```go
const (
    // Sessions
    RedisKeySession = "soins_suite_session:%s"
    RedisKeyUserSessions = "soins_suite_user_sessions:%s:%s" // userID:establishmentID

    // Setup
    RedisKeySetupState = "soins_suite_setup_state:%s"     // establishmentID
    RedisKeySetupLicense = "soins_suite_setup_license:%s" // establishmentID

    // Permissions
    RedisKeyPermissions = "soins_suite_permissions:%s:%s" // userID:establishmentID

    // Configuration
    RedisKeyEstablishmentConfig = "soins_suite_establishment_config:%s"
    RedisKeyEstablishmentModules = "soins_suite_establishment_modules:%s"

    // Cache
    RedisKeyFormSchema = "soins_suite_form_schema:%s:%s"    // module:formType
    RedisKeyLicenseCache = "soins_suite_license_cache:%s"   // licenseKey
)
```

### 6.2. TTL Standards

```go
const (
    // Sessions
    TTLSession         = 3600 * time.Second   // 1 heure
    TTLSessionSetup    = 7200 * time.Second   // 2 heures pour setup

    // Configuration
    TTLConfigEstablishment = 86400 * time.Second  // 24 heures
    TTLConfigModules      = 86400 * time.Second   // 24 heures

    // Cache
    TTLFormSchema      = 3600 * time.Second   // 1 heure
    TTLLicenseCache    = 1800 * time.Second   // 30 minutes

    // Temporaire
    TTLSetupTemp       = 7200 * time.Second   // 2 heures
)
```

### 6.3. Helpers Utilitaires

```go
// Générateurs de clés type-safe
func SessionKey(sessionHandle string) string {
    return fmt.Sprintf("soins_suite_session:%s", sessionHandle)
}

func PermissionsKey(userID, establishmentID string) string {
    return fmt.Sprintf("soins_suite_permissions:%s:%s", userID, establishmentID)
}

func SetupStateKey(establishmentID string) string {
    return fmt.Sprintf("soins_suite_setup_state:%s", establishmentID)
}
```

---

## 🖨️ 7. CONVENTIONS LOGGING

### 7.1. Logger Gin par Défaut

**✅ OBLIGATOIRE : Utiliser uniquement le logger Gin par défaut**

```go
// ✅ CORRECT - Configuration Gin avec logger par défaut
func NewGinEngine() *gin.Engine {
    // En développement : mode debug avec logs détaillés
    gin.SetMode(gin.DebugMode)

    r := gin.Default() // Inclut logger et recovery automatiquement

    return r
}

// ✅ CORRECT - Configuration production
func NewGinEngineProduction() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)

    r := gin.New()
    r.Use(gin.Logger())    // Logger Gin standard
    r.Use(gin.Recovery())  // Recovery automatique

    return r
}
```

### 7.2. Output Gin Standard

**Développement (gin.DebugMode) :**

```
[GIN-debug] GET    /api/v1/auth/login        --> main.(*AuthHandler).Login (3 handlers)
[GIN-debug] POST   /api/v1/setup/bootstrap   --> main.(*SetupHandler).Bootstrap (4 handlers)

[GIN] 2023/09/04 - 15:22:47 | 200 |   2.345ms |  192.168.1.100 | POST     "/api/v1/auth/login"
[GIN] 2023/09/04 - 15:22:48 | 400 |   1.234ms |  192.168.1.100 | POST     "/api/v1/setup/validate-license"
```

**Production (gin.ReleaseMode) :**

```
[GIN] 2023/09/04 - 15:22:47 | 200 |   2.345ms |  192.168.1.100 | POST     "/api/v1/auth/login"
[GIN] 2023/09/04 - 15:22:48 | 400 |   1.234ms |  192.168.1.100 | POST     "/api/v1/setup/validate-license"
```

### 7.3. Gestion des Erreurs dans les Handlers

```go
// ✅ CORRECT - Logging d'erreurs métier sans dépendance externe
func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
    var req dto.ValidateLicenseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // Gin log automatiquement la requête avec status 400
        c.JSON(400, gin.H{"error": "Données invalides"})
        return
    }

    license, err := h.licenseService.ValidateAndDecode(req.LicenseKey)
    if err != nil {
        // Log manuel pour erreurs métier importantes
        fmt.Printf("[LICENSE ERROR] %s: %v\n", time.Now().Format("15:04:05"), err)
        c.JSON(400, gin.H{"error": "Licence invalide"})
        return
    }

    // Gin log automatiquement la réponse 200
    c.JSON(200, gin.H{"success": true, "data": license})
}
```

### 7.4. Log Manuel pour Cas Critiques

```go
// ✅ CORRECT - Log manuel simple pour erreurs critiques
func (s *SetupService) Bootstrap(ctx context.Context, req *dto.BootstrapRequest) error {
    if err := s.migrationService.RunMigrations(ctx); err != nil {
        // Log critique : échec migration
        fmt.Printf("[CRITICAL] %s Migration failed: %v\n", time.Now().Format("15:04:05"), err)
        return fmt.Errorf("migration failed: %w", err)
    }

    if err := s.seedDefaultData(ctx, req); err != nil {
        // Log critique : échec seeding
        fmt.Printf("[CRITICAL] %s Seeding failed: %v\n", time.Now().Format("15:04:05"), err)
        return fmt.Errorf("seeding failed: %w", err)
    }

    fmt.Printf("[SUCCESS] %s Bootstrap completed\n", time.Now().Format("15:04:05"))
    return nil
}
```

### 7.5. Avantages du Logger Gin

✅ **Lisibilité** : Format simple et clair  
✅ **Performance** : Très léger, pas de sérialisation JSON  
✅ **Simplicité** : Aucune configuration requise  
✅ **Intégration** : Logging automatique des requêtes HTTP  
✅ **Debugging** : Routes affichées au démarrage en mode debug

### 7.6. Ne PAS Utiliser

```go
// ❌ INTERDIT - Loggers externes complexes
import "go.uber.org/zap"
import "github.com/sirupsen/logrus"

// ❌ INTERDIT - Injection de logger dans les structures
type AuthService struct {
    userRepo UserRepository
    logger   *zap.Logger  // ❌ Ne pas faire
}

// ❌ INTERDIT - Configuration complexe de logging
func NewComplexLogger() *zap.Logger {
    // Configuration verbose et illisible
}
```

---

## 📚 8. DOCUMENTATION ET COMMENTAIRES

### 8.1. Fonctions Publiques

```go
// ValidateLicense valide et décode une clé de licence de 16 caractères.
//
// La clé doit suivre le format XXXX-XXXX-XXXX-XXXX où:
// - Les 4 premiers caractères indiquent le type de licence
// - Les 4 suivants encodent les modules autorisés
// - Les 4 suivants encodent la date d'expiration (si licence standard)
// - Les 4 derniers constituent un checksum de validation
//
// Retourne les détails de la licence si validation réussie, erreur sinon.
func (s *LicenseService) ValidateLicense(licenseKey string) (*types.LicenseInfo, error)
```

### 8.2. Structures et Types

```go
// LicenseInfo contient les informations décodées d'une licence Soins Suite.
// Utilisée pour valider les permissions et modules autorisés.
type LicenseInfo struct {
    // Type de licence: "premium" pour accès illimité, "standard" pour licence limitée
    Type string `json:"type" validate:"required,oneof=premium standard"`

    // Liste des codes modules autorisés par cette licence
    Modules []string `json:"modules" validate:"required,min=1"`

    // Date d'expiration (nil pour licence premium, obligatoire pour standard)
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
```

### 8.3. Endpoints API

```go
// ValidateLicense valide une clé de licence et initie le processus de setup.
//
// POST /api/setup/validate-license
//
// Request: {"license_key": "XXXX-XXXX-XXXX-XXXX"}
// Response 200: Licence valide avec détails
// Response 400: Clé invalide ou corrompue
func (h *LicenseHandler) ValidateLicense(c *gin.Context)
```

### 8.4. Sécurité dans les Logs

```go
// ✅ CORRECT - Masquer données sensibles avec fmt.Printf
func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
    // Log sécurisé - préfixe seulement
    fmt.Printf("[LICENSE] %s Validation success: type=%s, modules=%d, prefix=%s\n",
        time.Now().Format("15:04:05"),
        licenseInfo.Type,
        len(licenseInfo.Modules),
        licenseKey[:4], // 4 premiers caractères seulement
    )
}

// ❌ INCORRECT - Données sensibles exposées
fmt.Printf("[AUTH] Token created: %s\n", fullToken)     // Token complet
fmt.Printf("[LICENSE] Key: %s\n", licenseKey)          // Clé complète
```

---

## 🗄️ 9. CONVENTIONS BASE DE DONNÉES

### 9.1. Infrastructure Postgres Centralisée

**✅ OBLIGATOIRE : Utiliser l'infrastructure postgres centralisée (client.go + transaction.go)**

```go
// ✅ CORRECT - Utilisation du Client centralisé
import "soins-suite-core/internal/infrastructure/database/postgres"

type UserRepository struct {
    db *postgres.Client
}

func NewUserRepository(db *postgres.Client) *UserRepository {
    return &UserRepository{
        db: db,
    }
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
    query := `SELECT id, name, email FROM users WHERE id = $1`

    // Utiliser l'abstraction Client
    row := r.db.QueryRow(ctx, query, userID)

    var user User
    err := row.Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, fmt.Errorf("erreur scan utilisateur: %w", err)
    }

    return &user, nil
}

// ❌ INTERDIT - Accès direct au pool
func (r *UserRepository) BadExample(ctx context.Context, userID string) (*User, error) {
    // NE PAS FAIRE : r.db.Pool().QueryRow()
    row := r.db.Pool().QueryRow(ctx, query, userID) // ❌ Interdit
}
```

### 9.2. Gestion des Transactions avec TransactionManager

```go
// ✅ CORRECT - Utilisation du TransactionManager centralisé
type OrderService struct {
    orderRepo   OrderRepository
    txManager   *postgres.TransactionManager
}

func NewOrderService(
    orderRepo OrderRepository,
    txManager *postgres.TransactionManager,
) *OrderService {
    return &OrderService{
        orderRepo: orderRepo,
        txManager: txManager,
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    // Transaction atomique avec rollback automatique
    return s.txManager.WithTransaction(ctx, func(tx *postgres.Transaction) error {
        // Toutes les opérations dans la transaction
        if err := s.createOrderRecord(ctx, order); err != nil {
            return err // Rollback automatique
        }

        if err := s.updateInventory(ctx, order.Items); err != nil {
            return err // Rollback automatique
        }

        return nil // Commit automatique
    })
}

// ❌ INTERDIT - Gestion manuelle des transactions
func (r *Repository) BadTransactionExample(ctx context.Context) error {
    tx, err := r.db.Pool().Begin(ctx) // ❌ Interdit
    defer tx.Rollback(ctx)
    // ...
}
```

### 9.3. Avantages de l'Infrastructure Centralisée

✅ **Abstraction** : Client unifié pour toutes les opérations  
✅ **Transactions** : TransactionManager avec rollback automatique  
✅ **Réutilisabilité** : Infrastructure commune à tous les modules  
✅ **Maintenance** : Configuration centralisée dans postgres.module.go  
✅ **Monitoring** : Métriques et health checks intégrés

### 9.4. Pattern d'Accès aux Données avec Client

```go
// ✅ CORRECT - Requêtes avec Client centralisé
func (r *Repository) GetMultipleRecords(ctx context.Context, ids []int) ([]Record, error) {
    query := `
        SELECT id, name, created_at
        FROM records
        WHERE id = ANY($1)
        ORDER BY created_at DESC
    `

    // Utiliser r.db.Query() au lieu de r.db.Pool().Query()
    rows, err := r.db.Query(ctx, query, ids)
    if err != nil {
        return nil, fmt.Errorf("erreur requête: %w", err)
    }
    defer rows.Close()

    var records []Record
    for rows.Next() {
        var record Record
        err := rows.Scan(&record.ID, &record.Name, &record.CreatedAt)
        if err != nil {
            return nil, fmt.Errorf("erreur scan: %w", err)
        }
        records = append(records, record)
    }

    return records, rows.Err()
}
```

---

## ✅ CHECKLIST VALIDATION

### Code Review

- [ ] **Nommage** : Packages lowercase, fichiers notation par points, variables camelCase
- [ ] **Structure** : Handlers/Services/Repositories suivent patterns standards
- [ ] **Modules Fx** : Fichiers `.module.go`, injection correcte, assemblage centralisé
- [ ] **API** : Format success/error respecté avec codes appropriés
- [ ] **Erreurs** : Types métier personnalisés avec wrapping contextuel
- [ ] **Redis** : Clés suivent pattern, TTL appropriés
- [ ] **Base de données** : pgxpool.Pool utilisé, injection via client PostgreSQL
- [ ] **Logging** : Logger Gin par défaut uniquement, pas de dépendances externes
- [ ] **Documentation** : Fonctions publiques documentées, exemples fournis
- [ ] **Sécurité** : Données sensibles masquées dans logs et réponses

Ces conventions garantissent un code cohérent, maintenable et sécurisé pour Soins Suite ! 🚀
