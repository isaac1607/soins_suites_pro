package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"soins-suite-core/internal/app/config"
	atlasInfra "soins-suite-core/internal/infrastructure/database/atlas"
)

// SimpleMigrationManager reproduction exacte de votre ancien code fonctionnel
type SimpleMigrationManager struct {
	atlasService     *atlasInfra.AtlasService
	config           *config.Config
	extensionManager *ExtensionManager
	mutex            sync.Mutex
	inProgress       bool
}

// NewSimpleMigrationManager crée le gestionnaire simple comme votre ancien projet
func NewSimpleMigrationManager(atlasService *atlasInfra.AtlasService, cfg *config.Config, extensionManager *ExtensionManager) *SimpleMigrationManager {
	return &SimpleMigrationManager{
		atlasService:     atlasService,
		config:           cfg,
		extensionManager: extensionManager,
	}
}

// EnsureMigrationsApplied vérifie l'état et génère automatiquement les migrations nécessaires
func (smm *SimpleMigrationManager) EnsureMigrationsApplied(ctx context.Context) error {
	// Protection concurrence
	smm.mutex.Lock()
	defer smm.mutex.Unlock()

	if smm.inProgress {
		return fmt.Errorf("migration déjà en cours")
	}

	smm.inProgress = true
	defer func() { smm.inProgress = false }()

	fmt.Printf("[MIGRATIONS] 🔍 Vérification état migrations Atlas\n")

	if !smm.atlasService.IsEnabled() {
		fmt.Printf("[MIGRATIONS] ⚠️  Atlas désactivé\n")
		return nil
	}

	// Vérifier si des migrations sont nécessaires
	return smm.checkAndApplyMigrations(ctx)
}

// getSimpleStatus récupération simple du statut comme votre ancien code
func (smm *SimpleMigrationManager) getSimpleStatus(ctx context.Context) (*MigrationStatus, error) {
	schemaManager := smm.atlasService.SchemaManager()
	if schemaManager == nil {
		return nil, fmt.Errorf("SchemaManager non disponible")
	}

	statusOutput, err := schemaManager.GetMigrationStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Parser simple comme votre migration/service.go:246-305
	return smm.parseAtlasStatus(statusOutput), nil
}

// parseAtlasStatus reproduction de votre migration/service.go:246-305
func (smm *SimpleMigrationManager) parseAtlasStatus(atlasOutput string) *MigrationStatus {
	status := &MigrationStatus{
		Status: "UNKNOWN",
	}

	lines := strings.Split(atlasOutput, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Migration Status:") {
			if strings.Contains(line, "PENDING") {
				status.Status = "PENDING"
			} else if strings.Contains(line, "UP TO DATE") || strings.Contains(line, "OK") {
				status.Status = "UP_TO_DATE"
			}
		}

		// Détecter aussi "Already at latest version"
		if strings.Contains(line, "Already at latest version") {
			status.Status = "UP_TO_DATE"
		}

		if strings.Contains(line, "Current Version:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.CurrentVersion = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Executed Files:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &status.ExecutedFiles)
			}
		}

		if strings.Contains(line, "Pending Files:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &status.PendingFiles)
			}
		}
	}

	// Traduction comme votre ancien code
	if status.CurrentVersion == "No migration applied yet" {
		status.CurrentVersion = "Aucune migration appliquée"
	}

	// Si pas de version et pas de pending = UP_TO_DATE
	if status.CurrentVersion == "Aucune migration appliquée" && status.PendingFiles == 0 {
		status.Status = "UP_TO_DATE"
	}

	status.AppliedCount = status.ExecutedFiles
	status.PendingCount = status.PendingFiles
	status.HasPendingMigrations = status.PendingFiles > 0

	return status
}

// checkAndApplyMigrations vérifie d'abord les migrations en attente, puis les changements schémas
func (smm *SimpleMigrationManager) checkAndApplyMigrations(ctx context.Context) error {
	// 1. Vérifier migrations en attente d'abord
	status, err := smm.getSimpleStatus(ctx)
	if err != nil {
		return fmt.Errorf("erreur vérification statut: %w", err)
	}

	fmt.Printf("[MIGRATIONS] 📊 Statut: %s, Version: %s, Pending: %d, Executed: %d\n",
		status.Status, status.CurrentVersion, status.PendingFiles, status.ExecutedFiles)

	// 2. S'il y a des migrations en attente, les appliquer
	if status.PendingFiles > 0 {
		fmt.Printf("[MIGRATIONS] 🔄 Application de %d migrations en attente\n", status.PendingFiles)
		return smm.applyPendingMigrations(ctx)
	}

	// 3. CAS CRITIQUE: Atlas dit "UP_TO_DATE" mais aucune migration appliquée = BDD vide
	if status.Status == "UP_TO_DATE" && (status.CurrentVersion == "Aucune migration appliquée" || status.ExecutedFiles == 0) {
		fmt.Printf("[MIGRATIONS] ⚠️  Détection BDD vide: Atlas à jour mais 0 migrations appliquées\n")
		fmt.Printf("[MIGRATIONS] 🔄 Génération migration initiale obligatoire depuis schémas\n")
		return smm.generateInitialMigration(ctx)
	}

	// 4. TOUJOURS vérifier les changements schémas vs BDD (même si "UP_TO_DATE")
	// Atlas peut dire "UP_TO_DATE" mais les fichiers schémas peuvent avoir changé
	fmt.Printf("[MIGRATIONS] 🔍 Vérification changements schémas vs base de données\n")
	return smm.checkSchemaChanges(ctx)
}

