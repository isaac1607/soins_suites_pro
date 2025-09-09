package tirauth

import (
	"go.uber.org/fx"
	"github.com/gin-gonic/gin"

	"soins-suite-core/internal/modules/tir/tir-auth/controllers"
	"soins-suite-core/internal/modules/tir/tir-auth/services"
)

// Module regroupe tous les providers du module TIR Auth
var Module = fx.Options(
	// Services (utilisent queries directement)
	fx.Provide(services.NewTIRAuthService),

	// Controllers
	fx.Provide(controllers.NewTIRAuthController),

	// Configuration des routes
	fx.Invoke(RegisterTIRAuthRoutes),
)

// RegisterTIRAuthRoutes configure les routes Gin pour l'authentification TIR
func RegisterTIRAuthRoutes(
	r *gin.Engine,
	ctrl *controllers.TIRAuthController,
) {
	// Routes publiques TIR (sans middleware)
	tirAuth := r.Group("/api/v1/tir/auth")
	{
		tirAuth.POST("/login", ctrl.Login)       // Connexion admin TIR
		tirAuth.POST("/logout", ctrl.Logout)     // Déconnexion admin TIR
		tirAuth.POST("/refresh", ctrl.Refresh)   // Renouvellement token TIR
		tirAuth.GET("/validate", ctrl.ValidateSession) // Validation session TIR
	}
}