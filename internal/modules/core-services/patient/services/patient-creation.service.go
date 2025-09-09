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

// PatientCreationService orchestre la création complète d'un patient
type PatientCreationService struct {
	db                 *postgres.Client
	txManager          *postgres.TransactionManager
	redis              *redis.Client
	redisKeys          *PatientRedisKeys
	codeGenerator      *PatientCodeGeneratorService
	validator          *PatientValidationService
}

// NewPatientCreationService crée une nouvelle instance du service
func NewPatientCreationService(
	db *postgres.Client,
	redis *redis.Client,
	codeGenerator *PatientCodeGeneratorService,
	validator *PatientValidationService,
) *PatientCreationService {
	return &PatientCreationService{
		db:            db,
		txManager:     postgres.NewTransactionManager(db),
		redis:         redis,
		redisKeys:     NewPatientRedisKeys(),
		codeGenerator: codeGenerator,
		validator:     validator,
	}
}

// CreatePatient crée un nouveau patient avec toutes les validations et vérifications
func (s *PatientCreationService) CreatePatient(
	ctx context.Context,
	etablissementCode string,
	etablissementID uuid.UUID,
	req *dto.CreatePatientRequest,
	userID uuid.UUID,
) (*dto.PatientCreationResult, error) {
	startTime := time.Now()
	var stepsExecuted []string

	// 1. Validation des données d'entrée
	stepsExecuted = append(stepsExecuted, "data_validation")
	validationResult, err := s.validator.ValidatePatientData(ctx, req, etablissementID)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if !validationResult.IsValid {
		return nil, fmt.Errorf("invalid patient data: %d errors", len(validationResult.Errors))
	}

	// 2. Vérification anti-doublon
	stepsExecuted = append(stepsExecuted, "duplicate_check")
	duplicateCheckReq := &dto.DuplicateCheckRequest{
		Nom:               req.Nom,
		Prenoms:           req.Prenoms,
		DateNaissance:     req.DateNaissance,
		TelephonePrincipal: &req.TelephonePrincipal,
		ScoreMinimum:      70,
		LimiteResultats:   5,
	}

	duplicateResult, err := s.validator.CheckPatientDuplicate(ctx, duplicateCheckReq)
	if err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	}

	// Bloquer si duplicata détecté avec score élevé
	if duplicateResult.ShouldBlock() {
		return nil, fmt.Errorf("patient duplicate detected with score %d - creation blocked", duplicateResult.HighestScore)
	}

	// 3. Génération du code patient unique
	stepsExecuted = append(stepsExecuted, "code_generation")
	codeGeneration, err := s.codeGenerator.GeneratePatientCode(ctx, etablissementCode)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// 4. Création du patient en transaction atomique
	stepsExecuted = append(stepsExecuted, "patient_creation")
	patientResponse, err := s.createPatientTransaction(ctx, req, etablissementID, codeGeneration.CodePatient, userID)
	if err != nil {
		return nil, fmt.Errorf("patient creation failed: %w", err)
	}

	// 5. Cache warming Redis (asynchrone pour performance)
	stepsExecuted = append(stepsExecuted, "cache_warming")
	go s.warmPatientCache(context.Background(), patientResponse)

	// Construire le résultat complet
	result := &dto.PatientCreationResult{
		Patient:              patientResponse,
		CodeGeneration:       codeGeneration,
		DuplicateCheckResult: duplicateResult,
		CreationTimeMs:       int(time.Since(startTime).Milliseconds()),
		StepsExecuted:        stepsExecuted,
	}

	// Log pour audit
	fmt.Printf("[AUDIT] Patient created - ID: %s, Code: %s, Etablissement: %s, Duration: %dms\n",
		patientResponse.ID, patientResponse.CodePatient, etablissementCode, result.CreationTimeMs)

	return result, nil
}

