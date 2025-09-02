# Architecture Go + Uber Fx Finale pour Soins Suite

## 🎯 Arborescence Finale Optimisée

Architecture Domain-Driven Design avec Uber Fx, inspirée de NestJS, optimisée pour Soins Suite.

```
soins-suite-core/
├── cmd/
│   └── api/
│       └── main.go                  # Bootstrap Fx principal

├── internal/
│   ├── app/                         # Configuration & Bootstrap Fx
│   │   ├── bootstrap/               # Système de bootstrap
│   │   │   ├── bootstrap.go         # Bootstrap principal
│   │   │   ├── extensions.go        # Extensions système
│   │   │   ├── migrations.go        # Gestion migrations
│   │   │   ├── migrations_simple.go # Migrations simples
│   │   │   └── seeding.go           # Système de seeding
│   │   ├── config/
│   │   │   └── config.go            # Configuration centralisée
│   │   ├── app.go                   # Application Fx struct
│   │   ├── modules.go               # Assemblage modules Fx
│   │   └── router.go                # Router Gin principal
│   │
│   ├── shared/                      # Code partagé
│   │   ├── constants/               # Constantes globales
│   │   │   └── system.go            # Constantes système
│   │   ├── types/                   # Types partagés (à créer selon besoins)
│   │   ├── utils/                   # Utilitaires
│   │   │   ├── crypto.go            # Utilitaires crypto
│   │   │   ├── license_decoder.go   # Décodeur licences
│   │   │   ├── module_mapper.go     # Mapping modules
│   │   │   └── redis_keys.go        # Génération clés Redis
│   │   ├── middleware/              # Middlewares organisés par domaine
│   │   │   ├── authentication/      # Middlewares d'authentification
│   │   │   │   ├── session.middleware.go      # Validation sessions/tokens
│   │   │   │   └── permission.middleware.go   # Permissions granulaires
│   │   │   ├── security/           # Middlewares de sécurité générale
│   │   │   │   ├── cors.go         # Configuration CORS
│   │   │   │   ├── security.go     # Headers sécurité (CSP, HSTS)
│   │   │   │   ├── recovery.go     # Recovery from panics
│   │   │   │   └── license.middleware.go  # Validation licence globale
│   │   │   ├── logging/            # Middlewares de logging
│   │   │   │   ├── logger.go       # Logger Gin principal
│   │   │   │   └── manual_logging.go  # Logging manuel/custom
│   │   │   ├── validation/         # Middlewares de validation
│   │   │   │   └── context_validator.go  # Validation contexte requests
│   │   │   └── middleware.module.go  # Module Fx principal
│   │   └── errors/                  # Gestion erreurs (à créer selon besoins)
│   │
│   ├── infrastructure/              # Infrastructure
│   │   ├── database/                # Connexions DB
│   │   │   ├── database.module.go   # Module Fx global database
│   │   │   ├── postgres/
│   │   │   │   ├── postgres.module.go # Module Fx PostgreSQL
│   │   │   │   ├── client.go        # Client PostgreSQL optimisé
│   │   │   │   └── transaction.go   # Gestionnaire transactions
│   │   │   ├── redis/
│   │   │   │   ├── redis.module.go  # Module Fx Redis
│   │   │   │   ├── client.go        # Client Redis
│   │   │   │   └── session.go       # Gestionnaire sessions
│   │   │   ├── mongodb/
│   │   │   │   ├── mongodb.module.go # Module Fx MongoDB
│   │   │   │   ├── client.go        # Client MongoDB
│   │   │   │   └── collection.go    # Gestionnaire collections
│   │   │   ├── atlas/               # Atlas migrations
│   │   │   │   ├── atlas.module.go  # Module Fx Atlas
│   │   │   │   ├── client.go        # Client Atlas
│   │   │   │   ├── config.go        # Configuration Atlas
│   │   │   │   ├── errors.go        # Erreurs Atlas
│   │   │   │   ├── logger.go        # Logger Atlas
│   │   │   │   ├── migration.go     # Gestionnaire migrations
│   │   │   │   ├── rollback.go      # Rollback migrations
│   │   │   │   └── schema_manager.go # Gestionnaire schémas
│   │   │   └── seeds/               # Système de seeding
│   │   │       ├── service.go       # Service seeding
│   │   │       ├── types.go         # Types seeding
│   │   │       └── errors.go        # Erreurs seeding
│   │   ├── logger/
│   │   │   ├── logger.module.go     # Module Fx Logger
│   │   │   └── middleware.go        # Middleware logging
│   │   └── http/
│   │       └── client.go            # Client HTTP externe
│   │
│   └── modules/                     # Modules métier organisés par interface
│       ├── auth/                    # Module Authentification (transversal)
│       │   ├── auth.module.go       # Module Fx Auth
│       │   ├── controllers/
│       │   │   └── auth.controller.go    # Contrôleur auth unifié
│       │   ├── services/
│       │   │   ├── auth.service.go             # Service auth principal (utilise queries directement)
│       │   │   ├── token.service.go            # Gestion tokens
│       │   │   └── session.service.go          # Gestion sessions
│       │   ├── dto/
│       │   │   └── login.dto.go                # DTOs login
│       │   └── queries/
│       │       ├── user.postgres.go            # Requêtes user PostgreSQL
│       │       └── session.redis.go            # Requêtes session Redis
│       │
│       ├── system/                  # Module Système (setup et configuration)
│       │   ├── system.module.go     # Module Fx System
│       │   ├── controllers/
│       │   │   └── license.controller.go       # Gestion licences
│       │   ├── services/
│       │   │   ├── license.service.go          # Service licences (utilise queries directement)
│       │   │   └── decoder.service.go          # Décodage licences
│       │   ├── dto/
│       │   │   └── license.dto.go              # DTOs licences
│       │   ├── errors/
│       │   │   └── system.errors.go            # Erreurs système
│       │   └── queries/
│       │       ├── license.postgres.go         # Requêtes licences PostgreSQL
│       │       └── cache.redis.go              # Requêtes cache Redis
│       │
│       ├── patients/                # Module Patients (en construction)
│       │   └── ... (structure à définir)
│       │
│       ├── back-office/             # Modules Back-Office (Administration)
│       │   ├── users/               # Gestion utilisateurs
│       │   │   ├── users.module.go
│       │   │   ├── controllers/
│       │   │   │   └── comptes_permissions/
│       │   │   │       └── comptes_permissions.controller.go
│       │   │   ├── services/
│       │   │   │   └── comptes_permissions/
│       │   │   │       └── comptes_permissions.service.go  # Utilise queries directement
│       │   │   ├── dto/
│       │   │   │   └── comptes_permissions/
│       │   │   │       └── comptes_permissions.dto.go
│       │   │   └── queries/
│       │   │       └── comptes_permissions/
│       │   │           └── comptes_permissions.postgres.go
│       │   │
│       │   └── establishment/       # Gestion établissements
│       │       ├── establishment.module.go
│       │       ├── controllers/
│       │       │   ├── infos_generale/
│       │       │   │   └── infos_generale.controller.go
│       │       │   ├── assurances/
│       │       │   │   └── assurances.controller.go
│       │       │   ├── infrastructures/
│       │       │   │   └── infrastructures.controller.go
│       │       │   ├── prestations/
│       │       │   │   └── prestations.controller.go
│       │       │   └── modules_services/
│       │       │       └── modules_services.controller.go
│       │       ├── services/
│       │       │   ├── infos_generale/
│       │       │   │   └── infos_generale.service.go    # Utilise queries directement
│       │       │   ├── assurances/
│       │       │   │   └── assurances.service.go        # Utilise queries directement
│       │       │   ├── infrastructures/
│       │       │   │   └── infrastructures.service.go   # Utilise queries directement
│       │       │   ├── prestations/
│       │       │   │   └── prestations.service.go       # Utilise queries directement
│       │       │   └── modules_services/
│       │       │       └── modules_services.service.go  # Utilise queries directement
│       │       ├── dto/
│       │       │   ├── infos_generale/
│       │       │   │   └── infos_generale.dto.go
│       │       │   ├── assurances/
│       │       │   │   └── assurances.dto.go
│       │       │   ├── infrastructures/
│       │       │   │   └── infrastructures.dto.go
│       │       │   ├── prestations/
│       │       │   │   └── prestations.dto.go
│       │       │   └── modules_services/
│       │       │       └── modules_services.dto.go
│       │       └── queries/
│       │           ├── infos_generale/
│       │           │   └── infos_generale.postgres.go
│       │           ├── assurances/
│       │           │   └── assurances.postgres.go
│       │           ├── infrastructures/
│       │           │   └── infrastructures.postgres.go
│       │           ├── prestations/
│       │           │   └── prestations.postgres.go
│       │           └── modules_services/
│       │               └── modules_services.postgres.go
│       │
│       └── front-office/            # Modules Front-Office (Métier) - À développer
│           └── ... (structure à définir selon besoins futurs)

├── database/                        # Infrastructure DB (externe)
│   ├── atlas.hcl                    # Configuration Atlas
│   ├── init/                        # Scripts d'initialisation
│   ├── migrations/                  # Migrations SQL
│   │   └── postgresql/              # Migrations PostgreSQL
│   ├── redis-schemas/               # Schémas Redis
│   ├── schemas/                     # Schémas SQL organisés
│   └── seeds/                       # Données initiales

├── bin/                             # Binaires compilés
├── configs/                         # Configuration par environnement
│   ├── .env                         # Variables d'environnement
│   ├── .env.example                 # Exemple de variables
│   ├── development.yaml             # Configuration développement
│   ├── docker.yaml                  # Configuration Docker
│   ├── production.yaml              # Configuration production
│   ├── staging.yaml                 # Configuration staging
│   └── README.md                    # Documentation config
└── scripts/                        # Scripts utilitaires
    ├── atlas-apply.sh               # Application migrations
    ├── atlas-diff.sh                # Différences schémas
    ├── atlas-test.sh                # Tests migrations
    ├── diagnose-migrations.sh       # Diagnostic migrations
    ├── reset-databases.sh           # Reset bases de données
    ├── seed-data.sh                 # Seeding données
    ├── test-cache-validation.sh     # Tests cache
    ├── test-logging.sh              # Tests logging
    └── validator/                   # Scripts de validation
```

