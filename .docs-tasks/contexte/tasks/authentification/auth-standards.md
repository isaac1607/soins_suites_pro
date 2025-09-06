# Standards d'Authentification - Soins Suite Core

## 🎯 Vision

**Simplicité, Performance, Sécurité Multi-tenant**

Système de tokens opaques stockés dans Redis avec isolation stricte par établissement, optimisé pour la performance et respectant les conventions du projet.

---

## 🔐 Architecture d'Authentification

### **Token Opaque**

Format standardisé pour l'écosystème Soins Suite :

- **Format** : UUID v4 (36 caractères)
- **Exemple** : `a1b2c3d4-e5f6-47h8-89i9-j0k1l2m3n4o5`
- **Header** : `Authorization: Bearer {token}`
- **Stockage** : Redis avec isolation multi-tenant

### **Headers Obligatoires**

```http
X-Establishment-Code: CENTREA
X-Client-Type: front-office  # ou back-office
Authorization: Bearer {token}
```

---

## 🔄 Workflows d'Authentification

### **Login - POST /api/v1/auth/login**

**Request :**

```http
POST /api/v1/auth/login
X-Establishment-Code: CENTREA
X-Client-Type: front-office
Content-Type: application/json

{
  "identifiant": "john.doe",
  "password": "SecurePass123!"
}
```

**Processus interne :**

1. Validation établissement via EstablishmentMiddleware
2. Vérification identifiants (PostgreSQL)
3. Contrôle cohérence client_type vs est_admin
4. Génération token UUID v4
5. Stockage session Redis multi-tenant
6. Cache permissions utilisateur

**Response 200 :**

```json
{
  "success": true,
  "data": {
    "token": "a1b2c3d4-e5f6-47h8-89i9-j0k1l2m3n4o5",
    "user": {
      "id": "user-uuid",
      "nom": "Doe",
      "prenoms": "John",
      "est_admin": false,
      "est_medecin": true
    },
    "expires_at": "2025-01-15T15:30:00Z"
  }
}
```

### **Logout - POST /api/v1/auth/logout**

**Request :**

```http
POST /api/v1/auth/logout
X-Establishment-Code: CENTREA
Authorization: Bearer {token}
```

**Response 200 :**

```json
{
  "success": true,
  "message": "Déconnexion réussie"
}
```

### **Refresh - POST /api/v1/auth/refresh**

**Request :**

```http
POST /api/v1/auth/refresh
X-Establishment-Code: CENTREA
Authorization: Bearer {old-token}
```

**Response 200 :**

```json
{
  "success": true,
  "data": {
    "token": "new-token-uuid",
    "expires_at": "2025-01-15T16:30:00Z"
  }
}
```

---

## 🔑 Système de Permissions

### **Concepts Fondamentaux**

#### **Module**

Domaine fonctionnel complet (ex: ACCUEIL, CAISSE, INFIRMERIE)

#### **Rubrique**

Sous-section d'un module pour contrôle granulaire

### **Logique de Permissions**

#### **Accès Module (90% des cas)**

```
User + Module = Accès TOUTES rubriques
```

#### **Accès Rubrique (10% des cas)**

```
User + Rubrique spécifique = Accès restreint
```

### **Structure Redis Permissions**

```
Clé : soins_suite_{etablissement}_auth_permissions:{user_id}
Type : SET
TTL : 3600s

Contenu :
- module:ACCUEIL           # Accès complet module
- module:CAISSE           # Accès complet module
- rubrique:INFIRMERIE:consultation  # Accès restreint
```

---

## 🗄️ Schémas Redis

### **Session**

```
Clé : soins_suite_{etablissement}_auth_session:{token}
Type : HASH
TTL : 3600s

Champs :
{
  "user_id": "uuid",
  "etablissement_id": "uuid",
  "etablissement_code": "CENTREA",
  "client_type": "front-office",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "created_at": "2025-01-15T14:30:00Z",
  "last_activity": "2025-01-15T14:45:00Z"
}
```

### **Permissions**