// createPatientTransaction effectue la création complète en transaction PostgreSQL
func (s *PatientCreationService) createPatientTransaction(
	ctx context.Context,
	req *dto.CreatePatientRequest,
	etablissementID uuid.UUID,
	codePatient string,
	userID uuid.UUID,
) (*dto.PatientResponse, error) {
	var patientResponse *dto.PatientResponse

	// Transaction avec isolation Serializable pour cohérence maximale
	err := s.txManager.WithTransactionIsolation(ctx, pgx.Serializable, func(tx *postgres.Transaction) error {
		// Générer l'ID du patient
		patientID := uuid.New()

		// Convertir les personnes à contacter en JSON
		personnesJSON, err := json.Marshal(req.PersonnesAContacter)
		if err != nil {
			return fmt.Errorf("failed to marshal personnes_a_contacter: %w", err)
		}

		// 1. Insertion du patient principal
		var createdPatient struct {
			ID                  uuid.UUID
			CodePatient         string
			Nom                 string
			Prenoms             string
			DateNaissance       time.Time
			Sexe                string
			TelephonePrincipal  string
			AdresseComplete     string
			EstAssure           bool
			Statut              string
			CreatedAt           time.Time
		}

		err = tx.QueryRow(ctx,
			queries.PatientCreationQueries.CreatePatientWithValidation,
			patientID,                           // $1
			codePatient,                         // $2
			etablissementID,                     // $3
			req.Nom,                            // $4
			req.Prenoms,                        // $5
			req.DateNaissance,                  // $6
			req.EstDateSupposee,                // $7
			req.Sexe,                           // $8
			req.NationaliteID,                  // $9
			req.SituationMatrimonialeID,        // $10
			req.TypePieceIdentiteID,            // $11
			req.CniNni,                         // $12
			req.NumeroPieceIdentite,            // $13
			req.LieuNaissance,                  // $14
			req.NomJeuneFille,                  // $15
			req.TelephonePrincipal,             // $16
			req.TelephoneSecondaire,            // $17
			req.Email,                          // $18
			req.AdresseComplete,                // $19
			req.Quartier,                       // $20
			req.Ville,                          // $21
			req.Commune,                        // $22
			req.PaysResidence,                  // $23
			req.ProfessionID,                   // $24
			string(personnesJSON),              // $25
			req.EstAssure,                      // $26
			"actif",                            // $27
			userID,                             // $28
		).Scan(
			&createdPatient.ID,
			&createdPatient.CodePatient,
			&createdPatient.Nom,
			&createdPatient.Prenoms,
			&createdPatient.DateNaissance,
			&createdPatient.Sexe,
			&createdPatient.TelephonePrincipal,
			&createdPatient.AdresseComplete,
			&createdPatient.EstAssure,
			&createdPatient.Statut,
			&createdPatient.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert patient: %w", err)
		}

		// 2. Insertion des assurances si le patient est assuré
		var assurances []dto.AssuranceResponse
		if req.EstAssure && len(req.Assurances) > 0 {
			assurances, err = s.insertPatientAssurances(ctx, tx, patientID, req.Assurances, userID)
			if err != nil {
				return fmt.Errorf("failed to insert patient assurances: %w", err)
			}
		}

		// 3. Construire la réponse
		patientResponse = &dto.PatientResponse{
			ID:                      createdPatient.ID,
			CodePatient:             createdPatient.CodePatient,
			Nom:                     createdPatient.Nom,
			Prenoms:                 createdPatient.Prenoms,
			DateNaissance:           createdPatient.DateNaissance,
			EstDateSupposee:         req.EstDateSupposee,
			Sexe:                    createdPatient.Sexe,
			TelephonePrincipal:      createdPatient.TelephonePrincipal,
			TelephoneSecondaire:     req.TelephoneSecondaire,
			Email:                   req.Email,
			AdresseComplete:         createdPatient.AdresseComplete,
			EstAssure:               createdPatient.EstAssure,
			Assurances:             assurances,
			EtablissementCreateurID: etablissementID,
			Statut:                  createdPatient.Statut,
			CreatedAt:               createdPatient.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return patientResponse, nil
}

// insertPatientAssurances insère les assurances d'un patient
func (s *PatientCreationService) insertPatientAssurances(
	ctx context.Context,
	tx *postgres.Transaction,
	patientID uuid.UUID,
	assurances []dto.CreateAssuranceData,
	userID uuid.UUID,
) ([]dto.AssuranceResponse, error) {
	// Préparer les arrays pour la requête bulk
	var assuranceIDs []uuid.UUID
	var numerosAssure []string
	var typesBeneficiaire []string
	var numerosAssurePrincipal []*string
	var liensAvecPrincipal []*string

	for _, assurance := range assurances {
		assuranceIDs = append(assuranceIDs, assurance.AssuranceID)
		numerosAssure = append(numerosAssure, assurance.NumeroAssure)
		typesBeneficiaire = append(typesBeneficiaire, assurance.TypeBeneficiaire)
		numerosAssurePrincipal = append(numerosAssurePrincipal, assurance.NumeroAssurePrincipal)
		liensAvecPrincipal = append(liensAvecPrincipal, assurance.LienAvecPrincipal)
	}

	// Exécuter l'insertion bulk
	rows, err := tx.Query(ctx,
		queries.PatientCreationQueries.InsertPatientAssurances,
		patientID,
		assuranceIDs,
		numerosAssure,
		typesBeneficiaire,
		numerosAssurePrincipal,
		liensAvecPrincipal,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert patient assurances: %w", err)
	}
	defer rows.Close()

	// Parser les résultats
	var result []dto.AssuranceResponse
	for rows.Next() {
		var assurance dto.AssuranceResponse
		var assuranceID uuid.UUID

		err := rows.Scan(
			&assurance.ID,
			&assuranceID,
			&assurance.NumeroAssure,
			&assurance.TypeBeneficiaire,
			&assurance.EstActif,
		)
		if err != nil {
			continue
		}

		// TODO: Enrichir avec le nom de l'assurance via une requête
		// assurance.AssuranceNom = "À récupérer"
		result = append(result, assurance)
	}

	return result, nil
}

// warmPatientCache met en cache Redis les données du patient créé
func (s *PatientCreationService) warmPatientCache(ctx context.Context, patient *dto.PatientResponse) {
	// Clé cache patient selon les conventions Redis
	cacheKey := fmt.Sprintf("soins_suite_patient_cache:%s", patient.CodePatient)

	// Convertir en map pour Redis HASH
	patientMap := map[string]interface{}{
		"id":                        patient.ID.String(),
		"code_patient":              patient.CodePatient,
		"nom":                       patient.Nom,
		"prenoms":                   patient.Prenoms,
		"date_naissance":            patient.DateNaissance.Format("2006-01-02"),
		"est_date_supposee":         fmt.Sprintf("%t", patient.EstDateSupposee),
		"sexe":                      patient.Sexe,
		"telephone_principal":       patient.TelephonePrincipal,
		"adresse_complete":          patient.AdresseComplete,
		"etablissement_createur_id": patient.EtablissementCreateurID.String(),
		"est_assure":                fmt.Sprintf("%t", patient.EstAssure),
		"statut":                    patient.Statut,
		"created_at":                patient.CreatedAt.Format(time.RFC3339),
		"updated_at":                patient.CreatedAt.Format(time.RFC3339),
	}

	// Ajouter téléphone secondaire et email si présents
	if patient.TelephoneSecondaire != nil {
		patientMap["telephone_secondaire"] = *patient.TelephoneSecondaire
	}
	if patient.Email != nil {
		patientMap["email"] = *patient.Email
	}

	// Ajouter assurances en JSON si présentes
	if len(patient.Assurances) > 0 {
		assurancesJSON, err := json.Marshal(patient.Assurances)
		if err == nil {
			patientMap["assurances"] = string(assurancesJSON)
		}
	}

	// Best effort cache - ignorer les erreurs
	s.redis.HMSet(ctx, cacheKey, patientMap)
	s.redis.Expire(ctx, cacheKey, time.Hour) // TTL 1 heure selon les spécifications

	fmt.Printf("[CACHE] Patient warmed in Redis - Code: %s, Key: %s\n", patient.CodePatient, cacheKey)
}

// GetPatientByCode récupère un patient par son code avec stratégie cache-first
func (s *PatientCreationService) GetPatientByCode(
	ctx context.Context,
	codePatient string,
) (*dto.PatientResponse, error) {
	// TODO: Implémentation de la récupération cache-first (CS-P-003)
	// Pour l'instant, on utilise directement PostgreSQL
	return s.getPatientFromDatabase(ctx, codePatient)
}

// getPatientFromDatabase récupère un patient directement depuis PostgreSQL
func (s *PatientCreationService) getPatientFromDatabase(
	ctx context.Context,
	codePatient string,
) (*dto.PatientResponse, error) {
	// Cette méthode sera complétée quand on implémentera CS-P-003
	return nil, fmt.Errorf("get patient from database not implemented yet - CS-P-003")
}