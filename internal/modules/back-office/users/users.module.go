package users

import (
	"go.uber.org/fx"
	"github.com/gin-gonic/gin"

	controllers "soins-suite-core/internal/modules/back-office/users/controllers/comptes"
	services "soins-suite-core/internal/modules/back-office/users/services/comptes"
	authMiddleware "soins-suite-core/internal/shared/middleware/auth"
)

var Module = fx.Options(
	fx.Provide(services.NewComptesService),
	fx.Provide(controllers.NewComptesController),
	fx.Invoke(RegisterUsersRoutes),
)

func RegisterUsersRoutes(
	r *gin.Engine, 
	ctrl *controllers.ComptesController,
	authStack *authMiddleware.AuthMiddlewareStack,
) {
	api := r.Group("/api/v1/back-office/users")
	api.Use(authMiddleware.RequireAdmin(authStack)...)
	{
		api.GET("", ctrl.ListUsers)
		api.GET("/:id", ctrl.GetUserDetails)
		api.POST("", ctrl.CreateUser)
		api.PUT("/:id/permissions", ctrl.ModifyUserPermissions)
	}
}
