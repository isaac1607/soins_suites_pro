package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreatePatientRequest représente les données requises pour créer un patient
type CreatePatientRequest struct {
	// IDENTITÉ (Obligatoires)
	Nom                     string    `json:"nom" validate:"required,min=2,max=255"`
	Prenoms                 string    `json:"prenoms" validate:"required,min=2,max=255"`
	DateNaissance           time.Time `json:"date_naissance" validate:"required"`
	EstDateSupposee         bool      `json:"est_date_supposee"`
	Sexe                    string    `json:"sexe" validate:"required,oneof=M F"`
	NationaliteID           uuid.UUID `json:"nationalite_id" validate:"required"`
	SituationMatrimonialeID uuid.UUID `json:"situation_matrimoniale_id" validate:"required"`

	// PIÈCE D'IDENTITÉ (Optionnels)
	TypePieceIdentiteID  *uuid.UUID `json:"type_piece_identite_id"`
	CniNni              *string    `json:"cni_nni"`
	NumeroPieceIdentite *string    `json:"numero_piece_identite"`
	LieuNaissance       *string    `json:"lieu_naissance"`
	NomJeuneFille       *string    `json:"nom_jeune_fille"`

	// CONTACT (Requis)
	TelephonePrincipal   string  `json:"telephone_principal" validate:"required,e164"`
	TelephoneSecondaire  *string `json:"telephone_secondaire"`
	Email               *string `json:"email" validate:"omitempty,email"`

	// LOCALISATION
	AdresseComplete string  `json:"adresse_complete" validate:"required"`
	Quartier       *string `json:"quartier"`
	Ville          *string `json:"ville"`
	Commune        *string `json:"commune"`
	PaysResidence  string  `json:"pays_residence" default:"Côte d'Ivoire"`

	// SOCIO-PROFESSIONNEL
	ProfessionID *uuid.UUID `json:"profession_id"`

	// PERSONNES À CONTACTER
	PersonnesAContacter []PersonneContact `json:"personnes_a_contacter"`

	// ASSURANCE
	EstAssure  bool                    `json:"est_assure"`
	Assurances []CreateAssuranceData   `json:"assurances" validate:"required_if=EstAssure true"`
}

// PersonneContact représente une personne à contacter
type PersonneContact struct {
	NomPrenoms           string     `json:"nom_prenoms" validate:"required"`
	Telephone           string     `json:"telephone" validate:"required,e164"`
	TelephoneSecondaire *string    `json:"telephone_secondaire"`
	AffiliationID       uuid.UUID  `json:"affiliation_id" validate:"required"`
}

// CreateAssuranceData représente les données d'assurance pour un patient
type CreateAssuranceData struct {
	AssuranceID              uuid.UUID `json:"assurance_id" validate:"required"`
	NumeroAssure            string    `json:"numero_assure" validate:"required"`
	TypeBeneficiaire        string    `json:"type_beneficiaire" validate:"oneof=principal ayant_droit" default:"principal"`
	NumeroAssurePrincipal   *string   `json:"numero_assure_principal"`
	LienAvecPrincipal       *string   `json:"lien_avec_principal" validate:"required_if=TypeBeneficiaire ayant_droit"`
}

// PatientResponse représente les données retournées après création/lecture d'un patient
type PatientResponse struct {
	ID           uuid.UUID `json:"id"`
	CodePatient  string    `json:"code_patient"`

	// Identité
	Nom                     string     `json:"nom"`
	Prenoms                 string     `json:"prenoms"`
	DateNaissance           time.Time  `json:"date_naissance"`
	EstDateSupposee         bool       `json:"est_date_supposee"`
	Sexe                    string     `json:"sexe"`

	// Contact
	TelephonePrincipal      string     `json:"telephone_principal"`
	TelephoneSecondaire     *string    `json:"telephone_secondaire"`
	Email                   *string    `json:"email"`
	AdresseComplete         string     `json:"adresse_complete"`

	// Assurance
	EstAssure               bool                  `json:"est_assure"`
	Assurances             []AssuranceResponse   `json:"assurances,omitempty"`

	// Métadonnées
	EtablissementCreateurID uuid.UUID `json:"etablissement_createur_id"`
	Statut                 string     `json:"statut"`
	CreatedAt              time.Time  `json:"created_at"`
	CreatedBy              *UserInfo  `json:"created_by,omitempty"`
}

// AssuranceResponse représente les données d'assurance d'un patient
type AssuranceResponse struct {
	ID                    uuid.UUID `json:"id"`
	AssuranceNom         string    `json:"assurance_nom"`
	NumeroAssure         string    `json:"numero_assure"`
	TypeBeneficiaire     string    `json:"type_beneficiaire"`
	EstActif             bool      `json:"est_actif"`
}

// UserInfo représente les informations basiques d'un utilisateur
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Nom      string    `json:"nom"`
	Prenoms  string    `json:"prenoms"`
}

// UpdatePatientRequest représente les données pour modification partielle d'un patient
type UpdatePatientRequest struct {
	// IDENTITÉ (Optionnels - seuls les fournis sont modifiés)
	Nom                     *string    `json:"nom,omitempty"`
	Prenoms                 *string    `json:"prenoms,omitempty"`
	DateNaissance           *time.Time `json:"date_naissance,omitempty"`
	EstDateSupposee         *bool      `json:"est_date_supposee,omitempty"`
	Sexe                    *string    `json:"sexe,omitempty" validate:"omitempty,oneof=M F"`
	NationaliteID           *uuid.UUID `json:"nationalite_id,omitempty"`
	SituationMatrimonialeID *uuid.UUID `json:"situation_matrimoniale_id,omitempty"`

	// CONTACT
	TelephonePrincipal   *string `json:"telephone_principal,omitempty" validate:"omitempty,e164"`
	TelephoneSecondaire  *string `json:"telephone_secondaire,omitempty"`
	Email               *string `json:"email,omitempty" validate:"omitempty,email"`

	// LOCALISATION
	AdresseComplete *string `json:"adresse_complete,omitempty"`
	Quartier       *string `json:"quartier,omitempty"`
	Ville          *string `json:"ville,omitempty"`
	Commune        *string `json:"commune,omitempty"`

	// STATUT & MÉTADONNÉES
	Statut            *string `json:"statut,omitempty" validate:"omitempty,oneof=actif inactif decede archive"`
	EstDecede         *bool   `json:"est_decede,omitempty"`
	DateDeces         *time.Time `json:"date_deces,omitempty"`

	// PERSONNES À CONTACTER (remplacement complet si fourni)
	PersonnesAContacter *[]PersonneContact `json:"personnes_a_contacter,omitempty"`
}

// PatientCreationResult représente le résultat de création d'un patient avec métadonnées
type PatientCreationResult struct {
	Patient              *PatientResponse           `json:"patient"`
	CodeGeneration       *CodeGenerationResponse    `json:"code_generation"`
	DuplicateCheckResult *DuplicateCheckResponse    `json:"duplicate_check"`
	CreationTimeMs       int                        `json:"creation_time_ms"`
	StepsExecuted        []string                   `json:"steps_executed"`
}