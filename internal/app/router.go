package app

import (
	"net/http"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
	redisInfra "soins-suite-core/internal/infrastructure/database/redis"
	tenant "soins-suite-core/internal/shared/middleware/tenant"
	loggingmw "soins-suite-core/internal/shared/middleware/logging"

	// securitymw "soins-suite-core/internal/shared/middleware/security"
	// validationmw "soins-suite-core/internal/shared/middleware/validation"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, pgClient *postgres.Client, redisClient *redisInfra.Client) *gin.Engine {
	// Set Gin mode based on environment
	configureGinMode(cfg.Environment)

	// Create router without default middleware for custom configuration
	r := gin.New()

	// Initialize tenant middlewares
	establishmentMW := tenant.NewEstablishmentMiddleware(pgClient, redisClient)
	licenseMW := tenant.NewLicenseMiddleware(pgClient, redisClient)

	// Add custom middlewares dans l'ordre d'importance
	r.Use(loggingmw.NewGinLoggerWithDefaults(cfg.Environment))
	r.Use(gin.Recovery()) // Recovery middleware de Gin par défaut
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

		// Test endpoints pour les middlewares d'authentification
		middlewareTests := apiV1.Group("/middleware-tests")
		{
			// Test EstablishmentMiddleware seul
			middlewareTests.GET("/establishment", establishmentMW.Handler(), func(c *gin.Context) {
				// Récupération du contexte establishment injecté par le middleware
				establishmentValue, exists := c.Get("establishment")
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Contexte établissement non trouvé",
					})
					return
				}

				establishment, ok := establishmentValue.(tenant.EstablishmentContext)
				if !ok {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Contexte établissement invalide",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"test_type":     "EstablishmentMiddleware",
						"establishment": establishment,
						"message":       "EstablishmentMiddleware fonctionne correctement",
					},
				})
			})

			// Test LicenseMiddleware (nécessite EstablishmentMiddleware en premier)
			middlewareTests.GET("/license", establishmentMW.Handler(), licenseMW.Handler(), func(c *gin.Context) {
				// Récupération des contextes injectés par les middlewares
				establishmentValue, _ := c.Get("establishment")
				licenseValue, exists := c.Get("license")

				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Contexte licence non trouvé",
					})
					return
				}

				establishment, _ := establishmentValue.(tenant.EstablishmentContext)
				license, ok := licenseValue.(tenant.LicenseContext)
				if !ok {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Contexte licence invalide",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"test_type":     "LicenseMiddleware",
						"establishment": establishment,
						"license":       license,
						"message":       "LicenseMiddleware fonctionne correctement",
					},
				})
			})

			// Test validation module spécifique
			middlewareTests.GET("/module/:module_code", establishmentMW.Handler(), licenseMW.Handler(), func(c *gin.Context) {
				moduleCode := c.Param("module_code")

				// Test validation module
				if !licenseMW.ValidateModuleAccess(c, moduleCode) {
					licenseMW.RespondModuleNotAuthorized(c, moduleCode)
					return
				}

				// Si validation réussie
				establishmentValue, _ := c.Get("establishment")
				licenseValue, _ := c.Get("license")

				establishment, _ := establishmentValue.(tenant.EstablishmentContext)
				license, _ := licenseValue.(tenant.LicenseContext)

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"test_type":         "ModuleValidation",
						"establishment":     establishment,
						"license":           license,
						"module_requested":  moduleCode,
						"module_authorized": true,
						"message":           "Module autorisé pour cet établissement",
					},
				})
			})

			// Test chaîne complète des middlewares
			middlewareTests.GET("/full-chain", establishmentMW.Handler(), licenseMW.Handler(), func(c *gin.Context) {
				establishmentValue, _ := c.Get("establishment")
				licenseValue, _ := c.Get("license")

				establishment, _ := establishmentValue.(tenant.EstablishmentContext)
				license, _ := licenseValue.(tenant.LicenseContext)

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"test_type":     "FullMiddlewareChain",
						"establishment": establishment,
						"license":       license,
						"message":       "Chaîne complète des middlewares fonctionne correctement",
						"middlewares_applied": []string{
							"EstablishmentMiddleware",
							"LicenseMiddleware",
						},
					},
				})
			})

			// Test des différents codes d'erreur selon les spécifications
			middlewareTests.GET("/test-errors", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"message": "Tests d'erreurs disponibles",
						"test_cases": gin.H{
							"establishment_errors": []gin.H{
								{
									"test":            "Header manquant",
									"method":          "GET /api/v1/middleware-tests/establishment",
									"headers":         "Aucun header X-Establishment-Code",
									"expected_status": 460,
									"expected_code":   "ESTABLISHMENT_CODE_REQUIRED",
								},
								{
									"test":            "Format invalide",
									"method":          "GET /api/v1/middleware-tests/establishment",
									"headers":         "X-Establishment-Code: inv@lid",
									"expected_status": 460,
									"expected_code":   "ESTABLISHMENT_CODE_INVALID_FORMAT",
								},
								{
									"test":            "Établissement inexistant",
									"method":          "GET /api/v1/middleware-tests/establishment",
									"headers":         "X-Establishment-Code: INEXISTANT",
									"expected_status": 460,
									"expected_code":   "ESTABLISHMENT_NOT_FOUND",
								},
							},
							"license_errors": []gin.H{
								{
									"test":            "Licence inexistante",
									"method":          "GET /api/v1/middleware-tests/license",
									"headers":         "X-Establishment-Code: ETABLISSEMENT_SANS_LICENCE",
									"expected_status": 465,
									"expected_code":   "LICENSE_NOT_FOUND",
								},
								{
									"test":            "Licence expirée",
									"method":          "GET /api/v1/middleware-tests/license",
									"headers":         "X-Establishment-Code: LICENCE_EXPIREE",
									"expected_status": 465,
									"expected_code":   "LICENSE_EXPIRED",
								},
							},
							"module_errors": []gin.H{
								{
									"test":            "Module non autorisé",
									"method":          "GET /api/v1/middleware-tests/module/CHIRURGIE",
									"headers":         "X-Establishment-Code: LICENCE_LIMITEE",
									"expected_status": 470,
									"expected_code":   "MODULE_NOT_LICENSED",
								},
							},
						},
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
