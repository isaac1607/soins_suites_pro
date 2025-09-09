package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateEstablishmentRequest - Données pour créer un établissement
type CreateEstablishmentRequest struct {
	CodeEtablissement   string `json:"code_etablissement" validate:"required,min=2,max=20"`
	Nom                 string `json:"nom" validate:"required,min=2,max=255"`
	NomCourt            string `json:"nom_court" validate:"required,min=2,max=100"`
	AdresseComplete     string `json:"adresse_complete" validate:"required,min=5,max=500"`
	TelephonePrincipal  string `json:"telephone_principal" validate:"required,min=8,max=20"`
	Ville               string `json:"ville" validate:"required,min=2,max=20"`
	Commune             string `json:"commune" validate:"required,min=2,max=20"`
	Email               string `json:"email" validate:"required,email,max=255"`
}

// EstablishmentResponse - Réponse après création/récupération
type EstablishmentResponse struct {
	ID                  uuid.UUID  `json:"id"`
	AppInstance         uuid.UUID  `json:"app_instance"`
	CodeEtablissement   string     `json:"code_etablissement"`
	Nom                 string     `json:"nom"`
	NomCourt            string     `json:"nom_court"`
	AdresseComplete     string     `json:"adresse_complete"`
	TelephonePrincipal  string     `json:"telephone_principal"`
	Ville               string     `json:"ville"`
	Commune             string     `json:"commune"`
	Email               string     `json:"email"`
	SecondTelephone     *string    `json:"second_telephone,omitempty"`
	RCCM                *string    `json:"rccm,omitempty"`
	CNPS                *string    `json:"cnps,omitempty"`
	LogoPrincipalURL    *string    `json:"logo_principal_url,omitempty"`
	LogoDocumentsURL    *string    `json:"logo_documents_url,omitempty"`
	DureeValiditeTicket *int       `json:"duree_validite_ticket_jours,omitempty"`
	NbSouchesParCaisse  *int       `json:"nb_souches_par_caisse,omitempty"`
	GardeHeureDebut     *string    `json:"garde_heure_debut,omitempty"`
	GardeHeureFin       *string    `json:"garde_heure_fin,omitempty"`
	Statut              string     `json:"statut"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAtAdminTir   *time.Time `json:"updated_at_admin_tir,omitempty"`
	UpdatedAtUser       *time.Time `json:"updated_at_user,omitempty"`
	CreatedBy           uuid.UUID  `json:"created_by"`
	UpdatedByAdminTir   *uuid.UUID `json:"updated_by_admin_tir,omitempty"`
	UpdatedByUser       *uuid.UUID `json:"updated_by_user,omitempty"`
}

// UpdateEstablishmentRequest - Données pour mettre à jour un établissement
type UpdateEstablishmentRequest struct {
	Nom                string `json:"nom" validate:"required,min=2,max=255"`
	NomCourt           string `json:"nom_court" validate:"required,min=2,max=100"`
	AdresseComplete    string `json:"adresse_complete" validate:"required,min=5,max=500"`
	TelephonePrincipal string `json:"telephone_principal" validate:"required,min=8,max=20"`
	Ville              string `json:"ville" validate:"required,min=2,max=20"`
	Commune            string `json:"commune" validate:"required,min=2,max=20"`
	Email              string `json:"email" validate:"required,email,max=255"`
}

// EstablishmentSummary - Résumé établissement pour listes
type EstablishmentSummary struct {
	ID                uuid.UUID `json:"id"`
	AppInstance       uuid.UUID `json:"app_instance"`
	CodeEtablissement string    `json:"code_etablissement"`
	Nom               string    `json:"nom"`
	NomCourt          string    `json:"nom_court"`
	Statut            string    `json:"statut"`
	CreatedAt         time.Time `json:"created_at"`
}

// EstablishmentCreationResult - Résultat de création d'établissement
type EstablishmentCreationResult struct {
	Establishment *EstablishmentResponse `json:"establishment"`
	Message       string                 `json:"message"`
}

// EstablishmentUpdateResult - Résultat de mise à jour d'établissement
type EstablishmentUpdateResult struct {
	ID                uuid.UUID  `json:"id"`
	AppInstance       uuid.UUID  `json:"app_instance"`
	CodeEtablissement string     `json:"code_etablissement"`
	Nom               string     `json:"nom"`
	NomCourt          string     `json:"nom_court"`
	UpdatedAt         *time.Time `json:"updated_at"`
	UpdatedBy         *uuid.UUID `json:"updated_by"`
	UpdatedByType     string     `json:"updated_by_type"` // "admin_tir" ou "user"
}

// EstablishmentHealthInfo - Informations sanitaires complètes d'un établissement
type EstablishmentHealthInfo struct {
	// Informations de base
	ID                uuid.UUID `json:"id"`
	AppInstance       uuid.UUID `json:"app_instance"`
	CodeEtablissement string    `json:"code_etablissement"`
	Nom               string    `json:"nom"`
	NomCourt          string    `json:"nom_court"`
	Statut            string    `json:"statut"`

	// Informations de contact
	AdresseComplete    string  `json:"adresse_complete"`
	TelephonePrincipal string  `json:"telephone_principal"`
	SecondTelephone    *string `json:"second_telephone,omitempty"`
	Email              string  `json:"email"`
	Ville              string  `json:"ville"`
	Commune            string  `json:"commune"`

	// Informations légales
	RCCM *string `json:"rccm,omitempty"`
	CNPS *string `json:"cnps,omitempty"`

	// Configuration sanitaire
	DureeValiditeTicket *int    `json:"duree_validite_ticket_jours,omitempty"`
	NbSouchesParCaisse  *int    `json:"nb_souches_par_caisse,omitempty"`
	GardeHeureDebut     *string `json:"garde_heure_debut,omitempty"`
	GardeHeureFin       *string `json:"garde_heure_fin,omitempty"`

	// Assets visuels
	LogoPrincipalURL *string `json:"logo_principal_url,omitempty"`
	LogoDocumentsURL *string `json:"logo_documents_url,omitempty"`

	// Métadonnées
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAtAdminTir *time.Time `json:"updated_at_admin_tir,omitempty"`
	UpdatedAtUser     *time.Time `json:"updated_at_user,omitempty"`
	LastModifiedBy    string     `json:"last_modified_by"` // "admin_tir" ou "user"
	LastModifiedAt    *time.Time `json:"last_modified_at,omitempty"`
}

// EstablishmentHealthInfoList - Liste des informations sanitaires (version résumée)
type EstablishmentHealthInfoList struct {
	Establishments []EstablishmentHealthInfoSummary `json:"establishments"`
	Total          int                              `json:"total"`
	Page           int                              `json:"page"`
	Limit          int                              `json:"limit"`
}

// EstablishmentHealthInfoSummary - Résumé pour listes d'établissements
type EstablishmentHealthInfoSummary struct {
	ID                uuid.UUID `json:"id"`
	CodeEtablissement string    `json:"code_etablissement"`
	Nom               string    `json:"nom"`
	NomCourt          string    `json:"nom_court"`
	Ville             string    `json:"ville"`
	Commune           string    `json:"commune"`
	Statut            string    `json:"statut"`
	TelephonePrincipal string   `json:"telephone_principal"`
	Email             string    `json:"email"`
	CreatedAt         time.Time `json:"created_at"`
}

// ===== DTOs pour la gestion des licences =====

// ModuleInfo - Information d'un module autorisé pour une licence
type ModuleInfo struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
}

// CreateLicenseRequest - Données pour créer une licence
type CreateLicenseRequest struct {
	EtablissementID  uuid.UUID `json:"etablissement_id" validate:"required"`
	ModeDeploiement  string    `json:"mode_deploiement" validate:"required,oneof=local online"`
	TypeLicence      string    `json:"type_licence" validate:"required,oneof=premium standard evaluation"`
	ModulesAutorises *[]string `json:"modules_autorises,omitempty"` // Codes des modules (optionnel pour mode local)
	DateExpiration   *time.Time `json:"date_expiration,omitempty"`   // Optionnel - calculé selon règles métier
}

// LicenseResponse - Réponse après création/récupération d'une licence
type LicenseResponse struct {
	ID                    uuid.UUID     `json:"id"`
	EtablissementID       uuid.UUID     `json:"etablissement_id"`
	ModeDeploiement       string        `json:"mode_deploiement"`
	TypeLicence           string        `json:"type_licence"`
	ModulesAutorises      []ModuleInfo  `json:"modules_autorises"`
	DateActivation        time.Time     `json:"date_activation"`
	DateExpiration        *time.Time    `json:"date_expiration,omitempty"`
	Statut                string        `json:"statut"`
	SyncInitialComplete   bool          `json:"sync_initial_complete"`
	DateSyncInitial       *time.Time    `json:"date_sync_initial,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
	CreatedBy             uuid.UUID     `json:"created_by"`
}

