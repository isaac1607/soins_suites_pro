package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/shared/utils"

	"github.com/jackc/pgx/v5"
)

// seedingService implémente SeedingService - Version simplifiée modules uniquement
type seedingService struct {
	pgClient *postgres.Client
	config   *config.Config
}

// NewSeedingService crée un nouveau service de seeding
func NewSeedingService(pgClient *postgres.Client, cfg *config.Config) SeedingService {
	return &seedingService{
		pgClient: pgClient,
		config:   cfg,
	}
}

// CheckSeedDataExists vérifie quelles données de seeding existent déjà
func (s *seedingService) CheckSeedDataExists(ctx context.Context) (*SeedDataStatus, error) {
	status := &SeedDataStatus{}

	// Vérifier modules/rubriques
	modulesExist, err := s.checkModulesExist(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification modules: %w", err)
	}
	status.ModulesExist = modulesExist

	// Vérifier super admin TIR
	superAdminExist, err := s.checkSuperAdminExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification super admin: %w", err)
	}
	status.SuperAdminExist = superAdminExist

	status.AllDataExists = status.ModulesExist && status.SuperAdminExist

	return status, nil
}

// ValidateRequiredTables valide que toutes les tables requises existent
func (s *seedingService) ValidateRequiredTables(ctx context.Context) error {
	requiredTables := []string{
		"base_module",
		"base_rubrique",
		"tir_admin_global",
	}

	for _, table := range requiredTables {
		exists, err := s.checkTableExists(ctx, table)
		if err != nil {
			return fmt.Errorf("erreur vérification table %s: %w", table, err)
		}
		if !exists {
			return ErrTableNotExists(table)
		}
	}

	return nil
}

// SeedModulesFromJSON seed les modules depuis un fichier JSON
func (s *seedingService) SeedModulesFromJSON(ctx context.Context, jsonPath string) error {
	// Charger les données depuis le fichier JSON
	modulesData, err := s.LoadModulesFromFile(jsonPath)
	if err != nil {
		return fmt.Errorf("chargement modules JSON: %w", err)
	}

	// Commencer une transaction
	tx, err := s.pgClient.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("début transaction modules: %w", err)
	}
	defer tx.Rollback(ctx)

	// Seed modules back-office
	for _, module := range modulesData.Modules.BackOffice {
		if err := s.seedModule(ctx, tx, &module); err != nil {
			return fmt.Errorf("seeding module back-office %s: %w", module.CodeModule, err)
		}
	}

	// Seed modules front-office
	for _, module := range modulesData.Modules.FrontOffice {
		if err := s.seedModule(ctx, tx, &module); err != nil {
			return fmt.Errorf("seeding module front-office %s: %w", module.CodeModule, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction modules: %w", err)
	}

	return nil
}

// LoadModulesFromFile charge les modules depuis un fichier JSON
func (s *seedingService) LoadModulesFromFile(jsonPath string) (*ModulesJSONStructure, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, ErrJSONLoad(jsonPath, err)
	}

	var modulesData ModulesJSONStructure
	if err := json.Unmarshal(data, &modulesData); err != nil {
		return nil, ErrJSONLoad(jsonPath, err)
	}

	return &modulesData, nil
}

// seedModule seed un module et ses rubriques (INSERT uniquement - pas de mise à jour)
func (s *seedingService) seedModule(ctx context.Context, tx pgx.Tx, module *ModuleJSONData) error {
	fmt.Printf("[SEEDING] 📦 Traitement module %s (médical: %t)\n",
		module.CodeModule, module.EstMedical)

	// Vérifier si le module existe déjà
	exists, err := s.checkModuleExists(ctx, module.CodeModule)
	if err != nil {
		return fmt.Errorf("vérification module existant: %w", err)
	}

	var moduleID string
	if exists {
		fmt.Printf("[SEEDING] ⏭️  Module %s existe déjà - Ignoré (pas de mise à jour)\n", module.CodeModule)
		// Récupérer l'ID du module existant pour traiter les rubriques
		moduleID, err = s.getModuleID(ctx, module.CodeModule)
		if err != nil {
			return fmt.Errorf("récupération ID module existant: %w", err)
		}
	} else {
		fmt.Printf("[SEEDING] ➕ Module %s nouveau - Création\n", module.CodeModule)
		moduleID, err = s.insertModule(ctx, tx, module)
		if err != nil {
			return fmt.Errorf("insertion module: %w", err)
		}
		fmt.Printf("[SEEDING] ✅ Module %s créé avec ID: %s\n", module.CodeModule, moduleID)
	}

	// Traiter les rubriques (INSERT uniquement pour les nouvelles)
	if err := s.seedRubriques(ctx, tx, moduleID, module.Rubriques); err != nil {
		return fmt.Errorf("traitement rubriques: %w", err)
	}

	return nil
}

