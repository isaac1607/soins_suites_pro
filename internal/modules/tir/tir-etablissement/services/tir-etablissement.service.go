package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	coreEstablishmentDTO "soins-suite-core/internal/modules/core-services/establishment/dto"
	coreEstablishmentServices "soins-suite-core/internal/modules/core-services/establishment/services"
	"soins-suite-core/internal/modules/tir/tir-etablissement/dto"
)

// TIREstablishmentService service pour gestion établissements par admin TIR
// Utilise les core-services establishment (pattern réutilisation)
type TIREstablishmentService struct {
	establishmentCreationService *coreEstablishmentServices.EstablishmentCreationService
}

// NewTIREstablishmentService constructeur Fx compatible
func NewTIREstablishmentService(
	establishmentCreationService *coreEstablishmentServices.EstablishmentCreationService,
) *TIREstablishmentService {
	return &TIREstablishmentService{
		establishmentCreationService: establishmentCreationService,
	}
}

// CreateEstablishment crée un établissement via core-service
func (s *TIREstablishmentService) CreateEstablishment(
	ctx context.Context,
	req dto.CreateEstablishmentTIRRequest,
	adminID uuid.UUID,
	adminInfo dto.AdminCreationInfo,
) (*dto.EstablishmentTIRCreationResult, error) {
	// Conversion DTO TIR vers DTO core-service
	coreReq := coreEstablishmentDTO.CreateEstablishmentRequest{
		CodeEtablissement:  req.CodeEtablissement,
		Nom:                req.Nom,
		NomCourt:           req.NomCourt,
		AdresseComplete:    req.AdresseComplete,
		TelephonePrincipal: req.TelephonePrincipal,
		Ville:              req.Ville,
		Commune:            req.Commune,
		Email:              req.Email,
	}

	// Appel core-service (logique business centralisée)
	coreResult, err := s.establishmentCreationService.CreateEstablishment(
		ctx,
		coreReq,
		adminID,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service création établissement: %w", err)
	}

	// Conversion DTO core-service vers DTO TIR
	tirResponse := &dto.EstablishmentTIRResponse{
		ID:                 coreResult.Establishment.ID,
		AppInstance:        coreResult.Establishment.AppInstance,
		CodeEtablissement:  coreResult.Establishment.CodeEtablissement,
		Nom:                coreResult.Establishment.Nom,
		NomCourt:           coreResult.Establishment.NomCourt,
		AdresseComplete:    coreResult.Establishment.AdresseComplete,
		TelephonePrincipal: coreResult.Establishment.TelephonePrincipal,
		Ville:              coreResult.Establishment.Ville,
		Commune:            coreResult.Establishment.Commune,
		Email:              &coreResult.Establishment.Email,
		Statut:             coreResult.Establishment.Statut,
		CreatedAt:          coreResult.Establishment.CreatedAt,
		CreatedBy:          coreResult.Establishment.CreatedBy,
		CreatedByAdmin:     adminInfo.Identifiant,
	}

	return &dto.EstablishmentTIRCreationResult{
		Success:       true,
		Establishment: tirResponse,
		Message:       "Établissement créé avec succès par admin TIR",
		AdminInfo:     adminInfo,
	}, nil
}

// GetEstablishmentByID récupère un établissement par ID via core-service
func (s *TIREstablishmentService) GetEstablishmentByID(
	ctx context.Context,
	establishmentID uuid.UUID,
) (*dto.EstablishmentTIRResponse, error) {
	// Appel core-service
	coreEstablishment, err := s.establishmentCreationService.GetEstablishmentByID(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service récupération établissement: %w", err)
	}

	// Conversion vers DTO TIR
	return &dto.EstablishmentTIRResponse{
		ID:                 coreEstablishment.ID,
		AppInstance:        coreEstablishment.AppInstance,
		CodeEtablissement:  coreEstablishment.CodeEtablissement,
		Nom:                coreEstablishment.Nom,
		NomCourt:           coreEstablishment.NomCourt,
		AdresseComplete:    coreEstablishment.AdresseComplete,
		TelephonePrincipal: coreEstablishment.TelephonePrincipal,
		Ville:              coreEstablishment.Ville,
		Commune:            coreEstablishment.Commune,
		Email:              &coreEstablishment.Email,
		Statut:             coreEstablishment.Statut,
		CreatedAt:          coreEstablishment.CreatedAt,
		CreatedBy:          coreEstablishment.CreatedBy,
		CreatedByAdmin:     "", // À enrichir si nécessaire
	}, nil
}

// GetEstablishmentByCode récupère un établissement par code via core-service
func (s *TIREstablishmentService) GetEstablishmentByCode(
	ctx context.Context,
	code string,
) (*dto.EstablishmentTIRSummary, error) {
	// Appel core-service
	coreEstablishment, err := s.establishmentCreationService.GetEstablishmentByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service récupération établissement par code: %w", err)
	}

	// Conversion vers DTO TIR Summary
	return &dto.EstablishmentTIRSummary{
		ID:                coreEstablishment.ID,
		CodeEtablissement: coreEstablishment.CodeEtablissement,
		Nom:               coreEstablishment.Nom,
		NomCourt:          coreEstablishment.NomCourt,
		Ville:             "", // Non disponible dans summary core
		Commune:           "", // Non disponible dans summary core
		Statut:            coreEstablishment.Statut,
		CreatedAt:         coreEstablishment.CreatedAt,
	}, nil
}

// UpdateEstablishmentByAdminTir met à jour un établissement via core-service
func (s *TIREstablishmentService) UpdateEstablishmentByAdminTir(
	ctx context.Context,
	establishmentID uuid.UUID,
	req dto.UpdateEstablishmentTIRRequest,
	adminID uuid.UUID,
	adminInfo dto.AdminCreationInfo,
) (*dto.EstablishmentTIRUpdateResult, error) {
	// Conversion DTO TIR vers DTO core-service
	coreReq := coreEstablishmentDTO.UpdateEstablishmentRequest{
		Nom:                req.Nom,
		NomCourt:           req.NomCourt,
		AdresseComplete:    req.AdresseComplete,
		TelephonePrincipal: req.TelephonePrincipal,
		Ville:              req.Ville,
		Commune:            req.Commune,
	}

	// Appel core-service
	coreResult, err := s.establishmentCreationService.UpdateEstablishmentByAdminTir(
		ctx,
		establishmentID,
		coreReq,
		adminID,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur core-service mise à jour établissement: %w", err)
	}

	// Conversion résultat vers DTO TIR
	// Note: UpdateResult ne contient que les champs mis à jour, pas toutes les données
	tirResponse := &dto.EstablishmentTIRResponse{
		ID:                coreResult.ID,
		AppInstance:       coreResult.AppInstance,
		CodeEtablissement: coreResult.CodeEtablissement,
		Nom:               coreResult.Nom,
		NomCourt:          coreResult.NomCourt,
		// Les autres champs ne sont pas disponibles dans UpdateResult
		// CreatedAt serait à récupérer via GetEstablishmentByID si nécessaire
	}

	return &dto.EstablishmentTIRUpdateResult{
		Success:       true,
		Establishment: tirResponse,
		Message:       "Établissement mis à jour avec succès par admin TIR",
		UpdatedBy:     adminInfo,
	}, nil
}
