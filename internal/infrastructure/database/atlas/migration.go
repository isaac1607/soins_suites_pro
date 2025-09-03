package atlas

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MigrationManager gère les migrations avec logging détaillé
type MigrationManager struct {
	client *Client
	logger AtlasLogger
}

// NewMigrationManager crée une nouvelle instance du gestionnaire de migrations
func NewMigrationManager(client *Client, logger AtlasLogger) *MigrationManager {
	if logger == nil {
		logger = NewGinCompatibleLogger()
	}

	return &MigrationManager{
		client: client,
		logger: logger,
	}
}

// ApplyMigrations applique toutes les migrations en attente
func (m *MigrationManager) ApplyMigrations(ctx context.Context) error {
	m.logger.Info("Début application des migrations")

	// Vérifier le statut avant migration
	statusBefore, err := m.client.GetStatus(ctx)
	if err != nil {
		m.logger.Error("Impossible de récupérer le statut avant migration", "error", err)
		return fmt.Errorf("échec récupération statut pré-migration: %w", err)
	}

	pendingCount := m.countPendingMigrations(statusBefore)
	m.logger.Info("Migrations en attente détectées", "count", pendingCount)

	if pendingCount == 0 {
		m.logger.Info("Aucune migration en attente")
		return nil
	}

	// Appliquer les migrations
	startTime := time.Now()
	cmd := m.client.buildCommand(ctx, "migrate", "apply", "--env", m.client.environment)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		m.logger.Error("Échec application des migrations",
			"error", err,
			"output", string(output),
			"duration_ms", duration.Milliseconds())
		return fmt.Errorf("échec application des migrations: %w (output: %s)", err, string(output))
	}

	// Vérifier le statut après migration
	statusAfter, err := m.client.GetStatus(ctx)
	if err != nil {
		m.logger.Warn("Impossible de vérifier le statut après migration", "error", err)
	} else {
		appliedCount := m.countAppliedMigrations(statusAfter)
		m.logger.Info("Migrations appliquées avec succès",
			"applied_count", appliedCount,
			"duration_ms", duration.Milliseconds(),
			"output", string(output))
	}

	return nil
}

// ApplySpecificMigration applique une migration spécifique jusqu'à une version donnée
func (m *MigrationManager) ApplySpecificMigration(ctx context.Context, targetVersion string) error {
	m.logger.Info("Application migration jusqu'à version spécifique", "target_version", targetVersion)

	startTime := time.Now()
	cmd := m.client.buildCommand(ctx, "migrate", "apply", "--env", m.client.environment, targetVersion)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		m.logger.Error("Échec application migration spécifique",
			"target_version", targetVersion,
			"error", err,
			"output", string(output),
			"duration_ms", duration.Milliseconds())
		return fmt.Errorf("échec application migration %s: %w (output: %s)", targetVersion, err, string(output))
	}

	m.logger.Info("Migration spécifique appliquée avec succès",
		"target_version", targetVersion,
		"duration_ms", duration.Milliseconds(),
		"output", string(output))

	return nil
}

// ValidateMigrations valide toutes les migrations sans les appliquer
func (m *MigrationManager) ValidateMigrations(ctx context.Context) error {
	m.logger.Info("Validation des migrations")

	startTime := time.Now()
	err := m.client.ValidateConfig(ctx)
	duration := time.Since(startTime)

	if err != nil {
		m.logger.Error("Validation des migrations échouée",
			"error", err,
			"duration_ms", duration.Milliseconds())
		return err
	}

	m.logger.Info("Validation des migrations réussie", "duration_ms", duration.Milliseconds())
	return nil
}

// GetMigrationHistory retourne l'historique complet des migrations
func (m *MigrationManager) GetMigrationHistory(ctx context.Context) ([]MigrationStatus, error) {
	m.logger.Info("Récupération historique des migrations")

	statuses, err := m.client.GetStatus(ctx)
	if err != nil {
		m.logger.Error("Impossible de récupérer l'historique des migrations", "error", err)
		return nil, err
	}

	appliedCount := m.countAppliedMigrations(statuses)
	pendingCount := m.countPendingMigrations(statuses)

	m.logger.Info("Historique des migrations récupéré",
		"total_migrations", len(statuses),
		"applied_count", appliedCount,
		"pending_count", pendingCount)

	return statuses, nil
}

// DryRun simule l'application des migrations sans les appliquer réellement
func (m *MigrationManager) DryRun(ctx context.Context) ([]string, error) {
	m.logger.Info("Simulation application des migrations (dry-run)")

	cmd := m.client.buildCommand(ctx, "migrate", "apply", "--dry-run", "--env", m.client.environment)

	output, err := cmd.Output()
	if err != nil {
		m.logger.Error("Échec simulation des migrations", "error", err)
		return nil, fmt.Errorf("échec simulation des migrations: %w", err)
	}

	changes := strings.Split(strings.TrimSpace(string(output)), "\n")
	m.logger.Info("Simulation terminée", "changes_count", len(changes))

	return changes, nil
}

// countPendingMigrations compte le nombre de migrations en attente
func (m *MigrationManager) countPendingMigrations(statuses []MigrationStatus) int {
	count := 0
	for _, status := range statuses {
		if !status.Applied {
			count++
		}
	}
	return count
}

// countAppliedMigrations compte le nombre de migrations appliquées
func (m *MigrationManager) countAppliedMigrations(statuses []MigrationStatus) int {
	count := 0
	for _, status := range statuses {
		if status.Applied {
			count++
		}
	}
	return count
}
