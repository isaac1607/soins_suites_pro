package authentication

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/shared/middleware/authentication/queries"
)

// LicenseContext contient les informations de licence injectées dans le contexte
type LicenseContext struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`              // premium|standard|evaluation
	Mode             string   `json:"mode"`              // local|online
	ModulesAutorises []string `json:"modules_autorises"`
	DateExpiration   *string  `json:"date_expiration"`
	EstActive        bool     `json:"est_active"`
}

// LicenseData représente les données de licence récupérées de la DB
// Optimisé pour cache : données critiques de validation
type LicenseData struct {
	ID               string    `json:"id"`
	TypeLicence      string    `json:"type_licence"`       // premium|standard|evaluation
	ModeDeploiement  string    `json:"mode_deploiement"`   // local|online
	Statut           string    `json:"statut"`             // actif|expiree|revoquee
	ModulesAutorises string `json:"modules_autorises"` // JSONB as string
	DateExpiration   *time.Time `json:"date_expiration"`   // NULL pour premium local
}

type LicenseMiddleware struct {
	db          *postgres.Client
	redisClient *redisInfra.Client
}

// NewLicenseMiddleware crée une nouvelle instance du middleware
func NewLicenseMiddleware(db *postgres.Client, redisClient *redisInfra.Client) *LicenseMiddleware {
	return &LicenseMiddleware{
		db:          db,
		redisClient: redisClient,
	}
}

// Handler retourne le middleware Gin pour la validation de licence
func (m *LicenseMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupération de l'établissement depuis le contexte (injecté par EstablishmentMiddleware)
		establishmentValue, exists := c.Get("establishment")
		if !exists {
			m.respondError(c, 465, "ESTABLISHMENT_CONTEXT_MISSING", 
				"Contexte établissement manquant", nil)
			return
		}

		establishment, ok := establishmentValue.(EstablishmentContext)
		if !ok {
			m.respondError(c, 465, "ESTABLISHMENT_CONTEXT_INVALID", 
				"Contexte établissement invalide", nil)
			return
		}

		// Vérification en cache Redis d'abord (utilise le code établissement pour cohérence)
		license, found := m.getLicenseFromCache(c.Request.Context(), establishment.Code)
		if !found {
			// Si pas en cache, récupération depuis la base (utilise l'ID pour la requête SQL)
			var err error
			license, err = m.getLicenseFromDB(c.Request.Context(), establishment.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					m.respondError(c, 465, "LICENSE_NOT_FOUND", 
						"Aucune licence trouvée", map[string]interface{}{
							"establishment_code": establishment.Code,
						})
					return
				}
				// Pour toute autre erreur de DB, on considère que la licence n'existe pas
				// selon les spécifications (plutôt qu'une erreur technique)
				m.respondError(c, 465, "LICENSE_NOT_FOUND", 
					"Aucune licence trouvée", map[string]interface{}{
						"establishment_code": establishment.Code,
					})
				return
			}

			// Mise en cache pour 24 heures (utilise le code établissement pour cohérence)
			m.cacheLicense(c.Request.Context(), establishment.Code, license)
		}

		// Validation du statut de la licence
		if license.Statut != "actif" {
			m.respondError(c, 465, "LICENSE_INACTIVE", 
				"Licence inactive", map[string]interface{}{
					"establishment_code": establishment.Code,
					"license_status":    license.Statut,
				})
			return
		}

		// Validation expiration (mode online uniquement)
		if license.ModeDeploiement == "online" && license.DateExpiration != nil {
			if time.Now().After(*license.DateExpiration) {
				daysExpired := int(time.Since(*license.DateExpiration).Hours() / 24)
				m.respondError(c, 465, "LICENSE_EXPIRED", 
					"Licence expirée", map[string]interface{}{
						"establishment_code": establishment.Code,
						"expiration_date":   license.DateExpiration.Format(time.RFC3339),
						"days_expired":      daysExpired,
					})
				return
			}
		}

		// Conversion JSONB string vers slice
		var modulesAutorises []string
		if license.ModulesAutorises != "" && license.ModulesAutorises != "[]" {
			err := json.Unmarshal([]byte(license.ModulesAutorises), &modulesAutorises)
			if err != nil {
				// Si erreur parsing JSONB, on continue avec slice vide
				modulesAutorises = []string{}
			}
		}

		// Conversion pour contexte Gin
		var dateExpiration *string
		if license.DateExpiration != nil {
			expStr := license.DateExpiration.Format(time.RFC3339)
			dateExpiration = &expStr
		}

		// Injection des données dans le contexte Gin
		licenseContext := LicenseContext{
			ID:               license.ID,
			Type:             license.TypeLicence,
			Mode:             license.ModeDeploiement,
			ModulesAutorises: modulesAutorises,
			DateExpiration:   dateExpiration,
			EstActive:        license.Statut == "actif",
		}
		c.Set("license", licenseContext)

		// Continue vers le controller suivant
		c.Next()
	}
}

// getLicenseFromCache récupère la licence depuis Redis selon les conventions
func (m *LicenseMiddleware) getLicenseFromCache(ctx context.Context, establishmentCode string) (*LicenseData, bool) {
	// Utilisation du pattern cache_middleware avec identifier "license" (TTL 24h automatique selon logique client Redis)
	jsonData, err := m.redisClient.GetWithPattern(ctx, "cache_middleware", establishmentCode, "license")
	if err != nil {
		// Pas trouvé en cache ou erreur technique Redis, on continue sans cache
		return nil, false
	}

	// Désérialisation des données
	var license LicenseData
	if err := json.Unmarshal([]byte(jsonData), &license); err != nil {
		// Erreur de désérialisation, on invalide le cache et continue sans
		m.redisClient.DelWithPattern(ctx, "cache_middleware", establishmentCode, "license")
		return nil, false
	}

	return &license, true
}

// cacheLicense met en cache les données de licence selon les conventions
func (m *LicenseMiddleware) cacheLicense(ctx context.Context, establishmentCode string, license *LicenseData) {
	// Sérialisation des données
	jsonData, err := json.Marshal(license)
	if err != nil {
		// Erreur de sérialisation, on continue sans mettre en cache
		return
	}

	// Mise en cache avec pattern cache_middleware et identifier "license" (TTL: 86400s = 24h selon logique client Redis)
	err = m.redisClient.SetWithPattern(ctx, "cache_middleware", establishmentCode, jsonData, "license")
	if err != nil {
		// Erreur Redis, on continue sans notification (non bloquant)
		return
	}
}

// getLicenseFromDB récupère la licence depuis PostgreSQL
func (m *LicenseMiddleware) getLicenseFromDB(ctx context.Context, establishmentID string) (*LicenseData, error) {
	row := m.db.QueryRow(ctx, queries.LicenseQueries.GetByEstablishmentID, establishmentID)
	
	var license LicenseData
	err := row.Scan(
		&license.ID,
		&license.TypeLicence,
		&license.ModeDeploiement,
		&license.Statut,
		&license.ModulesAutorises,
		&license.DateExpiration,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &license, nil
}

// ValidateModuleAccess valide l'accès à un module spécifique (utilisable par controllers)
func (m *LicenseMiddleware) ValidateModuleAccess(c *gin.Context, moduleCode string) bool {
	licenseValue, exists := c.Get("license")
	if !exists {
		return false
	}

	license, ok := licenseValue.(LicenseContext)
	if !ok || !license.EstActive {
		return false
	}

	// Vérification que le module est autorisé
	for _, authorizedModule := range license.ModulesAutorises {
		if authorizedModule == moduleCode {
			return true
		}
	}

	return false
}

// RespondModuleNotAuthorized envoie une erreur standardisée pour module non autorisé
func (m *LicenseMiddleware) RespondModuleNotAuthorized(c *gin.Context, moduleCode string) {
	establishmentValue, _ := c.Get("establishment")
	establishment, _ := establishmentValue.(EstablishmentContext)
	
	licenseValue, _ := c.Get("license")
	license, _ := licenseValue.(LicenseContext)

	m.respondError(c, 470, "MODULE_NOT_LICENSED", 
		"Module non autorisé par la licence", map[string]interface{}{
			"establishment_code":   establishment.Code,
			"module_requested":     moduleCode,
			"modules_authorized":   license.ModulesAutorises,
		})
}

// respondError envoie une réponse d'erreur standardisée
func (m *LicenseMiddleware) respondError(c *gin.Context, status int, code, message string, details map[string]interface{}) {
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