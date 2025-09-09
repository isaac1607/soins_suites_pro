package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"soins-suite-core/internal/infrastructure/database/postgres"
	redisClient "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/tir/tir-auth/dto"
	"soins-suite-core/internal/modules/tir/tir-auth/queries"
	"soins-suite-core/internal/shared/utils"
)

type TIRAuthService struct {
	db    *postgres.Client
	redis *redisClient.Client
}

func NewTIRAuthService(db *postgres.Client, redis *redisClient.Client) *TIRAuthService {
	return &TIRAuthService{
		db:    db,
		redis: redis,
	}
}

// Login authentifie un admin TIR et crée une session
func (s *TIRAuthService) Login(ctx context.Context, req dto.LoginTIRRequest, ipAddress, userAgent string) (*dto.LoginTIRResponse, error) {
	// 1. Récupérer l'admin par identifiant
	var adminID, identifiant, nom, prenoms, email, passwordHash, salt, niveauAdmin, statut string
	var peutGererLicences, peutGererEtablissements, peutAccederDonneesEtablissement, peutGererAdminsGlobaux bool
	var mustChangePassword bool
	var lastLoginAt *time.Time

	err := s.db.Pool().QueryRow(ctx, queries.TIRAuthQueries.GetAdminByIdentifiant, req.Identifiant).Scan(
		&adminID, &identifiant, &nom, &prenoms, &email,
		&passwordHash, &salt, &niveauAdmin,
		&peutGererLicences, &peutGererEtablissements, &peutAccederDonneesEtablissement, &peutGererAdminsGlobaux,
		&statut, &mustChangePassword, &lastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("identifiant ou mot de passe incorrect")
		}
		return nil, fmt.Errorf("erreur base de données: %w", err)
	}

	// 2. Vérifier le mot de passe
	if !utils.VerifyPasswordSHA512(req.Password, salt, passwordHash) {
		return nil, fmt.Errorf("identifiant ou mot de passe incorrect")
	}

	// 3. Générer token TIR avec préfixe
	tokenUUID := uuid.New()
	token := fmt.Sprintf("soins_suite_tir_admin_%s", tokenUUID.String())

	// 4. Calculer expiration (2h)
	expiresAt := time.Now().Add(2 * time.Hour)

	// 5. Créer session PostgreSQL
	_, err = s.db.Pool().Exec(ctx, queries.TIRAuthQueries.CreateSession,
		token, adminID, ipAddress, userAgent, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("échec création session PostgreSQL: %w", err)
	}

	// 6. Créer session Redis
	sessionData := dto.SessionTIRData{
		AdminID:                          adminID,
		Identifiant:                     identifiant,
		NiveauAdmin:                     niveauAdmin,
		PeutGererLicences:               boolToString(peutGererLicences),
		PeutGererEtablissements:         boolToString(peutGererEtablissements),
		PeutAccederDonneesEtablissement: boolToString(peutAccederDonneesEtablissement),
		PeutGererAdminsGlobaux:          boolToString(peutGererAdminsGlobaux),
		IPAddress:                       ipAddress,
		UserAgent:                       userAgent,
		CreatedAt:                       time.Now(),
		LastActivity:                    time.Now(),
		ExpiresAt:                       expiresAt,
	}

	sessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", token)
	err = s.redis.Client().HMSet(ctx, sessionKey, sessionData.ToMap()).Err()
	if err != nil {
		return nil, fmt.Errorf("échec création session Redis: %w", err)
	}

	// 7. Définir TTL Redis
	err = s.redis.Client().Expire(ctx, sessionKey, 2*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("échec définition TTL Redis: %w", err)
	}

	// 8. Mettre à jour last_login_at (optionnel, pas bloquant)
	go func() {
		ctxUpdate := context.Background()
		_, _ = s.db.Pool().Exec(ctxUpdate, 
			"UPDATE tir_admin_global SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1", 
			adminID,
		)
	}()

	return &dto.LoginTIRResponse{
		Token: token,
		Admin: dto.AdminTIRInfo{
			AdminID:                          adminID,
			Identifiant:                     identifiant,
			NiveauAdmin:                     niveauAdmin,
			PeutGererLicences:               peutGererLicences,
			PeutGererEtablissements:         peutGererEtablissements,
			PeutAccederDonneesEtablissement: peutAccederDonneesEtablissement,
			PeutGererAdminsGlobaux:          peutGererAdminsGlobaux,
		},
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateSession valide une session TIR depuis Redis (priorité) ou PostgreSQL (fallback)
func (s *TIRAuthService) ValidateSession(ctx context.Context, token string) (*dto.TIRSessionValidation, error) {
	// 1. Vérifier format token TIR
	if !strings.HasPrefix(token, "soins_suite_tir_admin_") {
		return &dto.TIRSessionValidation{
			Valid:       false,
			ErrorReason: "format token TIR invalide",
		}, nil
	}

	// 2. Essayer Redis en priorité
	sessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", token)
	sessionData := s.redis.Client().HGetAll(ctx, sessionKey).Val()

	if len(sessionData) > 0 {
		// Session trouvée dans Redis
		adminInfo, err := s.parseRedisSessionToAdminInfo(sessionData)
		if err != nil {
			return &dto.TIRSessionValidation{
				Valid:       false,
				ErrorReason: "erreur parsing session Redis",
			}, nil
		}

		// Mettre à jour last_activity
		s.redis.Client().HSet(ctx, sessionKey, "last_activity", time.Now().Format(time.RFC3339))

		return &dto.TIRSessionValidation{
			Valid:   true,
			AdminID: adminInfo.AdminID,
			Admin:   adminInfo,
			Token:   token,
		}, nil
	}

	// 3. Fallback PostgreSQL
	return s.validateSessionFromPostgreSQL(ctx, token)
}

// Logout révoque une session TIR
func (s *TIRAuthService) Logout(ctx context.Context, token string) error {
	// 1. Supprimer de Redis
	sessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", token)
	err := s.redis.Client().Del(ctx, sessionKey).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("échec suppression session Redis: %w", err)
	}

	// 2. Supprimer de PostgreSQL
	_, err = s.db.Pool().Exec(ctx, queries.TIRAuthQueries.DeleteSession, token)
	if err != nil {
		return fmt.Errorf("échec suppression session PostgreSQL: %w", err)
	}

	return nil
}

