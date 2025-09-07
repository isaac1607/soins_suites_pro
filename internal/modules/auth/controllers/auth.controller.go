package controllers

import (
	"net/http"
	"strings"

	"soins-suite-core/internal/modules/auth/dto"
	"soins-suite-core/internal/modules/auth/services"
	authMiddleware "soins-suite-core/internal/shared/middleware/auth"
	"soins-suite-core/internal/shared/middleware/tenant"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *services.AuthService
}

// NewAuthController crée une nouvelle instance du contrôleur d'authentification
func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Login - POST /api/v1/auth/login
func (c *AuthController) Login(ctx *gin.Context) {
	// Récupérer les headers requis
	// establishmentCode := ctx.GetHeader("X-Establishment-Code")
	clientType := ctx.GetHeader("X-Client-Type")

	// Récupérer l'établissement depuis le contexte (injecté par EstablishmentMiddleware)
	establishmentValue, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"code": "ESTABLISHMENT_CONTEXT_MISSING",
			},
		})
		return
	}

	establishment, ok := establishmentValue.(tenant.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Contexte établissement invalide",
			"details": map[string]interface{}{
				"code": "ESTABLISHMENT_CONTEXT_INVALID",
			},
		})
		return
	}

	// Validation du client type
	if clientType == "" || (clientType != "front-office" && clientType != "back-office") {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Type de client requis",
			"details": map[string]interface{}{
				"code":        "CLIENT_TYPE_REQUIRED",
				"valid_types": []string{"front-office", "back-office"},
			},
		})
		return
	}

	// Parser la requête
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données de connexion invalides",
			"details": map[string]interface{}{
				"code":              "INVALID_REQUEST_FORMAT",
				"validation_errors": err.Error(),
			},
		})
		return
	}

	// Validation des champs requis
	if strings.TrimSpace(req.Identifiant) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Identifiant requis",
			"details": map[string]interface{}{
				"code": "IDENTIFIANT_REQUIRED",
			},
		})
		return
	}

	if strings.TrimSpace(req.Password) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Mot de passe requis",
			"details": map[string]interface{}{
				"code": "PASSWORD_REQUIRED",
			},
		})
		return
	}

	// Récupérer les informations de la requête
	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	// Appel du service d'authentification
	result, err := c.authService.Login(
		ctx.Request.Context(),
		req,
		establishment.ID,
		establishment.Code,
		clientType,
		ipAddress,
		userAgent,
	)

	if err != nil {
		// Gestion des erreurs d'authentification
		if authErr, ok := err.(*dto.AuthError); ok {
			var statusCode int
			switch authErr.Code {
			case "INVALID_CREDENTIALS":
				statusCode = http.StatusUnauthorized
			case "CLIENT_TYPE_MISMATCH":
				statusCode = http.StatusForbidden
			case "RATE_LIMIT_EXCEEDED":
				statusCode = http.StatusTooManyRequests
			default:
				statusCode = http.StatusInternalServerError
			}

			ctx.JSON(statusCode, gin.H{
				"error":   authErr.Message,
				"details": authErr.Details,
			})
			return
		}

		// Erreur technique
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur interne lors de l'authentification",
			"details": map[string]interface{}{
				"code": "INTERNAL_ERROR",
			},
		})
		return
	}

	// Succès
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// Logout - POST /api/v1/auth/logout
func (c *AuthController) Logout(ctx *gin.Context) {
	// Récupérer le token depuis le header Authorization
	authHeader := ctx.GetHeader("Authorization")
	token := c.extractBearerToken(authHeader)

	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token d'authentification requis",
			"details": map[string]interface{}{
				"code": "TOKEN_REQUIRED",
			},
		})
		return
	}

	// Récupérer l'établissement depuis le contexte
	establishmentValue, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"code": "ESTABLISHMENT_CONTEXT_MISSING",
			},
		})
		return
	}

	establishment, ok := establishmentValue.(tenant.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Contexte établissement invalide",
			"details": map[string]interface{}{
				"code": "ESTABLISHMENT_CONTEXT_INVALID",
			},
		})
		return
	}

	// Effectuer la déconnexion directement (idempotent selon les spécifications)
	// Le service gère la récupération de l'userID depuis la session et la révocation complète
	err := c.authService.LogoutByToken(ctx.Request.Context(), token, establishment.Code)
	if err != nil {
		// Le logout ne doit jamais échouer selon les spécifications (idempotent)
		// On log l'erreur mais on retourne succès quand même
		// Car un token déjà révoqué ou inexistant = déconnexion réussie
	}

	// Succès (toujours selon les spécifications - idempotent)
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Déconnexion réussie",
	})
}

// Me - GET /api/v1/auth/me
// Endpoint protégé par SessionMiddleware - utilise directement le contexte enrichi
func (c *AuthController) Me(ctx *gin.Context) {
	// Les informations de session sont déjà validées et injectées par SessionMiddleware
	// Récupérer les informations de session depuis le contexte
	_, exists := ctx.Get("session")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Session context missing - SessionMiddleware required",
			"details": map[string]interface{}{
				"code": "SESSION_CONTEXT_MISSING",
			},
		})
		return
	}

	// Le SessionMiddleware garantit que la session est valide et le contexte enrichi
	userID := ctx.GetString("user_id")
	establishmentID := ctx.GetString("establishment_id")

	// Le establishment_code vient du EstablishmentMiddleware, pas SessionMiddleware
	establishmentValue, _ := ctx.Get("establishment")
	establishment, _ := establishmentValue.(tenant.EstablishmentContext)
	establishmentCode := establishment.Code

	if userID == "" || establishmentCode == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Session context incomplete",
			"details": map[string]interface{}{
				"code":               "SESSION_CONTEXT_INCOMPLETE",
				"user_id":            userID,
				"establishment_code": establishmentCode,
			},
		})
		return
	}

	// Récupérer les informations utilisateur optimisées (pas besoin de re-valider le token)
	result, err := c.authService.GetCurrentUserByID(ctx.Request.Context(), userID, establishmentID, establishmentCode)
	if err != nil {
		if authErr, ok := err.(*dto.AuthError); ok {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   authErr.Message,
				"details": authErr.Details,
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la récupération des informations utilisateur",
			"details": map[string]interface{}{
				"code": "USER_FETCH_ERROR",
			},
		})
		return
	}

	// Enrichir avec les informations de session depuis le SessionMiddleware
	if sessionValue, exists := ctx.Get("session"); exists {
		// Utiliser les données déjà validées par SessionMiddleware
		if sessionCtx, ok := sessionValue.(authMiddleware.SessionContext); ok {
			result.Session = dto.SessionInfo{
				Token:      sessionCtx.Token,
				ExpiresAt:  sessionCtx.ExpiresAt,
				ClientType: sessionCtx.ClientType,
			}
		}
	}

	// Succès
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// extractBearerToken extrait le token depuis le header Authorization
func (c *AuthController) extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}
