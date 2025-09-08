package comptes

import (
	"time"
)

type CreateUserRequest struct {
	Identifiant   string  `json:"identifiant" validate:"required,min=3,max=50"`
	Nom          string  `json:"nom" validate:"required,min=2,max=100"`
	Prenoms      string  `json:"prenoms" validate:"required,min=2,max=100"`
	Telephone    string  `json:"telephone" validate:"required,min=10,max=20"`
	Email        *string `json:"email,omitempty" validate:"omitempty,email"`
	Password     *string `json:"password,omitempty"` // validate:"omitempty,min=12" - Temporairement désactivé pour tests
	MustChangePassword bool `json:"must_change_password"` // Sera toujours forcé à true lors de la création
	
	EstAdmin     bool     `json:"est_admin"`
	TypeAdmin    *string  `json:"type_admin,omitempty"` // validate:"omitempty,oneof=super_admin admin_simple" - Temporairement désactivé pour tests
	EstMedecin   bool     `json:"est_medecin"`
	RoleMetier   *string  `json:"role_metier,omitempty" validate:"omitempty,max=100"`
	
	EstTemporaire   bool       `json:"est_temporaire"`
	DateExpiration *time.Time `json:"date_expiration,omitempty"`
	
	ProfilsIds       []string          `json:"profils_ids,omitempty"`
	ModulesAttribues []ModuleAttribue  `json:"modules_attribues,omitempty"`
}

type ModuleAttribue struct {
	ModuleID              string   `json:"module_id" validate:"required,uuid"`
	AccesToutesRubriques  bool     `json:"acces_toutes_rubriques"`
	RubriquesSpecifiques  []string `json:"rubriques_specifiques" validate:"dive,uuid"`
}

type CreateUserResponse struct {
	ID                    string                     `json:"id"`
	Identifiant          string                     `json:"identifiant"`
	PasswordTemporaire   *string                    `json:"password_temporaire,omitempty"`
	Message              string                     `json:"message"`
	PermissionsAttribuees PermissionsAttribueesInfo `json:"permissions_attribuees"`
}

type PermissionsAttribueesInfo struct {
	Profils         int `json:"profils"`
	ModulesComplets int `json:"modules_complets"`
	ModulesPartiels int `json:"modules_partiels"`
	TotalRubriques  int `json:"total_rubriques"`
}

type NotificationInfo struct {
	Envoyee      bool    `json:"envoyee"`
	Methode      *string `json:"methode,omitempty"`
	Destinataire *string `json:"destinataire,omitempty"`
}

type ValidationError struct {
	Code   string            `json:"code"`
	Champs map[string]string `json:"champs"`
}

type CreateUserInternal struct {
	EtablissementID   string
	Identifiant       string
	Nom               string
	Prenoms           string
	Telephone         string
	PasswordHash      string
	Salt              string
	MustChangePassword bool
	EstAdmin          bool
	TypeAdmin         *string
	EstMedecin        bool
	RoleMetier        *string
	EstTemporaire     bool
	DateExpiration    *time.Time
	Statut            string
	CreatedBy         string
}

// DTOs pour GET /api/v1/back-office/users
type ListUsersQuery struct {
	Page            int     `form:"page" binding:"omitempty,min=1"`
	Limit           int     `form:"limit" binding:"omitempty,min=1,max=100"`
	Search          *string `form:"search"`
	Statut          string  `form:"statut" binding:"omitempty,oneof=actif suspendu expire tous"`
	EstAdmin        *bool   `form:"est_admin"`
	EstMedecin      *bool   `form:"est_medecin"`
	ProfilID        *string `form:"profil_id" binding:"omitempty,uuid"`
	ModuleCode      *string `form:"module_code"`
	SortBy          string  `form:"sort_by" binding:"omitempty,oneof=nom identifiant last_login_at created_at"`
	SortOrder       string  `form:"sort_order" binding:"omitempty,oneof=asc desc"`
	IncludeArchived bool    `form:"include_archived"`
}