// applyPendingMigrations applique les migrations en attente
func (smm *SimpleMigrationManager) applyPendingMigrations(ctx context.Context) error {
	// 1. Assurer extensions UUID sur TOUTES les bases avant application
	if smm.extensionManager != nil {
		// Extension base principale d'abord
		if err := smm.extensionManager.EnsureMainDatabaseExtension(ctx); err != nil {
			return fmt.Errorf("extension UUID base principale requise: %w", err)
		}
		// Extension base Atlas ensuite
		if err := smm.extensionManager.EnsureAtlasExtensionReady(ctx); err != nil {
			return fmt.Errorf("extension UUID Atlas requise pour application: %w", err)
		}
		fmt.Printf("[MIGRATIONS] ✅ Extensions UUID prêtes sur toutes les bases\n")
	}

	schemaManager := smm.atlasService.SchemaManager()
	if schemaManager == nil {
		return fmt.Errorf("SchemaManager non disponible")
	}

	err := schemaManager.ApplyMigrationsOnly(ctx)
	if err != nil {
		return fmt.Errorf("application migrations en attente échouée: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Migrations en attente appliquées avec succès\n")
	return nil
}

// checkSchemaChanges vérifie automatiquement les changements entre schémas et base
func (smm *SimpleMigrationManager) checkSchemaChanges(ctx context.Context) error {
	// 1. Assurer extensions UUID sur TOUTES les bases avant comparaison
	if smm.extensionManager != nil {
		// Extension base principale d'abord
		if err := smm.extensionManager.EnsureMainDatabaseExtension(ctx); err != nil {
			return fmt.Errorf("extension UUID base principale requise: %w", err)
		}
		// Extension base Atlas ensuite
		if err := smm.extensionManager.EnsureAtlasExtensionReady(ctx); err != nil {
			return fmt.Errorf("extension UUID Atlas requise pour comparaison: %w", err)
		}
		fmt.Printf("[MIGRATIONS] ✅ Extensions UUID prêtes pour comparaison schémas\n")
	}

	schemaManager := smm.atlasService.SchemaManager()
	if schemaManager == nil {
		return fmt.Errorf("SchemaManager non disponible")
	}

	// 2. Utiliser DryRun pour détecter les changements sans appliquer
	changes, err := schemaManager.DryRun(ctx)
	if err != nil {
		return fmt.Errorf("détection changements schémas échouée: %w", err)
	}

	// 2. Si aucun changement détecté
	if len(changes) == 0 {
		fmt.Printf("[MIGRATIONS] ✅ Schémas synchronisés - aucune migration nécessaire\n")
		return nil
	}

	// 3. Des changements détectés - générer et appliquer automatiquement
	fmt.Printf("[MIGRATIONS] 🔄 %d changements détectés - génération migration automatique\n", len(changes))
	for i, change := range changes {
		if i < 3 { // Afficher max 3 changements pour éviter spam
			fmt.Printf("[MIGRATIONS]   - %s\n", change)
		}
	}
	if len(changes) > 3 {
		fmt.Printf("[MIGRATIONS]   ... et %d autres changements\n", len(changes)-3)
	}

	// 4. Re-vérifier extensions avant génération (opération critique)
	if smm.extensionManager != nil {
		// Double vérification base principale + Atlas
		if err := smm.extensionManager.EnsureMainDatabaseExtension(ctx); err != nil {
			return fmt.Errorf("extension UUID base principale requise: %w", err)
		}
		if err := smm.extensionManager.EnsureAtlasExtensionReady(ctx); err != nil {
			return fmt.Errorf("extension UUID Atlas requise pour génération: %w", err)
		}
	}

	// 5. Générer migration avec nom intelligent basé sur les changements
	migrationName := smm.generateMigrationName(changes)
	fmt.Printf("[MIGRATIONS] 🔄 Génération migration: %s\n", migrationName)

	err = schemaManager.GenerateAndApplyMigrations(ctx, migrationName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("[MIGRATIONS] ⚠️  Tables existantes détectées\n")
			fmt.Printf("[MIGRATIONS] 💡 Solution: atlas migrate hash --env development\n")
			return fmt.Errorf("tables existantes - exécuter: atlas migrate hash --env development")
		}
		return fmt.Errorf("génération migration automatique échouée: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Migration automatique générée et appliquée: %s\n", migrationName)
	return nil
}

// generateInitialMigration génère et applique la migration initiale depuis les schémas
func (smm *SimpleMigrationManager) generateInitialMigration(ctx context.Context) error {
	// 1. Assurer extensions UUID sur TOUTES les bases avant génération initiale
	if smm.extensionManager != nil {
		// Extension base principale d'abord
		if err := smm.extensionManager.EnsureMainDatabaseExtension(ctx); err != nil {
			return fmt.Errorf("extension UUID base principale requise: %w", err)
		}
		// Extension base Atlas ensuite
		if err := smm.extensionManager.EnsureAtlasExtensionReady(ctx); err != nil {
			return fmt.Errorf("extension UUID Atlas requise pour génération: %w", err)
		}
		fmt.Printf("[MIGRATIONS] ✅ Extensions UUID prêtes pour génération migration initiale\n")
	}

	schemaManager := smm.atlasService.SchemaManager()
	if schemaManager == nil {
		return fmt.Errorf("SchemaManager non disponible")
	}

	// 2. Générer migration initiale avec nom explicite
	migrationName := "initial_schema_from_sql_files"
	fmt.Printf("[MIGRATIONS] 🔄 Génération migration initiale: %s\n", migrationName)

	err := schemaManager.GenerateAndApplyMigrations(ctx, migrationName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("[MIGRATIONS] ⚠️  Tables existantes détectées lors de génération initiale\n")
			fmt.Printf("[MIGRATIONS] 💡 Solution: atlas migrate hash --env development\n")
			return fmt.Errorf("tables existantes - exécuter: atlas migrate hash --env development")
		}
		return fmt.Errorf("génération migration initiale échouée: %w", err)
	}

	fmt.Printf("[MIGRATIONS] ✅ Migration initiale générée et appliquée avec succès: %s\n", migrationName)
	return nil
}

