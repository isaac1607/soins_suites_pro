package patient

import (
	"go.uber.org/fx"

	"soins-suite-core/internal/modules/core-services/patient/services"
)

// Module regroupe tous les services métier centralisés du domaine Patient
// IMPORTANT: Ce module ne contient PAS de controllers (Core Services sans endpoints)
var Module = fx.Options(
	// Services Core - Ordre d'injection important (dépendances)
	fx.Provide(services.NewPatientCodeGeneratorService), // CS-P-001: Génération codes (CRITIQUE)
	fx.Provide(services.NewPatientValidationService),    // CS-P-005: Validation et anti-doublon
	fx.Provide(services.NewPatientCreationService),      // CS-P-002: Création complète de patient
	fx.Provide(services.NewPatientSearchService),        // CS-P-002: Recherche multi-critères
	fx.Provide(services.NewPatientCacheService),         // CS-P-003: Cache Redis intelligent

	// Services Core complètement implémentés selon spécifications
)