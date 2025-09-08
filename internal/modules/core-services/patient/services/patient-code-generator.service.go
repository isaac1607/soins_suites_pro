package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/patient/dto"
	"soins-suite-core/internal/modules/core-services/patient/queries"
)

// PatientCodeGeneratorService gère la génération atomique des codes patient uniques
type PatientCodeGeneratorService struct {
	db       *postgres.Client
	txManager *postgres.TransactionManager
	redis    *redis.Client
	redisKeys *PatientRedisKeys
	mu       sync.Map // Lock en mémoire par établissement pour éviter concurrence locale
}

// NewPatientCodeGeneratorService crée une nouvelle instance du service
func NewPatientCodeGeneratorService(db *postgres.Client, redis *redis.Client) *PatientCodeGeneratorService {
	return &PatientCodeGeneratorService{
		db:       db,
		txManager: postgres.NewTransactionManager(db),
		redis:    redis,
		redisKeys: NewPatientRedisKeys(),
		mu:       sync.Map{},
	}
}

// GeneratePatientCode génère un code patient unique atomiquement
// Format: {ETABLISSEMENT}-{YYYY}-{NNN}-{LLL}
// Exemple: CENTREA-2025-001-AAA
func (s *PatientCodeGeneratorService) GeneratePatientCode(
	ctx context.Context,
	etablissementCode string,
) (*dto.CodeGenerationResponse, error) {
	startTime := time.Now()
	year := time.Now().Year()

	// Validation établissement
	if err := s.validateEtablissementCode(etablissementCode); err != nil {
		return nil, err
	}

	// 1. Tentative rapide via Redis (99% des cas)
	if response, err := s.generateFromRedis(ctx, etablissementCode, year); err == nil {
		response.GenerationTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// 2. Fallback PostgreSQL si Redis indisponible ou première génération
	return s.generateFromPostgres(ctx, etablissementCode, year, startTime)
}

// generateFromRedis - Génération ultra-rapide via Redis avec locks distribués
func (s *PatientCodeGeneratorService) generateFromRedis(
	ctx context.Context,
	etablissementCode string,
	year int,
) (*dto.CodeGenerationResponse, error) {
	redisKey := s.redisKeys.PatientSequenceKey(etablissementCode, year)
	lockKey := s.redisKeys.PatientSequenceLockKey(etablissementCode, year)

	// Acquérir un lock Redis (protection concurrence)
	locked, err := s.redis.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil || !locked {
		return nil, fmt.Errorf("unable to acquire Redis lock: %w", err)
	}
	defer s.redis.Del(ctx, lockKey)

	// Récupérer la séquence actuelle
	current, err := s.redis.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// Première génération de l'année - initialiser depuis PostgreSQL
		return s.initializeRedisFromDB(ctx, etablissementCode, year)
	}
	if err != nil {
		return nil, fmt.Errorf("Redis get failed: %w", err)
	}

	// Parser et incrémenter
	var numero int
	var suffixe string
	_, scanErr := fmt.Sscanf(current, "%d:%s", &numero, &suffixe)
	if scanErr != nil {
		return nil, fmt.Errorf("invalid Redis sequence format: %w", scanErr)
	}

	numero, suffixe, err = s.incrementSequence(numero, suffixe)
	if err != nil {
		return nil, err
	}

	// Sauvegarder la nouvelle séquence
	newValue := fmt.Sprintf("%d:%s", numero, suffixe)
	ttl := s.calculateTTLUntilYearEnd()
	if err := s.redis.Set(ctx, redisKey, newValue, ttl).Err(); err != nil {
		return nil, fmt.Errorf("Redis set failed: %w", err)
	}

	// Mise à jour asynchrone PostgreSQL (fire-and-forget pour performance)
	go s.updatePostgresAsync(etablissementCode, year, numero, suffixe)

	// Formater le code final
	codePatient := fmt.Sprintf("%s-%d-%03d-%s", etablissementCode, year, numero, suffixe)

	return &dto.CodeGenerationResponse{
		CodePatient:       codePatient,
		EtablissementCode: etablissementCode,
		Annee:            year,
		Numero:           numero,
		Suffixe:          suffixe,
		GeneratedAt:      time.Now(),
		Source:           "redis",
	}, nil
}

