package security

import (
	"regexp"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"soins-suite-core/internal/app/config"
)

// CORSHandler type spécifique pour Fx
type CORSHandler gin.HandlerFunc

// CORSMiddleware configure les règles CORS pour multi-tenant
func CORSMiddleware(appConfig *config.Config) CORSHandler {
	corsConfig := appConfig.GetCORS()

	return CORSHandler(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// 1. Autoriser tous les sous-domaines *.soins-suite.tir.ci
			allowedPattern := regexp.MustCompile(
				`^https?://([a-zA-Z0-9-]+\.)?((back-office\.)?soins-suite\.tir\.ci|localhost:(3000|3001|8080))$`,
			)
			
			// Vérifier le pattern principal
			if allowedPattern.MatchString(origin) {
				return true
			}

			// 2. Vérifier les origins configurés dans l'environnement
			for _, allowedOrigin := range corsConfig.AllowedOrigins {
				if origin == allowedOrigin {
					return true
				}
			}

			return false
		},

		// Méthodes HTTP autorisées
		AllowMethods: corsConfig.AllowedMethods,

		// Headers autorisés (inclut les headers multi-tenant)
		AllowHeaders: append(corsConfig.AllowedHeaders, 
			"X-Establishment-Code", 
			"X-Client-Type",
			"X-Request-Id"),

		// Headers exposés au client
		ExposeHeaders: []string{
			"Content-Length", 
			"X-Request-Id",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},

		// Autoriser les credentials (cookies, tokens)
		AllowCredentials: corsConfig.AllowCredentials,

		// Cache de la réponse preflight
		MaxAge: time.Duration(corsConfig.MaxAge) * time.Second,
	}))
}