## 📝 Architecture Simplifiée pour MVP

### 🚀 **Structure Simplifiée (Sans Repositories/Interfaces)**

Pour un MVP, la structure est allégée :

```
module/
├── module.module.go          # Module Fx
├── controllers/
│   └── feature.controller.go # Contrôleurs
├── services/
│   └── feature.service.go    # Services (utilisent queries directement)
├── dto/
│   └── feature.dto.go        # DTOs
└── queries/
    └── feature.postgres.go   # Requêtes SQL natives
```

### ✅ **Avantages Architecture Simplifiée**

1. **Moins de code** : Suppression de 2 couches (repositories + interfaces)
2. **Plus rapide à développer** : Idéal pour MVP
3. **Plus simple à maintenir** : Moins de fichiers à gérer
4. **Performance** : Une couche d'abstraction en moins
5. **Go idiomatique** : Évite la sur-ingénierie

### ⚠️ **Quand Ajouter Repositories/Interfaces**

- **Repositories** : Quand tu as besoin de tester avec des mocks
- **Interfaces** : Quand tu as plusieurs implémentations d'un même service
- **Pour un MVP** : Ces couches sont souvent du sur-engineering

## 📝 Conventions de Nommage des Modules

### ✅ **Nomenclature des Fichiers Modules Fx**

