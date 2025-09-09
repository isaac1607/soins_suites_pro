package dto

import (
	"time"
)

// DuplicateCheckRequest représente une demande de vérification de doublon
type DuplicateCheckRequest struct {
	Nom             string    `json:"nom" validate:"required"`
	Prenoms         string    `json:"prenoms" validate:"required"`
	DateNaissance   time.Time `json:"date_naissance" validate:"required"`
	TelephonePrincipal *string `json:"telephone_principal"`

	// OPTIONS
	ScoreMinimum    int  `json:"score_minimum" default:"70"`
	LimiteResultats int  `json:"limite_resultats" default:"5"`
}

// DuplicateCheckResponse représente le résultat de vérification de doublon
type DuplicateCheckResponse struct {
	HasDuplicates    bool                    `json:"has_duplicates"`
	HighestScore     int                     `json:"highest_score"`
	Recommendation   string                  `json:"recommendation"` // "BLOCK", "WARN", "ALLOW"
	PotentialMatches []PotentialDuplicate    `json:"potential_matches"`
	CheckExecutedAt  time.Time               `json:"check_executed_at"`
}

// PotentialDuplicate représente un patient potentiellement en doublon
type PotentialDuplicate struct {
	Patient       PatientSearchResult `json:"patient"`
	Score         int                `json:"score"`
	MatchDetails  MatchDetail        `json:"match_details"`
}

// PatientSearchResult est défini dans patient-search.dto.go pour éviter la duplication

// MatchDetail représente les détails d'un match de doublon
type MatchDetail struct {
	NomMatch          int  `json:"nom_match"`           // Score 0-100
	PrenomsMatch      int  `json:"prenoms_match"`       // Score 0-100
	DateNaissanceMatch int `json:"date_naissance_match"` // Score 0-100
	TelephoneMatch    int  `json:"telephone_match"`     // Score 0-100
	ScoreGlobal      int  `json:"score_global"`        // Score 0-100
}

// ValidationError représente une erreur de validation métier
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResult représente le résultat d'une validation complète
type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors"`
}

// Constantes pour les recommandations de doublon
const (
	RecommendationBlock = "BLOCK"  // Score >= 85%
	RecommendationWarn  = "WARN"   // Score 70-84%
	RecommendationAllow = "ALLOW"  // Score < 70%
)

// Constantes pour les codes d'erreur de validation
const (
	ValidationErrorInvalidPhone     = "INVALID_PHONE"
	ValidationErrorInvalidEmail     = "INVALID_EMAIL"
	ValidationErrorInvalidDate      = "INVALID_DATE"
	ValidationErrorRequiredField    = "REQUIRED_FIELD"
	ValidationErrorInvalidFormat    = "INVALID_FORMAT"
	ValidationErrorDuplicateFound   = "DUPLICATE_FOUND"
	ValidationErrorReferenceNotFound = "REFERENCE_NOT_FOUND"
)

// GetRecommendation détermine la recommandation basée sur le score le plus élevé
func (r *DuplicateCheckResponse) GetRecommendation() string {
	if r.HighestScore >= 85 {
		return RecommendationBlock
	}
	if r.HighestScore >= 70 {
		return RecommendationWarn
	}
	return RecommendationAllow
}

// ShouldBlock indique si la création devrait être bloquée
func (r *DuplicateCheckResponse) ShouldBlock() bool {
	return r.GetRecommendation() == RecommendationBlock
}

// NewValidationError crée une nouvelle erreur de validation
func NewValidationError(field, code, message string, value interface{}) ValidationError {
	return ValidationError{
		Field:   field,
		Code:    code,
		Message: message,
		Value:   value,
	}
}