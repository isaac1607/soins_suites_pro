package services

import (
	"context"
	"fmt"
	"net"

	"github.com/google/uuid"

	coreEstablishmentDTO "soins-suite-core/internal/modules/core-services/establishment/dto"
	coreEstablishmentServices "soins-suite-core/internal/modules/core-services/establishment/services"
	"soins-suite-core/internal/modules/tir/tir-etablissement/dto"
)

// TIRLicenseService service pour gestion licences par admin TIR
// Utilise les core-services establishment (pattern réutilisation)
type TIRLicenseService struct {
	licenseCreationService    *coreEstablishmentServices.LicenseCreationService
	licenseConsultationService *coreEstablishmentServices.LicenseConsultationService
}

// NewTIRLicenseService constructeur Fx compatible
func NewTIRLicenseService(
	licenseCreationService *coreEstablishmentServices.LicenseCreationService,
	licenseConsultationService *coreEstablishmentServices.LicenseConsultationService,
) *TIRLicenseService {
	return &TIRLicenseService{
		licenseCreationService:    licenseCreationService,
		licenseConsultationService: licenseConsultationService,
	}
}

// CreateLicense crée une licence via core-service
func (s *TIRLicenseService) CreateLicense(
	ctx context.Context,
	req dto.CreateLicenseTIRRequest,
	adminID uuid.UUID,
	adminInfo dto.AdminCreationInfo,
	userIP *net.IP,
) (*dto.LicenseTIRCreationResult, error) {
	
	// Conversion DTO TIR vers DTO core-service
	coreReq := coreEstablishmentDTO.CreateLicenseRequest{
		EtablissementID:  req.EtablissementID,
		ModeDeploiement:  req.ModeDeploiement,
		TypeLicence:      req.TypeLicence,
		ModulesAutorises: req.ModulesAutorises,
		DateExpiration:   req.DateExpiration,
	}

	// Appel core-service (logique business centralisée)
	coreResult, err := s.licenseCreationService.CreateLicense(
		ctx,
		coreReq,
		adminID,
		userIP,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service création licence: %w", err)
	}

	// Conversion DTO core-service vers DTO TIR
	tirResponse := s.convertLicenseResponseToTIR(coreResult.License, adminInfo.Identifiant)

	return &dto.LicenseTIRCreationResult{
		Success:   true,
		License:   tirResponse,
		Message:   fmt.Sprintf("Licence créée avec succès par admin TIR. %s", coreResult.Message),
		AdminInfo: adminInfo,
	}, nil
}

// GetLicenseByID récupère une licence par ID via core-service
func (s *TIRLicenseService) GetLicenseByID(
	ctx context.Context,
	licenseID uuid.UUID,
	includeHistory bool,
	adminInfo *dto.AdminCreationInfo,
) (*dto.LicenseTIRConsultationResult, error) {
	
	// Appel core-service
	coreResult, err := s.licenseConsultationService.GetLicenseByID(ctx, licenseID, includeHistory)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service consultation licence: %w", err)
	}

	// Conversion vers DTO TIR
	tirResponse := s.convertLicenseDetailedResponseToTIR(coreResult.License, "")
	tirHistory := s.convertHistoryToTIR(coreResult.History)

	return &dto.LicenseTIRConsultationResult{
		Success:   true,
		License:   tirResponse,
		History:   tirHistory,
		Message:   coreResult.Message,
		AdminInfo: adminInfo,
	}, nil
}

// GetActiveLicenseByEstablishment récupère la licence active d'un établissement via core-service
func (s *TIRLicenseService) GetActiveLicenseByEstablishment(
	ctx context.Context,
	etablissementID uuid.UUID,
	includeHistory bool,
	adminInfo *dto.AdminCreationInfo,
) (*dto.LicenseTIRConsultationResult, error) {
	
	// Appel core-service
	coreResult, err := s.licenseConsultationService.GetActiveLicenseByEstablishment(ctx, etablissementID, includeHistory)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service consultation licence active: %w", err)
	}

	// Conversion vers DTO TIR
	tirResponse := s.convertLicenseDetailedResponseToTIR(coreResult.License, "")
	tirHistory := s.convertHistoryToTIR(coreResult.History)

	return &dto.LicenseTIRConsultationResult{
		Success:   true,
		License:   tirResponse,
		History:   tirHistory,
		Message:   coreResult.Message,
		AdminInfo: adminInfo,
	}, nil
}

