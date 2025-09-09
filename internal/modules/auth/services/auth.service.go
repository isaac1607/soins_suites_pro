package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/auth/dto"
	"soins-suite-core/internal/modules/auth/queries"
	"soins-suite-core/internal/shared/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthService struct {
	db             *postgres.Client
	redisClient    *redis.Client
	sessionService *SessionService
	permService    *PermissionService
}

// NewAuthService crée une nouvelle instance du service d'authentification
func NewAuthService(
	db *postgres.Client,
	redisClient *redis.Client,
	sessionService *SessionService,
	permService *PermissionService,
) *AuthService {
	return &AuthService{
		db:             db,
		redisClient:    redisClient,
		sessionService: sessionService,
		permService:    permService,
	}
}

// Login authentifie un utilisateur et crée une session
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, establishmentID, establishmentCode, clientType, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	// 1. Vérifier le rate limiting
	if err := s.checkRateLimit(ctx, establishmentCode, req.Identifiant); err != nil {
		return nil, err
	}

	// 2. Récupérer l'utilisateur
	var user struct {
		ID                 string
		Identifiant        string
		Nom                string
		Prenoms            string
		Telephone          string
		PasswordHash       string
		Salt               string
		EstAdmin           bool
		TypeAdmin          sql.NullString
		EstAdminTir        bool
		MustChangePassword bool
		EstMedecin         bool
		RoleMetier         sql.NullString
		Statut             string
		EtablissementCode  string
	}

	row := s.db.QueryRow(ctx, queries.UserQueries.GetByIdentifiant, req.Identifiant, establishmentID)
	err := row.Scan(
		&user.ID, &user.Identifiant, &user.Nom, &user.Prenoms, &user.Telephone,
		&user.PasswordHash, &user.Salt, &user.EstAdmin, &user.TypeAdmin,
		&user.EstAdminTir, &user.MustChangePassword, &user.EstMedecin,
		&user.RoleMetier, &user.Statut, &user.EtablissementCode,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Cas normal : utilisateur non trouvé - incrémenter rate limiting
			s.incrementFailedAttempt(ctx, establishmentCode, req.Identifiant)
			return nil, dto.NewAuthError("INVALID_CREDENTIALS", "Identifiant ou mot de passe incorrect", nil)
		}

		// Erreur technique de base de données (schéma, connexion, etc.)
		// Ne pas incrémenter le rate limiting car ce n'est pas une tentative malveillante
		log.Printf("Database error during login for user %s: %v", req.Identifiant, err)
		return nil, fmt.Errorf("erreur technique lors de la récupération de l'utilisateur: %w", err)
	}

	// 3. Vérifier le mot de passe
	if !utils.VerifyPasswordSHA512(req.Password, user.Salt, user.PasswordHash) {
		s.incrementFailedAttempt(ctx, establishmentCode, req.Identifiant)
		return nil, dto.NewAuthError("INVALID_CREDENTIALS", "Identifiant ou mot de passe incorrect", nil)
	}

	// 4. Vérifier la cohérence client type vs est_admin
	if err := s.validateClientTypeCoherence(clientType, user.EstAdmin); err != nil {
		return nil, err
	}

	// 5. Générer le token de session
	token := uuid.New().String()
	expiresAt := time.Now().Add(time.Hour)

	// 6. Créer la session
	sessionData := &dto.SessionData{
		UserID:            user.ID,
		EtablissementID:   establishmentID,
		EtablissementCode: establishmentCode,
		ClientType:        clientType,
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		CreatedAt:         time.Now().Format(time.RFC3339),
		LastActivity:      time.Now().Format(time.RFC3339),
		ExpiresAt:         expiresAt.Format(time.RFC3339),
	}

	if err := s.sessionService.CreateSession(ctx, token, sessionData); err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la session: %w", err)
	}

	// 7. Récupérer et cacher les permissions
	var permissions []dto.Permission
	if user.EstAdmin && user.TypeAdmin.Valid && user.TypeAdmin.String == "super_admin" && clientType == "back-office" {
		// Super admin back-office : récupérer tous les modules back-office
		permissions, err = s.permService.GetSuperAdminPermissions(ctx, establishmentCode, user.ID)
	} else {
		// Utilisateur normal : récupérer ses permissions spécifiques
		permissions, err = s.permService.GetUserPermissions(ctx, user.ID, establishmentID, establishmentCode)
	}

	if err != nil {
		// Nettoyer la session créée en cas d'erreur
		s.sessionService.DeleteSession(ctx, token, establishmentCode, user.ID)
		return nil, fmt.Errorf("erreur lors de la récupération des permissions: %w", err)
	}

	// 8. Construire les données utilisateur
	userData := dto.UserData{
		ID:                 user.ID,
		Identifiant:        user.Identifiant,
		Nom:                user.Nom,
		Prenoms:            user.Prenoms,
		Telephone:          user.Telephone,
		EstAdmin:           user.EstAdmin,
		TypeAdmin:          nil,
		EstAdminTir:        user.EstAdminTir,
		MustChangePassword: user.MustChangePassword,
		EstMedecin:         user.EstMedecin,
		RoleMetier:         nil,
	}

	if user.TypeAdmin.Valid {
		userData.TypeAdmin = &user.TypeAdmin.String
	}
	if user.RoleMetier.Valid {
		userData.RoleMetier = &user.RoleMetier.String
	}

	// 9. Récupérer les données setup si back-office
	var setupData *dto.SetupData
	if clientType == "back-office" {
		setupData, _ = s.getSetupState(ctx, establishmentID)
	}

	// 10. Nettoyer le compteur de rate limiting en cas de succès
	s.clearRateLimit(ctx, establishmentCode, req.Identifiant)

	// 11. Construire la réponse
	response := &dto.LoginResponse{
		Token:       token,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		FrontOffice: !user.EstAdmin,
		BackOffice:  user.EstAdmin,
		User:        userData,
		Permissions: permissions,
		Setup:       setupData,
	}

	return response, nil
}

