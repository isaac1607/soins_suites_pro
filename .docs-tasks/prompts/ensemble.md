# PROMPT TEMPLATE - IMPLÉMENTATION MODULE AUTHENTIFICATION

## Version pour Premier Endpoint (avec contexte complet)

```markdown
# CONTEXTE PROJET

Je travaille sur l'implémentation du module d'authentification de Soins Suite, une application médicale multi-tenant en Go/Gin avec Uber Fx.

# FICHIERS À ANALYSER

Analyse d'abord ces fichiers essentiels avant toute implémentation :

1. **Conventions & Architecture**

   - `.docs-tasks/contexte/rules/arborescence.md` - Structure projet
   - `.docs-tasks/contexte/rules/conventions.md` - Conventions code
   - `.docs-tasks/contexte/rules/conventions_bdd.md` - Conventions BDD

2. **Spécifications Module Auth**

   - `.docs-tasks/contexte/tasks/authentification/a1.specs.md` - Specs endpoints
   - `.docs-tasks/contexte/tasks/authentification/a2.middlewares_specs.md` - Specs middlewares
   - `.docs-tasks/contexte/tasks/authentification/auth-standards.md` - Standards auth

3. **Schémas Base de Données**

   - `database/schemas/00.initial.sql` - Tables établissements/licences
   - `database/schemas/02.user.sql` - Tables utilisateurs/permissions

4. **Schémas Redis**

   - `database/redis-schemas/00.core-patterns.md` - Patterns Redis
   - `database/redis-schemas/02.auth-keys.md` - Clés auth Redis

5. **Code Existant**
   - `internal/shared/middleware/logging/logger.go`
   - `internal/shared/middleware/authentication/establishment.middleware.go`
   - `internal/shared/middleware/authentication/license.middleware.go`
   - `internal/infrastructure/database/postgres/client.go`
   - `internal/infrastructure/database/redis/client.go`

# TÂCHE

Implémenter l'endpoint **POST /api/v1/auth/login** selon les spécifications.

# LIVRABLES REQUIS

1. **Module Fx** : `internal/modules/auth/auth.module.go`
2. **Controller** : `internal/modules/auth/controllers/auth.controller.go`
3. **Service** : `internal/modules/auth/services/auth.service.go`
4. **DTOs** : `internal/modules/auth/dto/auth.dto.go`
5. **Queries SQL** : `internal/modules/auth/queries/user.postgres.go`
6. **Service Session** : `internal/modules/auth/services/session.service.go`
7. **Service Permission** : `internal/modules/auth/services/permission.service.go`

# CONTRAINTES TECHNIQUES

- SQL natif PostgreSQL (pas d'ORM)
- Redis pour sessions et cache permissions
- Tokens UUID opaques (pas de JWT)
- Hashage : réutilise les fonctions définies dans `internal/shared/utils/crypto.go`
- Rate limiting sur login (5 tentatives/15 min)
- Response avec permissions structurées

# VÉRIFICATIONS

Avant de coder, vérifie que tu as bien compris :

1. Le format exact des réponses API
2. La structure des permissions (modules vs rubriques)
3. Les clés Redis multi-tenant
4. Les requêtes SQL avec CTEs pour les permissions
```

---

## Version pour Endpoints Suivants (après /clear)

```markdown
# CONTEXTE RAPIDE

Implémentation du module authentification Soins Suite - API Go/Gin avec Uber Fx, PostgreSQL, Redis.

# FICHIERS ESSENTIELS À REVOIR

1. **Specs** : `.docs-tasks/contexte/tasks/authentification/a1.specs.md`
2. **Schéma User** : `database/schemas/02.user.sql`
3. **Redis Auth** : `database/redis-schemas/02.auth-keys.md`
4. **Module existant** : `internal/modules/auth/auth.module.go`

# TÂCHE

Implémenter l'endpoint **[NOM_ENDPOINT]** selon les spécifications.

[Pour POST /api/v1/auth/logout]

- Révocation session Redis
- Ajout à blacklist
- Suppression index utilisateur

[Pour POST /api/v1/auth/refresh]

- Validation token actuel
- Génération nouveau token
- Migration session Redis

[Pour GET /api/v1/auth/me]

- Récupération infos utilisateur
- Retour permissions complètes
- Info session courante

# FICHIERS À CRÉER/MODIFIER

[Lister spécifiquement les fichiers concernés]

# POINTS D'ATTENTION

- Réutiliser les services existants (SessionService, PermissionService)
- Respecter les patterns établis dans le code existant
- Cohérence avec les autres endpoints déjà implémentés
```

