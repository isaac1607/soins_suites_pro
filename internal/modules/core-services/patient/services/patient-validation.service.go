package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/patient/dto"
	"soins-suite-core/internal/modules/core-services/patient/queries"
)

// PatientValidationService gère les validations métier et détection de doublons
type PatientValidationService struct {
	db *postgres.Client

	// Regex pour validations Côte d'Ivoire
	phoneRegex *regexp.Regexp
	emailRegex *regexp.Regexp
}

// NewPatientValidationService crée une nouvelle instance du service
func NewPatientValidationService(db *postgres.Client) *PatientValidationService {
	return &PatientValidationService{
		db:         db,
		phoneRegex: regexp.MustCompile(`^(\+225|00225)?[0-9]{10}$`),
		emailRegex: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
	}
}

// CheckPatientDuplicate vérifie l'existence de doublons potentiels avec scoring intelligent
func (s *PatientValidationService) CheckPatientDuplicate(
	ctx context.Context,
	req *dto.DuplicateCheckRequest,
) (*dto.DuplicateCheckResponse, error) {
	// startTime := time.Now()

	// Validation des paramètres
	if err := s.validateDuplicateCheckRequest(req); err != nil {
		return nil, err
	}

	// Exécution de la requête de détection avec scoring
	rows, err := s.db.Query(ctx,
		queries.PatientCreationQueries.CheckDuplicateWithScoring,
		req.Nom,
		req.Prenoms,
		req.DateNaissance,
		req.TelephonePrincipal,
		req.ScoreMinimum,
		req.LimiteResultats,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicates: %w", err)
	}
	defer rows.Close()

	var potentialMatches []dto.PotentialDuplicate
	var highestScore int

	// Parser les résultats
	for rows.Next() {
		var patient dto.PatientSearchResult
		var scoreGlobal, nomMatch, prenomsMatch, dateMatch, telephoneMatch int

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
			&scoreGlobal,
			&nomMatch,
			&prenomsMatch,
			&dateMatch,
			&telephoneMatch,
		)
		if err != nil {
			continue
		}

		// Mettre à jour le score le plus élevé
		if scoreGlobal > highestScore {
			highestScore = scoreGlobal
		}

		// Créer le détail du match
		matchDetail := dto.MatchDetail{
			NomMatch:           nomMatch,
			PrenomsMatch:       prenomsMatch,
			DateNaissanceMatch: dateMatch,
			TelephoneMatch:     telephoneMatch,
			ScoreGlobal:        scoreGlobal,
		}

		potentialMatch := dto.PotentialDuplicate{
			Patient:      patient,
			Score:        scoreGlobal,
			MatchDetails: matchDetail,
		}

		potentialMatches = append(potentialMatches, potentialMatch)
	}

	// Construire la réponse
	response := &dto.DuplicateCheckResponse{
		HasDuplicates:    len(potentialMatches) > 0,
		HighestScore:     highestScore,
		Recommendation:   s.getRecommendationFromScore(highestScore),
		PotentialMatches: potentialMatches,
		CheckExecutedAt:  time.Now(),
	}

	// Log pour debug si duplicatas détectés
	if response.HasDuplicates {
		fmt.Printf("[DEBUG] Duplicate check - Highest score: %d, Matches: %d, Recommendation: %s\n",
			highestScore, len(potentialMatches), response.Recommendation)
	}

	return response, nil
}

// ValidatePatientData effectue les validations métier complètes
func (s *PatientValidationService) ValidatePatientData(
	ctx context.Context,
	req *dto.CreatePatientRequest,
	etablissementID uuid.UUID,
) (*dto.ValidationResult, error) {
	var errors []dto.ValidationError

	// 1. Validations format
	errors = append(errors, s.validatePatientBasicData(req)...)

	// 2. Validations références existantes
	refErrors, err := s.validateReferenceData(ctx, req, etablissementID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate references: %w", err)
	}
	errors = append(errors, refErrors...)

	// 3. Validations cohérence métier
	errors = append(errors, s.validateBusinessRules(req)...)

	// Retourner le résultat
	return &dto.ValidationResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}, nil
}