// Logout révoque une session utilisateur
func (s *AuthService) Logout(ctx context.Context, token, establishmentCode, userID string) error {
	return s.sessionService.DeleteSession(ctx, token, establishmentCode, userID)
}

// LogoutByToken révoque une session uniquement par token (idempotent selon spécifications)
func (s *AuthService) LogoutByToken(ctx context.Context, token, establishmentCode string) error {
	// 1. Essayer de récupérer la session pour obtenir l'userID et les infos d'audit
	session, err := s.sessionService.GetSession(ctx, token, establishmentCode)

	var userID string
	var logoutInfo map[string]interface{}

	if err == nil && session != nil {
		userID = session.UserID
		// Informations pour l'audit
		logoutInfo = map[string]interface{}{
			"user_id":            session.UserID,
			"establishment_code": establishmentCode,
			"client_type":        session.ClientType,
			"session_duration":   s.calculateSessionDuration(session.CreatedAt),
			"ip_address":         session.IPAddress,
			"user_agent":         session.UserAgent,
		}
	} else {
		// Session déjà expirée/inexistante - logout idempotent
		logoutInfo = map[string]interface{}{
			"establishment_code": establishmentCode,
			"session_status":     "already_invalid",
		}
	}

	// 2. Log de l'événement logout avant suppression
	s.logLogoutEvent(userID, establishmentCode, logoutInfo)

	// 3. Effectuer le logout complet selon les spécifications Redis
	return s.sessionService.DeleteSessionIdempotent(ctx, token, establishmentCode, userID)
}

