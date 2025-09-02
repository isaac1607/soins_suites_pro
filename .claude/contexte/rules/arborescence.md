# Architecture Go + Uber Fx Finale pour Soins Suite

## 🎯 Arborescence Finale Optimisée

Architecture Domain-Driven Design avec Uber Fx, inspirée de NestJS, optimisée pour Soins Suite.

```
soins-suite-core/
├── cmd/
│   ├── api/
│   │   └── main.go                  # Bootstrap Fx principal
│   ├── migrate/
│   │   └── main.go                  # CLI migrations Atlas
│   └── seed/
│       └── main.go                  # CLI seeding données

├── internal/
│   ├── app/                         # Configuration & Bootstrap Fx
│   │   ├── app.go                   # Application Fx struct
│   │   ├── config.go                # Configuration centralisée
│   │   ├── modules.go               # Assemblage modules Fx
│   │   └── router.go                # Router Gin principal
│   │
│   ├── shared/                      # Code partagé
│   │   ├── constants/               # Constantes globales
│   │   │   ├── permissions.go       # Constantes permissions
│   │   │   ├── roles.go             # Constantes rôles
│   │   │   └── status.go            # Constantes statuts
│   │   ├── types/                   # Types partagés
│   │   │   ├── context.go           # Context enrichi
│   │   │   ├── pagination.go        # Types pagination
│   │   │   └── filter.go            # Types filtres
│   │   ├── utils/                   # Utilitaires
│   │   │   ├── crypto.go            # Utilitaires crypto
│   │   │   ├── validation.go        # Validateurs
│   │   │   └── converter.go         # Convertisseurs
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
│   │   └── errors/                  # Gestion erreurs
│   │       ├── codes.go             # Codes erreur
│   │       ├── errors.go            # Types erreur
│   │       └── handler.go           # Gestionnaire erreurs
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
│   │   │   └── mongodb/
│   │   │       ├── mongodb.module.go # Module Fx MongoDB
│   │   │       ├── client.go        # Client MongoDB
│   │   │       └── collection.go    # Gestionnaire collections
│   │   ├── logger/
│   │   │   ├── logger.module.go     # Module Fx Logger
│   │   │   ├── zap.go               # Implémentation Zap
│   │   │   └── middleware.go        # Middleware logging
│   │   └── http/
│   │       └── client.go            # Client HTTP externe
│   │
│   └── modules/                     # Modules métier organisés par interface
│       ├── auth/                    # Module Authentification (transversal)
│       │   ├── auth.module.go       # Module Fx Auth
│       │   ├── controllers/
│       │   │   ├── auth-back-office.controller.go     # Auth back-office
│       │   │   └── auth-front-office.controller.go  # Auth front-office
│       │   ├── services/
│       │   │   ├── auth.service.go             # Service auth principal
│       │   │   ├── token.service.go            # Gestion tokens
│       │   │   └── session.service.go          # Gestion sessions
│       │   ├── repositories/
│       │   │   ├── user.repository.go          # Repository utilisateurs
│       │   │   └── session.repository.go       # Repository sessions
│       │   ├── dto/
│       │   │   ├── login.dto.go                # DTOs login
│       │   │   └── token.dto.go                # DTOs tokens
│       │   ├── interfaces/
│       │   │   ├── auth.interface.go           # Interfaces auth
│       │   │   └── token.interface.go          # Interfaces tokens
│       │   └── queries/
│       │       ├── user.postgres.go            # Requêtes user PostgreSQL
│       │       └── session.redis.go            # Requêtes session Redis
│       │
│       ├── back-office/             # Modules Back-Office (Administration)
│       │   ├── setup/               # Configuration initiale système
│       │   │   ├── setup.module.go  # Module Fx Setup
│       │   │   ├── controllers/
│       │   │   │   └── setup.controller.go
│       │   │   ├── services/
│       │   │   │   ├── setup.service.go
│       │   │   │   └── bootstrap.service.go
│       │   │   ├── repositories/
│       │   │   │   ├── setup.repository.go
│       │   │   │   └── cache.repository.go
│       │   │   ├── dto/
│       │   │   │   ├── bootstrap.dto.go
│       │   │   │   └── establishment.dto.go
│       │   │   ├── interfaces/
│       │   │   │   └── setup.interface.go
│       │   │   └── queries/
│       │   │       ├── setup.postgres.go
│       │   │       └── cache.redis.go
│       │   │
│       │   ├── users/               # Gestion utilisateurs
│       │   │   ├── users.module.go
│       │   │   ├── controllers/
│       │   │   │   ├── admin.controller.go
│       │   │   │   └── permissions.controller.go
│       │   │   ├── services/
│       │   │   │   ├── user.service.go
│       │   │   │   └── permission.service.go
│       │   │   ├── repositories/
│       │   │   │   ├── user.repository.go
│       │   │   │   └── permission.repository.go
│       │   │   ├── dto/
│       │   │   │   ├── user.dto.go
│       │   │   │   └── permission.dto.go
│       │   │   ├── interfaces/
│       │   │   │   └── user.interface.go
│       │   │   └── queries/
│       │   │       ├── user.postgres.go
│       │   │       └── permission.postgres.go
│       │   │
│       │   ├── establishment/       # Gestion établissements
│       │   │   ├── establishment.module.go
│       │   │   ├── controllers/
│       │   │   │   └── establishment.controller.go
│       │   │   ├── services/
│       │   │   │   └── establishment.service.go
│       │   │   ├── repositories/
│       │   │   │   └── establishment.repository.go
│       │   │   ├── dto/
│       │   │   │   └── establishment.dto.go
│       │   │   ├── interfaces/
│       │   │   │   └── establishment.interface.go
│       │   │   └── queries/
│       │   │       └── establishment.postgres.go
│       │   │
│       │   ├── licenses/            # Gestion licences et modules
│       │   │   ├── licenses.module.go
│       │   │   ├── controllers/
│       │   │   │   └── license.controller.go
│       │   │   ├── services/
│       │   │   │   └── license.service.go
│       │   │   ├── repositories/
│       │   │   │   └── license.repository.go
│       │   │   ├── dto/
│       │   │   │   └── license.dto.go
│       │   │   ├── interfaces/
│       │   │   │   └── license.interface.go
│       │   │   └── queries/
│       │   │       └── license.postgres.go
│       │   │
│       │   └── reporting/           # Rapports et statistiques
│       │       ├── reporting.module.go
│       │       ├── controllers/
│       │       │   ├── dashboard.controller.go
│       │       │   └── analytics.controller.go
│       │       ├── services/
│       │       │   ├── report.service.go
│       │       │   └── analytics.service.go
│       │       ├── repositories/
│       │       │   └── analytics.repository.go
│       │       ├── dto/
│       │       │   └── report.dto.go
│       │       ├── interfaces/
│       │       │   └── report.interface.go
│       │       └── queries/
│       │           ├── dashboard.postgres.go
│       │           └── analytics.postgres.go
│       │
│       └── front-office/            # Modules Front-Office (Métier)
│           ├── patient/             # Gestion patients (module principal)
│           │   ├── patient.module.go # Module Fx Patient
│           │   ├── controllers/
│           │   │   ├── dossier.controller.go     # CRUD dossier patient
│           │   │   ├── consultation.controller.go # Consultations
│           │   │   ├── prescription.controller.go # Ordonnances
│           │   │   └── antecedents.controller.go  # Antécédents médicaux
│           │   ├── services/
│           │   │   ├── patient.service.go         # Service patient principal
│           │   │   ├── consultation.service.go    # Logique consultations
│           │   │   ├── prescription.service.go    # Logique prescriptions
│           │   │   └── medical-history.service.go # Antécédents
│           │   ├── repositories/
│           │   │   ├── patient.repository.go      # Repository patient
│           │   │   ├── consultation.repository.go # Repository consultations
│           │   │   └── prescription.repository.go # Repository prescriptions
│           │   ├── dto/
│           │   │   ├── patient.dto.go            # DTOs patient
│           │   │   ├── consultation.dto.go       # DTOs consultation
│           │   │   └── prescription.dto.go       # DTOs prescription
│           │   ├── interfaces/
│           │   │   ├── patient.interface.go      # Interfaces patient
│           │   │   └── medical.interface.go      # Interfaces médicales
│           │   └── queries/
│           │       ├── patient.postgres.go       # Requêtes patient
│           │       ├── consultation.postgres.go  # Requêtes consultation
│           │       ├── prescription.postgres.go  # Requêtes prescription
│           │       └── medical-forms.mongo.go    # Formulaires dynamiques
│           │
│           ├── medical/             # Modules médicaux avancés
│           │   ├── medical.module.go
│           │   ├── controllers/
│           │   │   ├── diagnostic.controller.go
│           │   │   ├── examination.controller.go
│           │   │   └── laboratory.controller.go
│           │   ├── services/
│           │   │   ├── diagnostic.service.go
│           │   │   ├── examination.service.go
│           │   │   └── laboratory.service.go
│           │   ├── repositories/
│           │   │   ├── diagnostic.repository.go
│           │   │   └── laboratory.repository.go
│           │   ├── dto/
│           │   │   ├── diagnostic.dto.go
│           │   │   └── examination.dto.go
│           │   ├── interfaces/
│           │   │   └── medical.interface.go
│           │   └── queries/
│           │       ├── diagnostic.postgres.go
│           │       ├── examination.postgres.go
│           │       └── lab-results.postgres.go
│           │
│           ├── appointment/         # Gestion rendez-vous
│           │   ├── appointment.module.go
│           │   ├── controllers/
│           │   │   └── appointment.controller.go
│           │   ├── services/
│           │   │   ├── appointment.service.go
│           │   │   └── calendar.service.go
│           │   ├── repositories/
│           │   │   └── appointment.repository.go
│           │   ├── dto/
│           │   │   └── appointment.dto.go
│           │   ├── interfaces/
│           │   │   └── appointment.interface.go
│           │   └── queries/
│           │       └── appointment.postgres.go
│           │
│           └── referentiel/         # Données de référence métier
│               ├── referentiel.module.go
│               ├── controllers/
│               │   ├── medication.controller.go
│               │   └── diagnosis.controller.go
│               ├── services/
│               │   ├── medication.service.go
│               │   └── diagnosis.service.go
│               ├── repositories/
│               │   ├── medication.repository.go
│               │   └── diagnosis.repository.go
│               ├── dto/
│               │   └── referentiel.dto.go
│               ├── interfaces/
│               │   └── referentiel.interface.go
│               └── queries/
│                   ├── medication.postgres.go
│                   └── diagnosis.postgres.go

├── database/                        # Infrastructure DB (externe)
│   ├── atlas.hcl                    # Configuration Atlas
│   ├── migrations/                  # Migrations SQL
│   ├── schemas/                     # Schémas SQL organisés
│   └── seeds/                       # Données initiales

├── configs/                         # Configuration par environnement
├── docs/                           # Documentation essentielle
├── scripts/                        # Scripts utilitaires
└── deployments/                    # Configuration déploiement
```

