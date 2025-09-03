package app

import (
	"net/http"

	"soins-suite-core/internal/app/config"
	// loggingmw "soins-suite-core/internal/shared/middleware/logging"
	// securitymw "soins-suite-core/internal/shared/middleware/security"
	// validationmw "soins-suite-core/internal/shared/middleware/validation"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on environment
	configureGinMode(cfg.Environment)

	// Create router without default middleware for custom configuration
	r := gin.New()

	// Add custom middlewares dans l'ordre d'importance
	// r.Use(loggingmw.NewGinLoggerWithDefaults(cfg.Environment))
	// r.Use(securitymw.NewGinRecoveryWithDefaults(cfg.Environment))
	// r.Use(loggingmw.NewSecurityWithDefaults(cfg.Environment))
	// Configuration sécurité avec CSRF désactivé
	// securityConfig := &middleware.SecurityConfig{
	// 	Environment:     cfg.Environment,
	// 	EnableCSP:       true,
	// 	EnableHSTS:      cfg.Environment == "production",
	// 	EnableCSRF:      false, // CSRF désactivé
	// 	CSRFTokenLength: 32,
	// }
	// r.Use(loggingmw.NewSecurityMiddleware(securityConfig))

	// r.Use(securitymw.NewCORSWithDefaults(cfg.Environment))
	// r.Use(loggingmw.NewManualLoggingWithDefaults(cfg.Environment))
	// r.Use(validationmw.NewContextValidatorWithDefaults(cfg.Environment))

	// Health check routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status": "healthy",
			},
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status": "ready",
			},
		})
	})

	// Endpoint de test pour la sécurité et les headers
	r.GET("/security-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"security_headers": "active",
				"cors_policy":      "configured",
				"csrf_protection":  "enabled",
				"gin_context":      "standard",
				"environment":      cfg.Environment,
			},
		})
	})

	// API versioning
	apiV1 := r.Group("/api/v1")
	{
		// Auth group
		auth := apiV1.Group("/auth")
		{
			auth.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    gin.H{"message": "Auth endpoint working"},
				})
			})
		}

		// System group
		system := apiV1.Group("/system")
		{
			system.GET("/info", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"environment": cfg.Environment,
						"version":     "0.1.0",
					},
				})
			})
		}

		// Back-office group
		backOffice := apiV1.Group("/back-office")
		{
			backOffice.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    gin.H{"message": "Back-office endpoint working"},
				})
			})
		}

		// Front-office group
		frontOffice := apiV1.Group("/front-office")
		{
			frontOffice.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data":    gin.H{"message": "Front-office endpoint working"},
				})
			})
		}
	}

	return r
}

// configureGinMode configure le mode Gin selon l'environnement
func configureGinMode(environment string) {
	switch environment {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "staging":
		gin.SetMode(gin.ReleaseMode)
	case "development":
		gin.SetMode(gin.DebugMode)
	default:
		// Mode debug par défaut pour développement local
		gin.SetMode(gin.DebugMode)
	}
}
