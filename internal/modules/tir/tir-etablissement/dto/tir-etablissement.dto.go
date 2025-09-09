package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateEstablishmentTIRRequest requête de création d'établissement par admin TIR
type CreateEstablishmentTIRRequest struct {
	CodeEtablissement   string `json:"code_etablissement" validate:"required,min=3,max=20"`
	Nom                 string `json:"nom" validate:"required,min=2,max=255"`
	NomCourt           string `json:"nom_court" validate:"required,min=2,max=100"`
	AdresseComplete    string `json:"adresse_complete" validate:"required,min=10,max=500"`
	TelephonePrincipal string `json:"telephone_principal" validate:"required,min=8,max=20"`
	Ville              string `json:"ville" validate:"required,min=2,max=20"`
	Commune            string `json:"commune" validate:"required,min=2,max=20"`
	Email              string `json:"email" validate:"omitempty,email,max=255"`
}

// EstablishmentTIRResponse réponse d'établissement pour endpoints TIR
type EstablishmentTIRResponse struct {
	ID                 uuid.UUID  `json:"id"`
	AppInstance        uuid.UUID  `json:"app_instance"`
	CodeEtablissement  string     `json:"code_etablissement"`
	Nom                string     `json:"nom"`
	NomCourt          string     `json:"nom_court"`
	AdresseComplete   string     `json:"adresse_complete"`
	TelephonePrincipal string     `json:"telephone_principal"`
	Ville             string     `json:"ville"`
	Commune           string     `json:"commune"`
	Email             *string    `json:"email"`
	Statut            string     `json:"statut"`
	CreatedAt         time.Time  `json:"created_at"`
	CreatedBy         uuid.UUID  `json:"created_by"`
	CreatedByAdmin    string     `json:"created_by_admin"` // Nom admin TIR qui a créé
}

// EstablishmentTIRCreationResult résultat de création d'établissement TIR
type EstablishmentTIRCreationResult struct {
	Success       bool                      `json:"success"`
	Establishment *EstablishmentTIRResponse `json:"establishment"`
	Message       string                    `json:"message"`
	AdminInfo     AdminCreationInfo         `json:"admin_info"`
}

// AdminCreationInfo informations sur l'admin TIR qui a créé
type AdminCreationInfo struct {
	AdminID     string `json:"admin_id"`
	Identifiant string `json:"identifiant"`
	NiveauAdmin string `json:"niveau_admin"`
}

// EstablishmentTIRListResponse réponse liste d'établissements TIR
type EstablishmentTIRListResponse struct {
	Establishments []EstablishmentTIRSummary `json:"establishments"`
	Total          int                       `json:"total"`
	Page           int                       `json:"page"`
	Limit          int                       `json:"limit"`
}

// EstablishmentTIRSummary résumé d'établissement pour listes TIR
type EstablishmentTIRSummary struct {
	ID                uuid.UUID `json:"id"`
	CodeEtablissement string    `json:"code_etablissement"`
	Nom               string    `json:"nom"`
	NomCourt         string    `json:"nom_court"`
	Ville            string    `json:"ville"`
	Commune          string    `json:"commune"`
	Statut           string    `json:"statut"`
	CreatedAt        time.Time `json:"created_at"`
}

// UpdateEstablishmentTIRRequest requête de mise à jour d'établissement par admin TIR
type UpdateEstablishmentTIRRequest struct {
	Nom                string  `json:"nom" validate:"omitempty,min=2,max=255"`
	NomCourt          string  `json:"nom_court" validate:"omitempty,min=2,max=100"`
	AdresseComplete   string  `json:"adresse_complete" validate:"omitempty,min=10,max=500"`
	TelephonePrincipal string  `json:"telephone_principal" validate:"omitempty,min=8,max=20"`
	Ville             string  `json:"ville" validate:"omitempty,min=2,max=20"`
	Commune           string  `json:"commune" validate:"omitempty,min=2,max=20"`
	Email             *string `json:"email" validate:"omitempty,email,max=255"`
}

// EstablishmentTIRUpdateResult résultat de mise à jour d'établissement TIR
type EstablishmentTIRUpdateResult struct {
	Success       bool                      `json:"success"`
	Establishment *EstablishmentTIRResponse `json:"establishment"`
	Message       string                    `json:"message"`
	UpdatedBy     AdminCreationInfo         `json:"updated_by"`
}

// ===== DTOs TIR pour la gestion des licences =====

// CreateLicenseTIRRequest - Requête de création de licence par admin TIR
type CreateLicenseTIRRequest struct {
	EtablissementID  uuid.UUID `json:"etablissement_id" validate:"required"`
	ModeDeploiement  string    `json:"mode_deploiement" validate:"required,oneof=local online"`
	TypeLicence      string    `json:"type_licence" validate:"required,oneof=premium standard evaluation"`
	ModulesAutorises *[]string `json:"modules_autorises,omitempty"` // Codes des modules (optionnel pour mode local)
	DateExpiration   *time.Time `json:"date_expiration,omitempty"`   // Optionnel - calculé selon règles métier
}

