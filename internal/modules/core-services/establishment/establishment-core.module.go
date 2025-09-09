package establishment

import (
	"go.uber.org/fx"
	
	"soins-suite-core/internal/modules/core-services/establishment/services"
)

// Module regroupe les services métier établissement (SANS endpoints)
// Core Service : logique business réutilisable entre modules
var Module = fx.Options(
	// Services métier uniquement
	fx.Provide(services.NewEstablishmentCreationService),
	fx.Provide(services.NewEstablishmentHealthInfoService),
	fx.Provide(services.NewLicenseCreationService),
	fx.Provide(services.NewLicenseConsultationService),
	
	// PAS de controllers, PAS de routes
)