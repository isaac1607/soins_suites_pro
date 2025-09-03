package bootstrap

import (
	"context"
	"fmt"
	"time"

	"soins-suite-core/internal/app/config"
	atlasInfra "soins-suite-core/internal/infrastructure/database/atlas"
	pgInfra "soins-suite-core/internal/infrastructure/database/postgres"

	"go.uber.org/fx"
)

// BootstrapSystem orchestré le processus de démarrage automatique
// Version simplifiée : 3 phases séquentielles sans surcomplexité
type BootstrapSystem struct {
	extensionManager *ExtensionManager
	migrationManager *MigrationManager
	seedingManager   *SeedingManager
	config           *config.Config
	timeout          time.Duration
}

// BootstrapResult contient le résultat d'exécution du bootstrap
type BootstrapResult struct {
	Success        bool          `json:"success"`
	TotalDuration  time.Duration `json:"total_duration"`
	PhasesExecuted []PhaseResult `json:"phases_executed"`
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// PhaseResult contient le résultat d'une phase du bootstrap
type PhaseResult struct {
	Phase       string        `json:"phase"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Description string        `json:"description"`
	Error       string        `json:"error,omitempty"`
}

// NewBootstrapSystem crée une nouvelle instance du système de bootstrap
func NewBootstrapSystem(
	extensionManager *ExtensionManager,
	migrationManager *MigrationManager,
	seedingManager *SeedingManager,
	config *config.Config,
) *BootstrapSystem {
	return &BootstrapSystem{
		extensionManager: extensionManager,
		migrationManager: migrationManager,
		seedingManager:   seedingManager,
		config:           config,
		timeout:          5 * time.Minute, // Timeout global 5 minutes
	}
}

// Execute lance le processus de bootstrap complet avec les 3 phases
func (bs *BootstrapSystem) Execute() (*BootstrapResult, error) {
	startTime := time.Now()

	// Context avec timeout global de 5 minutes
	ctx, cancel := context.WithTimeout(context.Background(), bs.timeout)
	defer cancel()

	fmt.Printf("[BOOTSTRAP] Démarrage BootstrapSystem (timeout: %v)\n", bs.timeout)

	result := &BootstrapResult{
		Success:        true,
		PhasesExecuted: []PhaseResult{},
	}

	// Phase 0: Extensions PostgreSQL
	phase0Result := bs.executePhase0(ctx)
	result.PhasesExecuted = append(result.PhasesExecuted, phase0Result)
	if !phase0Result.Success {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Phase 0 échouée: %s", phase0Result.Error)
		return bs.finalizeResult(result, startTime), fmt.Errorf("bootstrap failed at phase 0: %s", phase0Result.Error)
	}

	// Phase 1: Migrations Atlas (appliquer + générer si nécessaire)
	phase1Result := bs.executePhase1(ctx)
	result.PhasesExecuted = append(result.PhasesExecuted, phase1Result)
	if !phase1Result.Success {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Phase 1 échouée: %s", phase1Result.Error)
		return bs.finalizeResult(result, startTime), fmt.Errorf("bootstrap failed at phase 1: %s", phase1Result.Error)
	}

	// Phase 2: Seeding données
	phase2Result := bs.executePhase2(ctx)
	result.PhasesExecuted = append(result.PhasesExecuted, phase2Result)
	if !phase2Result.Success {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Phase 2 échouée: %s", phase2Result.Error)
		return bs.finalizeResult(result, startTime), fmt.Errorf("bootstrap failed at phase 2: %s", phase2Result.Error)
	}

	// Succès complet
	result = bs.finalizeResult(result, startTime)
	fmt.Printf("[BOOTSTRAP] ✅ BootstrapSystem terminé avec succès en %v\n", result.TotalDuration)
	fmt.Printf("[BOOTSTRAP] 🎯 Application prête pour démarrage serveur HTTP\n")

	return result, nil
}

// executePhase0 exécute la Phase 0: Extensions PostgreSQL
func (bs *BootstrapSystem) executePhase0(ctx context.Context) PhaseResult {
	startTime := time.Now()
	phase := "Phase 0: Extensions PostgreSQL"

	fmt.Printf("[BOOTSTRAP] 🔧 Démarrage %s\n", phase)

	err := bs.extensionManager.EnsureUUIDExtension(ctx)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("[BOOTSTRAP] ❌ %s échouée en %v: %v\n", phase, duration, err)
		return PhaseResult{
			Phase:       phase,
			Success:     false,
			Duration:    duration,
			Description: "Création extension uuid-ossp",
			Error:       err.Error(),
		}
	}

	fmt.Printf("[BOOTSTRAP] ✅ %s terminée en %v\n", phase, duration)
	return PhaseResult{
		Phase:       phase,
		Success:     true,
		Duration:    duration,
		Description: "Extension uuid-ossp créée avec succès",
	}
}

// executePhase1 exécute la Phase 1: Migrations Atlas (appliquer + générer)
func (bs *BootstrapSystem) executePhase1(ctx context.Context) PhaseResult {
	startTime := time.Now()
	phase := "Phase 1: Migrations Atlas"

	fmt.Printf("[BOOTSTRAP] 🗄️  Démarrage %s\n", phase)

	err := bs.migrationManager.EnsureMigrationsApplied(ctx)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("[BOOTSTRAP] ❌ %s échouée en %v: %v\n", phase, duration, err)
		return PhaseResult{
			Phase:       phase,
			Success:     false,
			Duration:    duration,
			Description: "Application et génération migrations",
			Error:       err.Error(),
		}
	}

	fmt.Printf("[BOOTSTRAP] ✅ %s terminée en %v\n", phase, duration)
	return PhaseResult{
		Phase:       phase,
		Success:     true,
		Duration:    duration,
		Description: "Migrations Atlas appliquées/générées avec succès",
	}
}

// executePhase2 exécute la Phase 2: Seeding données
func (bs *BootstrapSystem) executePhase2(ctx context.Context) PhaseResult {
	startTime := time.Now()
	phase := "Phase 2: Seeding données"

	fmt.Printf("[BOOTSTRAP] 🌱 Démarrage %s\n", phase)

	exists, err := bs.seedingManager.CheckSeedDataExists(ctx)
	if err != nil {
		duration := time.Since(startTime)
		fmt.Printf("[BOOTSTRAP] ❌ %s - Erreur vérification données en %v: %v\n", phase, duration, err)
		return PhaseResult{
			Phase:       phase,
			Success:     false,
			Duration:    duration,
			Description: "Vérification données existantes",
			Error:       fmt.Sprintf("data check failed: %v", err),
		}
	}

	err = bs.seedingManager.ApplySeeding(ctx, exists)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("[BOOTSTRAP] ❌ %s échouée en %v: %v\n", phase, duration, err)
		return PhaseResult{
			Phase:       phase,
			Success:     false,
			Duration:    duration,
			Description: "Application seeding données",
			Error:       err.Error(),
		}
	}

	fmt.Printf("[BOOTSTRAP] ✅ %s terminée en %v\n", phase, duration)
	return PhaseResult{
		Phase:       phase,
		Success:     true,
		Duration:    duration,
		Description: "Données TIR (établissement + modules + admin) créées avec succès",
	}
}

// finalizeResult finalise le résultat avec la durée totale
func (bs *BootstrapSystem) finalizeResult(result *BootstrapResult, startTime time.Time) *BootstrapResult {
	result.TotalDuration = time.Since(startTime)
	return result
}

// GetTimeout retourne le timeout configuré
func (bs *BootstrapSystem) GetTimeout() time.Duration {
	return bs.timeout
}

// SetTimeout configure un nouveau timeout (utile pour les tests)
func (bs *BootstrapSystem) SetTimeout(timeout time.Duration) {
	bs.timeout = timeout
}

// Providers Fx pour le système de bootstrap

// NewBootstrapExtensionManager provider pour le gestionnaire d'extensions
func NewBootstrapExtensionManager(pgClient *pgInfra.Client, cfg *config.Config) *ExtensionManager {
	return NewExtensionManager(pgClient, cfg)
}

// NewBootstrapMigrationManager provider pour le gestionnaire de migrations
func NewBootstrapMigrationManager(atlasService *atlasInfra.AtlasService, cfg *config.Config) *MigrationManager {
	return NewMigrationManager(atlasService, cfg)
}

// NewBootstrapSeedingManager provider pour le gestionnaire de seeding
func NewBootstrapSeedingManager(pgClient *pgInfra.Client, cfg *config.Config) *SeedingManager {
	return NewSeedingManager(pgClient, cfg)
}

// RegisterBootstrapLifecycle enregistre le système de bootstrap dans le cycle de vie Fx
func RegisterBootstrapLifecycle(
	lc fx.Lifecycle,
	bootstrap *BootstrapSystem,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Printf("[LIFECYCLE] 🚀 Démarrage BootstrapSystem AVANT serveur HTTP\n")

			// Exécuter le bootstrap avec son propre timeout
			result, err := bootstrap.Execute()
			if err != nil {
				fmt.Printf("[LIFECYCLE] ❌ Bootstrap échoué: %v\n", err)
				return fmt.Errorf("bootstrap system failed: %w", err)
			}

			fmt.Printf("[LIFECYCLE] ✅ Bootstrap terminé en %v\n", result.TotalDuration)
			fmt.Printf("[LIFECYCLE] 🎯 Système prêt pour démarrage serveur HTTP\n")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Printf("[LIFECYCLE] 🛑 Arrêt BootstrapSystem\n")
			return nil
		},
	})
}