// GetLicenseListByEstablishment récupère toutes les licences d'un établissement via core-service
func (s *TIRLicenseService) GetLicenseListByEstablishment(
	ctx context.Context,
	etablissementID uuid.UUID,
) (*dto.LicenseListTIRResponse, error) {
	
	// Appel core-service
	coreResult, err := s.licenseConsultationService.GetLicenseListByEstablishment(ctx, etablissementID)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service liste licences: %w", err)
	}

	// Conversion vers DTO TIR
	tirLicenses := make([]dto.LicenseSummaryTIR, len(coreResult.Licenses))
	for i, coreLicense := range coreResult.Licenses {
		tirLicenses[i] = s.convertLicenseSummaryToTIR(&coreLicense, "", "")
	}

	return &dto.LicenseListTIRResponse{
		Success:  true,
		Licenses: tirLicenses,
		Total:    coreResult.Total,
		Message:  fmt.Sprintf("Liste des licences récupérée avec succès. %d licence(s) trouvée(s).", coreResult.Total),
	}, nil
}

// GetAvailableFrontOfficeModules récupère les modules front-office disponibles via core-service
func (s *TIRLicenseService) GetAvailableFrontOfficeModules(
	ctx context.Context,
) (*dto.AvailableModulesTIRResponse, error) {
	
	// Appel core-service
	coreResult, err := s.licenseCreationService.GetAvailableFrontOfficeModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service modules disponibles: %w", err)
	}

	// Conversion vers DTO TIR
	tirModules := make([]dto.AvailableModuleTIRInfo, len(coreResult.Modules))
	for i, coreModule := range coreResult.Modules {
		tirModules[i] = dto.AvailableModuleTIRInfo{
			ID:          coreModule.ID,
			CodeModule:  coreModule.CodeModule,
			Nom:         coreModule.Nom,
			Description: coreModule.Description,
			EstActif:    coreModule.EstActif,
		}
	}

	return &dto.AvailableModulesTIRResponse{
		Success: true,
		Modules: tirModules,
		Total:   coreResult.Total,
		Message: fmt.Sprintf("Modules front-office disponibles récupérés avec succès. %d module(s) disponible(s).", coreResult.Total),
	}, nil
}

// convertLicenseResponseToTIR convertit une LicenseResponse core vers TIR
func (s *TIRLicenseService) convertLicenseResponseToTIR(
	coreLicense *coreEstablishmentDTO.LicenseResponse,
	adminIdentifiant string,
) *dto.LicenseTIRResponse {
	
	// Conversion des modules
	tirModules := make([]dto.LicenseModuleTIRInfo, len(coreLicense.ModulesAutorises))
	for i, module := range coreLicense.ModulesAutorises {
		tirModules[i] = dto.LicenseModuleTIRInfo{
			ID:   module.ID,
			Code: module.Code,
			Nom:  "", // Nom non disponible dans core ModuleInfo
		}
	}

	return &dto.LicenseTIRResponse{
		ID:                   coreLicense.ID,
		EtablissementID:      coreLicense.EtablissementID,
		EtablissementNom:     "", // À enrichir si nécessaire
		EtablissementCode:    "", // À enrichir si nécessaire
		ModeDeploiement:      coreLicense.ModeDeploiement,
		TypeLicence:          coreLicense.TypeLicence,
		ModulesAutorises:     tirModules,
		NombreModules:        len(tirModules),
		DateActivation:       coreLicense.DateActivation,
		DateExpiration:       coreLicense.DateExpiration,
		Statut:               coreLicense.Statut,
		StatutCalcule:        "", // Non disponible dans core LicenseResponse
		JoursAvantExpiration: nil, // Non disponible dans core LicenseResponse
		EstExpire:            false, // Non disponible dans core LicenseResponse
		EstBientotExpire:     false, // Non disponible dans core LicenseResponse
		SyncInitialComplete:  coreLicense.SyncInitialComplete,
		DateSyncInitial:      coreLicense.DateSyncInitial,
		CreatedAt:            coreLicense.CreatedAt,
		UpdatedAt:            coreLicense.UpdatedAt,
		CreatedBy:            coreLicense.CreatedBy,
		CreatedByAdmin:       adminIdentifiant,
	}
}

