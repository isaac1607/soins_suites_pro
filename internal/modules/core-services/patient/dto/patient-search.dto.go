package dto

import (
	"time"

	"github.com/google/uuid"
)

// SearchPatientRequest représente une demande de recherche multi-critères
type SearchPatientRequest struct {
	// RECHERCHE DIRECTE
	CodePatient string `json:"code_patient"`

	// RECHERCHE TEXTUELLE
	SearchTerm string `json:"search_term"` // Nom, prénom, téléphone, CNI

	// FILTRES PRÉCIS
	Nom                    *string    `json:"nom"`
	Prenoms               *string    `json:"prenoms"`
	TelephonePrincipal    *string    `json:"telephone_principal"`
	DateNaissance         *time.Time `json:"date_naissance"`
	DateNaissanceDebut    *time.Time `json:"date_naissance_debut"`
	DateNaissanceFin      *time.Time `json:"date_naissance_fin"`
	Sexe                  *string    `json:"sexe" validate:"omitempty,oneof=M F"`
	CniNni                *string    `json:"cni_nni"`

	// FILTRES STATUT
	Statut                []string   `json:"statut"` // actif, inactif, decede, archive
	EstAssure             *bool      `json:"est_assure"`
	EtablissementCreateur *uuid.UUID `json:"etablissement_createur_id"`

	// PAGINATION & TRI
	Page       int    `json:"page" validate:"min=1" default:"1"`
	Limit      int    `json:"limit" validate:"min=1,max=50" default:"20"`
	SortBy     string `json:"sort_by" validate:"oneof=nom created_at score" default:"score"`
	SortOrder  string `json:"sort_order" validate:"oneof=asc desc" default:"desc"`

	// OPTIONS
	IncludeAssurances bool `json:"include_assurances"`
}

// SearchPatientResponse représente le résultat d'une recherche de patients
type SearchPatientResponse struct {
	Patients   []PatientSearchResult `json:"patients"`
	Pagination PaginationInfo        `json:"pagination"`
	SearchInfo SearchMetadata        `json:"search_info"`
}

// PatientSearchResult représente un patient dans les résultats de recherche
type PatientSearchResult struct {
	ID               uuid.UUID `json:"id"`
	CodePatient      string    `json:"code_patient"`
	Nom              string    `json:"nom"`
	Prenoms          string    `json:"prenoms"`
	DateNaissance    time.Time `json:"date_naissance"`
	Sexe             string    `json:"sexe"`
	TelephonePrincipal string  `json:"telephone_principal"`
	AdresseComplete   string   `json:"adresse_complete"`
	EstAssure        bool      `json:"est_assure"`
	Statut           string    `json:"statut"`

	// MÉTADONNÉES RECHERCHE
	Score            *float64             `json:"score,omitempty"` // Score pertinence full-text
	Assurances       []AssuranceResponse  `json:"assurances,omitempty"`

	CreatedAt        time.Time `json:"created_at"`
}

// PaginationInfo contient les informations de pagination
type PaginationInfo struct {
	Page         int  `json:"page"`
	Limit        int  `json:"limit"`
	Total        int  `json:"total"`
	TotalPages   int  `json:"total_pages"`
	HasNext      bool `json:"has_next"`
	HasPrevious  bool `json:"has_previous"`
}

// SearchMetadata contient les métadonnées d'une recherche
type SearchMetadata struct {
	SearchType       string        `json:"search_type"` // "direct_code", "full_text", "criteria"
	ExecutionTimeMs  int           `json:"execution_time_ms"`
	CacheHit         bool          `json:"cache_hit"`
	TotalResults     int           `json:"total_results"`
	AppliedFilters   []string      `json:"applied_filters"`
}

// Constantes pour les types de recherche
const (
	SearchTypeDirectCode = "direct_code"
	SearchTypeFullText   = "full_text"
	SearchTypeCriteria   = "criteria"
)

// GetSearchType détermine le type de recherche basé sur les critères
func (r *SearchPatientRequest) GetSearchType() string {
	if r.CodePatient != "" {
		return SearchTypeDirectCode
	}
	if r.SearchTerm != "" {
		return SearchTypeFullText
	}
	return SearchTypeCriteria
}

// GetAppliedFilters retourne la liste des filtres appliqués
func (r *SearchPatientRequest) GetAppliedFilters() []string {
	var filters []string

	if r.CodePatient != "" {
		filters = append(filters, "code_patient")
	}
	if r.SearchTerm != "" {
		filters = append(filters, "search_term")
	}
	if r.Nom != nil {
		filters = append(filters, "nom")
	}
	if r.Prenoms != nil {
		filters = append(filters, "prenoms")
	}
	if r.TelephonePrincipal != nil {
		filters = append(filters, "telephone_principal")
	}
	if r.DateNaissance != nil {
		filters = append(filters, "date_naissance")
	}
	if r.DateNaissanceDebut != nil {
		filters = append(filters, "date_naissance_debut")
	}
	if r.DateNaissanceFin != nil {
		filters = append(filters, "date_naissance_fin")
	}
	if r.Sexe != nil {
		filters = append(filters, "sexe")
	}
	if r.CniNni != nil {
		filters = append(filters, "cni_nni")
	}
	if len(r.Statut) > 0 {
		filters = append(filters, "statut")
	}
	if r.EstAssure != nil {
		filters = append(filters, "est_assure")
	}
	if r.EtablissementCreateur != nil {
		filters = append(filters, "etablissement_createur")
	}

	return filters
}

// IsEmpty retourne true si la recherche n'a aucun critère
func (r *SearchPatientRequest) IsEmpty() bool {
	return r.CodePatient == "" &&
		r.SearchTerm == "" &&
		r.Nom == nil &&
		r.Prenoms == nil &&
		r.TelephonePrincipal == nil &&
		r.DateNaissance == nil &&
		r.DateNaissanceDebut == nil &&
		r.DateNaissanceFin == nil &&
		r.Sexe == nil &&
		r.CniNni == nil &&
		len(r.Statut) == 0 &&
		r.EstAssure == nil &&
		r.EtablissementCreateur == nil
}

// GetOffset calcule l'offset pour la pagination
func (r *SearchPatientRequest) GetOffset() int {
	return (r.Page - 1) * r.Limit
}

// SetDefaults définit les valeurs par défaut pour la recherche
func (r *SearchPatientRequest) SetDefaults() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 50 {
		r.Limit = 50
	}
	if r.SortBy == "" {
		r.SortBy = "score"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
	if len(r.Statut) == 0 {
		r.Statut = []string{"actif"} // Par défaut, rechercher seulement les patients actifs
	}
}

// NewPaginationInfo crée les informations de pagination
func NewPaginationInfo(page, limit, total int) PaginationInfo {
	totalPages := (total + limit - 1) / limit // Calcul du nombre total de pages (arrondi vers le haut)
	
	return PaginationInfo{
		Page:        page,
		Limit:       limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

// NewSearchMetadata crée les métadonnées de recherche
func NewSearchMetadata(searchType string, executionTimeMs int, totalResults int, appliedFilters []string) SearchMetadata {
	return SearchMetadata{
		SearchType:      searchType,
		ExecutionTimeMs: executionTimeMs,
		CacheHit:        searchType == SearchTypeDirectCode, // Cache hit seulement pour recherche directe par code
		TotalResults:    totalResults,
		AppliedFilters:  appliedFilters,
	}
}