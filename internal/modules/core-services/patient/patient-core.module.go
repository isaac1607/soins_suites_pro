package patient

import (
	"go.uber.org/fx"

	"soins-suite-core/internal/modules/core-services/patient/services"
)

// Module regroupe tous les services métier centralisés du domaine Patient
// IMPORTANT: Ce module ne contient PAS de controllers (Core Services sans endpoints)
var Module = fx.Options(
	// Services Core - Génération de codes patient (CRITIQUE)
	fx.Provide(services.NewPatientCodeGeneratorService),

	// TODO: Autres services Core à ajouter progressivement
	// fx.Provide(services.NewPatientValidationService),
	// fx.Provide(services.NewPatientCreationService),
	// fx.Provide(services.NewPatientSearchService),
	// fx.Provide(services.NewPatientCacheService),
)