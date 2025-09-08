package middleware

import (
	"go.uber.org/fx"
	"soins-suite-core/internal/shared/middleware/auth"
	"soins-suite-core/internal/shared/middleware/core"
	"soins-suite-core/internal/shared/middleware/security"
	"soins-suite-core/internal/shared/middleware/tenant"
)

// Module regroupe tous les providers des middlewares
var Module = fx.Options(
	// Core middlewares
	fx.Provide(core.RecoveryMiddleware),
	fx.Provide(security.CORSMiddleware),

	// Tenant middlewares
	fx.Provide(tenant.NewEstablishmentMiddleware),
	fx.Provide(tenant.NewLicenseMiddleware),

	// Auth middlewares
	auth.AuthMiddlewareModule,
)