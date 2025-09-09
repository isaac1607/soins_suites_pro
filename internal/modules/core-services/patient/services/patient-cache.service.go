package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/patient/dto"
	"soins-suite-core/internal/modules/core-services/patient/queries"
)

// PatientCacheService gère la récupération cache-first des patients avec données complètes
type PatientCacheService struct {
	db        *postgres.Client
	redis     *redis.Client
	redisKeys *PatientRedisKeys
}

// NewPatientCacheService crée une nouvelle instance du service
func NewPatientCacheService(
	db *postgres.Client,
	redis *redis.Client,
) *PatientCacheService {
	return &PatientCacheService{
		db:        db,
		redis:     redis,
		redisKeys: NewPatientRedisKeys(),
	}
}

// GetPatientByCode récupère un patient par son code avec stratégie cache-first
// Performance cible : < 1ms cache hit, < 50ms cache miss avec warming
func (s *PatientCacheService) GetPatientByCode(
	ctx context.Context,
	req *dto.GetPatientByCodeRequest,
) (*dto.PatientDetailResponse, error) {
	startTime := time.Now()

	// Validation de la requête
	if err := s.validateGetPatientRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 1. Stratégie cache-first (sauf si force refresh demandé)
	if !req.ForceRefreshCache {
		if cachedResponse, err := s.getPatientFromCache(ctx, req, startTime); err == nil {
			return cachedResponse, nil
		}
		// Erreur cache ignorée - fallback vers base de données
	}

	// 2. Fallback PostgreSQL + cache warming
	return s.getPatientFromDatabaseWithCaching(ctx, req, startTime)
}

// getPatientFromCache tente de récupérer le patient depuis Redis
func (s *PatientCacheService) getPatientFromCache(
	ctx context.Context,
	req *dto.GetPatientByCodeRequest,
	startTime time.Time,
) (*dto.PatientDetailResponse, error) {
	cacheKey := s.redisKeys.PatientDetailCacheKey(req.CodePatient)

	// Vérifier existence
	exists, err := s.redis.Exists(ctx, cacheKey).Result()
	if err != nil || exists == 0 {
		return nil, fmt.Errorf("patient not in cache")
	}

	// Récupérer toutes les données patient
	patientData, err := s.redis.HGetAll(ctx, cacheKey).Result()
	if err != nil || len(patientData) == 0 {
		return nil, fmt.Errorf("failed to get patient from cache")
	}

	// Parser les données cache
	detailResponse, err := s.parsePatientFromCacheData(patientData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cached patient: %w", err)
	}

	// Enrichir avec métadonnées cache
	detailResponse.LoadedFrom = "cache"
	detailResponse.LoadTime = int(time.Since(startTime).Milliseconds())

	// Audit access pour patients sensibles (asynchrone)
	go s.auditPatientAccess(context.Background(), req.CodePatient, "cache_hit")

	return detailResponse, nil
}

// getPatientFromDatabaseWithCaching récupère depuis PostgreSQL et met en cache
func (s *PatientCacheService) getPatientFromDatabaseWithCaching(
	ctx context.Context,
	req *dto.GetPatientByCodeRequest,
	startTime time.Time,
) (*dto.PatientDetailResponse, error) {
	// Vérification préalable existence + statut
	patientExists, err := s.checkPatientExistsAndStatus(ctx, req.CodePatient, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	if !patientExists {
		return nil, dto.NewPatientNotFoundError(req.CodePatient)
	}

	// Chargement complet depuis PostgreSQL
	detailResponse, err := s.loadPatientFromDatabase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load patient from database: %w", err)
	}

	// Enrichir avec métadonnées database
	detailResponse.LoadedFrom = "database"
	detailResponse.LoadTime = int(time.Since(startTime).Milliseconds())

	// Cache warming asynchrone pour optimiser les prochains accès
	go s.warmPatientInCache(context.Background(), detailResponse)

	// Audit access (asynchrone)
	go s.auditPatientAccess(context.Background(), req.CodePatient, "database_load")

	return detailResponse, nil
}

