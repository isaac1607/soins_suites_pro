package atlas

import "errors"

// Erreurs Atlas centralisées
var (
	ErrAtlasNotInstalled       = errors.New("Atlas CLI n'est pas installé ou accessible")
	ErrMissingDatabaseURL      = errors.New("URL de base de données manquante")
	ErrMissingDevDatabaseURL   = errors.New("URL de base de données de développement manquante")
	ErrInvalidTimeout          = errors.New("timeout invalide")
	ErrInvalidMigrationTimeout = errors.New("timeout de migration invalide")
	ErrConfigNotFound          = errors.New("fichier de configuration Atlas introuvable")
	ErrSchemasPathNotFound     = errors.New("répertoire de schémas introuvable")
	ErrMigrationsPathNotFound  = errors.New("répertoire de migrations introuvable")
	ErrNoMigrationsToApply     = errors.New("aucune migration à appliquer")
	ErrMigrationFailed         = errors.New("échec de migration")
	ErrRollbackFailed          = errors.New("échec de rollback")
	ErrInvalidVersion          = errors.New("version de migration invalide")
)