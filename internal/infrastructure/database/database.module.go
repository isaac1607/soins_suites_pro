package database

import (
	"soins-suite-core/internal/infrastructure/database/atlas"
	"soins-suite-core/internal/infrastructure/database/mongodb"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"

	"go.uber.org/fx"
)

var Module = fx.Options(

	// Modules database
	postgres.Module,
	redis.Module,
	mongodb.Module,
	atlas.Module,
)