// checkPatientExistsAndStatus vérifie l'existence et le statut d'un patient
func (s *PatientCacheService) checkPatientExistsAndStatus(
	ctx context.Context,
	codePatient string,
	includeInactive bool,
) (bool, error) {
	var patientID uuid.UUID
	var statut string
	var estDecede bool
	var updatedAt time.Time

	err := s.db.QueryRow(ctx,
		queries.PatientDetailQueries.CheckPatientExists,
		codePatient,
	).Scan(&patientID, &statut, &estDecede, &updatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil // Patient n'existe pas
		}
		return false, fmt.Errorf("failed to check patient existence: %w", err)
	}

	// Vérification statut selon spécifications CS-P-003
	if statut == "archive" && !includeInactive {
		return false, dto.NewPatientArchivedError(codePatient)
	}

	return true, nil
}

// loadPatientFromDatabase charge un patient complet depuis PostgreSQL
func (s *PatientCacheService) loadPatientFromDatabase(
	ctx context.Context,
	req *dto.GetPatientByCodeRequest,
) (*dto.PatientDetailResponse, error) {
	// Utiliser la requête principale pour chargement complet
	row := s.db.QueryRow(ctx,
		queries.PatientDetailQueries.GetPatientByCodeWithAllDetails,
		req.CodePatient,
		req.IncludeInactive,
	)

	// Variables pour scanner tous les champs
	var patient dto.PatientCacheData
	var nationaliteRef, situationRef dto.ReferenceInfo
	var pieceRefID, pieceRefCode, pieceRefNom *string
	var professionRefID, professionRefCode, professionRefNom *string
	var createdByID, createdByNom, createdByPrenoms *string
	var updatedByID, updatedByNom, updatedByPrenoms *string
	var assurancesJSON string

	err := row.Scan(
		// Données patient principales (32 champs)
		&patient.ID,
		&patient.CodePatient,
		&patient.EtablissementCreateur,
		&patient.Nom,
		&patient.Prenoms,
		&patient.DateNaissance,
		&patient.EstDateSupposee,
		&patient.Sexe,
		&patient.NationaliteID,
		&patient.SituationMatrimonialeID,
		&patient.TypePieceIdentiteID,
		&patient.CniNni,
		&patient.NumeroPieceIdentite,
		&patient.LieuNaissance,
		&patient.NomJeuneFille,
		&patient.TelephonePrincipal,
		&patient.TelephoneSecondaire,
		&patient.Email,
		&patient.AdresseComplete,
		&patient.Quartier,
		&patient.Ville,
		&patient.Commune,
		&patient.PaysResidence,
		&patient.ProfessionID,
		&patient.PersonnesAContacter,
		&patient.EstAssure,
		&patient.Statut,
		&patient.EstDecede,
		&patient.DateDeces,
		&patient.CreatedAt,
		&patient.UpdatedAt,
		&patient.CreatedBy,
		&patient.UpdatedBy,

		// Références enrichies - Nationalité (3 champs)
		&nationaliteRef.ID,
		&nationaliteRef.Code,
		&nationaliteRef.Nom,

		// Références enrichies - Situation matrimoniale (3 champs)
		&situationRef.ID,
		&situationRef.Code,
		&situationRef.Nom,

		// Références enrichies - Type pièce identité (3 champs, peuvent être NULL)
		&pieceRefID,
		&pieceRefCode,
		&pieceRefNom,

		// Références enrichies - Profession (3 champs, peuvent être NULL)
		&professionRefID,
		&professionRefCode,
		&professionRefNom,

		// Utilisateurs (6 champs, peuvent être NULL)
		&createdByID,
		&createdByNom,
		&createdByPrenoms,
		&updatedByID,
		&updatedByNom,
		&updatedByPrenoms,

		// Assurances JSON
		&assurancesJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dto.NewPatientNotFoundError(req.CodePatient)
		}
		return nil, fmt.Errorf("failed to scan patient data: %w", err)
	}

	// Construire les références optionnelles
	var pieceRef *dto.ReferenceInfo
	if pieceRefID != nil {
		pieceUUID, _ := uuid.Parse(*pieceRefID)
		pieceRef = &dto.ReferenceInfo{
			ID:   pieceUUID,
			Code: *pieceRefCode,
			Nom:  *pieceRefNom,
		}
	}

	var professionRef *dto.ReferenceInfo
	if professionRefID != nil {
		professionUUID, _ := uuid.Parse(*professionRefID)
		professionRef = &dto.ReferenceInfo{
			ID:   professionUUID,
			Code: *professionRefCode,
			Nom:  *professionRefNom,
		}
	}

	var createdByUser *dto.UserInfo
	if createdByID != nil {
		createdByUUID, _ := uuid.Parse(*createdByID)
		createdByUser = &dto.UserInfo{
			ID:      createdByUUID,
			Nom:     *createdByNom,
			Prenoms: *createdByPrenoms,
		}
	}

	var updatedByUser *dto.UserInfo
	if updatedByID != nil {
		updatedByUUID, _ := uuid.Parse(*updatedByID)
		updatedByUser = &dto.UserInfo{
			ID:      updatedByUUID,
			Nom:     *updatedByNom,
			Prenoms: *updatedByPrenoms,
		}
	}

	// Construire la réponse détaillée
	response := &dto.PatientDetailResponse{
		Patient: dto.PatientResponse{
			ID:                      patient.ID,
			CodePatient:             patient.CodePatient,
			Nom:                     patient.Nom,
			Prenoms:                 patient.Prenoms,
			DateNaissance:           patient.DateNaissance,
			EstDateSupposee:         patient.EstDateSupposee,
			Sexe:                    patient.Sexe,
			TelephonePrincipal:      patient.TelephonePrincipal,
			TelephoneSecondaire:     patient.TelephoneSecondaire,
			Email:                   patient.Email,
			AdresseComplete:         patient.AdresseComplete,
			EstAssure:               patient.EstAssure,
			EtablissementCreateurID: patient.EtablissementCreateur,
			Statut:                  patient.Statut,
			CreatedAt:               patient.CreatedAt,
			CreatedBy:               createdByUser,
		},
		Nationalite:         nationaliteRef,
		SituationMatrimoniale: situationRef,
		TypePieceIdentite:   pieceRef,
		Profession:          professionRef,
		LastUpdated:         patient.UpdatedAt,
	}

	// Stocker updatedByUser pour usage futur ou métadonnées
	_ = updatedByUser // Variable utilisée pour éviter l'erreur de compilation

	// Parser les personnes à contacter si demandé
	if req.IncludePersonnesContact {
		personnes, err := s.parsePersonnesAContacter(patient.PersonnesAContacter)
		if err == nil {
			response.PersonnesAContacter = personnes
		}
	}

	// Parser les assurances si demandé
	if req.IncludeAssurances && assurancesJSON != "[]" {
		var assurances []dto.AssuranceDetail
		if err := json.Unmarshal([]byte(assurancesJSON), &assurances); err == nil {
			response.AssurancesDetails = assurances
		}
	}

	return response, nil
}