// ChangePassword change le mot de passe d'un utilisateur
func (s *AuthService) ChangePassword(ctx context.Context, userID, establishmentID string, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error) {
	// 1. Validation des mots de passe
	if req.NewPassword != req.ConfirmPassword {
		return nil, dto.NewAuthError("PASSWORD_MISMATCH", "Les mots de passe ne correspondent pas", nil)
	}

	// 2. Commencer une transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du démarrage de la transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 3. Récupérer l'utilisateur pour vérifier le mot de passe actuel
	var user struct {
		ID           string
		PasswordHash string
		Salt         string
	}

	row := tx.QueryRow(ctx, `
		SELECT id::text, password_hash, salt 
		FROM user_utilisateur 
		WHERE id = $1::uuid AND etablissement_id = $2::uuid AND statut = 'actif'
		FOR UPDATE
	`, userID, establishmentID)

	err = row.Scan(&user.ID, &user.PasswordHash, &user.Salt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dto.NewAuthError("USER_NOT_FOUND", "Utilisateur non trouvé", nil)
		}
		return nil, fmt.Errorf("erreur lors de la récupération de l'utilisateur: %w", err)
	}

	// 3. Vérifier le mot de passe actuel
	if !utils.VerifyPasswordSHA512(req.CurrentPassword, user.Salt, user.PasswordHash) {
		return nil, dto.NewAuthError("INVALID_CURRENT_PASSWORD", "Mot de passe actuel incorrect", nil)
	}

	// 4. Générer nouveau hash et salt
	newSalt, _ := utils.GenerateSalt()
	newPasswordHash := utils.HashPasswordSHA512(req.NewPassword, newSalt)

	// 5. Mettre à jour en base
	var updatedUserID string
	var mustChangePassword bool
	var passwordChangedAt *time.Time

	err = tx.QueryRow(ctx, queries.UserQueries.ChangePassword,
		newPasswordHash, newSalt, userID, establishmentID).Scan(
		&updatedUserID, &mustChangePassword, &passwordChangedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dto.NewAuthError("USER_NOT_FOUND", "Utilisateur non trouvé ou inactif", nil)
		}
		return nil, fmt.Errorf("erreur lors du changement de mot de passe: %w", err)
	}

	// 6. Valider la transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("erreur lors de la validation de la transaction: %w", err)
	}

	return &dto.ChangePasswordResponse{
		Success:            true,
		Message:            "Mot de passe changé avec succès",
		MustChangePassword: mustChangePassword,
	}, nil
}

// GetCurrentUser récupère les informations de l'utilisateur courant
func (s *AuthService) GetCurrentUser(ctx context.Context, token, establishmentCode string) (*dto.MeResponse, error) {
	// Récupérer la session
	session, err := s.sessionService.GetSession(ctx, token, establishmentCode)
	if err != nil {
		return nil, err
	}

	// Déléguer à la méthode optimisée
	return s.GetCurrentUserByID(ctx, session.UserID, session.EtablissementID, establishmentCode)
}

