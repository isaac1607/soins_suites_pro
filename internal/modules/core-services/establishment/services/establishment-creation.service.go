package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/core-services/establishment/dto"
	"soins-suite-core/internal/modules/core-services/establishment/queries"
)

// EstablishmentCreationService - Service métier pour la création d'établissements
// Core Service : logique business réutilisable (SANS endpoints)
type EstablishmentCreationService struct {
	db          *postgres.Client
	redisClient *redisInfra.Client
}

// NewEstablishmentCreationService - Constructeur Fx compatible
func NewEstablishmentCreationService(db *postgres.Client, redisClient *redisInfra.Client) *EstablishmentCreationService {
	return &EstablishmentCreationService{
		db:          db,
		redisClient: redisClient,
	}
}

// CreateEstablishment - Crée un nouvel établissement par admin TIR
func (s *EstablishmentCreationService) CreateEstablishment(
	ctx context.Context,
	req dto.CreateEstablishmentRequest,
	createdByAdminTirID uuid.UUID,
) (*dto.EstablishmentCreationResult, error) {
	// Validation code établissement unique
	exists, err := s.checkCodeExists(ctx, req.CodeEtablissement)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification code établissement: %w", err)
	}
	if exists {
		return nil, &ServiceError{
			Type:    "conflict",
			Message: "Le code établissement existe déjà",
			Details: map[string]interface{}{
				"code_etablissement": req.CodeEtablissement,
			},
		}
	}

	// Création établissement
	var establishment dto.EstablishmentResponse
	err = s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.Create,
		req.CodeEtablissement,
		req.Nom,
		req.NomCourt,
		req.AdresseComplete,
		req.TelephonePrincipal,
		req.Ville,
		req.Commune,
		req.Email,
		createdByAdminTirID,
	).Scan(
		&establishment.ID,
		&establishment.AppInstance,
		&establishment.CodeEtablissement,
		&establishment.Nom,
		&establishment.NomCourt,
		&establishment.AdresseComplete,
		&establishment.TelephonePrincipal,
		&establishment.Ville,
		&establishment.Commune,
		&establishment.Email,
		&establishment.Statut,
		&establishment.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("erreur création établissement: %w", err)
	}

	// Compléter les métadonnées
	establishment.CreatedBy = createdByAdminTirID

	// Mise en cache immédiate selon conventions middleware Redis
	// Utilise la même clé que EstablishmentMiddleware pour cohérence
	s.cacheEstablishmentData(ctx, &establishment)

	return &dto.EstablishmentCreationResult{
		Establishment: &establishment,
		Message:       "Établissement créé avec succès",
	}, nil
}

// GetEstablishmentByID - Récupère un établissement par ID
func (s *EstablishmentCreationService) GetEstablishmentByID(
	ctx context.Context,
	establishmentID uuid.UUID,
) (*dto.EstablishmentResponse, error) {
	var establishment dto.EstablishmentResponse

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.GetByID,
		establishmentID,
	).Scan(
		&establishment.ID,
		&establishment.AppInstance,
		&establishment.CodeEtablissement,
		&establishment.Nom,
		&establishment.NomCourt,
		&establishment.AdresseComplete,
		&establishment.TelephonePrincipal,
		&establishment.Ville,
		&establishment.Commune,
		&establishment.Email,
		&establishment.SecondTelephone,
		&establishment.RCCM,
		&establishment.CNPS,
		&establishment.LogoPrincipalURL,
		&establishment.LogoDocumentsURL,
		&establishment.DureeValiditeTicket,
		&establishment.NbSouchesParCaisse,
		&establishment.GardeHeureDebut,
		&establishment.GardeHeureFin,
		&establishment.Statut,
		&establishment.CreatedAt,
		&establishment.UpdatedAtAdminTir,
		&establishment.UpdatedAtUser,
		&establishment.CreatedBy,
		&establishment.UpdatedByAdminTir,
		&establishment.UpdatedByUser,
	)

	if err == pgx.ErrNoRows {
		return nil, &ServiceError{
			Type:    "not_found",
			Message: "Établissement non trouvé",
			Details: map[string]interface{}{
				"establishment_id": establishmentID,
			},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("erreur récupération établissement: %w", err)
	}

	return &establishment, nil
}

// GetEstablishmentByCode - Récupère un établissement par code
func (s *EstablishmentCreationService) GetEstablishmentByCode(
	ctx context.Context,
	code string,
) (*dto.EstablishmentSummary, error) {
	var establishment dto.EstablishmentSummary

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.GetByCode,
		code,
	).Scan(
		&establishment.ID,
		&establishment.AppInstance,
		&establishment.CodeEtablissement,
		&establishment.Nom,
		&establishment.NomCourt,
		&establishment.Statut,
		&establishment.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, &ServiceError{
			Type:    "not_found",
			Message: "Établissement non trouvé",
			Details: map[string]interface{}{
				"code_etablissement": code,
			},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("erreur récupération établissement par code: %w", err)
	}

	return &establishment, nil
}

