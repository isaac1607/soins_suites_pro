package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	
	"soins-suite-core/internal/modules/tir/tir-auth/dto"
	"soins-suite-core/internal/modules/tir/tir-auth/services"
)

type TIRAuthController struct {
	service *services.TIRAuthService
}

func NewTIRAuthController(service *services.TIRAuthService) *TIRAuthController {
	return &TIRAuthController{
		service: service,
	}
}

// Login - POST /api/v1/tir/auth/login
func (c *TIRAuthController) Login(ctx *gin.Context) {
	var req dto.LoginTIRRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données de connexion invalides",
			"details": map[string]interface{}{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Récupérer métadonnées client
	ipAddress := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	// Appeler service d'authentification
	result, err := c.service.Login(ctx.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentification échouée",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Connexion admin TIR réussie",
	})
}

// Logout - POST /api/v1/tir/auth/logout
func (c *TIRAuthController) Logout(ctx *gin.Context) {
	// Récupérer token depuis header Authorization
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Token d'authentification requis",
			"details": map[string]interface{}{
				"missing_header": "Authorization",
			},
		})
		return
	}

	// Extraire token Bearer
	token := extractBearerToken(authHeader)
	if token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Format token invalide",
			"details": map[string]interface{}{
				"expected_format": "Bearer {token}",
			},
		})
		return
	}

	// Appeler service de déconnexion
	err := c.service.Logout(ctx.Request.Context(), token)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec déconnexion",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Déconnexion admin TIR réussie",
	})
}

// Refresh - POST /api/v1/tir/auth/refresh
func (c *TIRAuthController) Refresh(ctx *gin.Context) {
	// Récupérer token depuis header Authorization
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Token d'authentification requis",
			"details": map[string]interface{}{
				"missing_header": "Authorization",
			},
		})
		return
	}

	// Extraire token Bearer
	oldToken := extractBearerToken(authHeader)
	if oldToken == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Format token invalide",
			"details": map[string]interface{}{
				"expected_format": "Bearer {token}",
			},
		})
		return
	}

	// Appeler service de refresh
	result, err := c.service.RefreshToken(ctx.Request.Context(), oldToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Échec renouvellement token",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Token admin TIR renouvelé",
	})
}

// ValidateSession - GET /api/v1/tir/auth/validate
func (c *TIRAuthController) ValidateSession(ctx *gin.Context) {
	// Récupérer token depuis header Authorization
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Token d'authentification requis",
			"details": map[string]interface{}{
				"missing_header": "Authorization",
			},
		})
		return
	}

	// Extraire token Bearer
	token := extractBearerToken(authHeader)
	if token == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Format token invalide",
			"details": map[string]interface{}{
				"expected_format": "Bearer {token}",
			},
		})
		return
	}

	// Appeler service de validation
	validation, err := c.service.ValidateSession(ctx.Request.Context(), token)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur validation session",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	if !validation.Valid {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Session invalide ou expirée",
			"details": map[string]interface{}{
				"reason": validation.ErrorReason,
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    validation,
		"message": "Session admin TIR valide",
	})
}

// Helper pour extraire token Bearer
func extractBearerToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}