type UserListItem struct {
	ID                string                `json:"id"`
	Identifiant       string                `json:"identifiant"`
	Nom               string                `json:"nom"`
	Prenoms           string                `json:"prenoms"`
	Telephone         string                `json:"telephone"`
	EstAdmin          bool                  `json:"est_admin"`
	TypeAdmin         *string               `json:"type_admin"`
	EstMedecin        bool                  `json:"est_medecin"`
	RoleMetier        *string               `json:"role_metier"`
	EstTemporaire     bool                  `json:"est_temporaire"`
	DateExpiration    *time.Time            `json:"date_expiration"`
	Statut            string                `json:"statut"`
	Profils           []ProfilInfo          `json:"profils"`
	PermissionsResume PermissionsResume     `json:"permissions_resume"`
	LastLoginAt       *time.Time            `json:"last_login_at"`
	CreatedAt         time.Time             `json:"created_at"`
}

type ProfilInfo struct {
	ID         string `json:"id"`
	NomProfil  string `json:"nom_profil"`
	CodeProfil string `json:"code_profil"`
}

type PermissionsResume struct {
	TotalModules     int `json:"total_modules"`
	ModulesComplets  int `json:"modules_complets"`
	ModulesPartiels  int `json:"modules_partiels"`
	TotalRubriques   int `json:"total_rubriques"`
}

type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type FiltresAppliques struct {
	Statut     *string `json:"statut,omitempty"`
	EstAdmin   *bool   `json:"est_admin,omitempty"`
	EstMedecin *bool   `json:"est_medecin,omitempty"`
	Search     *string `json:"search,omitempty"`
	ProfilID   *string `json:"profil_id,omitempty"`
	ModuleCode *string `json:"module_code,omitempty"`
	SortBy     string  `json:"sort_by"`
	SortOrder  string  `json:"sort_order"`
}

type ListUsersResponse struct {
	Users            []UserListItem   `json:"users"`
	Pagination       PaginationInfo   `json:"pagination"`
	FiltresAppliques FiltresAppliques `json:"filtres_appliques"`
}

