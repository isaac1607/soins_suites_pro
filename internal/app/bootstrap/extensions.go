package bootstrap

import (
	"context"
	"fmt"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
)

// ExtensionManager gère la création des extensions PostgreSQL requises
// Version simplifiée : focus uniquement sur l'extension uuid-ossp de la base principale
type ExtensionManager struct {
	pgClient *postgres.Client
	config   *config.Config
}

// NewExtensionManager crée une nouvelle instance du gestionnaire d'extensions
func NewExtensionManager(pgClient *postgres.Client, cfg *config.Config) *ExtensionManager {
	return &ExtensionManager{
		pgClient: pgClient,
		config:   cfg,
	}
}

// EnsureUUIDExtension crée l'extension uuid-ossp si elle n'existe pas
// Approche simplifiée : une seule méthode, une seule base de données
func (em *ExtensionManager) EnsureUUIDExtension(ctx context.Context) error {
	fmt.Printf("[EXTENSIONS] Vérification et création extension uuid-ossp\n")

	// Vérifier si l'extension existe déjà
	exists, err := em.checkExtensionExists(ctx, "uuid-ossp")
	if err != nil {
		return fmt.Errorf("failed to check extension uuid-ossp: %w", err)
	}

	if exists {
		fmt.Printf("[EXTENSIONS] ✅ Extension uuid-ossp déjà installée\n")
		return nil
	}

	// Création de l'extension
	fmt.Printf("[EXTENSIONS] 🔧 Création extension uuid-ossp...\n")

	query := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`
	_, err = em.pgClient.Pool().Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create extension uuid-ossp: %w", err)
	}

	// Vérification post-création
	exists, err = em.checkExtensionExists(ctx, "uuid-ossp")
	if err != nil {
		return fmt.Errorf("failed to verify extension uuid-ossp after creation: %w", err)
	}

	if !exists {
		return fmt.Errorf("extension uuid-ossp was not created successfully")
	}

	fmt.Printf("[EXTENSIONS] ✅ Extension uuid-ossp créée avec succès\n")
	return nil
}

// checkExtensionExists vérifie si une extension PostgreSQL existe
func (em *ExtensionManager) checkExtensionExists(ctx context.Context, extensionName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pg_extension 
			WHERE extname = $1
		)
	`

	var exists bool
	err := em.pgClient.Pool().QueryRow(ctx, query, extensionName).Scan(&exists)
	return exists, err
}

// GetExtensionInfo retourne les informations de l'extension (optionnel, pour debug)
func (em *ExtensionManager) GetExtensionInfo(ctx context.Context, extensionName string) (map[string]string, error) {
	query := `
		SELECT 
			extname,
			extversion,
			nspname as schema_name
		FROM pg_extension e
		JOIN pg_namespace n ON e.extnamespace = n.oid
		WHERE extname = $1
	`

	row := em.pgClient.Pool().QueryRow(ctx, query, extensionName)

	var name, version, schema string
	err := row.Scan(&name, &version, &schema)

	if err != nil {
		return nil, fmt.Errorf("failed to get extension info for %s: %w", extensionName, err)
	}

	return map[string]string{
		"name":    name,
		"version": version,
		"schema":  schema,
	}, nil
}
