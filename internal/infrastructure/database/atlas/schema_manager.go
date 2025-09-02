package atlas

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SchemaManagerConfig configuration pour le gestionnaire de schémas
type SchemaManagerConfig struct {
	WorkingDir     string
	SchemasPath    string
	MigrationsPath string
	DatabaseURL    string
	DevDatabaseURL string
	Environment    string
	Timeout        time.Duration
}

// SchemaManagerLogger interface pour intégration avec système de logging existant
type SchemaManagerLogger interface {
	Info(message string, fields ...interface{})
	Error(message string, fields ...interface{})
	Warn(message string, fields ...interface{})
}

// SchemaManager gestionnaire pour opérations schema-driven Atlas
type SchemaManager struct {
	config *SchemaManagerConfig
	logger SchemaManagerLogger
}

// NewSchemaManager crée une nouvelle instance du gestionnaire de schémas
func NewSchemaManager(config *SchemaManagerConfig, logger SchemaManagerLogger) *SchemaManager {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &SchemaManager{
		config: config,
		logger: logger,
	}
}

// GenerateAndApplyMigrations analyse les schémas, génère et applique les migrations
func (sm *SchemaManager) GenerateAndApplyMigrations(ctx context.Context, migrationName string) error {
	sm.logger.Info("Début du processus migration schema-driven",
		"migration_name", migrationName,
		"schemas_path", sm.config.SchemasPath,
		"migrations_path", sm.config.MigrationsPath,
	)

	// Étape 1: Vérification des prérequis
	if err := sm.validatePrerequisites(); err != nil {
		return fmt.Errorf("validation prérequis échouée: %w", err)
	}

	// Étape 2: Génération de la migration
	migrationGenerated, err := sm.generateMigration(ctx, migrationName)
	if err != nil {
		return fmt.Errorf("génération migration échouée: %w", err)
	}

	if !migrationGenerated {
		sm.logger.Info("Aucune migration nécessaire - schémas à jour")
		return nil
	}

	// Étape 3: Application des migrations
	if err := sm.applyMigrations(ctx); err != nil {
		return fmt.Errorf("application migrations échouée: %w", err)
	}

	sm.logger.Info("Processus migration terminé avec succès", "migration_name", migrationName)
	return nil
}

// GenerateMigrationOnly génère uniquement la migration sans l'appliquer
func (sm *SchemaManager) GenerateMigrationOnly(ctx context.Context, migrationName string) (bool, error) {
	sm.logger.Info("Génération migration uniquement", "migration_name", migrationName)

	if err := sm.validatePrerequisites(); err != nil {
		return false, fmt.Errorf("validation prérequis échouée: %w", err)
	}

	return sm.generateMigration(ctx, migrationName)
}

// ApplyMigrationsOnly applique uniquement les migrations existantes
func (sm *SchemaManager) ApplyMigrationsOnly(ctx context.Context) error {
	sm.logger.Info("Application migrations uniquement")
	return sm.applyMigrations(ctx)
}

// GetMigrationStatus retourne le statut des migrations
func (sm *SchemaManager) GetMigrationStatus(ctx context.Context) (string, error) {
	sm.logger.Info("Récupération statut migrations")

	cmd := exec.CommandContext(ctx, "atlas", "migrate", "status",
		"--dir", fmt.Sprintf("file://%s", sm.config.MigrationsPath),
		"--url", sm.config.DatabaseURL,
	)

	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		sm.logger.Error("Erreur récupération statut migrations", "error", err, "output", string(output))
		return "", fmt.Errorf("impossible de récupérer le statut des migrations: %w", err)
	}

	status := strings.TrimSpace(string(output))
	sm.logger.Info("Statut migrations récupéré", "status", status)
	return status, nil
}