// GetCurrentUserByID récupère les informations utilisateur par ID (optimisé pour /me avec SessionMiddleware)
func (s *AuthService) GetCurrentUserByID(ctx context.Context, userID, establishmentID, establishmentCode string) (*dto.MeResponse, error) {
	// Récupérer les données utilisateur par ID directement (pas besoin de valider le token à nouveau)
	var user struct {
		ID                 string
		Identifiant        string
		Nom                string
		Prenoms            string
		Telephone          string
		EstAdmin           bool
		TypeAdmin          sql.NullString
		EstAdminTir        bool
		MustChangePassword bool
		EstMedecin         bool
		RoleMetier         sql.NullString
	}

	row := s.db.QueryRow(ctx, `
		SELECT 
			id, identifiant, nom, prenoms, telephone, 
			est_admin, type_admin, est_admin_tir, must_change_password,
			est_medecin, role_metier
		FROM user_utilisateur 
		WHERE id = $1 AND etablissement_id = $2 AND statut = 'actif'
	`, userID, establishmentID)

	err := row.Scan(
		&user.ID, &user.Identifiant, &user.Nom, &user.Prenoms, &user.Telephone,
		&user.EstAdmin, &user.TypeAdmin, &user.EstAdminTir, &user.MustChangePassword,
		&user.EstMedecin, &user.RoleMetier,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dto.NewAuthError("USER_NOT_FOUND", "Utilisateur non trouvé", nil)
		}
		return nil, fmt.Errorf("erreur lors de la récupération des données utilisateur: %w", err)
	}

	// Construire les données utilisateur complètes
	userData := dto.UserData{
		ID:                 user.ID,
		Identifiant:        user.Identifiant,
		Nom:                user.Nom,
		Prenoms:            user.Prenoms,
		Telephone:          user.Telephone,
		EstAdmin:           user.EstAdmin,
		TypeAdmin:          nil,
		EstAdminTir:        user.EstAdminTir,
		MustChangePassword: user.MustChangePassword,
		EstMedecin:         user.EstMedecin,
		RoleMetier:         nil,
	}

	if user.TypeAdmin.Valid {
		userData.TypeAdmin = &user.TypeAdmin.String
	}
	if user.RoleMetier.Valid {
		userData.RoleMetier = &user.RoleMetier.String
	}

	// Récupérer les permissions complètes avec cache
	var permissions []dto.Permission
	if user.EstAdmin && user.TypeAdmin.Valid && user.TypeAdmin.String == "super_admin" {
		// Super admin : récupérer tous les modules back-office
		permissions, err = s.permService.GetSuperAdminPermissions(ctx, establishmentCode, user.ID)
	} else {
		// Utilisateur normal : récupérer ses permissions spécifiques
		permissions, err = s.permService.GetUserPermissions(ctx, user.ID, establishmentID, establishmentCode)
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des permissions: %w", err)
	}

	// Récupérer les informations de session courante depuis Redis (pour /me)
	sessionInfo, err := s.getCurrentSessionInfo(ctx, userID, establishmentCode)
	if err != nil {
		// Session info non critique pour /me, on utilise des valeurs par défaut
		sessionInfo = dto.SessionInfo{
			Token:      "current-session",
			ExpiresAt:  time.Now().Add(time.Hour).Format(time.RFC3339),
			ClientType: "unknown",
		}
	}

	// Construire la réponse complète
	response := &dto.MeResponse{
		User:        userData,
		Permissions: permissions,
		Session:     sessionInfo,
	}

	return response, nil
}

// checkRateLimit vérifie les tentatives de connexion
func (s *AuthService) checkRateLimit(ctx context.Context, establishmentCode, identifiant string) error {
	key := fmt.Sprintf("soins_suite_%s_auth_ratelimit:%s", establishmentCode, identifiant)

	val, err := s.redisClient.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		// Si Redis est indisponible, on continue sans rate limiting
		return nil
	}

	if val != "" {
		// Parser la valeur
		var attempts int
		fmt.Sscanf(val, "%d", &attempts)

		if attempts >= 5 {
			ttl, _ := s.redisClient.Client().TTL(ctx, key).Result()
			return dto.NewAuthError("RATE_LIMIT_EXCEEDED", "Trop de tentatives de connexion", map[string]interface{}{
				"retry_after_seconds": int(ttl.Seconds()),
			})
		}
	}

	return nil
}

// incrementFailedAttempt incrémente le compteur d'échecs
func (s *AuthService) incrementFailedAttempt(ctx context.Context, establishmentCode, identifiant string) {
	key := fmt.Sprintf("soins_suite_%s_auth_ratelimit:%s", establishmentCode, identifiant)

	val := s.redisClient.Client().Incr(ctx, key).Val()
	if val == 1 {
		s.redisClient.Expire(ctx, key, 15*time.Minute)
	}
}

// clearRateLimit nettoie le compteur après succès
func (s *AuthService) clearRateLimit(ctx context.Context, establishmentCode, identifiant string) {
	key := fmt.Sprintf("soins_suite_%s_auth_ratelimit:%s", establishmentCode, identifiant)
	s.redisClient.Del(ctx, key)
}

