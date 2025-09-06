package utils

import "fmt"

// RedisKeyHelpers contient les helpers pour générer les clés Redis selon les conventions
// Pattern: soins_suite_{code_etablissement}_{domain}_{context}:{identifier}

// MiddlewareCacheKey génère une clé de cache pour les middlewares
func MiddlewareCacheKey(establishmentCode, dataType string) string {
	return fmt.Sprintf("soins_suite_%s_cache_middleware:%s", establishmentCode, dataType)
}

// AuthSessionKey génère une clé de session d'authentification
func AuthSessionKey(establishmentCode, token string) string {
	return fmt.Sprintf("soins_suite_%s_auth_session:%s", establishmentCode, token)
}

// AuthPermissionsKey génère une clé de cache des permissions utilisateur
func AuthPermissionsKey(establishmentCode, userID string) string {
	return fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
}

// AuthPermissionsDetailKey génère une clé de cache détaillé des permissions
func AuthPermissionsDetailKey(establishmentCode, userID string) string {
	return fmt.Sprintf("soins_suite_%s_auth_permissions_detail:%s", establishmentCode, userID)
}

// AuthUserSessionsKey génère une clé d'index des sessions utilisateur
func AuthUserSessionsKey(establishmentCode, userID string) string {
	return fmt.Sprintf("soins_suite_%s_auth_user_sessions:%s", establishmentCode, userID)
}

// AuthRateLimitKey génère une clé de rate limiting
func AuthRateLimitKey(establishmentCode, identifiant string) string {
	return fmt.Sprintf("soins_suite_%s_auth_ratelimit:%s", establishmentCode, identifiant)
}

// AuthBlacklistKey génère une clé de blacklist pour un token
func AuthBlacklistKey(establishmentCode, token string) string {
	return fmt.Sprintf("soins_suite_%s_auth_blacklist:%s", establishmentCode, token)
}