// warmPatientInCache met en cache un patient pour optimiser les prochains accès
func (s *PatientCacheService) warmPatientInCache(ctx context.Context, patient *dto.PatientDetailResponse) {
	cacheKey := s.redisKeys.PatientDetailCacheKey(patient.Patient.CodePatient)

	// Convertir en format cache
	cacheData := s.convertToPatientCacheData(patient)

	// Sérialiser en map Redis
	patientMap, err := s.serializePatientForCache(cacheData)
	if err != nil {
		return // Best effort - ignorer les erreurs
	}

	// Mettre en cache avec TTL 1 heure selon spécifications
	s.redis.HMSet(ctx, cacheKey, patientMap)
	s.redis.Expire(ctx, cacheKey, time.Hour)

	fmt.Printf("[CACHE] Patient warmed - Code: %s, TTL: 1h\n", patient.Patient.CodePatient)
}

// InvalidatePatientCache invalide le cache d'un patient (appelé lors des modifications)
func (s *PatientCacheService) InvalidatePatientCache(ctx context.Context, codePatient string) error {
	cacheKey := s.redisKeys.PatientDetailCacheKey(codePatient)
	
	_, err := s.redis.Del(ctx, cacheKey).Result()
	if err != nil {
		return fmt.Errorf("failed to invalidate patient cache: %w", err)
	}

	fmt.Printf("[CACHE] Patient cache invalidated - Code: %s\n", codePatient)
	return nil
}

// Méthodes utilitaires privées

func (s *PatientCacheService) validateGetPatientRequest(req *dto.GetPatientByCodeRequest) error {
	if req.CodePatient == "" {
		return fmt.Errorf("code_patient is required")
	}
	return nil
}