// convertLicenseDetailedResponseToTIR convertit une LicenseDetailedResponse core vers TIR
func (s *TIRLicenseService) convertLicenseDetailedResponseToTIR(
	coreLicense *coreEstablishmentDTO.LicenseDetailedResponse,
	adminIdentifiant string,
) *dto.LicenseTIRResponse {
	
	// Conversion des modules
	tirModules := make([]dto.LicenseModuleTIRInfo, len(coreLicense.ModulesAutorises))
	for i, module := range coreLicense.ModulesAutorises {
		tirModules[i] = dto.LicenseModuleTIRInfo{
			ID:   module.ID,
			Code: module.Code,
			Nom:  "", // Nom non disponible dans core ModuleInfo
		}
	}

	return &dto.LicenseTIRResponse{
		ID:                   coreLicense.ID,
		EtablissementID:      coreLicense.EtablissementID,
		EtablissementNom:     coreLicense.EtablissementNom,
		EtablissementCode:    coreLicense.EtablissementCode,
		ModeDeploiement:      coreLicense.ModeDeploiement,
		TypeLicence:          coreLicense.TypeLicence,
		ModulesAutorises:     tirModules,
		NombreModules:        coreLicense.NombreModules,
		DateActivation:       coreLicense.DateActivation,
		DateExpiration:       coreLicense.DateExpiration,
		Statut:               coreLicense.Statut,
		StatutCalcule:        coreLicense.StatutCalcule,
		JoursAvantExpiration: coreLicense.JoursAvantExpiration,
		EstExpire:            coreLicense.EstExpire,
		EstBientotExpire:     coreLicense.EstBientotExpire,
		SyncInitialComplete:  coreLicense.SyncInitialComplete,
		DateSyncInitial:      coreLicense.DateSyncInitial,
		CreatedAt:            coreLicense.CreatedAt,
		UpdatedAt:            coreLicense.UpdatedAt,
		CreatedBy:            coreLicense.CreatedBy,
		CreatedByAdmin:       adminIdentifiant,
	}
}

// convertLicenseSummaryToTIR convertit un LicenseSummary core vers TIR
func (s *TIRLicenseService) convertLicenseSummaryToTIR(
	coreLicense *coreEstablishmentDTO.LicenseSummary,
	etablissementNom string,
	etablissementCode string,
) dto.LicenseSummaryTIR {
	
	return dto.LicenseSummaryTIR{
		ID:                   coreLicense.ID,
		EtablissementNom:     etablissementNom, // À enrichir si nécessaire
		EtablissementCode:    etablissementCode, // À enrichir si nécessaire
		ModeDeploiement:      coreLicense.ModeDeploiement,
		TypeLicence:          coreLicense.TypeLicence,
		Statut:               coreLicense.Statut,
		StatutCalcule:        coreLicense.StatutCalcule,
		DateActivation:       coreLicense.DateActivation,
		DateExpiration:       coreLicense.DateExpiration,
		NombreModules:        coreLicense.NombreModules,
		EstExpire:            coreLicense.EstExpire,
		JoursAvantExpiration: coreLicense.JoursAvantExpiration,
		CreatedAt:            coreLicense.CreatedAt,
	}
}

// convertHistoryToTIR convertit l'historique core vers TIR
func (s *TIRLicenseService) convertHistoryToTIR(
	coreHistory []coreEstablishmentDTO.LicenseHistoryEntry,
) []dto.LicenseHistoryTIREntry {
	
	tirHistory := make([]dto.LicenseHistoryTIREntry, len(coreHistory))
	for i, entry := range coreHistory {
		tirHistory[i] = dto.LicenseHistoryTIREntry{
			ID:                entry.ID,
			TypeEvenement:     entry.TypeEvenement,
			StatutPrecedent:   entry.StatutPrecedent,
			StatutNouveau:     entry.StatutNouveau,
			MotifChangement:   entry.MotifChangement,
			UtilisateurAction: entry.UtilisateurAction,
			IPAction:          entry.IPAction,
			CreatedAt:         entry.CreatedAt,
		}
	}
	return tirHistory
}