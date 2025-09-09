package tir_etablissement

import (
	"go.uber.org/fx"
	"github.com/gin-gonic/gin"

	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	tirAuthMiddleware "soins-suite-core/internal/shared/middleware/tir-auth"
	"soins-suite-core/internal/modules/tir/tir-etablissement/controllers"
	"soins-suite-core/internal/modules/tir/tir-etablissement/services"
)

// Module regroupe tous les providers du domaine TIR Établissement
var Module = fx.Options(
	// Services (utilisent core-services directement)
	fx.Provide(services.NewTIREstablishmentService),
	fx.Provide(services.NewTIRLicenseService),

	// Controllers
	fx.Provide(controllers.NewTIREstablishmentController),
	fx.Provide(controllers.NewTIRLicenseController),

	// Configuration des routes TIR
	fx.Invoke(RegisterTIREstablishmentRoutes),
	fx.Invoke(RegisterTIRLicenseRoutes),
)

// RegisterTIREstablishmentRoutes configure les routes Gin pour TIR Établissement
func RegisterTIREstablishmentRoutes(
	r *gin.Engine,
	ctrl *controllers.TIREstablishmentController,
	redisClient *redisInfra.Client,
) {
	// Groupe routes TIR avec authentification
	tirAPI := r.Group("/api/v1/tir/establishments")

	// Middleware TIR Session (requis pour tous les endpoints TIR)
	tirAPI.Use(tirAuthMiddleware.TIRSessionMiddleware(redisClient))

	// Routes avec permission "gerer_etablissements"
	establishmentPermissionRoutes := tirAPI.Group("")
	establishmentPermissionRoutes.Use(
		tirAuthMiddleware.TIRPermissionMiddleware(tirAuthMiddleware.PermissionGererEtablissements),
	)
	{
		// POST /api/v1/tir/establishments - Créer établissement
		establishmentPermissionRoutes.POST("", ctrl.CreateEstablishment)

		// PUT /api/v1/tir/establishments/:id - Mettre à jour établissement
		establishmentPermissionRoutes.PUT("/:id", ctrl.UpdateEstablishment)
	}

	// Routes avec permission "acceder_donnees_etablissement" (lecture seule)
	readOnlyPermissionRoutes := tirAPI.Group("")
	readOnlyPermissionRoutes.Use(
		tirAuthMiddleware.TIRPermissionMiddleware(tirAuthMiddleware.PermissionAccederDonneesEtablissement),
	)
	{
		// GET /api/v1/tir/establishments/:id - Récupérer établissement par ID
		readOnlyPermissionRoutes.GET("/:id", ctrl.GetEstablishment)

		// GET /api/v1/tir/establishments/by-code/:code - Récupérer par code
		readOnlyPermissionRoutes.GET("/by-code/:code", ctrl.GetEstablishmentByCode)
	}
}

// RegisterTIRLicenseRoutes configure les routes Gin pour TIR Licences
func RegisterTIRLicenseRoutes(
	r *gin.Engine,
	ctrl *controllers.TIRLicenseController,
	redisClient *redisInfra.Client,
) {
	// Groupe routes TIR Licences avec authentification
	tirLicenseAPI := r.Group("/api/v1/tir")

	// Middleware TIR Session (requis pour tous les endpoints TIR)
	tirLicenseAPI.Use(tirAuthMiddleware.TIRSessionMiddleware(redisClient))

	// Routes licences avec permission "gerer_licences"
	licenseManagementRoutes := tirLicenseAPI.Group("")
	licenseManagementRoutes.Use(
		tirAuthMiddleware.TIRPermissionMiddleware(tirAuthMiddleware.PermissionGererLicences),
	)
	{
		// POST /api/v1/tir/licenses - Créer licence
		licenseManagementRoutes.POST("/licenses", ctrl.CreateLicense)

		// GET /api/v1/tir/licenses/available-modules - Modules disponibles
		licenseManagementRoutes.GET("/licenses/available-modules", ctrl.GetAvailableModules)
	}

	// Routes licences avec permission "acceder_donnees_etablissement" (lecture seule)
	licenseReadOnlyRoutes := tirLicenseAPI.Group("")
	licenseReadOnlyRoutes.Use(
		tirAuthMiddleware.TIRPermissionMiddleware(tirAuthMiddleware.PermissionAccederDonneesEtablissement),
	)
	{
		// GET /api/v1/tir/licenses/:id - Récupérer licence par ID
		licenseReadOnlyRoutes.GET("/licenses/:id", ctrl.GetLicense)

		// GET /api/v1/tir/establishments/:establishment_id/license/active - Licence active d'un établissement
		licenseReadOnlyRoutes.GET("/establishments/:establishment_id/license/active", ctrl.GetActiveLicenseByEstablishment)

		// GET /api/v1/tir/establishments/:establishment_id/licenses - Liste licences d'un établissement
		licenseReadOnlyRoutes.GET("/establishments/:establishment_id/licenses", ctrl.GetLicenseListByEstablishment)
	}
}