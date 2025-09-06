package authentication

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"

	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/shared/middleware/authentication/queries"

	"github.com/gin-gonic/gin"
)

// EstablishmentContext contient les informations de l'établissement injectées dans le contexte
type EstablishmentContext struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type EstablishmentMiddleware struct {
	db          *postgres.Client
	redisClient *redisInfra.Client
}

// NewEstablishmentMiddleware crée une nouvelle instance du middleware
func NewEstablishmentMiddleware(db *postgres.Client, redisClient *redisInfra.Client) *EstablishmentMiddleware {
	return &EstablishmentMiddleware{
		db:          db,
		redisClient: redisClient,
	}
}

// Handler retourne le middleware Gin pour la validation d'établissement
func (m *EstablishmentMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Exemptions - endpoints qui ne nécessitent pas de validation
		if m.isExemptEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extraction du code établissement depuis le header
		establishmentCode := c.GetHeader("X-Establishment-Code")
		if establishmentCode == "" {
			m.respondError(c, 460, "ESTABLISHMENT_CODE_REQUIRED",
				"Code établissement requis", map[string]interface{}{
					"header_required": "X-Establishment-Code",
				})
			return
		}

		// Validation du format du code
		if !m.isValidEstablishmentCodeFormat(establishmentCode) {
			m.respondError(c, 460, "ESTABLISHMENT_CODE_INVALID_FORMAT",
				"Format code établissement invalide", map[string]interface{}{
					"code_recu":     establishmentCode,
					"format_requis": "Alphanumérique, 3-20 caractères, majuscules",
				})
			return
		}

		// Vérification en cache Redis d'abord
		establishment, found := m.getEstablishmentFromCache(c.Request.Context(), establishmentCode)
		if !found {
			// Si pas en cache, récupération depuis la base
			var err error
			establishment, err = m.getEstablishmentFromDB(c.Request.Context(), establishmentCode)
			if err != nil {
				if err == sql.ErrNoRows {
					// Cas normal : établissement n'existe pas
					m.respondError(c, 460, "ESTABLISHMENT_NOT_FOUND",
						"Établissement non trouvé", map[string]interface{}{
							"establishment_code": establishmentCode,
						})
					return
				}
				// Erreur technique SQL : problème de requête, colonne manquante, etc.
				// En développement, on expose l'erreur pour diagnostic
				// En production, on peut masquer les détails techniques
				m.respondError(c, 500, "DATABASE_ERROR",
					"Erreur technique lors de la validation établissement", map[string]interface{}{
						"establishment_code": establishmentCode,
						"sql_error":          err.Error(), // Utile pour diagnostic
						"error_type":         "database_query_failed",
					})
				return
			}

			// Mise en cache pour 15 minutes
			m.cacheEstablishment(c.Request.Context(), establishmentCode, establishment)
		}

		// Vérification du statut de l'établissement
		if establishment.Statut == "suspendu" {
			m.respondError(c, 460, "ESTABLISHMENT_SUSPENDED",
				"Établissement suspendu", map[string]interface{}{
					"establishment_code": establishmentCode,
					"statut":             "suspendu",
					"motif":              "Licence expirée",
				})
			return
		}

		// Injection des données dans le contexte Gin
		establishmentContext := EstablishmentContext{
			ID:   establishment.ID,
			Code: establishment.Code,
		}
		c.Set("establishment", establishmentContext)

		// Continue vers le middleware suivant
		c.Next()
	}
}

// EstablishmentData représente les données d'établissement récupérées de la DB
// Optimisé pour cache : seuls les champs critiques et immuables
type EstablishmentData struct {
	ID          string `json:"id"`           // UUID - JAMAIS modifié
	AppInstance string `json:"app_instance"` // UUID - JAMAIS modifié
	Code        string `json:"code"`         // Code établissement - JAMAIS modifié
	Statut      string `json:"statut"`       // Seul champ pouvant changer
}

// isExemptEndpoint vérifie si l'endpoint est exempt de validation
func (m *EstablishmentMiddleware) isExemptEndpoint(path string) bool {
	exemptPaths := []string{
		"/health",
		"/ready",
		"/api/v1/system/ping",
		"/api/v1/system/offline/synchronised",
	}

	for _, exemptPath := range exemptPaths {
		if path == exemptPath {
			return true
		}
	}
	return false
}

// isValidEstablishmentCodeFormat valide le format du code établissement
func (m *EstablishmentMiddleware) isValidEstablishmentCodeFormat(code string) bool {
	// Alphanumérique, 3-20 caractères, majuscules
	matched, _ := regexp.MatchString(`^[A-Z0-9]{3,20}$`, code)
	return matched
}

// getEstablishmentFromCache récupère l'établissement depuis Redis selon les conventions
func (m *EstablishmentMiddleware) getEstablishmentFromCache(ctx context.Context, code string) (*EstablishmentData, bool) {
	// Utilisation du pattern cache_middleware avec les conventions
	jsonData, err := m.redisClient.GetWithPattern(ctx, "cache_middleware", code, "establishment")
	if err != nil {
		// Pas trouvé en cache ou erreur technique Redis, on continue sans cache
		return nil, false
	}

	// Désérialisation des données
	var establishment EstablishmentData
	if err := json.Unmarshal([]byte(jsonData), &establishment); err != nil {
		// Erreur de désérialisation, on invalide le cache et continue sans
		m.redisClient.DelWithPattern(ctx, "cache_middleware", code, "establishment")
		return nil, false
	}

	return &establishment, true
}

// cacheEstablishment met en cache les données d'établissement selon les conventions
func (m *EstablishmentMiddleware) cacheEstablishment(ctx context.Context, code string, establishment *EstablishmentData) {
	// Sérialisation des données
	jsonData, err := json.Marshal(establishment)
	if err != nil {
		// Erreur de sérialisation, on continue sans mettre en cache
		return
	}

	// Mise en cache avec pattern cache_middleware (TTL: 900s = 15 min selon config)
	err = m.redisClient.SetWithPattern(ctx, "cache_middleware", code, jsonData, "establishment")
	if err != nil {
		// Erreur Redis, on continue sans notification (non bloquant)
		return
	}
}

// getEstablishmentFromDB récupère l'établissement depuis PostgreSQL
func (m *EstablishmentMiddleware) getEstablishmentFromDB(ctx context.Context, code string) (*EstablishmentData, error) {
	row := m.db.QueryRow(ctx, queries.EstablishmentQueries.GetByCode, code)

	var establishment EstablishmentData
	err := row.Scan(
		&establishment.ID,
		&establishment.AppInstance,
		&establishment.Code,
		&establishment.Statut,
	)

	if err != nil {
		return nil, err
	}

	return &establishment, nil
}

// respondError envoie une réponse d'erreur standardisée
func (m *EstablishmentMiddleware) respondError(c *gin.Context, status int, code, message string, details map[string]interface{}) {
	response := gin.H{
		"error": message,
		"code":  code,
	}

	if details != nil {
		response["details"] = details
	}

	c.JSON(status, response)
	c.Abort()
}