// generateFromPostgres - Génération avec PostgreSQL (fallback robuste)
func (s *PatientCodeGeneratorService) generateFromPostgres(
	ctx context.Context,
	etablissementCode string,
	year int,
	startTime time.Time,
) (*dto.CodeGenerationResponse, error) {
	// Utiliser un lock en mémoire pour éviter la concurrence locale
	lockKey := fmt.Sprintf("%s-%d", etablissementCode, year)
	lockValue, _ := s.mu.LoadOrStore(lockKey, &sync.Mutex{})
	mutex := lockValue.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	var numero int
	var suffixe string
	var nombreGeneres int64

	// Transaction PostgreSQL avec TransactionManager
	err := s.txManager.WithTransactionIsolation(ctx, pgx.Serializable, func(tx *postgres.Transaction) error {
		// Génération atomique avec UPSERT
		return tx.QueryRow(ctx,
			queries.PatientCodeGenerationQueries.GenerateNextCodeFromPostgres,
			etablissementCode,
			year,
		).Scan(&numero, &suffixe, &nombreGeneres)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// Synchroniser avec Redis pour futures générations
	s.syncRedisFromPostgres(ctx, etablissementCode, year, numero, suffixe)

	// Formater le code final
	codePatient := fmt.Sprintf("%s-%d-%03d-%s", etablissementCode, year, numero, suffixe)

	return &dto.CodeGenerationResponse{
		CodePatient:       codePatient,
		EtablissementCode: etablissementCode,
		Annee:            year,
		Numero:           numero,
		Suffixe:          suffixe,
		NombreGeneres:    nombreGeneres,
		GeneratedAt:      time.Now(),
		Source:           "postgres",
		GenerationTimeMs: int(time.Since(startTime).Milliseconds()),
	}, nil
}

// incrementSequence - Logique d'incrémentation numéro/suffixe
func (s *PatientCodeGeneratorService) incrementSequence(numero int, suffixe string) (int, string, error) {
	numero++
	if numero > 999 {
		numero = 1
		newSuffixe, err := s.nextSuffix(suffixe)
		if err != nil {
			return 0, "", err
		}
		suffixe = newSuffixe
	}
	return numero, suffixe, nil
}

// nextSuffix - Calcul du suffixe suivant (AAA → AAB → ... → ZZZ)
func (s *PatientCodeGeneratorService) nextSuffix(current string) (string, error) {
	if current == "ZZZ" {
		return "", dto.NewCodeGenerationError(
			dto.ErrCodeCapaciteMaximale,
			"Capacité maximale atteinte pour l'année",
			"",
			time.Now().Year(),
		)
	}

	chars := []byte(strings.ToUpper(current))
	for i := 2; i >= 0; i-- {
		if chars[i] < 'Z' {
			chars[i]++
			break
		}
		chars[i] = 'A'
	}
	return string(chars), nil
}

// initializeRedisFromDB - Initialise Redis depuis PostgreSQL pour première génération
func (s *PatientCodeGeneratorService) initializeRedisFromDB(
	ctx context.Context,
	etablissementCode string,
	year int,
) (*dto.CodeGenerationResponse, error) {
	var numero int
	var suffixe string
	var nombreGeneres int64

	// Récupérer l'état depuis PostgreSQL
	err := s.db.QueryRow(ctx,
		queries.PatientCodeGenerationQueries.GetSequenceState,
		etablissementCode,
		year,
	).Scan(&numero, &suffixe, &nombreGeneres)

	if err == pgx.ErrNoRows {
		// Première génération absolue - utiliser fallback PostgreSQL
		return s.generateFromPostgres(ctx, etablissementCode, year, time.Now())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sequence state: %w", err)
	}

	// Synchroniser Redis
	redisKey := s.redisKeys.PatientSequenceKey(etablissementCode, year)
	redisValue := fmt.Sprintf("%d:%s", numero, suffixe)
	ttl := s.calculateTTLUntilYearEnd()

	if err := s.redis.Set(ctx, redisKey, redisValue, ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Maintenant générer via Redis
	return s.generateFromRedis(ctx, etablissementCode, year)
}

// syncRedisFromPostgres - Synchronise Redis avec l'état PostgreSQL
func (s *PatientCodeGeneratorService) syncRedisFromPostgres(
	ctx context.Context,
	etablissementCode string,
	year int,
	numero int,
	suffixe string,
) {
	redisKey := s.redisKeys.PatientSequenceKey(etablissementCode, year)
	redisValue := fmt.Sprintf("%d:%s", numero, suffixe)
	ttl := s.calculateTTLUntilYearEnd()

	// Best effort - on ignore les erreurs Redis en fallback
	s.redis.Set(ctx, redisKey, redisValue, ttl)
}

// updatePostgresAsync - Mise à jour asynchrone PostgreSQL depuis Redis
func (s *PatientCodeGeneratorService) updatePostgresAsync(
	etablissementCode string,
	year int,
	numero int,
	suffixe string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Best effort - on ignore les erreurs
	s.db.Exec(ctx,
		queries.PatientCodeGenerationQueries.UpdateSequenceAfterGeneration,
		etablissementCode,
		year,
		numero,
		suffixe,
	)
}

// calculateTTLUntilYearEnd - TTL jusqu'au 31 décembre 23:59:59
func (s *PatientCodeGeneratorService) calculateTTLUntilYearEnd() time.Duration {
	now := time.Now()
	endOfYear := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	return endOfYear.Sub(now)
}

// validateEtablissementCode - Validation du code établissement
func (s *PatientCodeGeneratorService) validateEtablissementCode(code string) error {
	if len(strings.TrimSpace(code)) == 0 {
		return dto.NewCodeGenerationError(
			dto.ErrCodeEtablissementInvalide,
			"Code établissement requis",
			code,
			0,
		)
	}
	if len(code) > 20 {
		return dto.NewCodeGenerationError(
			dto.ErrCodeEtablissementInvalide,
			"Code établissement trop long (max 20 caractères)",
			code,
			0,
		)
	}
	return nil
}

// GetSequenceStats - Récupère les statistiques de génération pour monitoring
func (s *PatientCodeGeneratorService) GetSequenceStats(
	ctx context.Context,
	etablissementCode string,
) (*dto.CodeGenerationStats, error) {
	year := time.Now().Year()
	
	var numero int
	var suffixe string
	var nombreGeneres int64

	err := s.db.QueryRow(ctx,
		queries.PatientCodeGenerationQueries.GetSequenceState,
		etablissementCode,
		year,
	).Scan(&numero, &suffixe, &nombreGeneres)

	if err == pgx.ErrNoRows {
		return &dto.CodeGenerationStats{
			EtablissementCode: etablissementCode,
			Annee:            year,
			NombreGeneres:    0,
			CapaciteUtilisee:  0.0,
			DernierCode:      "Aucun",
			ProchainCode:     fmt.Sprintf("%s-%d-001-AAA", etablissementCode, year),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Calculs statistiques
	capaciteMaximale := int64(17558424) // 999 * 17576 blocs
	capaciteUtilisee := (float64(nombreGeneres) / float64(capaciteMaximale)) * 100

	dernierCode := fmt.Sprintf("%s-%d-%03d-%s", etablissementCode, year, numero, suffixe)
	
	// Calculer le prochain code
	prochainNumero, prochainSuffixe, _ := s.incrementSequence(numero, suffixe)
	prochainCode := fmt.Sprintf("%s-%d-%03d-%s", etablissementCode, year, prochainNumero, prochainSuffixe)

	return &dto.CodeGenerationStats{
		EtablissementCode: etablissementCode,
		Annee:            year,
		NombreGeneres:    nombreGeneres,
		CapaciteUtilisee:  capaciteUtilisee,
		DernierCode:      dernierCode,
		ProchainCode:     prochainCode,
	}, nil
}