// UpdateEstablishmentByAdminTir - Met à jour un établissement par admin TIR
func (s *EstablishmentCreationService) UpdateEstablishmentByAdminTir(
	ctx context.Context,
	establishmentID uuid.UUID,
	req dto.UpdateEstablishmentRequest,
	updatedByAdminTirID uuid.UUID,
) (*dto.EstablishmentUpdateResult, error) {
	var result dto.EstablishmentUpdateResult

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.UpdateByAdminTir,
		establishmentID,
		req.Nom,
		req.NomCourt,
		req.AdresseComplete,
		req.TelephonePrincipal,
		req.Ville,
		req.Commune,
		req.Email,
		updatedByAdminTirID,
	).Scan(
		&result.ID,
		&result.AppInstance,
		&result.CodeEtablissement,
		&result.Nom,
		&result.NomCourt,
		&result.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, &ServiceError{
			Type:    "not_found",
			Message: "Établissement non trouvé",
			Details: map[string]interface{}{
				"establishment_id": establishmentID,
			},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("erreur mise à jour établissement par admin TIR: %w", err)
	}

	result.UpdatedBy = &updatedByAdminTirID
	result.UpdatedByType = "admin_tir"

	return &result, nil
}

// UpdateEstablishmentByUser - Met à jour un établissement par utilisateur
func (s *EstablishmentCreationService) UpdateEstablishmentByUser(
	ctx context.Context,
	establishmentID uuid.UUID,
	req dto.UpdateEstablishmentRequest,
	updatedByUserID uuid.UUID,
) (*dto.EstablishmentUpdateResult, error) {
	var result dto.EstablishmentUpdateResult

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.UpdateByUser,
		establishmentID,
		req.Nom,
		req.NomCourt,
		req.AdresseComplete,
		req.TelephonePrincipal,
		req.Ville,
		req.Commune,
		req.Email,
		updatedByUserID,
	).Scan(
		&result.ID,
		&result.AppInstance,
		&result.CodeEtablissement,
		&result.Nom,
		&result.NomCourt,
		&result.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, &ServiceError{
			Type:    "not_found",
			Message: "Établissement non trouvé",
			Details: map[string]interface{}{
				"establishment_id": establishmentID,
			},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("erreur mise à jour établissement par utilisateur: %w", err)
	}

	result.UpdatedBy = &updatedByUserID
	result.UpdatedByType = "user"

	return &result, nil
}

// checkCodeExists - Vérifie si un code établissement existe
func (s *EstablishmentCreationService) checkCodeExists(ctx context.Context, code string) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx, queries.EstablishmentQueries.CheckCodeExists, code).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// cacheEstablishmentData - Met en cache les données d'établissement selon les conventions middleware
// Utilise le même pattern que EstablishmentMiddleware pour cohérence
func (s *EstablishmentCreationService) cacheEstablishmentData(ctx context.Context, establishment *dto.EstablishmentResponse) {
	// Création de la structure conforme au middleware (données immuables uniquement)
	establishmentData := EstablishmentData{
		ID:          establishment.ID.String(),
		AppInstance: establishment.AppInstance.String(),
		Code:        establishment.CodeEtablissement,
		Statut:      establishment.Statut,
	}

	// Sérialisation JSON
	jsonData, err := json.Marshal(establishmentData)
	if err != nil {
		// Continue sans cache en cas d'erreur de sérialisation (non bloquant)
		return
	}

	// Mise en cache avec pattern cache_middleware selon conventions Redis
	// TTL infini car données immuables (ID, app_instance, code)
	err = s.redisClient.SetWithPattern(ctx, "cache_middleware", establishment.CodeEtablissement, jsonData, "establishment")
	if err != nil {
		// Continue sans notification en cas d'erreur Redis (non bloquant)
		return
	}
}

// EstablishmentData représente les données d'établissement pour cache Redis
// DOIT être identique à la structure utilisée dans EstablishmentMiddleware
type EstablishmentData struct {
	ID          string `json:"id"`           // UUID - JAMAIS modifié
	AppInstance string `json:"app_instance"` // UUID - JAMAIS modifié
	Code        string `json:"code"`         // Code établissement - JAMAIS modifié
	Statut      string `json:"statut"`       // Seul champ pouvant changer
}