// ValidateSchemas valide la syntaxe des schémas SQL
func (sm *SchemaManager) ValidateSchemas(ctx context.Context) error {
	sm.logger.Info("Validation schémas SQL", "schemas_path", sm.config.SchemasPath)

	cmd := exec.CommandContext(ctx, "atlas", "schema", "inspect",
		"--url", fmt.Sprintf("file://%s", sm.config.SchemasPath),
		"--dev-url", sm.config.DevDatabaseURL,
	)

	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()

	if err := cmd.Run(); err != nil {
		sm.logger.Error("Validation schémas échouée", "error", err, "schemas_path", sm.config.SchemasPath)
		return fmt.Errorf("validation des schémas échouée: %w", err)
	}

	sm.logger.Info("Validation schémas réussie")
	return nil
}

// validatePrerequisites vérifie que les prérequis sont satisfaits
func (sm *SchemaManager) validatePrerequisites() error {
	sm.logger.Info("Vérification prérequis Atlas")

	// Vérifier qu'Atlas CLI est installé
	if !sm.isAtlasInstalled() {
		return fmt.Errorf("Atlas CLI n'est pas installé ou accessible")
	}

	// Vérifier que les chemins existent
	if _, err := os.Stat(sm.config.SchemasPath); os.IsNotExist(err) {
		return fmt.Errorf("répertoire schémas n'existe pas: %s", sm.config.SchemasPath)
	}

	if _, err := os.Stat(sm.config.MigrationsPath); os.IsNotExist(err) {
		// Créer le répertoire migrations s'il n'existe pas
		sm.logger.Info("Création répertoire migrations", "path", sm.config.MigrationsPath)
		if err := os.MkdirAll(sm.config.MigrationsPath, 0755); err != nil {
			return fmt.Errorf("impossible de créer répertoire migrations: %w", err)
		}
	}

	sm.logger.Info("Prérequis validés avec succès")
	return nil
}

// generateMigration génère une migration depuis les schémas
func (sm *SchemaManager) generateMigration(ctx context.Context, migrationName string) (bool, error) {
	sm.logger.Info("Génération migration", "name", migrationName)

	// Debug: Afficher la commande exacte et les variables d'environnement
	cmd := exec.CommandContext(ctx, "atlas", "migrate", "diff", migrationName,
		"--dir", fmt.Sprintf("file://%s", sm.config.MigrationsPath),
		"--to", fmt.Sprintf("file://%s", sm.config.SchemasPath),
		"--dev-url", sm.config.DevDatabaseURL,
	)

	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()

	// Debug: Afficher la commande exacte
	fmt.Printf("[DEBUG] Commande Atlas: %s\n", strings.Join(cmd.Args, " "))
	fmt.Printf("[DEBUG] WorkingDir: %s\n", cmd.Dir)
	fmt.Printf("[DEBUG] DevDatabaseURL: %s\n", sm.config.DevDatabaseURL)
	
	// Debug: Afficher les variables d'environnement Atlas
	for _, env := range cmd.Env {
		if strings.Contains(env, "database_url") || strings.Contains(env, "dev_database_url") {
			fmt.Printf("[DEBUG] Env: %s\n", env)
		}
	}

	// Capturer stdout et stderr
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		sm.logger.Error("Erreur génération migration", "error", err, "output", outputStr)
		fmt.Printf("[DEBUG] Erreur complète: %v\n", err)
		fmt.Printf("[DEBUG] Output complet: %s\n", outputStr)
		return false, fmt.Errorf("génération migration échouée: %w", err)
	}

	// Vérifier si une migration a été générée
	if strings.Contains(outputStr, "is synced with the desired state, no changes to be made") {
		sm.logger.Info("Aucune migration nécessaire - schémas synchronisés")
		return false, nil
	}

	sm.logger.Info("Migration générée avec succès", "name", migrationName, "output", outputStr)
	
	// Post-traitement : Ajouter l'extension UUID si nécessaire
	if err := sm.ensureExtensionInMigration(migrationName); err != nil {
		sm.logger.Error("Erreur ajout extension dans migration", "error", err, "migration", migrationName)
		// Ne pas faire échouer pour cela, mais logger
	} else {
		// Recalculer les checksums Atlas après modification
		if err := sm.recalculateAtlasChecksum(); err != nil {
			sm.logger.Error("Erreur recalcul checksum Atlas", "error", err)
		}
	}
	
	return true, nil
}

