# CONTEXTE PROJET

Je travaille sur l'implémentation du module Gestion des utilisateurs destinées aux administrateurs du Back Office de Soins Suite, une application médicale multi-tenant avec le Backend en Go/Gin avec Uber Fx.

# FICHIERS À ANALYSER

Analyse d'abord ces fichiers essentiels avant toute implémentation :

1. **Conventions & Architecture**

   - `.docs-tasks/contexte/rules/arborescence.md` - Structure projet
   - `.docs-tasks/contexte/rules/conventions.md` - Conventions code
   - `.docs-tasks/contexte/rules/conventions_bdd.md` - Conventions BDD

2. **Spécifications Module Gestion des Utilisateurs**

   - `.docs-tasks/contexte/tasks/back_gestion_users/u1.specs.md` - Specs endpoints Rubrique

3. **Schémas Base de Données**

   - `database/schemas/00.initial.sql` - Tables établissements/licences/users
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

Implémenter l'endpoint **POST /api/v1/back-office/users** (Voir 🔧 US-GU-003 : Création d'un Utilisateur)selon les spécifications.
