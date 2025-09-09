package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/patient/dto"
	"soins-suite-core/internal/modules/core-services/patient/queries"
)

// PatientSearchService gère les recherches multi-critères de patients
type PatientSearchService struct {
	db        *postgres.Client
	redis     *redis.Client
	redisKeys *PatientRedisKeys
}

// NewPatientSearchService crée une nouvelle instance du service
func NewPatientSearchService(
	db *postgres.Client,
	redis *redis.Client,
) *PatientSearchService {
	return &PatientSearchService{
		db:        db,
		redis:     redis,
		redisKeys: NewPatientRedisKeys(),
	}
}

// SearchPatients effectue une recherche multi-critères de patients
func (s *PatientSearchService) SearchPatients(
	ctx context.Context,
	req *dto.SearchPatientRequest,
) (*dto.SearchPatientResponse, error) {
	startTime := time.Now()

	// Validation et normalisation de la requête
	if err := s.validateAndNormalizeRequest(req); err != nil {
		return nil, fmt.Errorf("invalid search request: %w", err)
	}

	// Stratégie de recherche basée sur le type
	searchType := req.GetSearchType()
	appliedFilters := req.GetAppliedFilters()

	switch searchType {
	case dto.SearchTypeDirectCode:
		return s.searchByCodeWithCache(ctx, req, startTime, appliedFilters)
	case dto.SearchTypeFullText:
		return s.searchByFullText(ctx, req, startTime, appliedFilters)
	default:
		return s.searchByCriteria(ctx, req, startTime, appliedFilters)
	}
}

// searchByCodeWithCache effectue une recherche directe par code avec stratégie cache-first
func (s *PatientSearchService) searchByCodeWithCache(
	ctx context.Context,
	req *dto.SearchPatientRequest,
	startTime time.Time,
	appliedFilters []string,
) (*dto.SearchPatientResponse, error) {
	// 1. Tentative cache Redis (< 5ms selon spécifications)
	cacheKey := fmt.Sprintf("soins_suite_patient_cache:%s", req.CodePatient)
	
	// Vérifier si le patient existe en cache
	exists, err := s.redis.Exists(ctx, cacheKey).Result()
	if err == nil && exists > 0 {
		// Patient trouvé en cache - construire réponse depuis cache
		patientData := s.redis.HGetAll(ctx, cacheKey).Val()
		if len(patientData) > 0 {
			patient, err := s.parsePatientFromCache(patientData)
			if err == nil {
				// Réponse depuis cache
				return &dto.SearchPatientResponse{
					Patients:   []dto.PatientSearchResult{*patient},
					Pagination: dto.NewPaginationInfo(req.Page, req.Limit, 1),
					SearchInfo: dto.NewSearchMetadata(
						dto.SearchTypeDirectCode,
						int(time.Since(startTime).Milliseconds()),
						1,
						appliedFilters,
					),
				}, nil
			}
		}
	}

	// 2. Fallback PostgreSQL + Cache warming
	return s.searchByCodeFromDatabase(ctx, req, startTime, appliedFilters)
}

// searchByCodeFromDatabase effectue une recherche par code depuis PostgreSQL
func (s *PatientSearchService) searchByCodeFromDatabase(
	ctx context.Context,
	req *dto.SearchPatientRequest,
	startTime time.Time,
	appliedFilters []string,
) (*dto.SearchPatientResponse, error) {
	var patient dto.PatientSearchResult
	var score *float64

	err := s.db.QueryRow(ctx,
		queries.PatientSearchQueries.GetPatientByCodeFromCache,
		req.CodePatient,
	).Scan(
		&patient.ID,
		&patient.CodePatient,
		&patient.Nom,
		&patient.Prenoms,
		&patient.DateNaissance,
		&patient.Sexe,
		&patient.TelephonePrincipal,
		&patient.AdresseComplete,
		&patient.EstAssure,
		&patient.Statut,
		&patient.CreatedAt,
		&score,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			// Patient non trouvé
			return &dto.SearchPatientResponse{
				Patients:   []dto.PatientSearchResult{},
				Pagination: dto.NewPaginationInfo(req.Page, req.Limit, 0),
				SearchInfo: dto.NewSearchMetadata(
					dto.SearchTypeDirectCode,
					int(time.Since(startTime).Milliseconds()),
					0,
					appliedFilters,
				),
			}, nil
		}
		return nil, fmt.Errorf("failed to search patient by code: %w", err)
	}

	patient.Score = score

	// Cache warming asynchrone pour optimiser les prochaines recherches
	go s.warmPatientInCache(context.Background(), &patient)

	return &dto.SearchPatientResponse{
		Patients:   []dto.PatientSearchResult{patient},
		Pagination: dto.NewPaginationInfo(req.Page, req.Limit, 1),
		SearchInfo: dto.NewSearchMetadata(
			dto.SearchTypeDirectCode,
			int(time.Since(startTime).Milliseconds()),
			1,
			appliedFilters,
		),
	}, nil
}

