package atlas

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RollbackManager gère les rollback des migrations Atlas
type RollbackManager struct {
	client *Client
	logger AtlasLogger
}

// NewRollbackManager crée une nouvelle instance du gestionnaire de rollback
func NewRollbackManager(client *Client, logger AtlasLogger) *RollbackManager {
	if logger == nil {
		logger = NewGinCompatibleLogger()
	}

	return &RollbackManager{
		client: client,
		logger: logger,
	}
}

// RollbackOptions options de configuration pour le rollback
type RollbackOptions struct {
	TargetVersion string        // Version cible pour le rollback
	DryRun        bool          // Simulation sans application
	Force         bool          // Forcer le rollback même en cas d'avertissement
	Timeout       time.Duration // Timeout pour l'opération
}

// RollbackToVersion effectue un rollback jusqu'à une version spécifique
func (r *RollbackManager) RollbackToVersion(ctx context.Context, targetVersion string) error {
	options := RollbackOptions{
		TargetVersion: targetVersion,
		DryRun:        false,
		Force:         false,
		Timeout:       30 * time.Second,
	}

	return r.RollbackWithOptions(ctx, options)
}

// RollbackWithOptions effectue un rollback avec options configurables
func (r *RollbackManager) RollbackWithOptions(ctx context.Context, options RollbackOptions) error {
	r.logger.Info("Début rollback des migrations",
		"target_version", options.TargetVersion,
		"dry_run", options.DryRun,
		"force", options.Force)

	// Valider la version cible
	if err := r.validateTargetVersion(ctx, options.TargetVersion); err != nil {
		r.logger.Error("Version cible invalide", "target_version", options.TargetVersion, "error", err)
		return fmt.Errorf("version cible invalide: %w", err)
	}

	// Obtenir l'état actuel
	currentStatus, err := r.client.GetStatus(ctx)
	if err != nil {
		r.logger.Error("Impossible de récupérer l'état actuel", "error", err)
		return fmt.Errorf("échec récupération état actuel: %w", err)
	}

	// Calculer les migrations à annuler
	migrationsToRollback := r.calculateMigrationsToRollback(currentStatus, options.TargetVersion)
	if len(migrationsToRollback) == 0 {
		r.logger.Info("Aucune migration à annuler")
		return nil
	}

	r.logger.Info("Migrations à annuler identifiées",
		"count", len(migrationsToRollback),
		"migrations", migrationsToRollback)

	// Dry run si demandé
	if options.DryRun {
		return r.performDryRun(ctx, options.TargetVersion)
	}

	// Effectuer le rollback
	return r.performRollback(ctx, options)
}

// RollbackLastMigration annule uniquement la dernière migration appliquée
func (r *RollbackManager) RollbackLastMigration(ctx context.Context) error {
	r.logger.Info("Rollback de la dernière migration")

	status, err := r.client.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("échec récupération statut: %w", err)
	}

	// Trouver la dernière migration appliquée
	var lastAppliedVersion string
	var lastAppliedTime *time.Time

	for _, migration := range status {
		if migration.Applied && (lastAppliedTime == nil || (migration.AppliedAt != nil && migration.AppliedAt.After(*lastAppliedTime))) {
			lastAppliedVersion = migration.Version
			lastAppliedTime = migration.AppliedAt
		}
	}

	if lastAppliedVersion == "" {
		r.logger.Info("Aucune migration appliquée à annuler")
		return nil
	}

	// Trouver la version précédente
	previousVersion, err := r.findPreviousVersion(status, lastAppliedVersion)
	if err != nil {
		return fmt.Errorf("impossible de trouver la version précédente: %w", err)
	}

	r.logger.Info("Rollback vers version précédente",
		"current_version", lastAppliedVersion,
		"target_version", previousVersion)

	return r.RollbackToVersion(ctx, previousVersion)
}

// validateTargetVersion valide que la version cible existe
func (r *RollbackManager) validateTargetVersion(ctx context.Context, targetVersion string) error {
	if targetVersion == "" {
		return fmt.Errorf("version cible vide")
	}

	status, err := r.client.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("échec validation version: %w", err)
	}

	// Vérifier que la version existe et est appliquée
	found := false
	for _, migration := range status {
		if migration.Version == targetVersion {
			found = true
			if !migration.Applied {
				return fmt.Errorf("version cible %s n'est pas appliquée", targetVersion)
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("version cible %s introuvable", targetVersion)
	}

	return nil
}

// calculateMigrationsToRollback calcule les migrations à annuler
func (r *RollbackManager) calculateMigrationsToRollback(status []MigrationStatus, targetVersion string) []string {
	var migrationsToRollback []string
	foundTarget := false

	// Parcourir les migrations en ordre inverse
	for i := len(status) - 1; i >= 0; i-- {
		migration := status[i]

		if migration.Version == targetVersion {
			foundTarget = true
			break
		}

		if migration.Applied {
			migrationsToRollback = append(migrationsToRollback, migration.Version)
		}
	}

	if !foundTarget {
		return nil
	}

	// Inverser l'ordre pour avoir l'ordre de rollback correct
	for i := 0; i < len(migrationsToRollback)/2; i++ {
		j := len(migrationsToRollback) - 1 - i
		migrationsToRollback[i], migrationsToRollback[j] = migrationsToRollback[j], migrationsToRollback[i]
	}

	return migrationsToRollback
}

// performDryRun effectue une simulation du rollback
func (r *RollbackManager) performDryRun(ctx context.Context, targetVersion string) error {
	r.logger.Info("Simulation du rollback", "target_version", targetVersion)

	cmd := r.client.buildCommand(ctx, "migrate", "down", "--dry-run", "--env", r.client.environment, targetVersion)

	output, err := cmd.Output()
	if err != nil {
		r.logger.Error("Échec simulation rollback", "error", err)
		return fmt.Errorf("échec simulation rollback: %w", err)
	}

	changes := strings.Split(strings.TrimSpace(string(output)), "\n")
	r.logger.Info("Simulation rollback terminée", "changes_count", len(changes), "changes", changes)

	return nil
}

// performRollback effectue le rollback réel
func (r *RollbackManager) performRollback(ctx context.Context, options RollbackOptions) error {
	startTime := time.Now()

	args := []string{"migrate", "down", "--env", r.client.environment}

	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, options.TargetVersion)

	cmd := r.client.buildCommand(ctx, args...)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		r.logger.Error("Échec rollback des migrations",
			"target_version", options.TargetVersion,
			"error", err,
			"output", string(output),
			"duration_ms", duration.Milliseconds())
		return fmt.Errorf("échec rollback vers %s: %w (output: %s)", options.TargetVersion, err, string(output))
	}

	r.logger.Info("Rollback des migrations réussi",
		"target_version", options.TargetVersion,
		"duration_ms", duration.Milliseconds(),
		"output", string(output))

	return nil
}

// findPreviousVersion trouve la version précédant la version donnée
func (r *RollbackManager) findPreviousVersion(status []MigrationStatus, currentVersion string) (string, error) {
	var previousVersion string

	for i, migration := range status {
		if migration.Version == currentVersion && i > 0 {
			// Trouver la migration appliquée précédente
			for j := i - 1; j >= 0; j-- {
				if status[j].Applied {
					previousVersion = status[j].Version
					break
				}
			}
			break
		}
	}

	if previousVersion == "" {
		return "", fmt.Errorf("aucune version précédente trouvée pour %s", currentVersion)
	}

	return previousVersion, nil
}
