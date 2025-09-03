package seeds

import "fmt"

// SeedingError représente une erreur de seeding
type SeedingError struct {
	Message string                 `json:"message"`
	Type    string                 `json:"type"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implémente l'interface error
func (e *SeedingError) Error() string {
	return e.Message
}

// NewSeedingError crée une nouvelle erreur de seeding
func NewSeedingError(message, errorType string, details map[string]interface{}) *SeedingError {
	return &SeedingError{
		Message: message,
		Type:    errorType,
		Details: details,
	}
}

// Erreurs prédéfinies pour le seeding
var (
	ErrEstablishmentExists = func(code string) error {
		return NewSeedingError(
			fmt.Sprintf("établissement avec code %s existe déjà", code),
			"establishment_exists",
			map[string]interface{}{"code": code},
		)
	}

	ErrAdminTIRExists = func(identifiant string) error {
		return NewSeedingError(
			fmt.Sprintf("admin TIR avec identifiant %s existe déjà", identifiant),
			"admin_tir_exists",
			map[string]interface{}{"identifiant": identifiant},
		)
	}

	ErrModuleExists = func(codeModule string) error {
		return NewSeedingError(
			fmt.Sprintf("module avec code %s existe déjà", codeModule),
			"module_exists",
			map[string]interface{}{"code_module": codeModule},
		)
	}

	ErrValidation = func(message string) error {
		return NewSeedingError(message, "validation_error", nil)
	}

	ErrJSONLoad = func(filePath string, err error) error {
		return NewSeedingError(
			fmt.Sprintf("impossible de charger le fichier JSON %s: %v", filePath, err),
			"json_load_error",
			map[string]interface{}{"file_path": filePath, "error": err.Error()},
		)
	}

	ErrTableNotExists = func(tableName string) error {
		return NewSeedingError(
			fmt.Sprintf("table %s n'existe pas", tableName),
			"table_not_exists",
			map[string]interface{}{"table_name": tableName},
		)
	}

	ErrDatabaseOperation = func(operation string, err error) error {
		return NewSeedingError(
			fmt.Sprintf("erreur base de données lors de %s: %v", operation, err),
			"database_error",
			map[string]interface{}{"operation": operation, "error": err.Error()},
		)
	}
)