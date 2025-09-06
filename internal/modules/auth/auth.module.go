package auth

import (
	"go.uber.org/fx"
	"github.com/gin-gonic/gin"

	"soins-suite-core/internal/modules/auth/controllers"
	"soins-suite-core/internal/modules/auth/services"
	authMiddleware "soins-suite-core/internal/shared/middleware/auth"
)

// Module regroupe tous les providers du domaine Auth
var Module = fx.Options(
	// Services (utilisent queries directement)
	fx.Provide(services.NewPermissionService),
	fx.Provide(services.NewSessionService),
	fx.Provide(services.NewAuthService),

	// Controllers
	fx.Provide(controllers.NewAuthController),

	// Configuration des routes
	fx.Invoke(RegisterAuthRoutes),
)

// RegisterAuthRoutes configure les routes Gin pour l'authentification
func RegisterAuthRoutes(
	r *gin.Engine,
	authController *controllers.AuthController,
	authStack *authMiddleware.AuthMiddlewareStack,
) {
	// Groupe API v1 pour l'authentification
	authAPI := r.Group("/api/v1/auth")
	{
		// Login - Nécessite EstablishmentMiddleware uniquement (appliqué globalement)
		authAPI.POST("/login", authController.Login)
		
		// Logout - Nécessite EstablishmentMiddleware uniquement (appliqué globalement)
		authAPI.POST("/logout", authController.Logout)
	}

	// Routes protégées par SessionMiddleware
	protectedAuthAPI := r.Group("/api/v1/auth")
	protectedAuthAPI.Use(authMiddleware.Protected(authStack)...)
	{
		// Me - Nécessite EstablishmentMiddleware + SessionMiddleware
		protectedAuthAPI.GET("/me", authController.Me)
	}
}