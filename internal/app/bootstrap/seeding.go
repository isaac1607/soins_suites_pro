package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/seeds"
)

// SeedingManager gère le seeding intelligent des données initiales
// Garde la même logique que l'original car elle est déjà bien conçue
type SeedingManager struct {
	pgClient    *postgres.Client
	config      *config.Config
	seedsPath   string
	seedService seeds.SeedingService
}

// NewSeedingManager crée une nouvelle instance du gestionnaire de seeding
func NewSeedingManager(pgClient *postgres.Client, cfg *config.Config) *SeedingManager {
	seedsPath := filepath.Join("database", "seeds")

	// Adapter config vers config.Config pour compatibility
	legacyConfig := adaptConfigToLegacy(cfg)

	// Créer le service de seeding avec config adaptée
	seedService := seeds.NewSeedingService(pgClient, legacyConfig)

	return &SeedingManager{
		pgClient:    pgClient,
		config:      cfg,
		seedsPath:   seedsPath,
		seedService: seedService,
	}
}

// adaptConfigToLegacy convertit config.Config vers config.Config
// pour maintenir la compatibilité avec le service seeds existant
func adaptConfigToLegacy(Config *config.Config) *config.Config {
	// Créer une structure config.Config avec les valeurs équivalentes
	legacyConfig := &config.Config{}

	// Adapter les champs nécessaires pour le seeding
	legacyConfig.System.AdminTIRPassword = Config.System.AdminTIRPassword

	// Adapter database config
	legacyConfig.Database.Host = Config.Database.Host
	legacyConfig.Database.Port = Config.Database.Port
	legacyConfig.Database.Database = Config.Database.Database
	legacyConfig.Database.Username = Config.Database.Username
	legacyConfig.Database.Password = Config.Database.Password
	legacyConfig.Database.SSLMode = Config.Database.SSLMode

	return legacyConfig
}

// CheckSeedDataExists vérifie quelles données de seeding existent déjà
func (sm *SeedingManager) CheckSeedDataExists(ctx context.Context) (*seeds.SeedDataStatus, error) {
	fmt.Printf("[SEEDING] Vérification données existantes\n")

	status, err := sm.seedService.CheckSeedDataExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification données seeding: %w", err)
	}

	fmt.Printf("[SEEDING] État données: modules=%t, super_admin=%t",
		status.ModulesExist, status.SuperAdminExist)

	return status, nil
}

// ApplySeeding applique le seeding intelligent selon les données manquantes
func (sm *SeedingManager) ApplySeeding(ctx context.Context, status *seeds.SeedDataStatus) error {
	if status.AllDataExists {
		fmt.Printf("[SEEDING] ✅ Toutes les données TIR sont déjà présentes\n")
		return nil
	}

	fmt.Printf("[SEEDING] 🌱 Application seeding données manquantes\n")

	// 1. Créer modules/rubriques si manquants (depuis JSON)
	if !status.ModulesExist {
		if err := sm.SeedModulesFromJSON(ctx); err != nil {
			return fmt.Errorf("échec seeding modules JSON: %w", err)
		}
	}

	// 2. Créer super admin TIR si manquant
	if !status.SuperAdminExist {
		if err := sm.SeedSuperAdminTIR(ctx); err != nil {
			return fmt.Errorf("échec seeding super admin TIR: %w", err)
		}
	}

	fmt.Printf("[SEEDING] ✅ Seeding terminé avec succès\n")
	return nil
}

// SeedModulesFromJSON exécute le seeding des modules depuis le fichier JSON
func (sm *SeedingManager) SeedModulesFromJSON(ctx context.Context) error {
	fmt.Printf("[SEEDING] 📋 Création modules et rubriques depuis JSON\n")

	jsonPath := filepath.Join(sm.seedsPath, "modules.json")

	if err := sm.seedService.SeedModulesFromJSON(ctx, jsonPath); err != nil {
		return fmt.Errorf("seeding modules JSON: %w", err)
	}

	fmt.Printf("[SEEDING] ✅ Modules et rubriques créés depuis JSON\n")
	return nil
}

// SeedSuperAdminTIR exécute le seeding du super admin TIR
func (sm *SeedingManager) SeedSuperAdminTIR(ctx context.Context) error {
	fmt.Printf("[SEEDING] 👤 Création super admin TIR par défaut\n")

	if err := sm.seedService.SeedSuperAdminTIR(ctx); err != nil {
		return fmt.Errorf("seeding super admin TIR: %w", err)
	}

	fmt.Printf("[SEEDING] ✅ Super admin TIR créé\n")
	return nil
}

// GetSeedsPath retourne le chemin vers les fichiers de seeding
func (sm *SeedingManager) GetSeedsPath() string {
	return sm.seedsPath
}

// ValidateSeedFiles vérifie que tous les fichiers de seeding sont présents
func (sm *SeedingManager) ValidateSeedFiles() error {
	requiredFiles := []string{
		"modules.json", // Fichier JSON pour modules et rubriques
	}

	fmt.Printf("[SEEDING] Validation fichiers de seeding\n")

	for _, file := range requiredFiles {
		fullPath := filepath.Join(sm.seedsPath, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("fichier de seeding manquant: %s", fullPath)
		}
	}

	fmt.Printf("[SEEDING] ✅ Tous les fichiers de seeding sont présents\n")
	return nil
}
