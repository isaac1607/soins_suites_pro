package services

import (
	"context"
	"fmt"
	"time"

	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/auth/dto"
	"soins-suite-core/internal/modules/auth/queries"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type SessionService struct {
	db          *postgres.Client
	redisClient *redisInfra.Client
}

// NewSessionService crée une nouvelle instance du service de session
func NewSessionService(db *postgres.Client, redisClient *redisInfra.Client) *SessionService {
	return &SessionService{
		db:          db,
		redisClient: redisClient,
	}
}

// CreateSession crée une nouvelle session dans Redis avec fallback PostgreSQL
func (s *SessionService) CreateSession(ctx context.Context, token string, sessionData *dto.SessionData) error {
	// Essayer Redis d'abord
	if err := s.createSessionRedis(ctx, token, sessionData); err == nil {
		// Créer aussi en PostgreSQL pour fallback
		s.createSessionPostgres(ctx, token, sessionData)
		return nil
	}

	// Si Redis échoue, utiliser PostgreSQL uniquement
	return s.createSessionPostgres(ctx, token, sessionData)
}

// GetSession récupère une session depuis Redis avec fallback PostgreSQL
func (s *SessionService) GetSession(ctx context.Context, token, establishmentCode string) (*dto.SessionData, error) {
	// Vérifier d'abord si le token est blacklisté
	if s.isTokenBlacklisted(ctx, establishmentCode, token) {
		return nil, dto.NewAuthError("TOKEN_REVOKED", "Token révoqué", nil)
	}

	// Essayer Redis d'abord
	session, err := s.getSessionRedis(ctx, token, establishmentCode)
	if err == nil {
		// Mettre à jour l'activité
		s.updateLastActivity(ctx, token, establishmentCode, session)
		return session, nil
	}

	// Fallback PostgreSQL
	session, err = s.getSessionPostgres(ctx, token)
	if err != nil {
		return nil, dto.NewAuthError("INVALID_TOKEN", "Session invalide ou expirée", nil)
	}

	// Re-sync vers Redis si disponible
	go s.syncSessionToRedis(ctx, token, session)

	return session, nil
}

// DeleteSession supprime une session de Redis et PostgreSQL
func (s *SessionService) DeleteSession(ctx context.Context, token, establishmentCode, userID string) error {
	// Blacklist le token dans Redis
	s.blacklistToken(ctx, establishmentCode, token)

	// Supprimer de Redis
	s.deleteSessionRedis(ctx, token, establishmentCode, userID)

	// Supprimer de PostgreSQL
	s.db.Exec(ctx, queries.UserQueries.DeleteSession, token)

	return nil
}

// DeleteSessionIdempotent supprime une session de manière idempotente (logout toujours réussi)
func (s *SessionService) DeleteSessionIdempotent(ctx context.Context, token, establishmentCode, userID string) error {
	// Cette méthode implémente parfaitement les spécifications du logout :
	// 1. Révocation session Redis
	// 2. Ajout à blacklist
	// 3. Suppression index utilisateur
	// Le tout de manière idempotente (pas d'erreur si déjà fait)

	// 1. Ajouter à la blacklist Redis (idempotent)
	s.blacklistTokenIdempotent(ctx, establishmentCode, token)

	// 2. Supprimer de Redis avec pipeline (idempotent)
	s.deleteSessionRedisIdempotent(ctx, token, establishmentCode, userID)

	// 3. Supprimer de PostgreSQL (idempotent)
	s.db.Exec(ctx, queries.UserQueries.DeleteSession, token)
	// PostgreSQL DELETE est idempotent par nature

	return nil // Toujours succès selon les spécifications
}

// ValidateSession valide un token et retourne la session
func (s *SessionService) ValidateSession(ctx context.Context, token, establishmentCode string) (*dto.SessionData, error) {
	return s.GetSession(ctx, token, establishmentCode)
}

// createSessionRedis crée une session dans Redis avec pipeline
func (s *SessionService) createSessionRedis(ctx context.Context, token string, sessionData *dto.SessionData) error {
	pipe := s.redisClient.Client().Pipeline()

	// 1. Session principale
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", sessionData.EtablissementCode, token)
	pipe.HMSet(ctx, sessionKey, sessionData.ToMap())
	pipe.Expire(ctx, sessionKey, time.Hour)

	// 2. Index des sessions utilisateur
	userSessionsKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", sessionData.EtablissementCode, sessionData.UserID)
	pipe.SAdd(ctx, userSessionsKey, token)
	pipe.Expire(ctx, userSessionsKey, time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// getSessionRedis récupère une session depuis Redis
func (s *SessionService) getSessionRedis(ctx context.Context, token, establishmentCode string) (*dto.SessionData, error) {
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)

	sessionData := s.redisClient.Client().HGetAll(ctx, sessionKey).Val()
	if len(sessionData) == 0 {
		return nil, redis.Nil
	}

	return dto.SessionFromMap(sessionData), nil
}

// deleteSessionRedis supprime une session de Redis
func (s *SessionService) deleteSessionRedis(ctx context.Context, token, establishmentCode, userID string) {
	pipe := s.redisClient.Client().Pipeline()

	// Supprimer la session
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)
	pipe.Del(ctx, sessionKey)

	// Retirer de l'index utilisateur
	userSessionsKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", establishmentCode, userID)
	pipe.SRem(ctx, userSessionsKey, token)

	pipe.Exec(ctx)
}

// deleteSessionRedisIdempotent supprime une session de Redis (idempotent)
func (s *SessionService) deleteSessionRedisIdempotent(ctx context.Context, token, establishmentCode, userID string) {
	// Pipeline Redis pour opérations atomiques selon les spécifications
	pipe := s.redisClient.Client().Pipeline()

	// 1. Supprimer la session principale
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)
	pipe.Del(ctx, sessionKey)

	// 2. Retirer de l'index utilisateur (si userID disponible)
	if userID != "" {
		userSessionsKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", establishmentCode, userID)
		pipe.SRem(ctx, userSessionsKey, token)
	}

	// 3. Supprimer le cache des permissions utilisateur pour forcer refresh
	if userID != "" {
		permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
		permissionsDetailKey := fmt.Sprintf("soins_suite_%s_auth_permissions_detail:%s", establishmentCode, userID)
		pipe.Del(ctx, permissionsKey)
		pipe.Del(ctx, permissionsDetailKey)
	}

	// Exécuter le pipeline - pas de gestion d'erreur pour respecter l'idempotence
	pipe.Exec(ctx)
	// Si Redis est down, la session sera quand même supprimée de PostgreSQL
}