// validateClientTypeCoherence vérifie la cohérence entre client_type et est_admin
func (s *AuthService) validateClientTypeCoherence(clientType string, isAdmin bool) error {
	if clientType == "back-office" && !isAdmin {
		return dto.NewAuthError("CLIENT_TYPE_MISMATCH", "Accès refusé à cette interface", map[string]interface{}{
			"reason": "Compte administrateur requis pour le back-office",
		})
	}

	if clientType == "front-office" && isAdmin {
		return dto.NewAuthError("CLIENT_TYPE_MISMATCH", "Accès refusé à cette interface", map[string]interface{}{
			"reason": "Compte utilisateur standard requis pour le front-office",
		})
	}

	return nil
}

// getSetupState récupère l'état du setup
func (s *AuthService) getSetupState(ctx context.Context, establishmentID string) (*dto.SetupData, error) {
	var estTermine bool
	var etapeActuelle int

	row := s.db.QueryRow(ctx, queries.UserQueries.GetSetupState, establishmentID)
	err := row.Scan(&estTermine, &etapeActuelle)
	if err != nil {
		return nil, err
	}

	return &dto.SetupData{
		EstTermine:    estTermine,
		EtapeActuelle: etapeActuelle,
		TotalEtapes:   5, // Total fixe selon les spécifications
	}, nil
}

// calculateSessionDuration calcule la durée d'une session
func (s *AuthService) calculateSessionDuration(createdAt string) string {
	if createdAt == "" {
		return "unknown"
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "invalid_format"
	}

	duration := time.Since(created)
	return fmt.Sprintf("%.0fs", duration.Seconds())
}

// logLogoutEvent enregistre un événement de logout pour audit
func (s *AuthService) logLogoutEvent(userID, establishmentCode string, info map[string]interface{}) {
	// Log structuré selon les conventions du projet (simple fmt.Printf pour MVP)
	// En production, ceci devrait utiliser un logger structuré (slog, logrus, etc.)

	logData := make(map[string]interface{})
	logData["event"] = "auth.logout"
	logData["timestamp"] = time.Now().Format(time.RFC3339)
	logData["user_id"] = userID
	logData["establishment_code"] = establishmentCode

	// Ajouter les informations supplémentaires
	for k, v := range info {
		logData[k] = v
	}

	// Log simple pour MVP (selon les conventions du projet)
	if userID != "" {
		log.Printf("[AUTH-LOGOUT] user_id=%s establishment=%s client_type=%v session_duration=%v",
			userID,
			establishmentCode,
			info["client_type"],
			info["session_duration"])
	} else {
		log.Printf("[AUTH-LOGOUT] establishment=%s status=%v",
			establishmentCode,
			info["session_status"])
	}
}

// getCurrentSessionInfo récupère les informations de la session courante (optimisé pour /me)
func (s *AuthService) getCurrentSessionInfo(ctx context.Context, userID, establishmentCode string) (dto.SessionInfo, error) {
	// Récupérer les sessions actives de l'utilisateur depuis Redis
	userSessionsKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", establishmentCode, userID)

	tokens, err := s.redisClient.Client().SMembers(ctx, userSessionsKey).Result()
	if err != nil || len(tokens) == 0 {
		return dto.SessionInfo{}, fmt.Errorf("aucune session active trouvée")
	}

	// Prendre la première session active (l'utilisateur peut avoir plusieurs sessions)
	// En production, on pourrait identifier la session courante par le token dans le contexte
	token := tokens[0]
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)

	sessionData := s.redisClient.Client().HGetAll(ctx, sessionKey).Val()
	if len(sessionData) == 0 {
		return dto.SessionInfo{}, fmt.Errorf("session non trouvée")
	}

	return dto.SessionInfo{
		Token:      token,
		ExpiresAt:  sessionData["expires_at"],
		ClientType: sessionData["client_type"],
	}, nil
}
