package middleware

import (
	"go.uber.org/fx"
	"soins-suite-core/internal/shared/middleware/authentication"
)

// Module regroupe tous les providers des middlewares
var Module = fx.Options(
	// Middlewares d'authentification
	fx.Provide(authentication.NewEstablishmentMiddleware),
)