```
Clé : soins_suite_{etablissement}_auth_permissions:{user_id}
Type : SET
TTL : 3600s

Membres :
- module:ACCUEIL
- module:CAISSE
- rubrique:INFIRMERIE:prescriptions
```

### **Index Sessions Utilisateur**

```
Clé : soins_suite_{etablissement}_auth_user_sessions:{user_id}
Type : SET
TTL : 3600s

Membres : [token1, token2, token3]
```

---

## 📋 Réponses API Standardisées

### **Erreurs Authentification**

#### **401 - Identifiants Incorrects**

```json
{
  "error": "Identifiant ou mot de passe incorrect",
  "details": {
    "code": "INVALID_CREDENTIALS"
  }
}
```

#### **403 - Client Type Incorrect**

```json
{
  "error": "Accès refusé à cette interface",
  "details": {
    "code": "CLIENT_TYPE_MISMATCH",
    "client_type_required": "back-office",
    "client_type_provided": "front-office"
  }
}
```

#### **460 - Token Invalide/Expiré**

```json
{
  "error": "Session expirée ou invalide",
  "details": {
    "code": "TOKEN_EXPIRED"
  }
}
```

#### **465 - Permissions Insuffisantes**

```json
{
  "error": "Permissions insuffisantes pour cette action",
  "details": {
    "code": "INSUFFICIENT_PERMISSIONS",
    "required": "module:CAISSE",
    "user_permissions": ["module:ACCUEIL"]
  }
}
```

---

## 🔧 Middlewares d'Authentification

### **SessionMiddleware**

Valide le token et enrichit le contexte :

```go
// Données injectées dans gin.Context
type SessionContext struct {
    UserID           string `json:"user_id"`
    EstablishmentID  string `json:"establishment_id"`
    ClientType       string `json:"client_type"`
    Token           string `json:"token"`
}

// Utilisation dans controller
session := c.MustGet("session").(SessionContext)
```

### **PermissionMiddleware**

Vérifie les permissions requises :

```go
// Configuration route
router.GET("/api/v1/caisse/operations",
    SessionMiddleware(),
    PermissionMiddleware("module:CAISSE"),
    controller.GetOperations,
)
```

---

## 🚀 Architecture Simplifiée

### **Structure Module Auth**

```
internal/modules/auth/
├── auth.module.go              # Module Fx
├── controllers/
│   └── auth.controller.go      # Login/Logout/Refresh
├── services/
│   ├── auth.service.go         # Logique auth
│   ├── session.service.go      # Gestion sessions Redis
│   └── permission.service.go   # Gestion permissions
├── dto/
│   └── auth.dto.go            # Request/Response types
└── queries/
    ├── user.postgres.go        # Requêtes utilisateurs
    └── permissions.postgres.go # Requêtes permissions
```

### **Queries SQL Natives**

```go
// queries/user.postgres.go
var UserQueries = struct {
    GetByIdentifiant string
    GetPermissions   string
}{
    GetByIdentifiant: `
        SELECT
            u.id, u.identifiant, u.nom, u.prenoms,
            u.password_hash, u.salt, u.est_admin,
            u.est_medecin, u.statut, u.etablissement_id,
            e.code_etablissement
        FROM user_utilisateur u
        JOIN base_etablissement e ON u.etablissement_id = e.id
        WHERE u.identifiant = $1
            AND u.etablissement_id = $2
            AND u.statut = 'actif'
    `,

    GetPermissions: `
        -- Modules complets
        SELECT 'module:' || m.code_module as permission
        FROM user_modules um
        JOIN base_module m ON um.module_id = m.id
        WHERE um.utilisateur_id = $1
            AND um.etablissement_id = $2
            AND um.est_active = TRUE
            AND um.acces_toutes_rubriques = TRUE

        UNION

        -- Rubriques spécifiques
        SELECT 'rubrique:' || m.code_module || ':' || r.code_rubrique
        FROM user_modules_rubriques umr
        JOIN base_module m ON umr.module_id = m.id
        JOIN base_rubrique r ON umr.rubrique_id = r.id
        WHERE umr.utilisateur_id = $1
            AND umr.etablissement_id = $2
            AND umr.est_active = TRUE
    `,
}
```