// validatePatientBasicData effectue les validations de format de base
func (s *PatientValidationService) validatePatientBasicData(req *dto.CreatePatientRequest) []dto.ValidationError {
	var errors []dto.ValidationError

	// Validation téléphone principal
	if !s.phoneRegex.MatchString(req.TelephonePrincipal) {
		errors = append(errors, dto.NewValidationError(
			"telephone_principal",
			dto.ValidationErrorInvalidPhone,
			"Format téléphone invalide pour la Côte d'Ivoire",
			req.TelephonePrincipal,
		))
	}

	// Validation email si fourni
	if req.Email != nil && *req.Email != "" && !s.emailRegex.MatchString(*req.Email) {
		errors = append(errors, dto.NewValidationError(
			"email",
			dto.ValidationErrorInvalidEmail,
			"Format email invalide",
			*req.Email,
		))
	}

	// Validation date de naissance
	if req.DateNaissance.After(time.Now()) {
		errors = append(errors, dto.NewValidationError(
			"date_naissance",
			dto.ValidationErrorInvalidDate,
			"La date de naissance ne peut pas être dans le futur",
			req.DateNaissance,
		))
	}

	// Validation âge réaliste (> 150 ans)
	if req.DateNaissance.Before(time.Now().AddDate(-150, 0, 0)) {
		errors = append(errors, dto.NewValidationError(
			"date_naissance",
			dto.ValidationErrorInvalidDate,
			"Âge non réaliste (plus de 150 ans)",
			req.DateNaissance,
		))
	}

	// Validation nom et prénoms (pas uniquement des espaces)
	if strings.TrimSpace(req.Nom) == "" {
		errors = append(errors, dto.NewValidationError(
			"nom",
			dto.ValidationErrorRequiredField,
			"Le nom est requis et ne peut être vide",
			req.Nom,
		))
	}

	if strings.TrimSpace(req.Prenoms) == "" {
		errors = append(errors, dto.NewValidationError(
			"prenoms",
			dto.ValidationErrorRequiredField,
			"Les prénoms sont requis et ne peuvent être vides",
			req.Prenoms,
		))
	}

	return errors
}

// validateReferenceData valide que les références existent en base
func (s *PatientValidationService) validateReferenceData(
	ctx context.Context,
	req *dto.CreatePatientRequest,
	etablissementID uuid.UUID,
) ([]dto.ValidationError, error) {
	var errors []dto.ValidationError

	// Vérifier les références via une seule requête
	var nationaliteExists, situationExists, pieceExists, professionExists, etablissementExists bool

	err := s.db.QueryRow(ctx,
		queries.PatientCreationQueries.ValidateReferenceDataExists,
		req.NationaliteID,
		req.SituationMatrimonialeID,
		req.TypePieceIdentiteID,
		req.ProfessionID,
		etablissementID,
	).Scan(&nationaliteExists, &situationExists, &pieceExists, &professionExists, &etablissementExists)

	if err != nil {
		return nil, fmt.Errorf("failed to validate reference data: %w", err)
	}

	// Vérifier chaque référence
	if !nationaliteExists {
		errors = append(errors, dto.NewValidationError(
			"nationalite_id",
			dto.ValidationErrorReferenceNotFound,
			"Nationalité introuvable ou inactive",
			req.NationaliteID,
		))
	}

	if !situationExists {
		errors = append(errors, dto.NewValidationError(
			"situation_matrimoniale_id",
			dto.ValidationErrorReferenceNotFound,
			"Situation matrimoniale introuvable ou inactive",
			req.SituationMatrimonialeID,
		))
	}

	if !pieceExists {
		errors = append(errors, dto.NewValidationError(
			"type_piece_identite_id",
			dto.ValidationErrorReferenceNotFound,
			"Type de pièce d'identité introuvable ou inactif",
			req.TypePieceIdentiteID,
		))
	}

	if !professionExists {
		errors = append(errors, dto.NewValidationError(
			"profession_id",
			dto.ValidationErrorReferenceNotFound,
			"Profession introuvable ou inactive",
			req.ProfessionID,
		))
	}

	if !etablissementExists {
		errors = append(errors, dto.NewValidationError(
			"etablissement_id",
			dto.ValidationErrorReferenceNotFound,
			"Établissement introuvable ou inactif",
			etablissementID,
		))
	}

	return errors, nil
}