// generateMigrationName génère un nom intelligent basé sur les changements détectés
func (smm *SimpleMigrationManager) generateMigrationName(changes []string) string {
	if len(changes) == 0 {
		return "empty_schema_sync"
	}

	// Analyser les types de changements pour générer un nom descriptif
	hasCreateTable := false
	hasDropTable := false
	hasAlterTable := false
	hasCreateIndex := false
	hasDropIndex := false

	tableNames := make(map[string]bool)

	for _, change := range changes {
		changeLower := strings.ToLower(change)

		if strings.Contains(changeLower, "create table") {
			hasCreateTable = true
			// Extraire nom de table
			if tableName := extractTableName(change, "create table"); tableName != "" {
				tableNames[tableName] = true
			}
		} else if strings.Contains(changeLower, "drop table") {
			hasDropTable = true
			if tableName := extractTableName(change, "drop table"); tableName != "" {
				tableNames[tableName] = true
			}
		} else if strings.Contains(changeLower, "alter table") {
			hasAlterTable = true
			if tableName := extractTableName(change, "alter table"); tableName != "" {
				tableNames[tableName] = true
			}
		} else if strings.Contains(changeLower, "create index") {
			hasCreateIndex = true
		} else if strings.Contains(changeLower, "drop index") {
			hasDropIndex = true
		}
	}

	// Générer nom basé sur les changements détectés
	var nameParts []string

	if hasCreateTable {
		nameParts = append(nameParts, "create_tables")
	}
	if hasDropTable {
		nameParts = append(nameParts, "drop_tables")
	}
	if hasAlterTable {
		nameParts = append(nameParts, "alter_tables")
	}
	if hasCreateIndex {
		nameParts = append(nameParts, "create_indexes")
	}
	if hasDropIndex {
		nameParts = append(nameParts, "drop_indexes")
	}

	// Si aucun pattern reconnu, utiliser générique
	if len(nameParts) == 0 {
		nameParts = append(nameParts, "schema_changes")
	}

	// Ajouter nom de table principale si une seule table affectée
	if len(tableNames) == 1 {
		for tableName := range tableNames {
			nameParts = append(nameParts, tableName)
			break
		}
	}

	// Limiter à 3 parties max et ajouter nombre de changements
	if len(nameParts) > 3 {
		nameParts = nameParts[:3]
	}

	baseName := strings.Join(nameParts, "_")
	return fmt.Sprintf("%s_%d_changes", baseName, len(changes))
}

// extractTableName extrait le nom de table d'une commande SQL
func extractTableName(sqlCommand, prefix string) string {
	sqlLower := strings.ToLower(sqlCommand)
	prefixLower := strings.ToLower(prefix)

	// Trouver la position après le préfixe
	index := strings.Index(sqlLower, prefixLower)
	if index == -1 {
		return ""
	}

	// Extraire la partie après le préfixe
	afterPrefix := strings.TrimSpace(sqlCommand[index+len(prefix):])

	// Prendre le premier mot (nom de table)
	parts := strings.Fields(afterPrefix)
	if len(parts) == 0 {
		return ""
	}

	tableName := parts[0]

	// Nettoyer les caractères spéciaux
	tableName = strings.Trim(tableName, "\"'`(")

	// Enlever préfixe schema si présent (ex: public.table_name -> table_name)
	if dotIndex := strings.LastIndex(tableName, "."); dotIndex != -1 {
		tableName = tableName[dotIndex+1:]
	}

	return tableName
}