// applyMigrations applique toutes les migrations en attente
func (sm *SchemaManager) applyMigrations(ctx context.Context) error {
	sm.logger.Info("Application des migrations", "database_url", sm.maskPassword(sm.config.DatabaseURL))

	cmd := exec.CommandContext(ctx, "atlas", "migrate", "apply",
		"--dir", fmt.Sprintf("file://%s", sm.config.MigrationsPath),
		"--url", sm.config.DatabaseURL,
	)

	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		sm.logger.Error("Erreur application migrations", "error", err, "output", outputStr)
		return fmt.Errorf("application migrations échouée: %w", err)
	}

	sm.logger.Info("Migrations appliquées avec succès", "output", outputStr)
	return nil
}

// isAtlasInstalled vérifie si Atlas CLI est accessible
func (sm *SchemaManager) isAtlasInstalled() bool {
	cmd := exec.Command("atlas", "version")
	return cmd.Run() == nil
}

// buildEnv construit l'environnement pour les commandes Atlas
func (sm *SchemaManager) buildEnv() []string {
	env := os.Environ()

	// Ajouter les variables Atlas requises avec les bons noms
	if sm.config.DatabaseURL != "" {
		env = append(env, fmt.Sprintf("database_url=%s", sm.config.DatabaseURL))
	}
	
	if sm.config.DevDatabaseURL != "" {
		env = append(env, fmt.Sprintf("dev_database_url=%s", sm.config.DevDatabaseURL))
	}

	return env
}

// maskPassword masque le mot de passe dans les URLs pour les logs
func (sm *SchemaManager) maskPassword(url string) string {
	// Simple masquage pour éviter de logger les mots de passe
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) >= 2 {
			// Masquer la partie credentials
			credentialsPart := parts[0]
			if strings.Contains(credentialsPart, ":") {
				credParts := strings.Split(credentialsPart, ":")
				if len(credParts) >= 3 {
					// postgres://user:password@host -> postgres://user:****@host
					credParts[2] = "****"
					parts[0] = strings.Join(credParts, ":")
					return strings.Join(parts, "@")
				}
			}
		}
	}
	return url
}

// GetWorkingDir retourne le répertoire de travail configuré
func (sm *SchemaManager) GetWorkingDir() string {
	return sm.config.WorkingDir
}

// GetSchemasPath retourne le chemin vers les schémas
func (sm *SchemaManager) GetSchemasPath() string {
	return sm.config.SchemasPath
}

// GetMigrationsPath retourne le chemin vers les migrations
func (sm *SchemaManager) GetMigrationsPath() string {
	return sm.config.MigrationsPath
}

// DryRun simule la génération de migrations pour détecter les changements
func (sm *SchemaManager) DryRun(ctx context.Context) ([]string, error) {
	sm.logger.Info("Simulation détection changements schémas (dry-run)")

	if err := sm.validatePrerequisites(); err != nil {
		return nil, fmt.Errorf("validation prérequis échouée: %w", err)
	}

	// Utiliser Atlas schema diff pour comparer l'état actuel avec les schémas sources
	// Spécifier explicitement le schéma dans l'URL source
	fromURL := sm.config.DatabaseURL
	if !strings.Contains(fromURL, "search_path=") {
		separator := "?"
		if strings.Contains(fromURL, "?") {
			separator = "&"
		}
		fromURL = fromURL + separator + "search_path=public"
	}
	
	cmd := exec.CommandContext(ctx, "atlas", "schema", "diff",
		"--from", fromURL,
		"--to", fmt.Sprintf("file://%s", sm.config.SchemasPath),
		"--dev-url", sm.config.DevDatabaseURL,
	)

	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		sm.logger.Error("Erreur schema diff", "error", err, "output", outputStr)
		return nil, fmt.Errorf("schema diff échoué: %w", err)
	}

	// Parser la sortie pour extraire les changements
	changes := sm.parseSchemaChanges(outputStr)
	
	if len(changes) > 0 {
		sm.logger.Info("Changements détectés", "count", len(changes))
	} else {
		sm.logger.Info("Aucun changement détecté - schémas synchronisés")
	}

	return changes, nil
}