// LicenseCreationResult - Résultat de création d'une licence
type LicenseCreationResult struct {
	License *LicenseResponse `json:"license"`
	Message string           `json:"message"`
}

// AvailableModule - Module disponible pour création de licence
type AvailableModule struct {
	ID          uuid.UUID `json:"id"`
	CodeModule  string    `json:"code_module"`
	Nom         string    `json:"nom_standard"`
	Description *string   `json:"description,omitempty"`
	EstActif    bool      `json:"est_actif"`
}

// AvailableModulesResponse - Liste des modules front-office disponibles
type AvailableModulesResponse struct {
	Modules []AvailableModule `json:"modules"`
	Total   int               `json:"total"`
}

// LicenseDetailedResponse - Réponse détaillée pour la consultation d'une licence
type LicenseDetailedResponse struct {
	// Informations licence
	ID                    uuid.UUID     `json:"id"`
	EtablissementID       uuid.UUID     `json:"etablissement_id"`
	ModeDeploiement       string        `json:"mode_deploiement"`
	TypeLicence           string        `json:"type_licence"`
	Statut                string        `json:"statut"`
	DateActivation        time.Time     `json:"date_activation"`
	DateExpiration        *time.Time    `json:"date_expiration,omitempty"`
	SyncInitialComplete   bool          `json:"sync_initial_complete"`
	DateSyncInitial       *time.Time    `json:"date_sync_initial,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
	CreatedBy             uuid.UUID     `json:"created_by"`
	
	// Modules autorisés
	ModulesAutorises      []ModuleInfo  `json:"modules_autorises"`
	NombreModules         int           `json:"nombre_modules"`
	
	// Informations établissement (dénormalisées)
	EtablissementNom      string        `json:"etablissement_nom"`
	EtablissementCode     string        `json:"etablissement_code"`
	EtablissementStatut   string        `json:"etablissement_statut"`
	
	// Statut calculé
	StatutCalcule         string        `json:"statut_calcule"`     // "actif", "expire", "bientot_expire"
	JoursAvantExpiration  *int          `json:"jours_avant_expiration,omitempty"`
	EstExpire             bool          `json:"est_expire"`
	EstBientotExpire      bool          `json:"est_bientot_expire"` // < 30 jours
}

// LicenseHistoryEntry - Entrée de l'historique d'une licence
type LicenseHistoryEntry struct {
	ID                uuid.UUID  `json:"id"`
	TypeEvenement     string     `json:"type_evenement"`
	StatutPrecedent   *string    `json:"statut_precedent,omitempty"`
	StatutNouveau     string     `json:"statut_nouveau"`
	MotifChangement   string     `json:"motif_changement"`
	UtilisateurAction *uuid.UUID `json:"utilisateur_action,omitempty"`
	IPAction          *string    `json:"ip_action,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// LicenseConsultationResult - Résultat complet pour la consultation d'une licence
type LicenseConsultationResult struct {
	License   *LicenseDetailedResponse  `json:"license"`
	History   []LicenseHistoryEntry     `json:"history,omitempty"`
	Message   string                    `json:"message"`
}

// LicenseListResponse - Liste des licences d'un établissement
type LicenseListResponse struct {
	Licenses []LicenseSummary `json:"licenses"`
	Total    int              `json:"total"`
}

// LicenseSummary - Résumé d'une licence pour les listes
type LicenseSummary struct {
	ID                    uuid.UUID  `json:"id"`
	ModeDeploiement       string     `json:"mode_deploiement"`
	TypeLicence           string     `json:"type_licence"`
	Statut                string     `json:"statut"`
	StatutCalcule         string     `json:"statut_calcule"`
	DateActivation        time.Time  `json:"date_activation"`
	DateExpiration        *time.Time `json:"date_expiration,omitempty"`
	NombreModules         int        `json:"nombre_modules"`
	EstExpire             bool       `json:"est_expire"`
	JoursAvantExpiration  *int       `json:"jours_avant_expiration,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}