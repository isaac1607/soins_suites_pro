package dto

import (
	"time"
)

// LoginTIRRequest structure pour la connexion admin TIR
type LoginTIRRequest struct {
	Identifiant string `json:"identifiant" validate:"required,min=3,max=255"`
	Password    string `json:"password" validate:"required,min=6"`
}

// AdminTIRInfo informations de l'admin TIR pour la session Redis et réponse API
type AdminTIRInfo struct {
	AdminID                          string `json:"admin_id"`
	Identifiant                     string `json:"identifiant"`
	NiveauAdmin                     string `json:"niveau_admin"`
	PeutGererLicences               bool   `json:"peut_gerer_licences"`
	PeutGererEtablissements         bool   `json:"peut_gerer_etablissements"`
	PeutAccederDonneesEtablissement bool   `json:"peut_acceder_donnees_etablissement"`
	PeutGererAdminsGlobaux          bool   `json:"peut_gerer_admins_globaux"`
}

// LoginTIRResponse réponse de connexion réussie
type LoginTIRResponse struct {
	Token     string       `json:"token"`
	Admin     AdminTIRInfo `json:"admin"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// RefreshTIRResponse réponse de refresh de token
type RefreshTIRResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionTIRData données de session pour Redis
type SessionTIRData struct {
	AdminID                          string    `json:"admin_id"`
	Identifiant                     string    `json:"identifiant"`
	NiveauAdmin                     string    `json:"niveau_admin"`
	PeutGererLicences               string    `json:"peut_gerer_licences"`               // Redis stocke en string
	PeutGererEtablissements         string    `json:"peut_gerer_etablissements"`         // Redis stocke en string
	PeutAccederDonneesEtablissement string    `json:"peut_acceder_donnees_etablissement"` // Redis stocke en string
	PeutGererAdminsGlobaux          string    `json:"peut_gerer_admins_globaux"`          // Redis stocke en string
	IPAddress                       string    `json:"ip_address"`
	UserAgent                       string    `json:"user_agent"`
	CreatedAt                       time.Time `json:"created_at"`
	LastActivity                    time.Time `json:"last_activity"`
	ExpiresAt                       time.Time `json:"expires_at"`
}

// ToMap convertit SessionTIRData en map pour Redis HSET
func (s *SessionTIRData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"admin_id":                            s.AdminID,
		"identifiant":                         s.Identifiant,
		"niveau_admin":                        s.NiveauAdmin,
		"peut_gerer_licences":                 s.PeutGererLicences,
		"peut_gerer_etablissements":           s.PeutGererEtablissements,
		"peut_acceder_donnees_etablissement":  s.PeutAccederDonneesEtablissement,
		"peut_gerer_admins_globaux":           s.PeutGererAdminsGlobaux,
		"ip_address":                          s.IPAddress,
		"user_agent":                          s.UserAgent,
		"created_at":                          s.CreatedAt.Format(time.RFC3339),
		"last_activity":                       s.LastActivity.Format(time.RFC3339),
		"expires_at":                          s.ExpiresAt.Format(time.RFC3339),
	}
}

// ToAdminInfo convertit SessionTIRData en AdminTIRInfo
func (s *SessionTIRData) ToAdminInfo() AdminTIRInfo {
	return AdminTIRInfo{
		AdminID:                          s.AdminID,
		Identifiant:                     s.Identifiant,
		NiveauAdmin:                     s.NiveauAdmin,
		PeutGererLicences:               s.PeutGererLicences == "true",
		PeutGererEtablissements:         s.PeutGererEtablissements == "true",
		PeutAccederDonneesEtablissement: s.PeutAccederDonneesEtablissement == "true",
		PeutGererAdminsGlobaux:          s.PeutGererAdminsGlobaux == "true",
	}
}

// TIRSessionValidation structure pour la validation de session depuis Redis
type TIRSessionValidation struct {
	Valid       bool         `json:"valid"`
	AdminID     string       `json:"admin_id,omitempty"`
	Admin       AdminTIRInfo `json:"admin,omitempty"`
	Token       string       `json:"token,omitempty"`
	ExpiresAt   time.Time    `json:"expires_at,omitempty"`
	ErrorReason string       `json:"error_reason,omitempty"`
}