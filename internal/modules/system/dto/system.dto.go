package dto

import (
	"time"
)

// EstablishmentInfoDTO représente les informations de l'établissement
type EstablishmentInfoDTO struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	Nom            string    `json:"nom"`
	NomCourt       string    `json:"nom_court"`
	Ville          string    `json:"ville"`
	SetupTermine   bool      `json:"setup_termine"`
	SetupEtape     int       `json:"setup_etape"`
	CreatedAt      time.Time `json:"created_at"`
}

// LicenseInfoDTO représente les informations de licence
type LicenseInfoDTO struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	ModeDeploiement  string     `json:"mode_deploiement"`
	Statut           string     `json:"statut"` // actif, expiree, revoquee, non_configuree
	DateActivation   *time.Time `json:"date_activation,omitempty"`
	DateExpiration   *time.Time `json:"date_expiration,omitempty"`
	JoursRestants    *int       `json:"jours_restants,omitempty"`
	ModulesAutorises []string   `json:"modules_autorises"`
}

// ModuleInfoDTO représente un module disponible
type ModuleInfoDTO struct {
	CodeModule        string `json:"code_module"`
	NomStandard       string `json:"nom_standard"`
	EstMedical        bool   `json:"est_medical"`
	PeutPrendreTicket bool   `json:"peut_prendre_ticket"`
	EstActif          bool   `json:"est_actif"`
}

// AuthorizedModuleDTO représente un module autorisé pour la navigation
type AuthorizedModuleDTO struct {
	CodeModule           string `json:"code_module"`
	NomStandard          string `json:"nom_standard"`
	EstMedical           bool   `json:"est_medical"`
	PeutPrendreTicket    bool   `json:"peut_prendre_ticket"`
	EstModuleBackOffice  bool   `json:"est_module_back_office"`
}

// SystemConfigurationDTO représente la configuration système
type SystemConfigurationDTO struct {
	DureeValiditeTicketJours int     `json:"duree_validite_ticket_jours"`
	NbSouchesParCaisse       int     `json:"nb_souches_par_caisse"`
	GardeActive              bool    `json:"garde_active"`
	GardeHeureDebut          *string `json:"garde_heure_debut"`
	GardeHeureFin            *string `json:"garde_heure_fin"`
}

// SystemInfoResponse représente la réponse complète de /api/v1/system/info
type SystemInfoResponse struct {
	Etablissement        EstablishmentInfoDTO      `json:"etablissement"`
	Licence              LicenseInfoDTO            `json:"licence"`
	ModulesDisponibles   []ModuleInfoDTO           `json:"modules_disponibles"`
	Configuration        SystemConfigurationDTO   `json:"configuration"`
}

// StandardAPIResponse représente la structure standard des réponses API
type StandardAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Alertes []AlerteDTO `json:"alertes,omitempty"`
}

