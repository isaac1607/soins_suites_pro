package utils

import "fmt"

// RedisKeyHelpers contient les helpers pour générer les clés Redis selon les conventions
// Pattern: soins_suite_{code_etablissement}_{domain}_{context}:{identifier}
// Note : Seuls les helpers pour les clés réellement implémentées sont définis

// MiddlewareCacheKey génère une clé de cache pour les middlewares
// Seule clé Redis actuellement implémentée dans le projet
func MiddlewareCacheKey(establishmentCode, dataType string) string {
	return fmt.Sprintf("soins_suite_%s_cache_middleware:%s", establishmentCode, dataType)
}