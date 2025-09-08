package auth

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

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
	// Groupe API v1 pour l'authentification avec EstablishmentMiddleware
	authAPI := r.Group("/api/v1/auth")
	authAPI.Use(authStack.EstablishmentMiddleware.Handler())
	{
		// Login - Nécessite EstablishmentMiddleware uniquement (rate limiting géré par AuthService)
		authAPI.POST("/login", authController.Login)

		// Logout - Nécessite EstablishmentMiddleware uniquement
		authAPI.POST("/logout", authController.Logout)
	}

	// Routes protégées par SessionMiddleware
	protectedAuthAPI := r.Group("/api/v1/auth")
	protectedAuthAPI.Use(authMiddleware.Protected(authStack)...)
	{
		// Me - Nécessite EstablishmentMiddleware + SessionMiddleware
		protectedAuthAPI.GET("/me", authController.Me)

		// Change Password - Nécessite EstablishmentMiddleware + SessionMiddleware
		protectedAuthAPI.POST("/change-password", authController.ChangePassword)
	}

}