// LicenseTIRResponse - Réponse de licence pour endpoints TIR
type LicenseTIRResponse struct {
	ID                    uuid.UUID                `json:"id"`
	EtablissementID       uuid.UUID                `json:"etablissement_id"`
	EtablissementNom      string                   `json:"etablissement_nom"`
	EtablissementCode     string                   `json:"etablissement_code"`
	ModeDeploiement       string                   `json:"mode_deploiement"`
	TypeLicence           string                   `json:"type_licence"`
	ModulesAutorises      []LicenseModuleTIRInfo   `json:"modules_autorises"`
	NombreModules         int                      `json:"nombre_modules"`
	DateActivation        time.Time                `json:"date_activation"`
	DateExpiration        *time.Time               `json:"date_expiration,omitempty"`
	Statut                string                   `json:"statut"`
	StatutCalcule         string                   `json:"statut_calcule"`
	JoursAvantExpiration  *int                     `json:"jours_avant_expiration,omitempty"`
	EstExpire             bool                     `json:"est_expire"`
	EstBientotExpire      bool                     `json:"est_bientot_expire"`
	SyncInitialComplete   bool                     `json:"sync_initial_complete"`
	DateSyncInitial       *time.Time               `json:"date_sync_initial,omitempty"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
	CreatedBy             uuid.UUID                `json:"created_by"`
	CreatedByAdmin        string                   `json:"created_by_admin"` // Nom admin TIR
}

// LicenseModuleTIRInfo - Information d'un module pour licence TIR
type LicenseModuleTIRInfo struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Nom  string    `json:"nom,omitempty"`
}

// LicenseTIRCreationResult - Résultat de création de licence TIR
type LicenseTIRCreationResult struct {
	Success   bool                `json:"success"`
	License   *LicenseTIRResponse `json:"license"`
	Message   string              `json:"message"`
	AdminInfo AdminCreationInfo   `json:"admin_info"`
}

// LicenseTIRConsultationResult - Résultat de consultation de licence TIR
type LicenseTIRConsultationResult struct {
	Success   bool                      `json:"success"`
	License   *LicenseTIRResponse       `json:"license"`
	History   []LicenseHistoryTIREntry  `json:"history,omitempty"`
	Message   string                    `json:"message"`
	AdminInfo *AdminCreationInfo        `json:"admin_info,omitempty"`
}

// LicenseHistoryTIREntry - Entrée d'historique de licence pour TIR
type LicenseHistoryTIREntry struct {
	ID                uuid.UUID  `json:"id"`
	TypeEvenement     string     `json:"type_evenement"`
	StatutPrecedent   *string    `json:"statut_precedent,omitempty"`
	StatutNouveau     string     `json:"statut_nouveau"`
	MotifChangement   string     `json:"motif_changement"`
	UtilisateurAction *uuid.UUID `json:"utilisateur_action,omitempty"`
	IPAction          *string    `json:"ip_action,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// AvailableModulesTIRResponse - Liste des modules disponibles pour TIR
type AvailableModulesTIRResponse struct {
	Success bool                        `json:"success"`
	Modules []AvailableModuleTIRInfo    `json:"modules"`
	Total   int                         `json:"total"`
	Message string                      `json:"message"`
}

// AvailableModuleTIRInfo - Module disponible pour création de licence TIR
type AvailableModuleTIRInfo struct {
	ID          uuid.UUID `json:"id"`
	CodeModule  string    `json:"code_module"`
	Nom         string    `json:"nom_standard"`
	Description *string   `json:"description,omitempty"`
	EstActif    bool      `json:"est_actif"`
}

// LicenseListTIRResponse - Liste des licences d'un établissement pour TIR
type LicenseListTIRResponse struct {
	Success   bool                   `json:"success"`
	Licenses  []LicenseSummaryTIR    `json:"licenses"`
	Total     int                    `json:"total"`
	Message   string                 `json:"message"`
}

// LicenseSummaryTIR - Résumé de licence pour listes TIR
type LicenseSummaryTIR struct {
	ID                   uuid.UUID  `json:"id"`
	EtablissementNom     string     `json:"etablissement_nom"`
	EtablissementCode    string     `json:"etablissement_code"`
	ModeDeploiement      string     `json:"mode_deploiement"`
	TypeLicence          string     `json:"type_licence"`
	Statut               string     `json:"statut"`
	StatutCalcule        string     `json:"statut_calcule"`
	DateActivation       time.Time  `json:"date_activation"`
	DateExpiration       *time.Time `json:"date_expiration,omitempty"`
	NombreModules        int        `json:"nombre_modules"`
	EstExpire            bool       `json:"est_expire"`
	JoursAvantExpiration *int       `json:"jours_avant_expiration,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}