package redis

import (
	"fmt"
	"regexp"
	"strings"
)

// RedisKeyGenerator génère et valide les clés Redis selon les standards
type RedisKeyGenerator struct {
	environment string
	namespace   string
}

// NewRedisKeyGenerator crée une nouvelle instance du générateur
func NewRedisKeyGenerator(environment string) *RedisKeyGenerator {
	return &RedisKeyGenerator{
		environment: environment,
		namespace:   "soins_suite",
	}
}

// RedisKeyPattern définit les patterns standards des clés
type RedisKeyPattern struct {
	Module     string
	Feature    string
	TTL        int // TTL en secondes, 0 = pas d'expiration
	Persistent bool
}

// Patterns prédéfinis pour chaque module
var RedisKeyPatterns = map[string]RedisKeyPattern{
	// Module System - Pattern unique simplifié
	"system_status": {Module: "system", Feature: "status", TTL: 1800, Persistent: false}, // 30min

	// Module Auth - Patterns réels utilisés dans l'implémentation
	"auth_session":       {Module: "auth", Feature: "session", TTL: 3600, Persistent: false},       // 1h - sessions actives
	"auth_user_sessions": {Module: "auth", Feature: "user_sessions", TTL: 3600, Persistent: false}, // 1h - index sessions par utilisateur
	"auth_permissions":   {Module: "auth", Feature: "permissions", TTL: 3600, Persistent: false},   // 1h - cache permissions utilisateur
}

// GenerateKey génère une clé Redis standardisée
func (rkg *RedisKeyGenerator) GenerateKey(patternName string, identifier ...string) (string, error) {
	pattern, exists := RedisKeyPatterns[patternName]
	if !exists {
		return "", fmt.Errorf("pattern Redis non trouvé: %s", patternName)
	}

	// Construction de la clé
	var keyParts []string
	keyParts = append(keyParts, rkg.environment)
	keyParts = append(keyParts, rkg.namespace)
	keyParts = append(keyParts, pattern.Module)
	keyParts = append(keyParts, pattern.Feature)

	// Ajout des identifiants
	if len(identifier) > 0 {
		keyParts = append(keyParts, identifier...)
	}

	key := strings.Join(keyParts, ":")

	// Validation finale
	if err := rkg.ValidateKey(key); err != nil {
		return "", fmt.Errorf("clé générée invalide: %w", err)
	}

	return key, nil
}

// GetTTL récupère le TTL d'un pattern
func (rkg *RedisKeyGenerator) GetTTL(patternName string) (int, error) {
	pattern, exists := RedisKeyPatterns[patternName]
	if !exists {
		return 0, fmt.Errorf("pattern Redis non trouvé: %s", patternName)
	}
	return pattern.TTL, nil
}

// IsPersistent vérifie si une clé doit être persistante
func (rkg *RedisKeyGenerator) IsPersistent(patternName string) bool {
	pattern, exists := RedisKeyPatterns[patternName]
	if !exists {
		return false
	}
	return pattern.Persistent
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

	// Vérification format
	validKeyRegex := regexp.MustCompile(`^[a-zA-Z0-9_:\-]+$`)
	if !validKeyRegex.MatchString(key) {
		return fmt.Errorf("clé contient des caractères invalides: %s", key)
	}

	// Vérification structure
	parts := strings.Split(key, ":")
	if len(parts) < 4 {
		return fmt.Errorf("structure clé invalide (min 4 parties): %s", key)
	}

	// Vérification environnement
	if parts[0] != rkg.environment {
		return fmt.Errorf("environnement incorrect: attendu %s, reçu %s", rkg.environment, parts[0])
	}

	// Vérification namespace
	if parts[1] != rkg.namespace {
		return fmt.Errorf("namespace incorrect: attendu %s, reçu %s", rkg.namespace, parts[1])
	}

	return nil
}

// ListAllPatterns retourne tous les patterns disponibles
func (rkg *RedisKeyGenerator) ListAllPatterns() map[string]RedisKeyPattern {
	return RedisKeyPatterns
}

// GenerateWildcardPattern génère un pattern wildcard pour recherche
func (rkg *RedisKeyGenerator) GenerateWildcardPattern(module string, feature string) string {
	return fmt.Sprintf("%s:%s:%s:%s:*", rkg.environment, rkg.namespace, module, feature)
}

// RedisKeyInfo contient les informations d'une clé analysée
type RedisKeyInfo struct {
	Environment string
	Namespace   string
	Module      string
	Feature     string
	Identifier  []string
	IsValid     bool
	Error       string
}

// AnalyzeKey analyse et décompose une clé Redis
func (rkg *RedisKeyGenerator) AnalyzeKey(key string) RedisKeyInfo {
	parts := strings.Split(key, ":")
	info := RedisKeyInfo{
		IsValid: false,
	}

	if len(parts) < 4 {
		info.Error = "Structure insuffisante"
		return info
	}

	info.Environment = parts[0]
	info.Namespace = parts[1]
	info.Module = parts[2]
	info.Feature = parts[3]

	if len(parts) > 4 {
		info.Identifier = parts[4:]
	}

	// Validation
	if err := rkg.ValidateKey(key); err != nil {
		info.Error = err.Error()
		return info
	}

	info.IsValid = true
	return info
}
