package system

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"soins-suite-core/internal/modules/system/controllers"
	"soins-suite-core/internal/modules/system/services"
	middlewareAuth "soins-suite-core/internal/shared/middleware/tenant"
)

// Module regroupe tous les providers du domaine System
var Module = fx.Options(
	// Services (utilisent queries directement)
	fx.Provide(services.NewSystemService),

	// Controllers
	fx.Provide(controllers.NewSystemController),

	// Configuration des routes
	fx.Invoke(RegisterSystemRoutes),
)

// RegisterSystemRoutes configure les routes Gin pour System
func RegisterSystemRoutes(
	r *gin.Engine,
	ctrl *controllers.SystemController,
	establishmentMiddleware *middlewareAuth.EstablishmentMiddleware,
) {
	// Routes système avec middleware d'authentification requis
	api := r.Group("/api/v1/system")

	// Application du middleware EstablishmentMiddleware
	// Validation du header X-Establishment-Code et injection du contexte
	api.Use(establishmentMiddleware.Handler())

	{
		// Endpoint principal - GET /api/v1/system/info
		// Protégé par EstablishmentMiddleware
		api.GET("/info", ctrl.GetSystemInfo)

		// Endpoint modules autorisés - GET /api/v1/system/modules/authorized
		// Protégé par EstablishmentMiddleware
		api.GET("/modules/authorized", ctrl.GetAuthorizedModules)

		// Endpoint activation offline - POST /api/v1/system/activated
		// Protégé par EstablishmentMiddleware
		api.POST("/activated", ctrl.ActivateOffline)

		// Endpoint synchronisation offline - POST /api/v1/system/offline/synchronised
		// Protégé par EstablishmentMiddleware
		api.POST("/offline/synchronised", ctrl.SynchronizeOffline)
	}
}
