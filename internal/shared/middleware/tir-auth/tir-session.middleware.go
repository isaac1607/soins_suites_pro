package tir_auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
)

// TIRSessionData représente les données de session TIR depuis Redis
type TIRSessionData struct {
	AdminID                          string `json:"admin_id"`
	Identifiant                     string `json:"identifiant"`
	NiveauAdmin                     string `json:"niveau_admin"`
	PeutGererLicences               string `json:"peut_gerer_licences"`
	PeutGererEtablissements         string `json:"peut_gerer_etablissements"`
	PeutAccederDonneesEtablissement string `json:"peut_acceder_donnees_etablissement"`
	PeutGererAdminsGlobaux          string `json:"peut_gerer_admins_globaux"`
	IPAddress                       string `json:"ip_address"`
	UserAgent                       string `json:"user_agent"`
	CreatedAt                       string `json:"created_at"`
	LastActivity                    string `json:"last_activity"`
	ExpiresAt                       string `json:"expires_at"`
}

// TIRSessionMiddleware valide les sessions administrateurs TIR via Redis
func TIRSessionMiddleware(redisClient *redisInfra.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extraire token TIR depuis Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token TIR requis",
				"details": map[string]interface{}{
					"missing_header": "Authorization",
					"expected_format": "Bearer soins_suite_tir_admin_{token}",
				},
			})
			return
		}

		// 2. Extraire token Bearer
		token := extractBearerToken(authHeader)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Format token invalide",
				"details": map[string]interface{}{
					"expected_format": "Bearer {token}",
				},
			})
			return
		}

		// 3. Valider le préfixe TIR
		if !strings.HasPrefix(token, "soins_suite_tir_admin_") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token TIR invalide - préfixe incorrect",
				"details": map[string]interface{}{
					"expected_prefix": "soins_suite_tir_admin_",
				},
			})
			return
		}

		// 4. Construire clé Redis selon le schéma défini
		sessionKey := fmt.Sprintf("soins_suite_tir_admin_session:%s", token)

		// 5. Récupérer session depuis Redis
		sessionData, err := redisClient.HGetAll(context.Background(), sessionKey)
		if err == redis.Nil || len(sessionData) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Session TIR invalide ou expirée",
				"details": map[string]interface{}{
					"redis_key": sessionKey,
					"status": "not_found",
				},
			})
			return
		}

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Erreur validation session TIR",
				"details": map[string]interface{}{
					"redis_error": err.Error(),
				},
			})
			return
		}

		// 6. Vérifier expiration de session
		if isSessionExpired(sessionData["expires_at"]) {
			// Supprimer session expirée
			redisClient.Del(context.Background(), sessionKey)
			
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Session TIR expirée",
				"details": map[string]interface{}{
					"expires_at": sessionData["expires_at"],
				},
			})
			return
		}

		// 7. Mettre à jour last_activity
		err = redisClient.HSet(context.Background(), sessionKey, "last_activity", time.Now().Format(time.RFC3339))
		if err != nil {
			// Log error mais continue (non bloquant)
		}

		// 8. Enrichir contexte Gin avec données session TIR
		c.Set("tir_session", sessionData)
		c.Set("tir_admin_id", sessionData["admin_id"])
		c.Set("tir_identifiant", sessionData["identifiant"])
		c.Set("tir_niveau_admin", sessionData["niveau_admin"])

		// 9. Enrichir permissions TIR pour utilisation dans TIRPermissionMiddleware
		c.Set("peut_gerer_licences", sessionData["peut_gerer_licences"] == "true")
		c.Set("peut_gerer_etablissements", sessionData["peut_gerer_etablissements"] == "true")
		c.Set("peut_acceder_donnees_etablissement", sessionData["peut_acceder_donnees_etablissement"] == "true")
		c.Set("peut_gerer_admins_globaux", sessionData["peut_gerer_admins_globaux"] == "true")

		c.Next()
	}
}

// extractBearerToken extrait le token depuis l'header Authorization Bearer
func extractBearerToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// isSessionExpired vérifie si la session est expirée
func isSessionExpired(expiresAtStr string) bool {
	if expiresAtStr == "" {
		return true
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return true
	}

	return time.Now().After(expiresAt)
}