**Format :** `nom-dossier.module.go`

| Dossier     | Fichier Module       | Description                |
| ----------- | -------------------- | -------------------------- |
| `database/` | `database.module.go` | Module Fx global database  |
| `postgres/` | `postgres.module.go` | Module Fx PostgreSQL       |
| `redis/`    | `redis.module.go`    | Module Fx Redis            |
| `auth/`     | `auth.module.go`     | Module Fx authentification |
| `users/`    | `users.module.go`    | Module Fx utilisateurs     |
| `products/` | `products.module.go` | Module Fx produits         |

### ✅ **Avantages de cette Convention**

1. **Clarté** : Identification immédiate des modules Fx
2. **Cohérence** : Standard uniforme dans toute l'application
3. **Go idiomatique** : Utilisation du trait d'union `-` conforme aux bonnes pratiques
4. **Lisibilité** : Distinction claire modules vs autres fichiers

### ✅ **Template Standard Module Fx Simplifié**

```go
// internal/modules/products/products.module.go
package products

import (
    "go.uber.org/fx"
)

// Module regroupe tous les providers du domaine Products
var Module = fx.Options(
    // Services (utilisent queries directement)
    fx.Provide(NewProductService),

    // Controllers
    fx.Provide(NewProductController),

    // Configuration des routes
    fx.Invoke(RegisterProductRoutes),
)

// RegisterProductRoutes configure les routes Gin pour Products
func RegisterProductRoutes(
    r *gin.Engine,
    ctrl *ProductController,
) {
    api := r.Group("/api/products")
    api.GET("", ctrl.GetProducts)
    api.GET("/:id", ctrl.GetProduct)
    api.POST("", ctrl.CreateProduct)
    api.PUT("/:id", ctrl.UpdateProduct)
    api.DELETE("/:id", ctrl.DeleteProduct)
}
```

