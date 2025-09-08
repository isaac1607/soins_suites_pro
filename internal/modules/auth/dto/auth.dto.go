package dto

// LoginRequest représente la requête de connexion
type LoginRequest struct {
	Identifiant string `json:"identifiant" validate:"required,min=3,max=50"`
	Password    string `json:"password" validate:"required,min=6"`
}

// LoginResponse représente la réponse de connexion réussie
type LoginResponse struct {
	Token       string       `json:"token"`
	ExpiresAt   string       `json:"expires_at"`
	FrontOffice bool         `json:"front_office"`
	BackOffice  bool         `json:"back_office"`
	User        UserData     `json:"user"`
	Permissions []Permission `json:"permissions"`
	Setup       *SetupData   `json:"setup,omitempty"` // Uniquement pour back-office
}

// UserData représente les informations utilisateur
type UserData struct {
	ID                 string  `json:"id"`
	Identifiant        string  `json:"identifiant"`
	Nom                string  `json:"nom"`
	Prenoms            string  `json:"prenoms"`
	Telephone          string  `json:"telephone"`
	EstAdmin           bool    `json:"est_admin"`
	TypeAdmin          *string `json:"type_admin"`
	EstAdminTir        bool    `json:"est_admin_tir"`
	MustChangePassword bool    `json:"must_change_password"`
	EstMedecin         bool    `json:"est_medecin"`
	RoleMetier         *string `json:"role_metier"`
}

// Permission représente un module avec ses rubriques
type Permission struct {
	ID                  string     `json:"id"`                    // ID du module
	CodeModule          string     `json:"code_module"`
	NomStandard         string     `json:"nom_standard"`
	NomPersonnalise     *string    `json:"nom_personnalise"`
	Description         string     `json:"description"`
	AccesToutesRubriques bool      `json:"acces_toutes_rubriques"` // true = accès complet, false = accès restreint (depuis DB)
	Rubriques           []Rubrique `json:"rubriques"`             // Liste des rubriques (toutes si accès complet, spécifiques sinon)
}

// Rubrique représente une rubrique d'un module
type Rubrique struct {
	CodeRubrique   string `json:"code_rubrique"`
	Nom            string `json:"nom"`
	Description    string `json:"description"`
	OrdreAffichage int    `json:"ordre_affichage"`
}

// SetupData représente l'état du setup (back-office uniquement)
type SetupData struct {
	EstTermine    bool `json:"est_termine"`
	EtapeActuelle int  `json:"etape_actuelle"`
	TotalEtapes   int  `json:"total_etapes"`
}

// LogoutResponse représente la réponse de déconnexion
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MeResponse représente la réponse du endpoint /me
type MeResponse struct {
	User        UserData     `json:"user"`
	Permissions []Permission `json:"permissions"`
	Session     SessionInfo  `json:"session"`
}

// SessionInfo représente les informations de session
type SessionInfo struct {
	Token      string `json:"token"`
	ExpiresAt  string `json:"expires_at"`
	ClientType string `json:"client_type"`
}

// SessionData représente les données de session Redis
type SessionData struct {
	UserID            string `json:"user_id"`
	EtablissementID   string `json:"etablissement_id"`
	EtablissementCode string `json:"etablissement_code"`
	ClientType        string `json:"client_type"`
	IPAddress         string `json:"ip_address"`
	UserAgent         string `json:"user_agent"`
	CreatedAt         string `json:"created_at"`
	LastActivity      string `json:"last_activity"`
	ExpiresAt         string `json:"expires_at"`
}

// ToMap convertit SessionData en map pour Redis HMSET
func (s *SessionData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"user_id":            s.UserID,
		"etablissement_id":   s.EtablissementID,
		"etablissement_code": s.EtablissementCode,
		"client_type":        s.ClientType,
		"ip_address":         s.IPAddress,
		"user_agent":         s.UserAgent,
		"created_at":         s.CreatedAt,
		"last_activity":      s.LastActivity,
		"expires_at":         s.ExpiresAt,
	}
}

// SessionFromMap créé SessionData depuis map Redis
func SessionFromMap(data map[string]string) *SessionData {
	return &SessionData{
		UserID:            data["user_id"],
		EtablissementID:   data["etablissement_id"],
		EtablissementCode: data["etablissement_code"],
		ClientType:        data["client_type"],
		IPAddress:         data["ip_address"],
		UserAgent:         data["user_agent"],
		CreatedAt:         data["created_at"],
		LastActivity:      data["last_activity"],
		ExpiresAt:         data["expires_at"],
	}
}

// AuthError représente les erreurs d'authentification
type AuthError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// NewAuthError crée une nouvelle erreur d'authentification
func NewAuthError(code, message string, details map[string]interface{}) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ChangePasswordRequest représente la demande de changement de mot de passe
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8,max=100"`
}

// ChangePasswordResponse représente la réponse après changement de mot de passe
type ChangePasswordResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	MustChangePassword bool   `json:"must_change_password"`
}