---

## ⚡ Règles de Performance

### **Cache Strategy**

- **Sessions** : TTL 1h, refresh automatique sur activité
- **Permissions** : TTL 1h, invalidation sur modification
- **Établissement** : Cache infini (données immuables)

### **Optimisations Redis**

```go
// Utilisation de PIPELINE pour opérations multiples
func (s *SessionService) CreateSession(ctx context.Context, data SessionData) error {
    pipe := s.redis.Pipeline()

    // Session
    sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s",
        data.EstablishmentCode, data.Token)
    pipe.HMSet(ctx, sessionKey, data.ToMap())
    pipe.Expire(ctx, sessionKey, time.Hour)

    // Index utilisateur
    indexKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s",
        data.EstablishmentCode, data.UserID)
    pipe.SAdd(ctx, indexKey, data.Token)
    pipe.Expire(ctx, indexKey, time.Hour)

    _, err := pipe.Exec(ctx)
    return err
}
```

---

## 🛡️ Sécurité

### **Hashage Mots de Passe**

```go
// Utilisation bcrypt avec salt
func HashPassword(password string) (hash string, salt string, err error) {
    salt = generateRandomSalt(32)
    combined := password + salt
    hashedBytes, err := bcrypt.GenerateFromPassword([]byte(combined), 12)
    return string(hashedBytes), salt, err
}
```

### **Validation Token**

```go
func ValidateTokenFormat(token string) error {
    // UUID v4 format validation
    _, err := uuid.Parse(token)
    if err != nil {
        return ErrInvalidTokenFormat
    }
    return nil
}
```

### **Rate Limiting Login**

```
Clé : soins_suite_{etablissement}_auth_ratelimit:{identifiant}
TTL : 900s (15 minutes)
Max : 5 tentatives
```

---

## 🔄 Fallback PostgreSQL

### **Stratégie de Continuité**

```go
func (s *SessionService) ValidateSession(ctx context.Context, token string) (*Session, error) {
    // 1. Tentative Redis (< 1ms)
    session, err := s.getSessionFromRedis(ctx, token)
    if err == nil {
        return session, nil
    }

    // 2. Si Redis down, fallback PostgreSQL
    if errors.Is(err, ErrRedisUnavailable) {
        session, err = s.getSessionFromPostgres(ctx, token)
        if err != nil {
            return nil, err
        }

        // 3. Re-sync vers Redis si disponible
        go s.syncSessionToRedis(ctx, session)
        return session, nil
    }

    return nil, ErrSessionNotFound
}
```

---

## 📊 Métriques & Monitoring

### **Logs Structurés**

```go
// Log connexion réussie
log.Info("auth.login.success",
    "user_id", userID,
    "establishment_code", establishmentCode,
    "client_type", clientType,
    "ip", ipAddress,
)

// Log échec authentification
log.Warn("auth.login.failed",
    "identifiant", identifiant,
    "establishment_code", establishmentCode,
    "reason", "invalid_credentials",
)
```

### **Métriques Prometheus**

- `auth_login_total{status="success|failed"}`
- `auth_session_active_total{establishment}`
- `auth_token_validation_duration_seconds`
- `auth_redis_fallback_total`

---

## ✅ Checklist Implémentation

- [ ] **EstablishmentMiddleware** : Validation code établissement
- [ ] **SessionMiddleware** : Validation token + enrichissement contexte
- [ ] **PermissionMiddleware** : Contrôle permissions modules/rubriques
- [ ] **AuthService** : Login/Logout/Refresh avec Redis
- [ ] **SessionService** : Gestion sessions multi-tenant
- [ ] **PermissionService** : Cache et validation permissions
- [ ] **Fallback PostgreSQL** : Table user_session pour continuité
- [ ] **Rate Limiting** : Protection brute-force
- [ ] **Monitoring** : Logs structurés + métriques

---

**Ces standards garantissent un système d'authentification simple, performant et sécurisé, respectant l'architecture multi-tenant de Soins Suite.**