// insertModule insère un nouveau module
func (s *seedingService) insertModule(ctx context.Context, tx pgx.Tx, module *ModuleJSONData) (string, error) {
	moduleQuery := `
		INSERT INTO base_module (
			code_module, nom_standard, nom_personnalise,
			description, est_medical, est_obligatoire, est_actif,
			est_module_back_office, peut_prendre_ticket, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id
	`

	var moduleID string
	err := tx.QueryRow(ctx, moduleQuery,
		module.CodeModule, module.NomStandard, module.NomPersonnalise,
		module.Description, module.EstMedical, module.EstObligatoire, module.EstActif,
		module.EstModuleBackOffice, module.PeutPrendreTicket, time.Now(),
	).Scan(&moduleID)

	if err != nil {
		return "", ErrDatabaseOperation("insertion module", err)
	}

	return moduleID, nil
}

// getModuleID récupère l'ID d'un module existant par son code
func (s *seedingService) getModuleID(ctx context.Context, codeModule string) (string, error) {
	query := `SELECT id FROM base_module WHERE code_module = $1`

	var moduleID string
	err := s.pgClient.Pool().QueryRow(ctx, query, codeModule).Scan(&moduleID)
	if err != nil {
		return "", ErrDatabaseOperation("récupération ID module", err)
	}

	return moduleID, nil
}

// seedRubriques traite toutes les rubriques d'un module (INSERT uniquement pour les nouvelles)
func (s *seedingService) seedRubriques(ctx context.Context, tx pgx.Tx, moduleID string, rubriques []RubriqueJSONData) error {
	for _, rubrique := range rubriques {
		// Vérifier si la rubrique existe
		exists, err := s.checkRubriqueExists(ctx, moduleID, rubrique.CodeRubrique)
		if err != nil {
			return fmt.Errorf("vérification rubrique existante: %w", err)
		}

		if exists {
			// Rubrique existe déjà - pas de mise à jour
			fmt.Printf("[SEEDING]   ⏭️  Rubrique %s existe déjà - Ignorée\n", rubrique.CodeRubrique)
		} else {
			// Insertion nouvelle rubrique uniquement
			if err := s.insertRubrique(ctx, tx, moduleID, &rubrique); err != nil {
				return fmt.Errorf("insertion rubrique %s: %w", rubrique.CodeRubrique, err)
			}
			fmt.Printf("[SEEDING]   ➕ Rubrique %s créée\n", rubrique.CodeRubrique)
		}
	}
	return nil
}