## 📝 Structure Standard des Queries

### Nomenclature des Fichiers Queries

```
queries/
├── feature-1.postgres.go           # Requêtes PostgreSQL pour feature-1
├── feature-2.mongo.go              # Requêtes MongoDB pour feature-2
├── feature-3.redis.go              # Requêtes Redis pour feature-3
└── feature-complex.postgres.go     # Requêtes PostgreSQL complexes
```

### Template Standard pour PostgreSQL

```go
// queries/accueil.postgres.go
package queries

// AccueilQueries regroupe toutes les requêtes SQL pour le module Accueil
var AccueilQueries = struct {
    GetSemainesCollecte30Jours   string
    GetSemainesCollecteJours     string
    GetSemaineCollecteEnCours    string
    GetDernierEtatCollecte       string
}{
    /**
     * Récupère les semaines de collecte actives ou futures dans les 30 prochains jours
     * (inclut les semaines en cours et celles qui commencent dans les 30 jours)
     * Paramètres: aucun
     */
    GetSemainesCollecte30Jours: `
        SELECT
            id,
            nom,
            date_debut,
            date_fin,
            experimentation,
            CASE
                WHEN date_debut > CURRENT_DATE THEN 'a_venir'
                WHEN date_debut <= CURRENT_DATE AND date_fin >= CURRENT_DATE THEN 'en_cours'
                ELSE 'termine'
            END as statut,
            CASE
                WHEN date_debut > CURRENT_DATE THEN (date_debut - CURRENT_DATE)
                WHEN date_debut <= CURRENT_DATE AND date_fin >= CURRENT_DATE THEN (date_fin - CURRENT_DATE)
                ELSE 0
            END as jours_restants
        FROM calendrier_semaine_collecte
        WHERE date_fin >= CURRENT_DATE
            AND date_debut <= CURRENT_DATE + INTERVAL '30 days'
        ORDER BY date_debut ASC
    `,

    /**
     * Récupère les semaines de collecte actives ou futures dans les N prochains jours
     * (inclut les semaines en cours et celles qui commencent dans les N jours)
     * Paramètres: $1 = nombre de jours
     */
    GetSemainesCollecteJours: `
        SELECT
            id,
            date_debut,
            date_fin,
            experimentation
        FROM calendrier_semaine_collecte
        WHERE date_fin >= CURRENT_DATE
            AND date_debut <= CURRENT_DATE + INTERVAL '1 day' * $1
        ORDER BY date_debut ASC
    `,

    /**
     * Récupère la semaine de collecte en cours (si elle existe)
     * Paramètres: aucun
     */
    GetSemaineCollecteEnCours: `
        SELECT id
        FROM calendrier_semaine_collecte
        WHERE date_debut <= CURRENT_DATE
            AND date_fin >= CURRENT_DATE
        ORDER BY date_debut DESC
        LIMIT 1
    `,

    /**
     * Récupère le dernier état de collecte d'un agent pour une localité et une semaine spécifique
     * Paramètres: $1 = user_id, $2 = localite_id, $3 = semaine_collecte_id
     */
    GetDernierEtatCollecte: `
        SELECT
            id,
            semaine_collecte_id,
            localite_id,
            user_id,
            type_experimentation,
            dernier_segment,
            dernier_sous_segment,
            dernier_vegetal,
            is_completed,
            created_at,
            updated_at
        FROM collecte_etat_collecte
        WHERE user_id = $1
            AND localite_id = $2
            AND semaine_collecte_id = $3
        ORDER BY updated_at DESC
        LIMIT 1
    `,
}
```