---

## Version pour Middlewares

```markdown
# CONTEXTE

Implémentation des middlewares d'authentification pour Soins Suite.

# FICHIERS À ANALYSER

1. **Specs Middlewares** : `.docs-tasks/contexte/tasks/authentification/a2.middlewares_specs.md`
2. **Redis Patterns** : `database/redis-schemas/02.auth-keys.md`
3. **Middlewares existants** : `internal/shared/middleware/authentication/`

# TÂCHE

Implémenter le middleware **[NOM_MIDDLEWARE]**.

[Pour SessionMiddleware]

- Validation token Bearer
- Vérification blacklist
- Enrichissement contexte Gin
- Update last_activity

[Pour PermissionMiddleware]

- Vérification permissions module/rubrique
- Support accès complet vs restreint
- Cache Redis SET

[Pour RateLimitMiddleware]

- Protection par endpoint/user
- Sliding window Redis
- Headers informatifs

# EMPLACEMENT

`internal/shared/middleware/auth/[nom].middleware.go`

# INTÉGRATION

Montrer comment l'intégrer dans les routes avec les autres middlewares.
```

---

## Checklist de Progression

```markdown
## 📋 PROGRESSION IMPLÉMENTATION AUTH

### Endpoints API

- [ ] POST /api/v1/auth/login
- [ ] POST /api/v1/auth/logout
- [ ] POST /api/v1/auth/refresh
- [ ] GET /api/v1/auth/me

### Middlewares

- [ ] SessionMiddleware
- [ ] PermissionMiddleware
- [ ] RateLimitMiddleware
- [ ] ClientTypeMiddleware

### Services Core

- [ ] AuthService
- [ ] SessionService
- [ ] PermissionService
- [ ] RateLimitService

### Intégration

- [ ] Module Fx complet
- [ ] Routes configurées
- [ ] Tests unitaires
- [ ] Documentation API
```

---

## Instructions d'Usage

### Pour le PREMIER endpoint (login) :

1. Utiliser la version complète du prompt
2. Demander l'implémentation complète avec tous les services
3. Vérifier la cohérence globale

### Pour les endpoints SUIVANTS :

1. Faire `/clear` dans Claude Code
2. Utiliser la version simplifiée du prompt
3. Adapter la section TÂCHE selon l'endpoint
4. Référencer les services existants

### Pour les MIDDLEWARES :

1. Faire `/clear` si nécessaire
2. Utiliser la version middleware du prompt
3. Un middleware à la fois
4. Tester l'intégration après chaque ajout

### Ordre d'implémentation recommandé :

1. **Login** (base complète)
2. **SessionMiddleware** (pour tester login)
3. **Me** (utilise session)
4. **PermissionMiddleware** (pour routes protégées)
5. **Logout** (révocation)
6. **Refresh** (renouvellement)
7. **RateLimitMiddleware** (protection finale)

---

## Exemple de Commande Complète

```bash
# Premier endpoint
"Implémente l'endpoint POST /api/v1/auth/login en suivant strictement les specs.
Crée tous les fichiers nécessaires du module auth."

# Endpoints suivants (après /clear)
"Implémente l'endpoint POST /api/v1/auth/logout en réutilisant
SessionService et les patterns établis."

# Middleware
"Implémente SessionMiddleware selon les specs, avec validation
token et enrichissement contexte Gin."
```
