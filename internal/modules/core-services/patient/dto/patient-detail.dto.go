package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PatientDetailResponse représente un patient avec toutes ses données enrichies
type PatientDetailResponse struct {
	// PATIENT PRINCIPAL
	Patient PatientResponse `json:"patient"`

	// RÉFÉRENCES ENRICHIES
	Nationalite         ReferenceInfo  `json:"nationalite"`
	SituationMatrimoniale ReferenceInfo  `json:"situation_matrimoniale"`
	TypePieceIdentite   *ReferenceInfo `json:"type_piece_identite,omitempty"`
	Profession          *ReferenceInfo `json:"profession,omitempty"`

	// PERSONNES À CONTACTER
	PersonnesAContacter []PersonneContactDetail `json:"personnes_a_contacter"`

	// ASSURANCES DÉTAILLÉES
	AssurancesDetails []AssuranceDetail `json:"assurances_details"`

	// MÉTADONNÉES
	LoadedFrom   string    `json:"loaded_from"` // "cache" ou "database"
	LoadTime     int       `json:"load_time_ms"`
	LastUpdated  time.Time `json:"last_updated"`
}

// PersonneContactDetail représente une personne à contacter avec références enrichies
type PersonneContactDetail struct {
	NomPrenoms           string        `json:"nom_prenoms"`
	Telephone           string        `json:"telephone"`
	TelephoneSecondaire *string       `json:"telephone_secondaire"`
	Affiliation         ReferenceInfo `json:"affiliation"`
}

// AssuranceDetail représente une assurance avec toutes les informations détaillées
type AssuranceDetail struct {
	ID                    uuid.UUID     `json:"id"`
	Assurance            ReferenceInfo `json:"assurance"`
	NumeroAssure         string        `json:"numero_assure"`
	TypeBeneficiaire     string        `json:"type_beneficiaire"`
	NumeroAssurePrincipal *string       `json:"numero_assure_principal"`
	LienAvecPrincipal    *string       `json:"lien_avec_principal"`
	EstActif             bool          `json:"est_actif"`
	CreatedAt            time.Time     `json:"created_at"`
}

// ReferenceInfo représente les informations d'une table de référence
type ReferenceInfo struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Nom  string    `json:"nom"`
}

// PatientCacheData représente les données patient stockées en cache Redis
type PatientCacheData struct {
	// Données patient principales
	ID                  uuid.UUID `json:"id"`
	CodePatient         string    `json:"code_patient"`
	EtablissementCreateur uuid.UUID `json:"etablissement_createur_id"`

	// Identité
	Nom                     string    `json:"nom"`
	Prenoms                 string    `json:"prenoms"`
	DateNaissance           time.Time `json:"date_naissance"`
	EstDateSupposee         bool      `json:"est_date_supposee"`
	Sexe                    string    `json:"sexe"`
	NationaliteID           uuid.UUID `json:"nationalite_id"`
	SituationMatrimonialeID uuid.UUID `json:"situation_matrimoniale_id"`

	// Pièce d'identité (optionnel)
	TypePieceIdentiteID  *uuid.UUID `json:"type_piece_identite_id,omitempty"`
	CniNni              *string    `json:"cni_nni,omitempty"`
	NumeroPieceIdentite *string    `json:"numero_piece_identite,omitempty"`
	LieuNaissance       *string    `json:"lieu_naissance,omitempty"`
	NomJeuneFille       *string    `json:"nom_jeune_fille,omitempty"`

	// Contact
	TelephonePrincipal   string  `json:"telephone_principal"`
	TelephoneSecondaire  *string `json:"telephone_secondaire,omitempty"`
	Email               *string `json:"email,omitempty"`

	// Localisation
	AdresseComplete string  `json:"adresse_complete"`
	Quartier       *string `json:"quartier,omitempty"`
	Ville          *string `json:"ville,omitempty"`
	Commune        *string `json:"commune,omitempty"`
	PaysResidence  string  `json:"pays_residence"`

	// Profession (optionnel)
	ProfessionID *uuid.UUID `json:"profession_id,omitempty"`

	// Personnes à contacter (JSON)
	PersonnesAContacter string `json:"personnes_a_contacter"` // JSON serialized

	// Assurance
	EstAssure bool `json:"est_assure"`

	// Statut
	Statut    string     `json:"statut"`
	EstDecede bool       `json:"est_decede"`
	DateDeces *time.Time `json:"date_deces,omitempty"`

	// Métadonnées
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
}

// PatientCacheMetadata représente les métadonnées du cache
type PatientCacheMetadata struct {
	CachedAt          time.Time `json:"cached_at"`
	LastDatabaseSync  time.Time `json:"last_database_sync"`
	Version           string    `json:"version"`
	TTLSeconds        int       `json:"ttl_seconds"`
}

// GetPatientByCodeRequest représente une demande de récupération par code
type GetPatientByCodeRequest struct {
	CodePatient           string `json:"code_patient" validate:"required"`
	IncludeInactive       bool   `json:"include_inactive" default:"false"`
	IncludeAssurances     bool   `json:"include_assurances" default:"true"`
	IncludePersonnesContact bool   `json:"include_personnes_contact" default:"true"`
	ForceRefreshCache     bool   `json:"force_refresh_cache" default:"false"`
}

// PatientNotFoundError représente une erreur lorsque le patient n'est pas trouvé
type PatientNotFoundError struct {
	CodePatient string `json:"code_patient"`
	Message     string `json:"message"`
}

// Error implémente l'interface error
func (e *PatientNotFoundError) Error() string {
	return e.Message
}

// NewPatientNotFoundError crée une nouvelle erreur patient non trouvé
func NewPatientNotFoundError(codePatient string) *PatientNotFoundError {
	return &PatientNotFoundError{
		CodePatient: codePatient,
		Message:     fmt.Sprintf("Patient avec le code '%s' introuvable", codePatient),
	}
}

// PatientArchivedError représente une erreur lorsque le patient est archivé
type PatientArchivedError struct {
	CodePatient string `json:"code_patient"`
	Message     string `json:"message"`
}

// Error implémente l'interface error
func (e *PatientArchivedError) Error() string {
	return e.Message
}

// NewPatientArchivedError crée une nouvelle erreur patient archivé
func NewPatientArchivedError(codePatient string) *PatientArchivedError {
	return &PatientArchivedError{
		CodePatient: codePatient,
		Message:     fmt.Sprintf("Patient avec le code '%s' est archivé", codePatient),
	}
}