### Template Standard pour MongoDB

```go
// queries/forms.mongo.go
package queries

import (
    "go.mongodb.org/mongo-driver/bson"
)

// FormsQueries regroupe toutes les requêtes MongoDB pour les formulaires dynamiques
var FormsQueries = struct {
    FindFormByType        bson.M
    FindActiveFormsByUser bson.M
    UpdateFormStatus      bson.M
}{
    /**
     * Recherche un formulaire par type
     * Paramètres: type de formulaire
     */
    FindFormByType: bson.M{
        "type":      1, // Paramètre à remplacer
        "is_active": true,
    },

    /**
     * Recherche les formulaires actifs d'un utilisateur
     * Paramètres: user_id, statuts
     */
    FindActiveFormsByUser: bson.M{
        "user_id": 1, // Paramètre à remplacer
        "status": bson.M{
            "$in": []string{}, // Paramètres statuts à remplacer
        },
        "created_at": bson.M{
            "$gte": 1, // Paramètre date à remplacer
        },
    },

    /**
     * Met à jour le statut d'un formulaire
     * Paramètres: form_id, nouveau statut
     */
    UpdateFormStatus: bson.M{
        "$set": bson.M{
            "status":     1, // Paramètre à remplacer
            "updated_at": 1, // Paramètre timestamp à remplacer
        },
    },
}
```

### Template Standard pour Redis

```go
// queries/session.redis.go
package queries

// SessionQueries regroupe toutes les patterns Redis pour les sessions
var SessionQueries = struct {
    SessionKeyPattern     string
    UserSessionsPattern   string
    SessionExpiry         int
    LockKeyPattern        string
}{
    /**
     * Pattern pour les clés de session
     * Format: soins_suite:session:{sessionId}
     */
    SessionKeyPattern: "soins_suite:session:%s",

    /**
     * Pattern pour les sessions utilisateur
     * Format: soins_suite:user_sessions:{userId}
     */
    UserSessionsPattern: "soins_suite:user_sessions:%s",

    /**
     * Durée d'expiration par défaut des sessions (en secondes)
     * 24 heures = 86400 secondes
     */
    SessionExpiry: 86400,

    /**
     * Pattern pour les verrous de session
     * Format: soins_suite:lock:session:{sessionId}
     */
    LockKeyPattern: "soins_suite:lock:session:%s",
}
```

## 🔄 Assemblage Fx Principal

```go
// internal/app/modules.go
package app

import (
    "go.uber.org/fx"

    "soins-suite-core/internal/app/bootstrap"
    "soins-suite-core/internal/app/config"
    "soins-suite-core/internal/infrastructure/database"
    "soins-suite-core/internal/infrastructure/logger"
    "soins-suite-core/internal/modules/auth"
    "soins-suite-core/internal/modules/back-office/establishment"
    "soins-suite-core/internal/modules/back-office/users"
    "soins-suite-core/internal/modules/system"
    "soins-suite-core/internal/shared/middleware"
    "soins-suite-core/internal/shared/utils"
)

// AppModule assemble tous les modules Fx de l'application
var AppModule = fx.Options(
    // Configuration (doit être fournie en premier)
    fx.Provide(config.NewConfig),
    fx.Provide(config.NewDatabaseConfigProvider),
    fx.Provide(config.NewAtlasConfigFromApp),
    fx.Provide(config.NewPostgresConfig),
    fx.Provide(config.NewRedisConfig),
    fx.Provide(config.NewMongoConfig),

    // Utilitaires partagés (après config, avant infrastructure)
    fx.Provide(NewRedisKeyGenerator),

    // Infrastructure
    database.Module,
    logger.Module,

    // Middlewares partagés (après infrastructure, avant modules métier)
    middleware.Module,

    // Modules métier
    system.Module,
    auth.Module,
    establishment.Module,
    users.Module,

    // Bootstrap System - Providers
    fx.Provide(bootstrap.NewBootstrapExtensionManager),
    fx.Provide(bootstrap.NewBootstrapMigrationManager),
    fx.Provide(bootstrap.NewBootstrapSeedingManager),
    fx.Provide(bootstrap.NewBootstrapSystem),

    // Router
    fx.Provide(NewRouter),

    // Application
    fx.Provide(NewApplication),

    // Lifecycle management
    fx.Invoke(bootstrap.RegisterBootstrapLifecycle),
    fx.Invoke((*Application).Start),
)
```