// searchByFullText effectue une recherche full-text avec search_vector
func (s *PatientSearchService) searchByFullText(
	ctx context.Context,
	req *dto.SearchPatientRequest,
	startTime time.Time,
	appliedFilters []string,
) (*dto.SearchPatientResponse, error) {
	// 1. Compter le nombre total de résultats
	var total int
	err := s.db.QueryRow(ctx,
		queries.PatientSearchQueries.CountPatientsFullText,
		req.Statut,                // $1
		req.SearchTerm,            // $2
		req.Nom,                   // $3
		req.Prenoms,               // $4
		req.TelephonePrincipal,    // $5
		req.DateNaissanceDebut,    // $6
		req.DateNaissanceFin,      // $7
		req.Sexe,                  // $8
		req.CniNni,                // $9
		req.EstAssure,             // $10
		req.EtablissementCreateur, // $11
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// 2. Récupérer les résultats paginés
	var patients []dto.PatientSearchResult

	if req.IncludeAssurances {
		patients, err = s.searchWithAssurances(ctx, req)
	} else {
		patients, err = s.searchWithoutAssurances(ctx, req, queries.PatientSearchQueries.SearchPatientsFullText)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}

	return &dto.SearchPatientResponse{
		Patients:   patients,
		Pagination: dto.NewPaginationInfo(req.Page, req.Limit, total),
		SearchInfo: dto.NewSearchMetadata(
			dto.SearchTypeFullText,
			int(time.Since(startTime).Milliseconds()),
			total,
			appliedFilters,
		),
	}, nil
}

// searchByCriteria effectue une recherche par critères spécifiques
func (s *PatientSearchService) searchByCriteria(
	ctx context.Context,
	req *dto.SearchPatientRequest,
	startTime time.Time,
	appliedFilters []string,
) (*dto.SearchPatientResponse, error) {
	// 1. Compter le nombre total de résultats
	var total int
	err := s.db.QueryRow(ctx,
		queries.PatientSearchQueries.CountPatientsByCriteria,
		req.Statut,                // $1
		req.Nom,                   // $2
		req.Prenoms,               // $3
		req.TelephonePrincipal,    // $4
		req.DateNaissanceDebut,    // $5
		req.DateNaissanceFin,      // $6
		req.DateNaissance,         // $7
		req.Sexe,                  // $8
		req.CniNni,                // $9
		req.EstAssure,             // $10
		req.EtablissementCreateur, // $11
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// 2. Récupérer les résultats paginés
	patients, err := s.searchWithoutAssurances(ctx, req, queries.PatientSearchQueries.SearchPatientsByCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to execute criteria search: %w", err)
	}

	return &dto.SearchPatientResponse{
		Patients:   patients,
		Pagination: dto.NewPaginationInfo(req.Page, req.Limit, total),
		SearchInfo: dto.NewSearchMetadata(
			dto.SearchTypeCriteria,
			int(time.Since(startTime).Milliseconds()),
			total,
			appliedFilters,
		),
	}, nil
}

// searchWithoutAssurances exécute une recherche sans inclure les assurances
func (s *PatientSearchService) searchWithoutAssurances(
	ctx context.Context,
	req *dto.SearchPatientRequest,
	query string,
) ([]dto.PatientSearchResult, error) {
	var patients []dto.PatientSearchResult

	// Adapter les paramètres selon le type de requête
	var err error
	var rows interface {
		Close()
		Next() bool
		Scan(dest ...interface{}) error
	}

	if query == queries.PatientSearchQueries.SearchPatientsFullText {
		rows, err = s.db.Query(ctx, query,
			req.SearchTerm,            // $1
			req.Statut,                // $2
			req.Nom,                   // $3
			req.Prenoms,               // $4
			req.TelephonePrincipal,    // $5
			req.DateNaissanceDebut,    // $6
			req.DateNaissanceFin,      // $7
			req.Sexe,                  // $8
			req.CniNni,                // $9
			req.EstAssure,             // $10
			req.EtablissementCreateur, // $11
			req.SortBy,                // $12
			req.SortOrder,             // $13
			req.Limit,                 // $14
			req.GetOffset(),           // $15
		)
	} else {
		rows, err = s.db.Query(ctx, query,
			req.Statut,                // $1
			req.Nom,                   // $2
			req.Prenoms,               // $3
			req.TelephonePrincipal,    // $4
			req.DateNaissanceDebut,    // $5
			req.DateNaissanceFin,      // $6
			req.DateNaissance,         // $7
			req.Sexe,                  // $8
			req.CniNni,                // $9
			req.EstAssure,             // $10
			req.EtablissementCreateur, // $11
			req.SortBy,                // $12
			req.SortOrder,             // $13
			req.Limit,                 // $14
			req.GetOffset(),           // $15
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var patient dto.PatientSearchResult
		var score *float64

		err := rows.Scan(
			&patient.ID,
			&patient.CodePatient,
			&patient.Nom,
			&patient.Prenoms,
			&patient.DateNaissance,
			&patient.Sexe,
			&patient.TelephonePrincipal,
			&patient.AdresseComplete,
			&patient.EstAssure,
			&patient.Statut,
			&patient.CreatedAt,
			&score,
		)
		if err != nil {
			continue
		}

		patient.Score = score
		patients = append(patients, patient)
	}

	return patients, nil
}

// searchWithAssurances exécute une recherche en incluant les assurances
func (s *PatientSearchService) searchWithAssurances(
	ctx context.Context,
	req *dto.SearchPatientRequest,
) ([]dto.PatientSearchResult, error) {
	var patients []dto.PatientSearchResult

	rows, err := s.db.Query(ctx,
		queries.PatientSearchQueries.SearchPatientsWithAssurances,
		req.SearchTerm,            // $1
		req.Statut,                // $2
		req.Nom,                   // $3
		req.Prenoms,               // $4
		req.TelephonePrincipal,    // $5
		req.DateNaissanceDebut,    // $6
		req.DateNaissanceFin,      // $7
		req.Sexe,                  // $8
		req.CniNni,                // $9
		req.EstAssure,             // $10
		req.EtablissementCreateur, // $11
		req.SortBy,                // $12
		req.SortOrder,             // $13
		req.Limit,                 // $14
		req.GetOffset(),           // $15
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var patient dto.PatientSearchResult
		var score *float64
		var assurancesJSON string

		err := rows.Scan(
			&patient.ID,
			&patient.CodePatient,
			&patient.Nom,
			&patient.Prenoms,
			&patient.DateNaissance,
			&patient.Sexe,
			&patient.TelephonePrincipal,
			&patient.AdresseComplete,
			&patient.EstAssure,
			&patient.Statut,
			&patient.CreatedAt,
			&score,
			&assurancesJSON,
		)
		if err != nil {
			continue
		}

		patient.Score = score

		// Parser les assurances JSON
		if assurancesJSON != "[]" {
			var assurances []dto.AssuranceResponse
			if err := json.Unmarshal([]byte(assurancesJSON), &assurances); err == nil {
				patient.Assurances = assurances
			}
		}

		patients = append(patients, patient)
	}

	return patients, nil
}

// validateAndNormalizeRequest valide et normalise la requête de recherche
func (s *PatientSearchService) validateAndNormalizeRequest(req *dto.SearchPatientRequest) error {
	// Définir les valeurs par défaut
	req.SetDefaults()

	// Validation des limites
	if req.Limit > 50 {
		return fmt.Errorf("limit cannot exceed 50")
	}

	// Validation des dates
	if req.DateNaissanceDebut != nil && req.DateNaissanceFin != nil {
		if req.DateNaissanceDebut.After(*req.DateNaissanceFin) {
			return fmt.Errorf("date_naissance_debut cannot be after date_naissance_fin")
		}
	}

	// Vérification qu'au moins un critère est fourni
	if req.IsEmpty() {
		return fmt.Errorf("at least one search criteria is required")
	}

	return nil
}

// parsePatientFromCache convertit les données Redis en PatientSearchResult
func (s *PatientSearchService) parsePatientFromCache(data map[string]string) (*dto.PatientSearchResult, error) {
	// Implementation simplifiée - à compléter selon les besoins
	// Cette fonction devrait parser les données Redis et créer un PatientSearchResult
	return nil, fmt.Errorf("cache parsing not implemented yet")
}

// warmPatientInCache met en cache un patient pour optimiser les recherches futures
func (s *PatientSearchService) warmPatientInCache(ctx context.Context, patient *dto.PatientSearchResult) {
	// Cache warming asynchrone selon les conventions Redis
	cacheKey := fmt.Sprintf("soins_suite_patient_cache:%s", patient.CodePatient)
	
	patientMap := map[string]interface{}{
		"id":                 patient.ID.String(),
		"code_patient":       patient.CodePatient,
		"nom":                patient.Nom,
		"prenoms":            patient.Prenoms,
		"date_naissance":     patient.DateNaissance.Format("2006-01-02"),
		"sexe":               patient.Sexe,
		"telephone_principal": patient.TelephonePrincipal,
		"adresse_complete":   patient.AdresseComplete,
		"est_assure":         fmt.Sprintf("%t", patient.EstAssure),
		"statut":             patient.Statut,
		"created_at":         patient.CreatedAt.Format(time.RFC3339),
	}

	// Best effort - ignorer les erreurs
	s.redis.HMSet(ctx, cacheKey, patientMap)
	s.redis.Expire(ctx, cacheKey, time.Hour) // TTL 1 heure selon spécifications
}