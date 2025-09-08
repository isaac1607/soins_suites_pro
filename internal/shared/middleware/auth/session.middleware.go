package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/auth/dto"
	"soins-suite-core/internal/modules/auth/services"
	"soins-suite-core/internal/shared/middleware/tenant"
)

// SessionContext contient les informations de session injectées dans le contexte Gin
type SessionContext struct {
	UserID           string `json:"user_id"`
	EtablissementID  string `json:"etablissement_id"`
	EtablissementCode string `json:"etablissement_code"`
	ClientType       string `json:"client_type"`
	Token           string `json:"token"`
	IPAddress       string `json:"ip_address"`
	UserAgent       string `json:"user_agent"`
	CreatedAt       string `json:"created_at"`
	LastActivity    string `json:"last_activity"`
	ExpiresAt       string `json:"expires_at"`
}

type SessionMiddleware struct {
	sessionService *services.SessionService
}

// NewSessionMiddleware crée une nouvelle instance du middleware de session
func NewSessionMiddleware(db *postgres.Client, redisClient *redis.Client) *SessionMiddleware {
	sessionService := services.NewSessionService(db, redisClient)
	return &SessionMiddleware{
		sessionService: sessionService,
	}
}

// Handler retourne le middleware Gin pour la validation de session
func (m *SessionMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extraire le token Bearer
		authHeader := c.GetHeader("Authorization")
		token := m.extractBearerToken(authHeader)
		
		if token == "" {
			m.respondError(c, 480, "TOKEN_REQUIRED", 
				"Token d'authentification requis", map[string]interface{}{
					"header_format": "Authorization: Bearer {token}",
				})
			return
		}

		// 2. Récupérer l'établissement depuis le contexte (injecté par EstablishmentMiddleware)
		establishmentValue, exists := c.Get("establishment")
		if !exists {
			m.respondError(c, http.StatusInternalServerError, "ESTABLISHMENT_CONTEXT_MISSING",
				"Contexte établissement manquant", nil)
			return
		}

		establishment, ok := establishmentValue.(tenant.EstablishmentContext)
		if !ok {
			m.respondError(c, http.StatusInternalServerError, "ESTABLISHMENT_CONTEXT_INVALID",
				"Contexte établissement invalide", nil)
			return
		}

		// 3. Valider la session
		session, err := m.sessionService.ValidateSession(c.Request.Context(), token, establishment.Code)
		if err != nil {
			// Gestion des erreurs d'authentification
			if authErr, ok := err.(*dto.AuthError); ok {
				var statusCode int
				switch authErr.Code {
				case "TOKEN_REVOKED":
					statusCode = 480 // Code custom pour token révoqué
				case "INVALID_TOKEN", "SESSION_EXPIRED", "SESSION_NOT_FOUND":
					statusCode = 480 // Code custom pour session invalide/expirée
				default:
					statusCode = 480 // Par défaut, utiliser 480 pour tous les problèmes d'authentification
				}

				m.respondError(c, statusCode, authErr.Code, authErr.Message, authErr.Details)
				return
			}

			// Erreur technique
			m.respondError(c, 480, "SESSION_VALIDATION_ERROR",
				"Erreur lors de la validation de la session", map[string]interface{}{
					"token_format": "Vérifiez le format du token",
				})
			return
		}

		// 4. Vérifier que la session appartient au bon établissement
		if session.EtablissementCode != establishment.Code {
			m.respondError(c, http.StatusForbidden, "ESTABLISHMENT_MISMATCH",
				"Token non valide pour cet établissement", map[string]interface{}{
					"establishment_code": establishment.Code,
				})
			return
		}

		// 5. Mettre à jour la dernière activité (non bloquant)
		go m.updateLastActivity(c.Request.Context(), token, establishment.Code, session)

		// 6. Enrichir le contexte Gin avec les données de session
		sessionContext := SessionContext{
			UserID:            session.UserID,
			EtablissementID:   session.EtablissementID,
			EtablissementCode: session.EtablissementCode,
			ClientType:        session.ClientType,
			Token:            token,
			IPAddress:        session.IPAddress,
			UserAgent:        session.UserAgent,
			CreatedAt:        session.CreatedAt,
			LastActivity:     session.LastActivity,
			ExpiresAt:        session.ExpiresAt,
		}

		c.Set("session", sessionContext)
		c.Set("user_id", session.UserID)
		c.Set("establishment_id", session.EtablissementID)
		c.Set("client_type", session.ClientType)

		// 7. Continuer vers le middleware/handler suivant
		c.Next()
	}
}

// extractBearerToken extrait le token depuis le header Authorization
func (m *SessionMiddleware) extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// updateLastActivity met à jour la dernière activité de la session
func (m *SessionMiddleware) updateLastActivity(ctx context.Context, token, establishmentCode string, session *dto.SessionData) {
	// Cette opération est faite en arrière-plan pour ne pas bloquer la requête
	// La mise à jour est déjà gérée dans SessionService.ValidateSession
	// Cette méthode pourrait être utilisée pour des statistiques supplémentaires
}

// respondError envoie une réponse d'erreur standardisée
func (m *SessionMiddleware) respondError(c *gin.Context, status int, code, message string, details map[string]interface{}) {
	response := gin.H{
		"error": message,
		"details": gin.H{
			"code": code,
		},
	}

	if details != nil {
		if detailsMap, ok := response["details"].(gin.H); ok {
			for k, v := range details {
				detailsMap[k] = v
			}
		}
	}

	c.JSON(status, response)
	c.Abort()
}