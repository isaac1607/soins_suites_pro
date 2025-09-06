package seeds

import (
	"context"
)

// SeedDataStatus représente l'état des données de seeding
type SeedDataStatus struct {
	ModulesExist  bool `json:"modules_exist"`
	AllDataExists bool `json:"all_data_exists"`
}

// Types pour établissement et admin supprimés - Focus modules uniquement

// ModuleJSONData représente un module dans le fichier JSON
type ModuleJSONData struct {
	CodeModule          string             `json:"code_module"`
	NomStandard         string             `json:"nom_standard"`
	NomPersonnalise     *string            `json:"nom_personnalise"`
	Description         string             `json:"description"`
	EstMedical          bool               `json:"est_medical"`
	EstObligatoire      bool               `json:"est_obligatoire"`
	EstActif            bool               `json:"est_actif"`
	EstModuleBackOffice bool               `json:"est_module_back_office"`
	PeutPrendreTicket   bool               `json:"peut_prendre_ticket"`
	Rubriques           []RubriqueJSONData `json:"rubriques"`
}

// RubriqueJSONData représente une rubrique dans le fichier JSON
type RubriqueJSONData struct {
	CodeRubrique   string `json:"code_rubrique"`
	Nom            string `json:"nom"`
	Description    string `json:"description"`
	OrdreAffichage int    `json:"ordre_affichage"`
	EstObligatoire bool   `json:"est_obligatoire"`
	EstActif       bool   `json:"est_actif"`
}

// Type SpecialiteJSONData supprimé - Les spécialités ne sont plus gérées séparément

// ModulesJSONStructure représente la structure complète du fichier JSON modules
type ModulesJSONStructure struct {
	Modules struct {
		BackOffice  []ModuleJSONData `json:"back_office"`
		FrontOffice []ModuleJSONData `json:"front_office"`
	} `json:"modules"`
}

// SeedingService interface simplifiée pour la gestion des modules uniquement
type SeedingService interface {
	// Vérifications d'état
	CheckSeedDataExists(ctx context.Context) (*SeedDataStatus, error)
	ValidateRequiredTables(ctx context.Context) error

	// Seeding des modules
	SeedModulesFromJSON(ctx context.Context, jsonPath string) error

	// Utilitaires
	LoadModulesFromFile(jsonPath string) (*ModulesJSONStructure, error)
}

// IsComplete vérifie si le seeding des modules est complet
func (s *SeedDataStatus) IsComplete() bool {
	return s.ModulesExist
}

// GetMissingSeeds retourne la liste des seeds manquants
func (s *SeedDataStatus) GetMissingSeeds() []string {
	var missing []string

	if !s.ModulesExist {
		missing = append(missing, "modules")
	}

	return missing
}
