package atlas

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"
)

// Module Fx pour Atlas avec injection automatique
var Module = fx.Options(
	// Providers
	fx.Provide(NewAtlasLogger),
	fx.Provide(NewAtlasClient),
	fx.Provide(NewAtlasMigrationManager),
	fx.Provide(NewAtlasRollbackManager),
	fx.Provide(NewAtlasSchemaManager),
	fx.Provide(NewAtlasService),

	// Lifecycle hooks
	fx.Invoke(RegisterAtlasLifecycle),
)

// NewAtlasLogger provider pour le logger Atlas compatible Gin
func NewAtlasLogger() AtlasLogger {
	return NewGinCompatibleLogger()
}

// NewAtlasClient provider pour le client Atlas
func NewAtlasClient(config *AtlasConfig) *Client {
	return NewClient(
		config.WorkingDir,
		config.GetAbsoluteConfigPath(),
		config.Environment,
	)
}

// NewAtlasMigrationManager provider pour le gestionnaire de migrations
func NewAtlasMigrationManager(client *Client, logger AtlasLogger) *MigrationManager {
	return NewMigrationManager(client, logger)
}

// NewAtlasRollbackManager provider pour le gestionnaire de rollback
func NewAtlasRollbackManager(client *Client, logger AtlasLogger) *RollbackManager {
	return NewRollbackManager(client, logger)
}

// NewAtlasSchemaManager provider pour le gestionnaire de schémas
func NewAtlasSchemaManager(config *AtlasConfig, logger AtlasLogger) *SchemaManager {
	// Debug: vérifier les variables d'environnement
	fmt.Printf("[DEBUG] os.Getenv(ATLAS_DEV_DATABASE_URL): '%s'\n", os.Getenv("ATLAS_DEV_DATABASE_URL"))
	fmt.Printf("[DEBUG] config.DevDatabaseURL: '%s'\n", config.DevDatabaseURL)
	fmt.Printf("[DEBUG] config.DatabaseURL: '%s'\n", config.DatabaseURL)

	devURL := os.Getenv("ATLAS_DEV_DATABASE_URL")
	if devURL == "" {
		devURL = config.DevDatabaseURL
	}

	schemaConfig := &SchemaManagerConfig{
		WorkingDir:     config.WorkingDir,
		SchemasPath:    config.GetAbsoluteSchemasPath(),
		MigrationsPath: config.GetAbsoluteMigrationsPath(),
		DatabaseURL:    config.DatabaseURL,
		DevDatabaseURL: devURL,
		Environment:    config.Environment,
		Timeout:        config.MigrationTimeout,
	}

	fmt.Printf("[DEBUG] schemaConfig.DevDatabaseURL final: '%s'\n", schemaConfig.DevDatabaseURL)

	return NewSchemaManager(schemaConfig, logger)
}

// AtlasService service principal qui orchestre tous les managers Atlas
type AtlasService struct {
	config       *AtlasConfig
	client       *Client
	migrationMgr *MigrationManager
	rollbackMgr  *RollbackManager
	schemaMgr    *SchemaManager
	logger       AtlasLogger
}

// NewAtlasService crée le service principal Atlas
func NewAtlasService(
	config *AtlasConfig,
	client *Client,
	migrationMgr *MigrationManager,
	rollbackMgr *RollbackManager,
	schemaMgr *SchemaManager,
	logger AtlasLogger,
) *AtlasService {
	return &AtlasService{
		config:       config,
		client:       client,
		migrationMgr: migrationMgr,
		rollbackMgr:  rollbackMgr,
		schemaMgr:    schemaMgr,
		logger:       logger,
	}
}

// MigrationManager retourne le gestionnaire de migrations
func (s *AtlasService) MigrationManager() *MigrationManager {
	return s.migrationMgr
}

// RollbackManager retourne le gestionnaire de rollback
func (s *AtlasService) RollbackManager() *RollbackManager {
	return s.rollbackMgr
}

// SchemaManager retourne le gestionnaire de schémas
func (s *AtlasService) SchemaManager() *SchemaManager {
	return s.schemaMgr
}

// IsEnabled retourne si Atlas est activé
func (s *AtlasService) IsEnabled() bool {
	return s.config.Enabled
}

// GetMigrationStatus retourne le statut des migrations via le MigrationManager
func (s *AtlasService) GetMigrationStatus(ctx context.Context) ([]MigrationStatus, error) {
	return s.migrationMgr.GetMigrationHistory(ctx)
}

// ApplyMigrations applique les migrations en attente via le MigrationManager
func (s *AtlasService) ApplyMigrations(ctx context.Context) error {
	return s.migrationMgr.ApplyMigrations(ctx)
}

// RegisterAtlasLifecycle enregistre les hooks de lifecycle pour Atlas
func RegisterAtlasLifecycle(
	lc fx.Lifecycle,
	config *AtlasConfig,
	client *Client,
	logger AtlasLogger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !config.Enabled {
				logger.Info("Atlas désactivé - skip validation")
				return nil
			}

			logger.Info("Validation Atlas au démarrage")

			// Vérifier qu'Atlas CLI est installé
			if !client.IsInstalled() {
				logger.Error("Atlas CLI non trouvé")
				return ErrAtlasNotInstalled
			}

			// Valider la configuration
			if err := config.Validate(); err != nil {
				logger.Error("Configuration Atlas invalide", "error", err)
				return fmt.Errorf("configuration Atlas invalide: %w", err)
			}

			// Test de connectivité avec timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			version, err := client.GetVersion(timeoutCtx)
			if err != nil {
				logger.Warn("Impossible de récupérer la version Atlas", "error", err)
			} else {
				logger.Info("Atlas CLI validé", "version", version)
			}

			// Test de ping base de données
			if err := client.Ping(timeoutCtx); err != nil {
				logger.Warn("Test connexion base de données Atlas échoué", "error", err)
				// Ne pas faire échouer le démarrage pour problème de connectivité DB
				// L'application peut démarrer même si Atlas n'arrive pas à se connecter
			} else {
				logger.Info("Connexion base de données Atlas validée")
			}

			logger.Info("Atlas initialisé avec succès")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if !config.Enabled {
				return nil
			}

			logger.Info("Arrêt Atlas")
			return client.Close()
		},
	})
}