func (s *PatientCacheService) parsePatientFromCacheData(data map[string]string) (*dto.PatientDetailResponse, error) {
	// Implementation simplifiée - à compléter selon les besoins
	// Cette fonction devrait parser les données Redis et créer un PatientDetailResponse complet
	return nil, fmt.Errorf("cache parsing not fully implemented yet")
}

func (s *PatientCacheService) parsePersonnesAContacter(jsonData string) ([]dto.PersonneContactDetail, error) {
	if jsonData == "" || jsonData == "[]" {
		return []dto.PersonneContactDetail{}, nil
	}

	var personnes []dto.PersonneContact
	if err := json.Unmarshal([]byte(jsonData), &personnes); err != nil {
		return nil, err
	}

	// Convertir en PersonneContactDetail avec références enrichies
	var result []dto.PersonneContactDetail
	for _, p := range personnes {
		// TODO: Enrichir avec les données d'affiliation depuis la base
		detail := dto.PersonneContactDetail{
			NomPrenoms:           p.NomPrenoms,
			Telephone:           p.Telephone,
			TelephoneSecondaire: p.TelephoneSecondaire,
			Affiliation: dto.ReferenceInfo{
				ID:   p.AffiliationID,
				Code: "TODO", // À récupérer depuis ref_affiliation
				Nom:  "TODO", // À récupérer depuis ref_affiliation
			},
		}
		result = append(result, detail)
	}

	return result, nil
}

func (s *PatientCacheService) convertToPatientCacheData(patient *dto.PatientDetailResponse) *dto.PatientCacheData {
	// Convertir PatientDetailResponse en PatientCacheData pour le cache
	return &dto.PatientCacheData{
		ID:                      patient.Patient.ID,
		CodePatient:             patient.Patient.CodePatient,
		EtablissementCreateur:   patient.Patient.EtablissementCreateurID,
		Nom:                     patient.Patient.Nom,
		Prenoms:                 patient.Patient.Prenoms,
		DateNaissance:           patient.Patient.DateNaissance,
		EstDateSupposee:         patient.Patient.EstDateSupposee,
		Sexe:                    patient.Patient.Sexe,
		NationaliteID:           patient.Nationalite.ID,
		SituationMatrimonialeID: patient.SituationMatrimoniale.ID,
		TelephonePrincipal:      patient.Patient.TelephonePrincipal,
		TelephoneSecondaire:     patient.Patient.TelephoneSecondaire,
		Email:                   patient.Patient.Email,
		AdresseComplete:         patient.Patient.AdresseComplete,
		EstAssure:               patient.Patient.EstAssure,
		Statut:                  patient.Patient.Statut,
		CreatedAt:               patient.Patient.CreatedAt,
		UpdatedAt:               patient.LastUpdated,
	}
}

func (s *PatientCacheService) serializePatientForCache(cacheData *dto.PatientCacheData) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":                        cacheData.ID.String(),
		"code_patient":              cacheData.CodePatient,
		"etablissement_createur_id": cacheData.EtablissementCreateur.String(),
		"nom":                       cacheData.Nom,
		"prenoms":                   cacheData.Prenoms,
		"date_naissance":            cacheData.DateNaissance.Format("2006-01-02"),
		"est_date_supposee":         fmt.Sprintf("%t", cacheData.EstDateSupposee),
		"sexe":                      cacheData.Sexe,
		"nationalite_id":            cacheData.NationaliteID.String(),
		"situation_matrimoniale_id": cacheData.SituationMatrimonialeID.String(),
		"telephone_principal":       cacheData.TelephonePrincipal,
		"adresse_complete":          cacheData.AdresseComplete,
		"est_assure":                fmt.Sprintf("%t", cacheData.EstAssure),
		"statut":                    cacheData.Statut,
		"created_at":                cacheData.CreatedAt.Format(time.RFC3339),
		"updated_at":                cacheData.UpdatedAt.Format(time.RFC3339),
		"cached_at":                 time.Now().Format(time.RFC3339),
	}, nil
}

func (s *PatientCacheService) auditPatientAccess(ctx context.Context, codePatient, accessType string) {
	// Log pour audit access selon spécifications CS-P-003
	fmt.Printf("[AUDIT] Patient accessed - Code: %s, Type: %s, Time: %s\n",
		codePatient, accessType, time.Now().Format(time.RFC3339))
}