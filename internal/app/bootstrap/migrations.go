package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"soins-suite-core/internal/app/config"
	atlasInfra "soins-suite-core/internal/infrastructure/database/atlas"
)

// MigrationManager gère les migrations Atlas avec la logique complète :
// 1. Appliquer migrations en attente en priorité
// 2. Vérifier changements schémas et générer nouvelles migrations si nécessaire
// Version simplifiée mais gardant la logique essentielle
type MigrationManager struct {
	atlasService *atlasInfra.AtlasService
	config       *config.Config
}

// MigrationStatus représente l'état des migrations
type MigrationStatus struct {
	Status               string `json:"status"`          // PENDING, APPLIED, UP_TO_DATE
	CurrentVersion       string `json:"current_version"` // Version actuelle
	ExecutedFiles        int    `json:"executed_files"`  // Nombre de fichiers appliqués
	PendingFiles         int    `json:"pending_files"`   // Nombre de fichiers en attente
	HasPendingMigrations bool   `json:"has_pending_migrations"`
}

// NewMigrationManager crée une nouvelle instance du gestionnaire de migrations
func NewMigrationManager(atlasService *atlasInfra.AtlasService, cfg *config.Config) *MigrationManager {
	return &MigrationManager{
		atlasService: atlasService,
		config:       cfg,
	}
}

// EnsureMigrationsApplied exécute la logique complète :
// 1. Appliquer migrations en attente
// 2. Vérifier et générer nouvelles migrations
func (mm *MigrationManager) EnsureMigrationsApplied(ctx context.Context) error {
	fmt.Printf("[MIGRATIONS] 🔍 Vérification état migrations Atlas\n")

	if !mm.atlasService.IsEnabled() {
		fmt.Printf("[MIGRATIONS] ⚠️  Atlas désactivé - skip migrations\n")
		return nil
	}

	// Étape 1: Vérifier le statut actuel
	status, err := mm.getMigrationStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("[MIGRATIONS] 📊 Statut: %s, Version: %s, Pending: %d, Executed: %d\n",
		status.Status, status.CurrentVersion, status.PendingFiles, status.ExecutedFiles)

	// Étape 2: Appliquer migrations en attente (priorité absolue)
	if status.PendingFiles > 0 {
		fmt.Printf("[MIGRATIONS] 🔄 Application de %d migrations en attente\n", status.PendingFiles)
		if err := mm.applyPendingMigrations(ctx); err != nil {
			return fmt.Errorf("failed to apply pending migrations: %w", err)
		}

		// Re-vérifier le statut après application
		status, err = mm.getMigrationStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get status after applying pending: %w", err)
		}
	}

	// Étape 3: Vérifier changements schémas et générer nouvelles migrations si nécessaire
	if err := mm.checkAndGenerateNewMigrations(ctx, status); err != nil {
		return fmt.Errorf("failed to check and generate new migrations: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Toutes les migrations sont à jour\n")
	return nil
}

// getMigrationStatus récupère le statut des migrations
func (mm *MigrationManager) getMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	// Utiliser directement AtlasService (version simplifiée)
	statuses, err := mm.atlasService.GetMigrationStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Atlas migration status: %w", err)
	}

	// Convertir []MigrationStatus vers notre format simplifié
	return mm.convertAtlasStatus(statuses), nil
}

// convertAtlasStatus convertit []atlasInfra.MigrationStatus vers MigrationStatus
func (mm *MigrationManager) convertAtlasStatus(statuses []atlasInfra.MigrationStatus) *MigrationStatus {
	status := &MigrationStatus{
		Status:         "UP_TO_DATE",
		CurrentVersion: "Aucune migration appliquée",
		ExecutedFiles:  0,
		PendingFiles:   0,
	}

	appliedCount := 0
	pendingCount := 0

	for _, s := range statuses {
		if s.Applied {
			appliedCount++
			status.CurrentVersion = s.Version // Dernière version appliquée
		} else {
			pendingCount++
		}
	}

	status.ExecutedFiles = appliedCount
	status.PendingFiles = pendingCount
	status.HasPendingMigrations = pendingCount > 0

	if pendingCount > 0 {
		status.Status = "PENDING"
	} else if appliedCount > 0 {
		status.Status = "UP_TO_DATE"
	}

	return status
}