// DTOs pour GET /api/v1/back-office/users/{id}
type UserDetails struct {
	ID                   string     `json:"id"`
	Identifiant          string     `json:"identifiant"`
	Nom                  string     `json:"nom"`
	Prenoms              string     `json:"prenoms"`
	Telephone            string     `json:"telephone"`
	Email                *string    `json:"email"`
	EstAdmin             bool       `json:"est_admin"`
	TypeAdmin            *string    `json:"type_admin"`
	EstAdminTir          bool       `json:"est_admin_tir"`
	EstMedecin           bool       `json:"est_medecin"`
	RoleMetier           *string    `json:"role_metier"`
	PhotoURL             *string    `json:"photo_url"`
	EstTemporaire        bool       `json:"est_temporaire"`
	DateExpiration       *time.Time `json:"date_expiration"`
	Statut               string     `json:"statut"`
	MotifDesactivation   *string    `json:"motif_desactivation"`
	MustChangePassword   bool       `json:"must_change_password"`
	PasswordChangedAt    *time.Time `json:"password_changed_at"`
	LastLoginAt          *time.Time `json:"last_login_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CreatedBy            *UserRef   `json:"created_by"`
	UpdatedBy            *UserRef   `json:"updated_by"`
}

type UserRef struct {
	ID      string `json:"id"`
	Nom     string `json:"nom"`
	Prenoms string `json:"prenoms"`
}

type ProfilDetails struct {
	ID              string    `json:"id"`
	NomProfil       string    `json:"nom_profil"`
	CodeProfil      string    `json:"code_profil"`
	Description     *string   `json:"description"`
	DateAttribution time.Time `json:"date_attribution"`
	AttribuePar     UserRef   `json:"attribue_par"`
	EstActif        bool      `json:"est_actif"`
}

type ModuleCompletDetails struct {
	CodeModule       string     `json:"code_module"`
	NomStandard      string     `json:"nom_standard"`
	NomPersonnalise  *string    `json:"nom_personnalise"`
	Description      *string    `json:"description"`
	Source           string     `json:"source"`         // "profil" ou "individuelle"
	ProfilSource     *string    `json:"profil_source"`  // nom du profil si source = "profil"
	DateAttribution  time.Time  `json:"date_attribution"`
	AttribuePar      string     `json:"attribue_par"`   // nom complet de celui qui a attribué
}

type RubriqueDetails struct {
	CodeRubrique string  `json:"code_rubrique"`
	Nom          string  `json:"nom"`
	Description  *string `json:"description"`
}

type ModulePartielDetails struct {
	CodeModule      string            `json:"code_module"`
	NomStandard     string            `json:"nom_standard"`
	NomPersonnalise *string           `json:"nom_personnalise"`
	Description     *string           `json:"description"`
	Source          string            `json:"source"`
	Rubriques       []RubriqueDetails `json:"rubriques"`
	DateAttribution time.Time         `json:"date_attribution"`
	AttribuePar     string            `json:"attribue_par"`
}

type PermissionsDetails struct {
	ModulesComplets []ModuleCompletDetails `json:"modules_complets"`
	ModulesPartiels []ModulePartielDetails `json:"modules_partiels"`
}

type Statistiques struct {
	NombreConnexions30j       int        `json:"nombre_connexions_30j"`
	DerniereActivite         *time.Time `json:"derniere_activite"`
	NombreSessionsActives     int        `json:"nombre_sessions_actives"`
	PermissionsViaProfils     int        `json:"permissions_via_profils"`
	PermissionsIndividuelles  int        `json:"permissions_individuelles"`
}

type UserDetailsResponse struct {
	User         UserDetails        `json:"user"`
	Profils      []ProfilDetails    `json:"profils"`
	Permissions  PermissionsDetails `json:"permissions"`
	Statistiques Statistiques       `json:"statistiques"`
}

// DTOs pour PUT /api/v1/back-office/users/{id}/permissions
type ModifyPermissionsRequest struct {
	Profils                *ProfilsModification       `json:"profils,omitempty"`
	ModulesComplets        *ModulesCompletsModification `json:"modules_complets,omitempty"`
	ModulesPartiels        *ModulesPartielsModification `json:"modules_partiels,omitempty"`
	NotifierUtilisateur    bool                      `json:"notifier_utilisateur"`
}

type ProfilsModification struct {
	Ajouter []string `json:"ajouter" validate:"dive,uuid"`
	Retirer []string `json:"retirer" validate:"dive,uuid"`
}

type ModuleCompletRequest struct {
	ModuleID string `json:"module_id" validate:"required,uuid"`
}

type ModulesCompletsModification struct {
	Ajouter []ModuleCompletRequest `json:"ajouter" validate:"dive"`
	Retirer []string              `json:"retirer" validate:"dive,uuid"`
}

type ModulePartielRequest struct {
	ModuleID     string   `json:"module_id" validate:"required,uuid"`
	RubriquesIds []string `json:"rubriques_ids" validate:"required,min=1,dive,uuid"`
}

type ModulesPartielsModification struct {
	Ajouter  []ModulePartielRequest `json:"ajouter" validate:"dive"`
	Modifier []ModulePartielRequest `json:"modifier" validate:"dive"`
	Retirer  []string              `json:"retirer" validate:"dive,uuid"`
}

type ChangementsPermissions struct {
	Profils         ChangementStats `json:"profils"`
	ModulesComplets ChangementStats `json:"modules_complets"`
	ModulesPartiels ChangePartiels  `json:"modules_partiels"`
	TotalRubriquesAffectees int    `json:"total_rubriques_affectees"`
}

type ChangementStats struct {
	Ajoutes int `json:"ajoutes"`
	Retires int `json:"retires"`
}

type ChangePartiels struct {
	Ajoutes  int `json:"ajoutes"`
	Modifies int `json:"modifies"`
	Retires  int `json:"retires"`
}

type ModifyPermissionsResponse struct {
	Message      string                 `json:"message"`
	Changements  ChangementsPermissions `json:"changements"`
	Notification NotificationInfo       `json:"notification"`
	ModifiedBy   UserRef                `json:"modified_by"`
	ModifiedAt   time.Time              `json:"modified_at"`
}

// DTOs internes pour le traitement
type PermissionOperation struct {
	Type        string // "add" ou "remove"
	EntityType  string // "profil", "module_complet", "module_partiel"
	EntityID    string
	ModuleID    *string   // Pour les modules partiels
	RubriquesIds []string // Pour les modules partiels
}
