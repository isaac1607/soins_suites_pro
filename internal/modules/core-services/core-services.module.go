package core_services

import (
	"go.uber.org/fx"

	"soins-suite-core/internal/modules/core-services/establishment"
	"soins-suite-core/internal/modules/core-services/patient"
)

// Module regroupe tous les services métier centralisés (Core Services)
// Ces services sont réutilisables par plusieurs modules sans avoir d'endpoints propres
var Module = fx.Options(
	// Patient Core Services (Génération codes, validation, etc.)
	patient.Module,

	// Establishment Core Services (Création, validation, etc.)
	establishment.Module,

	// TODO: Autres domaines Core Services à ajouter selon besoins
	// user.Module,          // Services utilisateur centralisés
)