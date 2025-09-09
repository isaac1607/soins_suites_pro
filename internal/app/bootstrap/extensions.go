package bootstrap

import (
	"context"
	"fmt"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
)

// ExtensionManager gère la création des extensions PostgreSQL requises
// Extensions supportées : uuid-ossp et pg_trgm
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

// EnsureRequiredExtensions crée toutes les extensions requises
func (em *ExtensionManager) EnsureRequiredExtensions(ctx context.Context) error {
	fmt.Printf("[EXTENSIONS] Création des extensions PostgreSQL requises\n")

	// Créer uuid-ossp
	if err := em.ensureExtension(ctx, "uuid-ossp"); err != nil {
		return fmt.Errorf("failed to ensure uuid-ossp extension: %w", err)
	}

	// Créer pg_trgm
	if err := em.ensureExtension(ctx, "pg_trgm"); err != nil {
		return fmt.Errorf("failed to ensure pg_trgm extension: %w", err)
	}

	fmt.Printf("[EXTENSIONS] ✅ Toutes les extensions requises sont installées\n")
	return nil
}

// EnsureUUIDExtension crée l'extension uuid-ossp si elle n'existe pas
// Méthode conservée pour compatibilité
func (em *ExtensionManager) EnsureUUIDExtension(ctx context.Context) error {
	return em.EnsureRequiredExtensions(ctx)
}

// ensureExtension crée une extension PostgreSQL spécifique si elle n'existe pas
func (em *ExtensionManager) ensureExtension(ctx context.Context, extensionName string) error {
	fmt.Printf("[EXTENSIONS] Vérification et création extension %s\n", extensionName)

	// Vérifier si l'extension existe déjà
	exists, err := em.checkExtensionExists(ctx, extensionName)
	if err != nil {
		return fmt.Errorf("failed to check extension %s: %w", extensionName, err)
	}

	if exists {
		fmt.Printf("[EXTENSIONS] ✅ Extension %s déjà installée\n", extensionName)
		return nil
	}

	// Création de l'extension
	fmt.Printf("[EXTENSIONS] 🔧 Création extension %s...\n", extensionName)

	query := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, extensionName)
	_, err = em.pgClient.Pool().Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create extension %s: %w", extensionName, err)
	}

	// Vérification post-création
	exists, err = em.checkExtensionExists(ctx, extensionName)
	if err != nil {
		return fmt.Errorf("failed to verify extension %s after creation: %w", extensionName, err)
	}

	if !exists {
		return fmt.Errorf("extension %s was not created successfully", extensionName)
	}

	fmt.Printf("[EXTENSIONS] ✅ Extension %s créée avec succès\n", extensionName)
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