// insertRubrique insère une nouvelle rubrique
func (s *seedingService) insertRubrique(ctx context.Context, tx pgx.Tx, moduleID string, rubrique *RubriqueJSONData) error {
	rubriqueQuery := `
		INSERT INTO base_rubrique (
			module_id, code_rubrique, nom, description,
			ordre_affichage, est_obligatoire, est_actif, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	_, err := tx.Exec(ctx, rubriqueQuery,
		moduleID, rubrique.CodeRubrique, rubrique.Nom, rubrique.Description,
		rubrique.OrdreAffichage, rubrique.EstObligatoire, rubrique.EstActif, time.Now(),
	)

	if err != nil {
		return ErrDatabaseOperation("insertion rubrique", err)
	}

	return nil
}

// Méthodes utilitaires privées
func (s *seedingService) checkModulesExist(ctx context.Context) (bool, error) {
	// Vérifier les modules essentiels basés sur les codes du JSON
	moduleQuery := `
		SELECT EXISTS(
			SELECT 1 FROM base_module 
			WHERE code_module IN ('GESTION_ETABLISSEMENT', 'ACCUEIL', 'CAISSE')
		)
	`

	var modulesExist bool
	err := s.pgClient.Pool().QueryRow(ctx, moduleQuery).Scan(&modulesExist)
	if err != nil {
		return false, err
	}

	return modulesExist, nil
}

func (s *seedingService) checkTableExists(ctx context.Context, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1
		)
	`

	var exists bool
	err := s.pgClient.Pool().QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

func (s *seedingService) checkModuleExists(ctx context.Context, codeModule string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM base_module WHERE code_module = $1)`

	var exists bool
	err := s.pgClient.Pool().QueryRow(ctx, query, codeModule).Scan(&exists)
	return exists, err
}

func (s *seedingService) checkRubriqueExists(ctx context.Context, moduleID, codeRubrique string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM base_rubrique WHERE module_id = $1 AND code_rubrique = $2)`

	var exists bool
	err := s.pgClient.Pool().QueryRow(ctx, query, moduleID, codeRubrique).Scan(&exists)
	return exists, err
}

// SeedSuperAdminTIR crée le super admin TIR par défaut
func (s *seedingService) SeedSuperAdminTIR(ctx context.Context) error {
	fmt.Printf("[SEEDING] 👤 Création super admin TIR\n")

	// Vérifier si le super admin existe déjà
	exists, err := s.checkSuperAdminExists(ctx)
	if err != nil {
		return fmt.Errorf("vérification super admin existant: %w", err)
	}

	if exists {
		fmt.Printf("[SEEDING] ⏭️  Super admin TIR existe déjà - Ignoré\n")
		return nil
	}

	// Récupérer le mot de passe depuis la config
	if s.config.System.AdminTIRPassword == "" {
		return fmt.Errorf("AdminTIRPassword requis en configuration")
	}

	// Générer salt et hash du mot de passe
	salt, err := utils.GenerateSalt()
	if err != nil {
		return fmt.Errorf("génération salt: %w", err)
	}

	passwordHash := utils.HashPasswordSHA512(s.config.System.AdminTIRPassword, salt)

	// Commencer une transaction
	tx, err := s.pgClient.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("début transaction super admin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insérer le super admin TIR
	if err := s.insertSuperAdminTIR(ctx, tx, salt, passwordHash); err != nil {
		return fmt.Errorf("insertion super admin TIR: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction super admin: %w", err)
	}

	fmt.Printf("[SEEDING] ✅ Super admin TIR créé avec succès\n")
	return nil
}

// insertSuperAdminTIR insère le super admin TIR avec valeurs fixes et toutes les permissions
func (s *seedingService) insertSuperAdminTIR(ctx context.Context, tx pgx.Tx, salt, passwordHash string) error {
	query := `
		INSERT INTO tir_admin_global (
			identifiant, nom, prenoms, email,
			password_hash, salt, niveau_admin,
			peut_gerer_licences, peut_gerer_etablissements,
			peut_acceder_donnees_etablissement, peut_gerer_admins_globaux,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := tx.Exec(ctx, query,
		"admin_tir",                    // identifiant
		"Admin",                        // nom
		"TIR",                          // prenoms
		"admin@tir-system.local",       // email
		passwordHash,                   // password_hash
		salt,                          // salt
		"super_admin_tir",             // niveau_admin
		true,                          // peut_gerer_licences
		true,                          // peut_gerer_etablissements
		true,                          // peut_acceder_donnees_etablissement
		true,                          // peut_gerer_admins_globaux
		time.Now(),                    // created_at
	)

	if err != nil {
		return ErrDatabaseOperation("insertion super admin TIR", err)
	}

	return nil
}

// checkSuperAdminExists vérifie si un super admin TIR existe
func (s *seedingService) checkSuperAdminExists(ctx context.Context) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tir_admin_global WHERE niveau_admin = 'super_admin_tir')`

	var exists bool
	err := s.pgClient.Pool().QueryRow(ctx, query).Scan(&exists)
	return exists, err
}