// applyPendingMigrations applique toutes les migrations en attente
func (mm *MigrationManager) applyPendingMigrations(ctx context.Context) error {
	// Utiliser directement AtlasService (version simplifiée)
	err := mm.atlasService.ApplyMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply pending migrations: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Migrations en attente appliquées avec succès\n")
	return nil
}

// checkAndGenerateNewMigrations vérifie s'il faut générer de nouvelles migrations
func (mm *MigrationManager) checkAndGenerateNewMigrations(ctx context.Context, currentStatus *MigrationStatus) error {
	fmt.Printf("[MIGRATIONS] 🔍 Vérification changements schémas vs base de données\n")

	// Utiliser le SchemaManager d'AtlasService pour les opérations avancées
	schemaManager := mm.atlasService.SchemaManager()

	// Utiliser DryRun pour détecter les changements potentiels
	changes, err := schemaManager.DryRun(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect schema changes: %w", err)
	}

	// Si aucun changement détecté
	if len(changes) == 0 {
		fmt.Printf("[MIGRATIONS] ✅ Schémas synchronisés - aucune nouvelle migration nécessaire\n")
		return nil
	}

	// Cas spécial : BDD vide mais schémas définis = génération migration initiale
	if currentStatus.Status == "UP_TO_DATE" &&
		(currentStatus.CurrentVersion == "Aucune migration appliquée" || currentStatus.ExecutedFiles == 0) {
		fmt.Printf("[MIGRATIONS] 🔄 BDD vide détectée - génération migration initiale\n")
		return mm.generateInitialMigration(ctx, changes, schemaManager)
	}

	// Des changements détectés - générer nouvelle migration
	fmt.Printf("[MIGRATIONS] 🔄 %d changements détectés - génération nouvelle migration\n", len(changes))
	return mm.generateNewMigration(ctx, changes, schemaManager)
}

// generateInitialMigration génère la migration initiale pour une BDD vide
func (mm *MigrationManager) generateInitialMigration(ctx context.Context, changes []string, schemaManager *atlasInfra.SchemaManager) error {
	migrationName := "initial_schema"
	fmt.Printf("[MIGRATIONS] 🔄 Génération migration initiale: %s\n", migrationName)

	err := schemaManager.GenerateAndApplyMigrations(ctx, migrationName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("[MIGRATIONS] ⚠️  Tables existantes détectées\n")
			fmt.Printf("[MIGRATIONS] 💡 Solution: atlas migrate hash --env %s\n", mm.config.Atlas.Environment)
			return fmt.Errorf("tables existantes - exécuter: atlas migrate hash --env %s", mm.config.Atlas.Environment)
		}
		return fmt.Errorf("failed to generate initial migration: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Migration initiale générée et appliquée: %s\n", migrationName)
	return nil
}

// generateNewMigration génère une nouvelle migration basée sur les changements détectés
func (mm *MigrationManager) generateNewMigration(ctx context.Context, changes []string, schemaManager *atlasInfra.SchemaManager) error {
	// Générer nom de migration intelligent basé sur timestamp
	migrationName := fmt.Sprintf("schema_changes_%d", time.Now().Unix())

	// Afficher aperçu des changements (limité pour éviter spam)
	fmt.Printf("[MIGRATIONS] 📝 Changements détectés:\n")
	for i, change := range changes {
		if i < 3 { // Limiter à 3 changements affichés
			fmt.Printf("[MIGRATIONS]   - %s\n", change)
		}
	}
	if len(changes) > 3 {
		fmt.Printf("[MIGRATIONS]   ... et %d autres changements\n", len(changes)-3)
	}

	fmt.Printf("[MIGRATIONS] 🔄 Génération migration: %s\n", migrationName)

	err := schemaManager.GenerateAndApplyMigrations(ctx, migrationName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("[MIGRATIONS] ⚠️  Tables existantes détectées\n")
			fmt.Printf("[MIGRATIONS] 💡 Solution: atlas migrate hash --env %s\n", mm.config.Atlas.Environment)
			return fmt.Errorf("tables existantes - exécuter: atlas migrate hash --env %s", mm.config.Atlas.Environment)
		}
		return fmt.Errorf("failed to generate new migration: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Nouvelle migration générée et appliquée: %s\n", migrationName)
	return nil
}

// IsAtlasEnabled retourne si Atlas est activé
func (mm *MigrationManager) IsAtlasEnabled() bool {
	return mm.atlasService.IsEnabled()
}
