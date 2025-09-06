package redis

import (
	"fmt"
	"regexp"
	"strings"
)

// RedisKeyGenerator génère et valide les clés Redis selon les conventions Soins Suite
type RedisKeyGenerator struct{}

// NewRedisKeyGenerator crée une nouvelle instance du générateur
func NewRedisKeyGenerator() *RedisKeyGenerator {
	return &RedisKeyGenerator{}
}

// RedisKeyPattern définit les patterns standards des clés selon les conventions
// Pattern: soins_suite_{code_etablissement}_{domain}_{context}:{identifier}
type RedisKeyPattern struct {
	Domain  string // auth, cache, setup, etc.
	Context string // session, license, state, etc.
	TTL     int    // TTL en secondes, 0 = pas d'expiration
}

// Patterns prédéfinis selon les conventions du projet
// Note : Seuls les patterns réellement implémentés sont listés ici
var RedisKeyPatterns = map[string]RedisKeyPattern{
	// Cache - Middlewares avec TTL différenciés
	"cache_middleware": {Domain: "cache", Context: "middleware", TTL: 0}, // TTL infini pour établissement (données immuables)
}

// GenerateKey génère une clé Redis selon la convention : soins_suite_{etablissement}_{domain}_{context}:{identifier}
func (rkg *RedisKeyGenerator) GenerateKey(patternName, establishmentCode string, identifier ...string) (string, error) {
	pattern, exists := RedisKeyPatterns[patternName]
	if !exists {
		return "", fmt.Errorf("pattern Redis non trouvé: %s", patternName)
	}

	// Validation du code établissement
	if establishmentCode == "" {
		return "", fmt.Errorf("code établissement requis pour la génération de clé")
	}
	if !rkg.isValidEstablishmentCode(establishmentCode) {
		return "", fmt.Errorf("code établissement invalide: %s", establishmentCode)
	}

	// Construction de la clé selon la convention
	// Format: soins_suite_{etablissement}_{domain}_{context}:{identifier}
	prefix := fmt.Sprintf("soins_suite_%s_%s_%s", establishmentCode, pattern.Domain, pattern.Context)
	
	if len(identifier) > 0 {
		// Joindre les identifiants avec "_" s'il y en a plusieurs
		identifierStr := strings.Join(identifier, "_")
		return fmt.Sprintf("%s:%s", prefix, identifierStr), nil
	}

	// Si pas d'identifier, retourner juste le préfixe (pour les clés singleton)
	return prefix, nil
}

// GetTTL récupère le TTL d'un pattern
func (rkg *RedisKeyGenerator) GetTTL(patternName string) (int, error) {
	pattern, exists := RedisKeyPatterns[patternName]
	if !exists {
		return 0, fmt.Errorf("pattern Redis non trouvé: %s", patternName)
	}
	return pattern.TTL, nil
}

// ValidateKey valide qu'une clé respecte les conventions
func (rkg *RedisKeyGenerator) ValidateKey(key string) error {
	// Vérifications de base
	if len(key) == 0 {
		return fmt.Errorf("clé vide")
	}

	if len(key) > 250 {
		return fmt.Errorf("clé trop longue (max 250 caractères): %d", len(key))
	}

	// Vérification format général
	validKeyRegex := regexp.MustCompile(`^[a-zA-Z0-9_:\-]+$`)
	if !validKeyRegex.MatchString(key) {
		return fmt.Errorf("clé contient des caractères invalides: %s", key)
	}

	// Vérification convention soins_suite_{etablissement}_{domain}_{context}
	if !strings.HasPrefix(key, "soins_suite_") {
		return fmt.Errorf("clé doit commencer par 'soins_suite_': %s", key)
	}

	// Extraction des parties pour validation
	parts := strings.SplitN(key, ":", 2)
	prefix := parts[0]
	
	// Vérification structure du préfixe
	prefixParts := strings.Split(prefix, "_")
	if len(prefixParts) < 4 {
		return fmt.Errorf("structure préfixe invalide (format: soins_suite_etablissement_domain_context): %s", prefix)
	}

	if prefixParts[0] != "soins" || prefixParts[1] != "suite" {
		return fmt.Errorf("préfixe incorrect: doit commencer par 'soins_suite': %s", prefix)
	}

	// Validation code établissement
	establishmentCode := prefixParts[2]
	if !rkg.isValidEstablishmentCode(establishmentCode) {
		return fmt.Errorf("code établissement invalide: %s", establishmentCode)
	}

	return nil
}

// isValidEstablishmentCode valide le format du code établissement
func (rkg *RedisKeyGenerator) isValidEstablishmentCode(code string) bool {
	// Même validation que le middleware : alphanumérique, 3-20 caractères, majuscules
	matched, _ := regexp.MatchString(`^[A-Z0-9]{3,20}$`, code)
	return matched
}

// ListAllPatterns retourne tous les patterns disponibles
func (rkg *RedisKeyGenerator) ListAllPatterns() map[string]RedisKeyPattern {
	return RedisKeyPatterns
}

// GenerateWildcardPattern génère un pattern wildcard pour recherche par domaine/context
func (rkg *RedisKeyGenerator) GenerateWildcardPattern(establishmentCode, domain, context string) string {
	return fmt.Sprintf("soins_suite_%s_%s_%s*", establishmentCode, domain, context)
}

// RedisKeyInfo contient les informations d'une clé analysée selon les conventions
type RedisKeyInfo struct {
	EstablishmentCode string
	Domain            string
	Context           string
	Identifier        string
	IsValid           bool
	Error             string
}

// AnalyzeKey analyse et décompose une clé Redis selon les conventions
func (rkg *RedisKeyGenerator) AnalyzeKey(key string) RedisKeyInfo {
	info := RedisKeyInfo{
		IsValid: false,
	}

	// Validation préliminaire
	if err := rkg.ValidateKey(key); err != nil {
		info.Error = err.Error()
		return info
	}

	// Découpage de la clé
	parts := strings.SplitN(key, ":", 2)
	prefix := parts[0]
	
	if len(parts) > 1 {
		info.Identifier = parts[1]
	}

	// Analyse du préfixe soins_suite_etablissement_domain_context
	prefixParts := strings.Split(prefix, "_")
	if len(prefixParts) >= 4 {
		info.EstablishmentCode = prefixParts[2]
		info.Domain = prefixParts[3]
		if len(prefixParts) >= 5 {
			// Si plus de 4 parties, joindre le reste pour le context
			info.Context = strings.Join(prefixParts[4:], "_")
		}
	}

	info.IsValid = true
	return info
}

// Helper functions for patterns management

// GetKeyInfo retourne les informations d'une clé (alias pour AnalyzeKey)
func (rkg *RedisKeyGenerator) GetKeyInfo(key string) RedisKeyInfo {
	return rkg.AnalyzeKey(key)
}

// BuildEstablishmentCacheKey construit une clé de cache pour un établissement
func (rkg *RedisKeyGenerator) BuildEstablishmentCacheKey(establishmentCode, cacheType string, identifier ...string) (string, error) {
	patternName := "cache_" + cacheType
	return rkg.GenerateKey(patternName, establishmentCode, identifier...)
}