// AlerteDTO représente une alerte système
type AlerteDTO struct {
	Type    string                 `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// AuthorizedModulesResponse représente la réponse pour /api/v1/system/modules/authorized
type AuthorizedModulesResponse struct {
	ModulesAutorises []AuthorizedModuleDTO `json:"modules_autorises"`
}

// ActivateOfflineRequest représente la requête POST /api/v1/system/activated
type ActivateOfflineRequest struct {
	CodeEtablissement string `json:"code_etablissement" validate:"required"`
}

// ActivateOfflineResponse représente la réponse POST /api/v1/system/activated
type ActivateOfflineResponse struct {
	SynchronisationEffectuee bool   `json:"synchronisation_effectuee"`
	EtablissementExiste      bool   `json:"etablissement_existe"`
	Message                  string `json:"message"`
}

// SyncOfflineRequest représente la requête POST /api/v1/system/offline/synchronised
type SyncOfflineRequest struct {
	AppInstance string `json:"app_instance" validate:"required"`
}

// SyncOfflineResponse représente la réponse POST /api/v1/system/offline/synchronised
type SyncOfflineResponse struct {
	Etablissement EstablissementSyncDTO `json:"etablissement"`
	Licence       LicenceSyncDTO        `json:"licence"`
	Modules       []ModuleSyncDTO       `json:"modules"`
	SuperAdmin    SuperAdminSyncDTO     `json:"super_admin"`
	SyncMetadata  SyncMetadataDTO       `json:"sync_metadata"`
}

// EstablissementSyncDTO pour la synchronisation
type EstablissementSyncDTO struct {
	ID                         string `json:"id"`
	AppInstance                string `json:"app_instance"`
	EtablissementCode          string `json:"etablissement_code"`
	Nom                        string `json:"nom"`
	NomCourt                   string `json:"nom_court"`
	AdresseComplete            string `json:"adresse_complete"`
	Ville                      string `json:"ville"`
	Commune                    string `json:"commune"`
	TelephonePrincipal         string `json:"telephone_principal"`
	Email                      string `json:"email"`
	DureeValiditeTicketJours   int    `json:"duree_validite_ticket_jours"`
	NbSouchesParCaisse         int    `json:"nb_souches_par_caisse"`
	GardeHeureDebut            string `json:"garde_heure_debut"`
	GardeHeureFin              string `json:"garde_heure_fin"`
	CreatedAt                  string `json:"created_at"`
}

// LicenceSyncDTO pour la synchronisation
type LicenceSyncDTO struct {
	ID                     string   `json:"id"`
	EtablissementID        string   `json:"etablissement_id"`
	ModeDeploiement        string   `json:"mode_deploiement"`
	TypeLicence            string   `json:"type_licence"`
	ModulesAutorises       []string `json:"modules_autorises"`
	DateActivation         string   `json:"date_activation"`
	DateExpiration         *string  `json:"date_expiration"`
	Statut                 string   `json:"statut"`
	SyncInitialComplete    bool     `json:"sync_initial_complete"`
	CreatedAt              string   `json:"created_at"`
}

// RubriqueDTO pour les rubriques des modules
type RubriqueDTO struct {
	ID             string `json:"id"`
	CodeRubrique   string `json:"code_rubrique"`
	Nom            string `json:"nom"`
	Description    string `json:"description"`
	OrdreAffichage int    `json:"ordre_affichage"`
	EstObligatoire bool   `json:"est_obligatoire"`
	EstActif       bool   `json:"est_actif"`
}

// ModuleSyncDTO pour la synchronisation avec rubriques
type ModuleSyncDTO struct {
	ID                    string        `json:"id"`
	NumeroModule          int           `json:"numero_module"`
	CodeModule            string        `json:"code_module"`
	NomStandard           string        `json:"nom_standard"`
	Description           string        `json:"description"`
	EstMedical            bool          `json:"est_medical"`
	EstObligatoire        bool          `json:"est_obligatoire"`
	EstActif              bool          `json:"est_actif"`
	EstModuleBackOffice   bool          `json:"est_module_back_office"`
	PeutPrendreTicket     bool          `json:"peut_prendre_ticket"`
	Rubriques             []RubriqueDTO `json:"rubriques"`
}

// SuperAdminSyncDTO pour l'administrateur
type SuperAdminSyncDTO struct {
	ID                  string `json:"id"`
	EtablissementID     string `json:"etablissement_id"`
	Identifiant         string `json:"identifiant"`
	Nom                 string `json:"nom"`
	Prenoms             string `json:"prenoms"`
	Telephone           string `json:"telephone"`
	PasswordHash        string `json:"password_hash"`
	Salt                string `json:"salt"`
	MustChangePassword  bool   `json:"must_change_password"`
	EstAdmin            bool   `json:"est_admin"`
	TypeAdmin           string `json:"type_admin"`
	EstAdminTir         bool   `json:"est_admin_tir"`
	EstTemporaire       bool   `json:"est_temporaire"`
	Statut              string `json:"statut"`
	CreatedAt           string `json:"created_at"`
}

// SyncMetadataDTO pour les métadonnées de synchronisation
type SyncMetadataDTO struct {
	SyncTimestamp   string `json:"sync_timestamp"`
	TotalModules    int    `json:"total_modules"`
	TotalRubriques  int    `json:"total_rubriques"`
	VersionDonnees  string `json:"version_donnees"`
}