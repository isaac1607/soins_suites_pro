package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/auth/services"
)

type PermissionMiddleware struct {
	permissionService *services.PermissionService
}

// NewPermissionMiddleware crée une nouvelle instance du middleware de permissions
func NewPermissionMiddleware(db *postgres.Client, redisClient *redis.Client) *PermissionMiddleware {
	permissionService := services.NewPermissionService(db, redisClient)
	return &PermissionMiddleware{
		permissionService: permissionService,
	}
}

// RequireModule retourne un middleware qui vérifie l'accès à un module complet
func (m *PermissionMiddleware) RequireModule(moduleCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer les informations de session
		session, exists := c.Get("session")
		if !exists {
			m.respondPermissionError(c, "SESSION_REQUIRED", 
				"Session requise pour vérifier les permissions", nil)
			return
		}

		sessionCtx, ok := session.(SessionContext)
		if !ok {
			m.respondPermissionError(c, "INVALID_SESSION_CONTEXT",
				"Contexte de session invalide", nil)
			return
		}

		// Vérifier la permission
		hasAccess, err := m.permissionService.HasModuleAccess(
			c.Request.Context(),
			sessionCtx.UserID,
			sessionCtx.EtablissementID,
			sessionCtx.EtablissementCode,
			moduleCode,
		)

		if err != nil {
			m.respondPermissionError(c, "PERMISSION_CHECK_ERROR",
				"Erreur lors de la vérification des permissions", map[string]interface{}{
					"module_code": moduleCode,
					"error":      err.Error(),
				})
			return
		}

		if !hasAccess {
			m.respondPermissionError(c, "INSUFFICIENT_PERMISSIONS",
				"Permissions insuffisantes pour cette action", map[string]interface{}{
					"required_permission": "module:" + moduleCode,
					"user_id":            sessionCtx.UserID,
				})
			return
		}

		// Permission accordée, continuer
		c.Next()
	}
}

// RequireRubrique retourne un middleware qui vérifie l'accès à une rubrique spécifique
func (m *PermissionMiddleware) RequireRubrique(moduleCode, rubriqueCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer les informations de session
		session, exists := c.Get("session")
		if !exists {
			m.respondPermissionError(c, "SESSION_REQUIRED",
				"Session requise pour vérifier les permissions", nil)
			return
		}

		sessionCtx, ok := session.(SessionContext)
		if !ok {
			m.respondPermissionError(c, "INVALID_SESSION_CONTEXT",
				"Contexte de session invalide", nil)
			return
		}

		// Vérifier la permission
		hasAccess, err := m.permissionService.HasRubriqueAccess(
			c.Request.Context(),
			sessionCtx.UserID,
			sessionCtx.EtablissementID,
			sessionCtx.EtablissementCode,
			moduleCode,
			rubriqueCode,
		)

		if err != nil {
			m.respondPermissionError(c, "PERMISSION_CHECK_ERROR",
				"Erreur lors de la vérification des permissions", map[string]interface{}{
					"module_code":   moduleCode,
					"rubrique_code": rubriqueCode,
					"error":        err.Error(),
				})
			return
		}

		if !hasAccess {
			m.respondPermissionError(c, "INSUFFICIENT_PERMISSIONS",
				"Permissions insuffisantes pour cette action", map[string]interface{}{
					"required_permission": "rubrique:" + moduleCode + ":" + rubriqueCode,
					"user_id":            sessionCtx.UserID,
				})
			return
		}

		// Permission accordée, continuer
		c.Next()
	}
}

// RequirePermission retourne un middleware qui vérifie une permission au format string
// Formats supportés :
// - "module:MODULE_CODE" pour accès module complet
// - "rubrique:MODULE_CODE:RUBRIQUE_CODE" pour accès rubrique spécifique
func (m *PermissionMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parser le format de permission
		parts := strings.Split(permission, ":")
		if len(parts) < 2 {
			m.respondPermissionError(c, "INVALID_PERMISSION_FORMAT",
				"Format de permission invalide", map[string]interface{}{
					"provided_permission": permission,
					"expected_formats": []string{
						"module:MODULE_CODE",
						"rubrique:MODULE_CODE:RUBRIQUE_CODE",
					},
				})
			return
		}

		permissionType := parts[0]
		moduleCode := parts[1]

		switch permissionType {
		case "module":
			// Déléguer au middleware de module
			m.RequireModule(moduleCode)(c)
		case "rubrique":
			if len(parts) != 3 {
				m.respondPermissionError(c, "INVALID_RUBRIQUE_PERMISSION_FORMAT",
					"Format de permission rubrique invalide", map[string]interface{}{
						"provided_permission": permission,
						"expected_format":    "rubrique:MODULE_CODE:RUBRIQUE_CODE",
					})
				return
			}
			rubriqueCode := parts[2]
			// Déléguer au middleware de rubrique
			m.RequireRubrique(moduleCode, rubriqueCode)(c)
		default:
			m.respondPermissionError(c, "UNSUPPORTED_PERMISSION_TYPE",
				"Type de permission non supporté", map[string]interface{}{
					"provided_type":   permissionType,
					"supported_types": []string{"module", "rubrique"},
				})
			return
		}
	}
}

