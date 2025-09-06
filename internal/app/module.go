package app

import (
	"soins-suite-core/internal/app/bootstrap"
	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database"
	"soins-suite-core/internal/infrastructure/logger"
	"soins-suite-core/internal/shared/middleware"
	"soins-suite-core/internal/modules/system"

	"go.uber.org/fx"
)

// La fonction NewRedisKeyGenerator est maintenant fournie directement par le module Redis

var AppModule = fx.Options(
	// Configuration (doit être fournie en premier)
	fx.Provide(config.NewConfig),
	fx.Provide(config.NewDatabaseConfigProvider),
	fx.Provide(config.NewAtlasConfigFromApp),
	fx.Provide(config.NewPostgresConfig),
	fx.Provide(config.NewRedisConfig),
	fx.Provide(config.NewMongoConfig),

	// Utilitaires partagés (après config, avant infrastructure)
	// NewRedisKeyGenerator est maintenant fourni par redis.Module

	// Infrastructure
	database.Module,
	logger.Module,

	// Middlewares partagés (après infrastructure, avant modules métier)
	middleware.Module,

	// Modules métier
	system.Module,

	// Bootstrap System - Providers
	fx.Provide(bootstrap.NewBootstrapExtensionManager),
	fx.Provide(bootstrap.NewBootstrapMigrationManager),
	fx.Provide(bootstrap.NewBootstrapSeedingManager),
	fx.Provide(bootstrap.NewBootstrapSystem),

	// Router
	fx.Provide(NewRouter),

	// Application
	fx.Provide(NewApplication),

	// Lifecycle management
	fx.Invoke(bootstrap.RegisterBootstrapLifecycle),
	fx.Invoke((*Application).Start),
)
