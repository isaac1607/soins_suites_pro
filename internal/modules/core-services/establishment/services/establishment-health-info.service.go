package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/core-services/establishment/dto"
	"soins-suite-core/internal/modules/core-services/establishment/queries"
)

// EstablishmentHealthInfoService - Service métier pour les informations sanitaires d'établissements
// Core Service : logique business réutilisable (SANS endpoints)
type EstablishmentHealthInfoService struct {
	db          *postgres.Client
	redisClient *redisInfra.Client
}

// NewEstablishmentHealthInfoService - Constructeur Fx compatible
func NewEstablishmentHealthInfoService(db *postgres.Client, redisClient *redisInfra.Client) *EstablishmentHealthInfoService {
	return &EstablishmentHealthInfoService{
		db:          db,
		redisClient: redisClient,
	}
}

// GetEstablishmentHealthInfo - Récupère les informations sanitaires complètes d'un établissement
func (s *EstablishmentHealthInfoService) GetEstablishmentHealthInfo(
	ctx context.Context,
	establishmentID uuid.UUID,
) (*dto.EstablishmentHealthInfo, error) {
	var info dto.EstablishmentHealthInfo
	var lastModifiedBy *string
	var totalCount int // Non utilisé mais nécessaire pour la query
	_ = totalCount

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.GetHealthInfo,
		establishmentID,
	).Scan(
		&info.ID,
		&info.AppInstance,
		&info.CodeEtablissement,
		&info.Nom,
		&info.NomCourt,
		&info.Statut,
		&info.AdresseComplete,
		&info.TelephonePrincipal,
		&info.SecondTelephone,
		&info.Email,
		&info.Ville,
		&info.Commune,
		&info.RCCM,
		&info.CNPS,
		&info.DureeValiditeTicket,
		&info.NbSouchesParCaisse,
		&info.GardeHeureDebut,
		&info.GardeHeureFin,
		&info.LogoPrincipalURL,
		&info.LogoDocumentsURL,
		&info.CreatedAt,
		&info.UpdatedAtAdminTir,
		&info.UpdatedAtUser,
		&lastModifiedBy,
		&info.LastModifiedAt,
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
		return nil, fmt.Errorf("erreur récupération informations sanitaires: %w", err)
	}

	// Gestion du champ last_modified_by
	if lastModifiedBy != nil {
		info.LastModifiedBy = *lastModifiedBy
	}

	return &info, nil
}

// GetEstablishmentHealthInfoByCode - Récupère les informations sanitaires par code établissement
func (s *EstablishmentHealthInfoService) GetEstablishmentHealthInfoByCode(
	ctx context.Context,
	codeEtablissement string,
) (*dto.EstablishmentHealthInfo, error) {
	var info dto.EstablishmentHealthInfo
	var lastModifiedBy *string

	err := s.db.QueryRow(
		ctx,
		queries.EstablishmentQueries.GetHealthInfoByCode,
		codeEtablissement,
	).Scan(
		&info.ID,
		&info.AppInstance,
		&info.CodeEtablissement,
		&info.Nom,
		&info.NomCourt,
		&info.Statut,
		&info.AdresseComplete,
		&info.TelephonePrincipal,
		&info.SecondTelephone,
		&info.Email,
		&info.Ville,
		&info.Commune,
		&info.RCCM,
		&info.CNPS,
		&info.DureeValiditeTicket,
		&info.NbSouchesParCaisse,
		&info.GardeHeureDebut,
		&info.GardeHeureFin,
		&info.LogoPrincipalURL,
		&info.LogoDocumentsURL,
		&info.CreatedAt,
		&info.UpdatedAtAdminTir,
		&info.UpdatedAtUser,
		&lastModifiedBy,
		&info.LastModifiedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, &ServiceError{
			Type:    "not_found",
			Message: "Établissement non trouvé",
			Details: map[string]interface{}{
				"code_etablissement": codeEtablissement,
			},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("erreur récupération informations sanitaires par code: %w", err)
	}

	// Gestion du champ last_modified_by
	if lastModifiedBy != nil {
		info.LastModifiedBy = *lastModifiedBy
	}

	return &info, nil
}

// GetEstablishmentHealthInfoList - Récupère la liste paginée des informations sanitaires
func (s *EstablishmentHealthInfoService) GetEstablishmentHealthInfoList(
	ctx context.Context,
	page int,
	limit int,
) (*dto.EstablishmentHealthInfoList, error) {
	// Validation paramètres
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20 // Default
	}

	offset := (page - 1) * limit

	rows, err := s.db.Query(
		ctx,
		queries.EstablishmentQueries.GetHealthInfoList,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération liste informations sanitaires: %w", err)
	}
	defer rows.Close()

	var establishments []dto.EstablishmentHealthInfoSummary
	var totalCount int

	for rows.Next() {
		var establishment dto.EstablishmentHealthInfoSummary

		err := rows.Scan(
			&establishment.ID,
			&establishment.CodeEtablissement,
			&establishment.Nom,
			&establishment.NomCourt,
			&establishment.Ville,
			&establishment.Commune,
			&establishment.Statut,
			&establishment.TelephonePrincipal,
			&establishment.Email,
			&establishment.CreatedAt,
			&totalCount, // COUNT(*) OVER()
		)
		if err != nil {
			return nil, fmt.Errorf("erreur scan établissement: %w", err)
		}

		establishments = append(establishments, establishment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur itération rows: %w", err)
	}

	return &dto.EstablishmentHealthInfoList{
		Establishments: establishments,
		Total:          totalCount,
		Page:           page,
		Limit:          limit,
	}, nil
}

// GetActiveEstablishmentsCount - Compte le nombre d'établissements actifs
func (s *EstablishmentHealthInfoService) GetActiveEstablishmentsCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM base_etablissement WHERE statut = 'actif'`,
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("erreur comptage établissements actifs: %w", err)
	}

	return count, nil
}

// GetEstablishmentsByCity - Récupère les établissements par ville
func (s *EstablishmentHealthInfoService) GetEstablishmentsByCity(
	ctx context.Context,
	ville string,
) ([]dto.EstablishmentHealthInfoSummary, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT 
			id, code_etablissement, nom, nom_court, ville, commune, 
			statut, telephone_principal, email, created_at
		FROM base_etablissement 
		WHERE ville ILIKE $1 AND statut != 'archive'
		ORDER BY nom ASC`,
		"%"+ville+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération établissements par ville: %w", err)
	}
	defer rows.Close()

	var establishments []dto.EstablishmentHealthInfoSummary

	for rows.Next() {
		var establishment dto.EstablishmentHealthInfoSummary

		err := rows.Scan(
			&establishment.ID,
			&establishment.CodeEtablissement,
			&establishment.Nom,
			&establishment.NomCourt,
			&establishment.Ville,
			&establishment.Commune,
			&establishment.Statut,
			&establishment.TelephonePrincipal,
			&establishment.Email,
			&establishment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur scan établissement par ville: %w", err)
		}

		establishments = append(establishments, establishment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur itération rows par ville: %w", err)
	}

	return establishments, nil
}