// RefreshToken renouvelle un token TIR existant
func (s *TIRAuthService) RefreshToken(ctx context.Context, oldToken string) (*dto.RefreshTIRResponse, error) {
	// 1. Valider session existante
	validation, err := s.ValidateSession(ctx, oldToken)
	if err != nil {
		return nil, fmt.Errorf("erreur validation session: %w", err)
	}

	if !validation.Valid {
		return nil, fmt.Errorf("session invalide pour refresh")
	}

	// 2. Générer nouveau token
	tokenUUID := uuid.New()
	newToken := fmt.Sprintf("soins_suite_tir_admin_%s", tokenUUID.String())
	newExpiresAt := time.Now().Add(2 * time.Hour)

	// 3. Récupérer données session actuelle depuis Redis
	oldSessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", oldToken)
	sessionData := s.redis.Client().HGetAll(ctx, oldSessionKey).Val()

	// 4. Créer nouvelle session Redis
	newSessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", newToken)
	sessionData["expires_at"] = newExpiresAt.Format(time.RFC3339)
	sessionData["last_activity"] = time.Now().Format(time.RFC3339)

	err = s.redis.Client().HMSet(ctx, newSessionKey, sessionData).Err()
	if err != nil {
		return nil, fmt.Errorf("échec création nouvelle session Redis: %w", err)
	}

	err = s.redis.Client().Expire(ctx, newSessionKey, 2*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("échec TTL nouvelle session: %w", err)
	}

	// 5. Supprimer ancienne session
	s.redis.Client().Del(ctx, oldSessionKey)
	s.db.Pool().Exec(ctx, queries.TIRAuthQueries.DeleteSession, oldToken)

	// 6. Créer nouvelle session PostgreSQL
	_, err = s.db.Pool().Exec(ctx, queries.TIRAuthQueries.CreateSession,
		newToken, validation.AdminID, sessionData["ip_address"], sessionData["user_agent"], newExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("échec création session PostgreSQL: %w", err)
	}

	return &dto.RefreshTIRResponse{
		Token:     newToken,
		ExpiresAt: newExpiresAt,
	}, nil
}

// Helpers privées

func (s *TIRAuthService) parseRedisSessionToAdminInfo(sessionData map[string]string) (dto.AdminTIRInfo, error) {
	return dto.AdminTIRInfo{
		AdminID:                          sessionData["admin_id"],
		Identifiant:                     sessionData["identifiant"],
		NiveauAdmin:                     sessionData["niveau_admin"],
		PeutGererLicences:               sessionData["peut_gerer_licences"] == "true",
		PeutGererEtablissements:         sessionData["peut_gerer_etablissements"] == "true",
		PeutAccederDonneesEtablissement: sessionData["peut_acceder_donnees_etablissement"] == "true",
		PeutGererAdminsGlobaux:          sessionData["peut_gerer_admins_globaux"] == "true",
	}, nil
}

func (s *TIRAuthService) validateSessionFromPostgreSQL(ctx context.Context, token string) (*dto.TIRSessionValidation, error) {
	var sessionToken, adminID, ipAddress, userAgent, identifiant, niveauAdmin string
	var lastActivity, expiresAt, createdAt time.Time
	var peutGererLicences, peutGererEtablissements, peutAccederDonneesEtablissement, peutGererAdminsGlobaux bool

	err := s.db.Pool().QueryRow(ctx, queries.TIRAuthQueries.GetSessionByToken, token).Scan(
		&sessionToken, &adminID, &ipAddress, &userAgent,
		&lastActivity, &expiresAt, &createdAt,
		&identifiant, &niveauAdmin,
		&peutGererLicences, &peutGererEtablissements, &peutAccederDonneesEtablissement, &peutGererAdminsGlobaux,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return &dto.TIRSessionValidation{
				Valid:       false,
				ErrorReason: "session non trouvée",
			}, nil
		}
		return nil, fmt.Errorf("erreur recherche session PostgreSQL: %w", err)
	}

	// Mettre à jour last_activity PostgreSQL
	s.db.Pool().Exec(ctx, queries.TIRAuthQueries.UpdateLastActivity, token)

	return &dto.TIRSessionValidation{
		Valid:   true,
		AdminID: adminID,
		Admin: dto.AdminTIRInfo{
			AdminID:                          adminID,
			Identifiant:                     identifiant,
			NiveauAdmin:                     niveauAdmin,
			PeutGererLicences:               peutGererLicences,
			PeutGererEtablissements:         peutGererEtablissements,
			PeutAccederDonneesEtablissement: peutAccederDonneesEtablissement,
			PeutGererAdminsGlobaux:          peutGererAdminsGlobaux,
		},
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}