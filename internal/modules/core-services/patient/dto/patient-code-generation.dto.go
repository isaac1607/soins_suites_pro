package dto

import "time"

// CodeGenerationRequest représente une demande de génération de code patient
type CodeGenerationRequest struct {
	EtablissementCode string `json:"etablissement_code" validate:"required,min=1,max=20"`
}

// CodeGenerationResponse représente le résultat de génération d'un code patient
type CodeGenerationResponse struct {
	CodePatient       string    `json:"code_patient"`
	EtablissementCode string    `json:"etablissement_code"`
	Annee            int       `json:"annee"`
	Numero           int       `json:"numero"`
	Suffixe          string    `json:"suffixe"`
	NombreGeneres    int64     `json:"nombre_generes"`
	GeneratedAt      time.Time `json:"generated_at"`
	Source           string    `json:"source"` // "redis" ou "postgres"
	GenerationTimeMs int       `json:"generation_time_ms"`
}

// SequenceState représente l'état actuel d'une séquence pour un établissement/année
type SequenceState struct {
	EtablissementCode string `json:"etablissement_code"`
	Annee            int    `json:"annee"`
	DernierNumero    int    `json:"dernier_numero"`
	DernierSuffixe   string `json:"dernier_suffixe"`
	NombreGeneres    int64  `json:"nombre_generes"`
}

// CodeGenerationStats représente les statistiques de génération pour monitoring
type CodeGenerationStats struct {
	EtablissementCode     string  `json:"etablissement_code"`
	Annee                int     `json:"annee"`
	NombreGeneres        int64   `json:"nombre_generes"`
	CapaciteUtilisee     float64 `json:"capacite_utilisee_pct"` // Pourcentage de 17.5M
	DernierCode          string  `json:"dernier_code"`
	ProchainCode         string  `json:"prochain_code"`
	JoursAvantEpuisement *int    `json:"jours_avant_epuisement,omitempty"`
}

// CodeGenerationError représente les erreurs spécifiques à la génération de codes
type CodeGenerationError struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	EtablissementCode string `json:"etablissement_code"`
	Annee         int    `json:"annee,omitempty"`
}

// Constantes pour les erreurs de génération
const (
	ErrCodeCapaciteMaximale     = "CAPACITY_EXCEEDED"
	ErrCodeEtablissementInvalide = "INVALID_ESTABLISHMENT"
	ErrCodeRedisIndisponible    = "REDIS_UNAVAILABLE"
	ErrCodePostgresIndisponible = "POSTGRES_UNAVAILABLE"
	ErrCodeLockTimeout          = "LOCK_TIMEOUT"
	ErrCodeFormatInvalide       = "INVALID_FORMAT"
)

// NewCodeGenerationError crée une nouvelle erreur de génération de code
func NewCodeGenerationError(code, message, etablissementCode string, annee int) *CodeGenerationError {
	return &CodeGenerationError{
		Code:              code,
		Message:           message,
		EtablissementCode: etablissementCode,
		Annee:            annee,
	}
}

// Error implémente l'interface error
func (e *CodeGenerationError) Error() string {
	return e.Message
}