## 📝 Conventions de Nommage des Modules

### ✅ **Nomenclature des Fichiers Modules Fx**

**Format :** `nom-dossier.module.go`

| Dossier          | Fichier Module            | Description                |
| ---------------- | ------------------------- | -------------------------- |
| `database/`      | `database.module.go`      | Module Fx global database  |
| `postgres/`      | `postgres.module.go`      | Module Fx PostgreSQL       |
| `redis/`         | `redis.module.go`         | Module Fx Redis            |
| `auth/`          | `auth.module.go`          | Module Fx authentification |
| `setup/`         | `setup.module.go`         | Module Fx setup            |
| `patient/`       | `patient.module.go`       | Module Fx patient          |
| `establishment/` | `establishment.module.go` | Module Fx établissement    |

### ✅ **Avantages de cette Convention**

1. **Clarté** : Identification immédiate des modules Fx
2. **Cohérence** : Standard uniforme dans toute l'application
3. **Go idiomatique** : Utilisation du trait d'union `-` conforme aux bonnes pratiques
4. **Lisibilité** : Distinction claire modules vs autres fichiers

### ✅ **Template Standard Module Fx**

```go
// internal/modules/auth/auth.module.go
package auth

import (
    "go.uber.org/fx"
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
) {
    // Configuration des routes auth...
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

    // Infrastructure consolidée
    "soins-suite-core/internal/infrastructure/database"
    "soins-suite-core/internal/infrastructure/logger"

    // Modules métier
    "soins-suite-core/internal/modules/auth"
    "soins-suite-core/internal/modules/back-office/setup"
    "soins-suite-core/internal/modules/back-office/users"
    "soins-suite-core/internal/modules/back-office/establishment"
    "soins-suite-core/internal/modules/back-office/licenses"
    "soins-suite-core/internal/modules/back-office/reporting"
    "soins-suite-core/internal/modules/front-office/patient"
    "soins-suite-core/internal/modules/front-office/medical"
    "soins-suite-core/internal/modules/front-office/appointment"
    "soins-suite-core/internal/modules/front-office/referentiel"
)

// AppModule assemble tous les modules Fx de l'application
var AppModule = fx.Options(
    // Infrastructure (PostgreSQL + Redis + MongoDB + Logger)
    database.Module,
    logger.Module,

    // Module authentification (transversal)
    auth.Module,

    // Modules back-office
    setup.Module,
    users.Module,
    establishment.Module,
    licenses.Module,
    reporting.Module,

    // Modules front-office
    patient.Module,
    medical.Module,
    appointment.Module,
    referentiel.Module,

    // Bootstrap application
    fx.Provide(NewGinEngine),
    fx.Provide(NewConfig),
    fx.Invoke(RegisterRoutes),
    fx.Invoke(StartServer),
)
```

## 🚀 Avantages de cette Architecture

### ✅ **Séparation Claire des Responsabilités**

- **auth/** : Transversal, géré une seule fois
- **back-office/** : Administration, configuration, rapports
- **front-office/** : Métier médical, patients, consultations

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