// validateBusinessRules effectue les validations de cohérence métier
func (s *PatientValidationService) validateBusinessRules(req *dto.CreatePatientRequest) []dto.ValidationError {
	var errors []dto.ValidationError

	// Validation assurance : si est_assure = true, au moins une assurance requise
	if req.EstAssure && len(req.Assurances) == 0 {
		errors = append(errors, dto.NewValidationError(
			"assurances",
			dto.ValidationErrorRequiredField,
			"Au moins une assurance est requise si le patient est assuré",
			req.EstAssure,
		))
	}

	// Validation assurance : cohérence ayant-droit
	for i, assurance := range req.Assurances {
		if assurance.TypeBeneficiaire == "ayant_droit" {
			if assurance.NumeroAssurePrincipal == nil || *assurance.NumeroAssurePrincipal == "" {
				errors = append(errors, dto.NewValidationError(
					fmt.Sprintf("assurances[%d].numero_assure_principal", i),
					dto.ValidationErrorRequiredField,
					"Numéro d'assuré principal requis pour un ayant-droit",
					assurance.NumeroAssurePrincipal,
				))
			}
			if assurance.LienAvecPrincipal == nil || *assurance.LienAvecPrincipal == "" {
				errors = append(errors, dto.NewValidationError(
					fmt.Sprintf("assurances[%d].lien_avec_principal", i),
					dto.ValidationErrorRequiredField,
					"Lien avec l'assuré principal requis pour un ayant-droit",
					assurance.LienAvecPrincipal,
				))
			}
		}
	}

	// Validation personnes à contacter : maximum 5
	if len(req.PersonnesAContacter) > 5 {
		errors = append(errors, dto.NewValidationError(
			"personnes_a_contacter",
			dto.ValidationErrorInvalidFormat,
			"Maximum 5 personnes à contacter autorisées",
			len(req.PersonnesAContacter),
		))
	}

	return errors
}

// validateDuplicateCheckRequest valide la requête de vérification de doublon
func (s *PatientValidationService) validateDuplicateCheckRequest(req *dto.DuplicateCheckRequest) error {
	if strings.TrimSpace(req.Nom) == "" {
		return fmt.Errorf("nom requis pour la vérification de doublon")
	}
	if strings.TrimSpace(req.Prenoms) == "" {
		return fmt.Errorf("prénoms requis pour la vérification de doublon")
	}
	if req.DateNaissance.IsZero() {
		return fmt.Errorf("date de naissance requise pour la vérification de doublon")
	}
	if req.ScoreMinimum < 0 || req.ScoreMinimum > 100 {
		return fmt.Errorf("score minimum doit être entre 0 et 100")
	}
	if req.LimiteResultats < 1 || req.LimiteResultats > 50 {
		return fmt.Errorf("limite résultats doit être entre 1 et 50")
	}
	return nil
}

// getRecommendationFromScore détermine la recommandation basée sur le score
func (s *PatientValidationService) getRecommendationFromScore(score int) string {
	if score >= 85 {
		return dto.RecommendationBlock
	}
	if score >= 70 {
		return dto.RecommendationWarn
	}
	return dto.RecommendationAllow
}

// UpdatePatient effectue la mise à jour partielle d'un patient avec validations
func (s *PatientValidationService) UpdatePatient(
	ctx context.Context,
	codePatient string,
	req *dto.UpdatePatientRequest,
) (*dto.PatientResponse, error) {
	// TODO: Implémentation de la mise à jour partielle
	// Pour l'instant, on retourne une erreur indiquant que cette fonctionnalité n'est pas encore implémentée
	return nil, fmt.Errorf("update patient not implemented yet - CS-P-004")
}