// parseSchemaChanges parse la sortie d'Atlas schema diff
func (sm *SchemaManager) parseSchemaChanges(output string) []string {
	if strings.TrimSpace(output) == "" {
		return []string{}
	}

	// Si la sortie contient des indications qu'il n'y a pas de changements
	if strings.Contains(output, "Schemas are synced") ||
	   strings.Contains(output, "No changes") ||
	   strings.Contains(output, "up to date") {
		return []string{}
	}

	// Diviser la sortie en lignes non-vides
	lines := strings.Split(output, "\n")
	changes := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && 
		   !strings.HasPrefix(line, "--") && // Ignorer les commentaires SQL
		   !strings.HasPrefix(line, "/*") {   // Ignorer les commentaires multi-lignes
			changes = append(changes, line)
		}
	}

	return changes
}

// ensureExtensionInMigration ajoute l'extension UUID au début d'une migration si elle utilise uuid_generate_v4()
func (sm *SchemaManager) ensureExtensionInMigration(migrationName string) error {
	// Trouver le fichier de migration le plus récent
	migrationFiles, err := os.ReadDir(sm.config.MigrationsPath)
	if err != nil {
		return fmt.Errorf("lecture répertoire migrations: %w", err)
	}
	
	var targetFile string
	for _, file := range migrationFiles {
		if strings.Contains(file.Name(), migrationName) && strings.HasSuffix(file.Name(), ".sql") {
			targetFile = fmt.Sprintf("%s/%s", sm.config.MigrationsPath, file.Name())
			break
		}
	}
	
	if targetFile == "" {
		return fmt.Errorf("fichier migration non trouvé pour: %s", migrationName)
	}
	
	// Lire le contenu du fichier
	content, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("lecture fichier migration: %w", err)
	}
	
	contentStr := string(content)
	
	// Vérifier si uuid_generate_v4() est utilisé
	if !strings.Contains(contentStr, "uuid_generate_v4()") {
		sm.logger.Info("Migration ne nécessite pas d'extension UUID", "file", targetFile)
		return nil
	}
	
	// Vérifier si l'extension est déjà présente
	if strings.Contains(contentStr, "CREATE EXTENSION") && strings.Contains(contentStr, "uuid-ossp") {
		sm.logger.Info("Extension UUID déjà présente dans migration", "file", targetFile)
		return nil
	}
	
	// Ajouter l'extension au début
	extensionSQL := "-- Extension UUID requise\nCREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\n\n"
	newContent := extensionSQL + contentStr
	
	// Écrire le fichier modifié
	err = os.WriteFile(targetFile, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("écriture fichier migration: %w", err)
	}
	
	sm.logger.Info("Extension UUID ajoutée à la migration", "file", targetFile)
	return nil
}

// recalculateAtlasChecksum recalcule les checksums Atlas après modification des migrations
func (sm *SchemaManager) recalculateAtlasChecksum() error {
	sm.logger.Info("Recalcul checksums Atlas")
	
	cmd := exec.Command("atlas", "migrate", "hash",
		"--dir", fmt.Sprintf("file://%s", sm.config.MigrationsPath),
	)
	
	cmd.Dir = sm.config.WorkingDir
	cmd.Env = sm.buildEnv()
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		sm.logger.Error("Erreur recalcul checksums", "error", err, "output", string(output))
		return fmt.Errorf("recalcul checksums Atlas échoué: %w", err)
	}
	
	sm.logger.Info("Checksums Atlas recalculés avec succès")
	return nil
}