## 🚀 Avantages de cette Architecture

### ✅ **Séparation Claire des Responsabilités**

- **system/** : Gestion des licences et configuration système
- **auth/** : Authentification transversale
- **back-office/** : Administration (établissements, utilisateurs)
- **front-office/** : Métier médical (à développer selon besoins futurs)
- **patients/** : Module patients spécialisé (en construction)

### ✅ **Structure Modulaire Évolutive**

- Chaque module peut évoluer indépendamment
- Organisation des fichiers cohérente dans tous les modules
- Queries organisées par technologie (PostgreSQL, MongoDB, Redis)

### ✅ **Conformité avec NestJS**

- Structure familière pour les développeurs NestJS
- Conventions de nommage cohérentes
- Séparation controllers/services/repositories maintenue

### ✅ **Performance et Maintenabilité**

- Injection Fx optimisée et automatique
- Queries SQL documentées et organisées
- Modules indépendants pour tests et déploiement

Cette architecture finale combine le meilleur de NestJS avec la puissance d'Uber Fx et la performance de Go !

---

## 🎯 EXEMPLE COMPLET : Endpoint Simplifié pour MVP

Voici un exemple complet d'endpoint **GET /api/products** avec l'architecture simplifiée :

### 1. Queries (SQL Natif)

```go
// internal/modules/products/queries/products.postgres.go
package queries

var ProductsQueries = struct {
    GetAll       string
    GetByID      string
    Create       string
    Update       string
    Delete       string
}{
    /**
     * Récupère tous les produits avec pagination
     * Paramètres: $1 = limit, $2 = offset
     */
    GetAll: `
        SELECT
            id,
            name,
            description,
            price,
            created_at,
            updated_at
        FROM products
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `,

    /**
     * Récupère un produit par ID
     * Paramètres: $1 = product_id
     */
    GetByID: `
        SELECT
            id,
            name,
            description,
            price,
            created_at,
            updated_at
        FROM products
        WHERE id = $1 AND deleted_at IS NULL
    `,

    /**
     * Crée un nouveau produit
     * Paramètres: $1 = name, $2 = description, $3 = price
     */
    Create: `
        INSERT INTO products (name, description, price, created_at, updated_at)
        VALUES ($1, $2, $3, NOW(), NOW())
        RETURNING id, name, description, price, created_at, updated_at
    `,
}
```

### 2. DTOs

```go
// internal/modules/products/dto/products.dto.go
package dto

import "time"

type CreateProductRequest struct {
    Name        string  `json:"name" validate:"required,min=2,max=100"`
    Description string  `json:"description" validate:"max=500"`
    Price       float64 `json:"price" validate:"required,min=0"`
}

type ProductResponse struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Price       float64   `json:"price"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ProductsListResponse struct {
    Products []ProductResponse `json:"products"`
    Total    int               `json:"total"`
}
```

### 3. Service (Utilise Queries Directement)