// RequireAdmin retourne un middleware qui vérifie si l'utilisateur est administrateur
func (m *PermissionMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer les informations de session
		session, exists := c.Get("session")
		if !exists {
			m.respondPermissionError(c, "SESSION_REQUIRED",
				"Session requise pour vérifier les permissions", nil)
			return
		}

		sessionCtx, ok := session.(SessionContext)
		if !ok {
			m.respondPermissionError(c, "INVALID_SESSION_CONTEXT",
				"Contexte de session invalide", nil)
			return
		}

		// Vérifier le client type (seuls les back-office sont admin)
		if sessionCtx.ClientType != "back-office" {
			m.respondPermissionError(c, "ADMIN_ACCESS_REQUIRED",
				"Accès administrateur requis", map[string]interface{}{
					"current_client_type": sessionCtx.ClientType,
					"required_client_type": "back-office",
				})
			return
		}

		// Permission accordée, continuer
		c.Next()
	}
}

// RequireClientType retourne un middleware qui vérifie le type de client
func (m *PermissionMiddleware) RequireClientType(requiredClientType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer les informations de session
		session, exists := c.Get("session")
		if !exists {
			m.respondPermissionError(c, "SESSION_REQUIRED",
				"Session requise pour vérifier les permissions", nil)
			return
		}

		sessionCtx, ok := session.(SessionContext)
		if !ok {
			m.respondPermissionError(c, "INVALID_SESSION_CONTEXT",
				"Contexte de session invalide", nil)
			return
		}

		// Vérifier le client type
		if sessionCtx.ClientType != requiredClientType {
			m.respondPermissionError(c, "CLIENT_TYPE_MISMATCH",
				"Type de client incorrect", map[string]interface{}{
					"current_client_type":  sessionCtx.ClientType,
					"required_client_type": requiredClientType,
				})
			return
		}

		// Permission accordée, continuer
		c.Next()
	}
}

// respondPermissionError envoie une réponse d'erreur de permission standardisée
func (m *PermissionMiddleware) respondPermissionError(c *gin.Context, code, message string, details map[string]interface{}) {
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

	// Code 465 pour les erreurs de permissions selon les spécifications
	c.JSON(465, response)
	c.Abort()
}

// Helper functions pour utilisation dans les controllers

// CheckModuleAccess vérifie l'accès à un module (utilisable dans les controllers)
func (m *PermissionMiddleware) CheckModuleAccess(c *gin.Context, moduleCode string) bool {
	session, exists := c.Get("session")
	if !exists {
		return false
	}

	sessionCtx, ok := session.(SessionContext)
	if !ok {
		return false
	}

	hasAccess, err := m.permissionService.HasModuleAccess(
		c.Request.Context(),
		sessionCtx.UserID,
		sessionCtx.EtablissementID,
		sessionCtx.EtablissementCode,
		moduleCode,
	)

	return err == nil && hasAccess
}

// CheckRubriqueAccess vérifie l'accès à une rubrique (utilisable dans les controllers)
func (m *PermissionMiddleware) CheckRubriqueAccess(c *gin.Context, moduleCode, rubriqueCode string) bool {
	session, exists := c.Get("session")
	if !exists {
		return false
	}

	sessionCtx, ok := session.(SessionContext)
	if !ok {
		return false
	}

	hasAccess, err := m.permissionService.HasRubriqueAccess(
		c.Request.Context(),
		sessionCtx.UserID,
		sessionCtx.EtablissementID,
		sessionCtx.EtablissementCode,
		moduleCode,
		rubriqueCode,
	)

	return err == nil && hasAccess
}