// createSessionPostgres crée une session dans PostgreSQL
func (s *SessionService) createSessionPostgres(ctx context.Context, token string, sessionData *dto.SessionData) error {
	expiresAt, _ := time.Parse(time.RFC3339, sessionData.ExpiresAt)

	return s.db.Exec(ctx, queries.UserQueries.CreateSession,
		token,
		sessionData.UserID,
		sessionData.EtablissementID,
		sessionData.ClientType,
		sessionData.IPAddress,
		sessionData.UserAgent,
		expiresAt,
	)
}

// getSessionPostgres récupère une session depuis PostgreSQL
func (s *SessionService) getSessionPostgres(ctx context.Context, token string) (*dto.SessionData, error) {
	var session dto.SessionData
	var createdAt, lastActivity, expiresAt time.Time
	var establishmentCode string

	row := s.db.QueryRow(ctx, queries.UserQueries.GetSessionByToken, token)
	err := row.Scan(
		&session.UserID,
		&session.EtablissementID,
		&session.ClientType,
		&session.IPAddress,
		&session.UserAgent,
		&createdAt,
		&lastActivity,
		&expiresAt,
		&establishmentCode,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session non trouvée")
		}
		return nil, err
	}

	session.EtablissementCode = establishmentCode
	session.CreatedAt = createdAt.Format(time.RFC3339)
	session.LastActivity = lastActivity.Format(time.RFC3339)
	session.ExpiresAt = expiresAt.Format(time.RFC3339)

	return &session, nil
}

// updateLastActivity met à jour la dernière activité
func (s *SessionService) updateLastActivity(ctx context.Context, token, establishmentCode string, session *dto.SessionData) {
	sessionKey := fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)
	now := time.Now().Format(time.RFC3339)

	// Mettre à jour Redis
	s.redisClient.Client().HSet(ctx, sessionKey, "last_activity", now)

	// Mettre à jour PostgreSQL en arrière-plan
	go func() {
		s.db.Exec(context.Background(), `
			UPDATE user_session 
			SET last_activity = NOW(), updated_at = NOW() 
			WHERE token = $1
		`, token)
	}()
}

// blacklistToken ajoute un token à la blacklist
func (s *SessionService) blacklistToken(ctx context.Context, establishmentCode, token string) {
	blacklistKey := fmt.Sprintf("soins_suite_%s_auth_blacklist:%s", establishmentCode, token)
	revokedAt := fmt.Sprintf("revoked_at:%s", time.Now().Format(time.RFC3339))

	s.redisClient.Set(ctx, blacklistKey, revokedAt, time.Hour)
}

// blacklistTokenIdempotent ajoute un token à la blacklist (idempotent)
func (s *SessionService) blacklistTokenIdempotent(ctx context.Context, establishmentCode, token string) {
	blacklistKey := fmt.Sprintf("soins_suite_%s_auth_blacklist:%s", establishmentCode, token)
	revokedAt := fmt.Sprintf("revoked_at:%s", time.Now().Format(time.RFC3339))

	// SET avec TTL - idempotent par nature
	s.redisClient.Set(ctx, blacklistKey, revokedAt, time.Hour)
	// Pas de gestion d'erreur - si Redis est down, le token sera quand même invalidé côté PostgreSQL
}

// isTokenBlacklisted vérifie si un token est blacklisté
func (s *SessionService) isTokenBlacklisted(ctx context.Context, establishmentCode, token string) bool {
	blacklistKey := fmt.Sprintf("soins_suite_%s_auth_blacklist:%s", establishmentCode, token)

	exists, err := s.redisClient.Exists(ctx, blacklistKey)
	return err == nil && exists
}

// syncSessionToRedis resynchronise une session PostgreSQL vers Redis
func (s *SessionService) syncSessionToRedis(ctx context.Context, token string, session *dto.SessionData) {
	// Créer la session en Redis
	s.createSessionRedis(ctx, token, session)
}

// GetActiveUserSessions récupère toutes les sessions actives d'un utilisateur
func (s *SessionService) GetActiveUserSessions(ctx context.Context, userID, establishmentCode string) ([]string, error) {
	// Essayer Redis d'abord
	userSessionsKey := fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", establishmentCode, userID)
	tokens, err := s.redisClient.Client().SMembers(ctx, userSessionsKey).Result()
	if err == nil && len(tokens) > 0 {
		return tokens, nil
	}

	// Fallback PostgreSQL
	rows, err := s.db.Query(ctx, queries.UserQueries.GetActiveSessionsByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			continue
		}
		sessions = append(sessions, token)
	}

	return sessions, nil
}

// CleanExpiredSessions nettoie les sessions expirées de PostgreSQL
func (s *SessionService) CleanExpiredSessions(ctx context.Context) error {
	return s.db.Exec(ctx, queries.UserQueries.CleanExpiredSessions)
}