```go
// internal/modules/products/services/products.service.go
package services

import (
    "context"

    "soins-suite-core/internal/infrastructure/database/postgres"
    "soins-suite-core/internal/modules/products/dto"
    "soins-suite-core/internal/modules/products/queries"
)

type ProductService struct {
    db *postgres.Client
}

func NewProductService(db *postgres.Client) *ProductService {
    return &ProductService{db: db}
}

// GetProducts récupère la liste des produits avec pagination
func (s *ProductService) GetProducts(ctx context.Context, limit, offset int) (*dto.ProductsListResponse, error) {
    rows, err := s.db.Query(ctx, queries.ProductsQueries.GetAll, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var products []dto.ProductResponse
    for rows.Next() {
        var product dto.ProductResponse
        err := rows.Scan(
            &product.ID,
            &product.Name,
            &product.Description,
            &product.Price,
            &product.CreatedAt,
            &product.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        products = append(products, product)
    }

    return &dto.ProductsListResponse{
        Products: products,
        Total:    len(products),
    }, nil
}

// CreateProduct crée un nouveau produit
func (s *ProductService) CreateProduct(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
    var product dto.ProductResponse

    err := s.db.QueryRow(
        ctx,
        queries.ProductsQueries.Create,
        req.Name,
        req.Description,
        req.Price,
    ).Scan(
        &product.ID,
        &product.Name,
        &product.Description,
        &product.Price,
        &product.CreatedAt,
        &product.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return &product, nil
}
```

### 4. Controller

```go
// internal/modules/products/controllers/products.controller.go
package controllers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "soins-suite-core/internal/modules/products/dto"
    "soins-suite-core/internal/modules/products/services"
)

type ProductController struct {
    service *services.ProductService
}

func NewProductController(service *services.ProductService) *ProductController {
    return &ProductController{service: service}
}

// GetProducts - GET /api/products
func (c *ProductController) GetProducts(ctx *gin.Context) {
    // Paramètres de pagination
    limit := 20
    offset := 0

    if l := ctx.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
            limit = parsed
        }
    }

    if o := ctx.Query("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Appel du service
    products, err := c.service.GetProducts(ctx.Request.Context(), limit, offset)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch products",
        })
        return
    }

    ctx.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    products,
    })
}

// CreateProduct - POST /api/products
func (c *ProductController) CreateProduct(ctx *gin.Context) {
    var req dto.CreateProductRequest

    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request format",
        })
        return
    }

    product, err := c.service.CreateProduct(ctx.Request.Context(), req)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to create product",
        })
        return
    }

    ctx.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    product,
    })
}
```

### 5. Module Fx

```go
// internal/modules/products/products.module.go
package products

import (
    "go.uber.org/fx"
    "github.com/gin-gonic/gin"

    "soins-suite-core/internal/modules/products/controllers"
    "soins-suite-core/internal/modules/products/services"
)

var Module = fx.Options(
    // Services (injectent le client PostgreSQL centralisé)
    fx.Provide(services.NewProductService),

    // Controllers
    fx.Provide(controllers.NewProductController),

    // Routes
    fx.Invoke(RegisterProductRoutes),
)

func RegisterProductRoutes(r *gin.Engine, ctrl *controllers.ProductController) {
    api := r.Group("/api/products")
    {
        api.GET("", ctrl.GetProducts)
        api.POST("", ctrl.CreateProduct)
    }
}
```

### 🎯 **Résultat**

Avec cette architecture simplifiée :

- **5 fichiers** au lieu de 8-10 (sans repositories/interfaces)
- **Code direct** : Service → Queries → Client PostgreSQL centralisé
- **Infrastructure standardisée** : Utilise `*postgres.Client` selon les conventions
- **Très rapide à développer** pour MVP
- **Facile à maintenir** et comprendre
- **Cohérent** avec le reste du projet Soins Suite

**Endpoint prêt :** `GET /api/products?limit=10&offset=0`

### 📋 **Points Clés Architecture**

✅ **Client PostgreSQL centralisé** : `*postgres.Client` injecté par Uber Fx  
✅ **Méthodes standardisées** : `db.Query()`, `db.QueryRow()` au lieu de `database/sql`  
✅ **Cohérence** : Suit exactement les conventions définies dans `conventions.md`  
✅ **Infrastructure réutilisable** : S'appuie sur l'infrastructure existante de Soins Suite
