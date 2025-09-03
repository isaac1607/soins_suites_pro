package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"soins-suite-core/internal/app/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// ✅ APPLICATION  SIMPLIFIÉE
// Configuration uniquement via variables d'environnement
// Compatible avec le nouveau système config_

// Application version simplifiée de l'application
type Application struct {
	config *config.Config
	router *gin.Engine
	server *http.Server
}

// NewApplication crée une nouvelle instance de l'application
func NewApplication(cfg *config.Config, router *gin.Engine) *Application {
	return &Application{
		config: cfg,
		router: router,
	}
}

// Start démarre l'application  avec lifecycle Fx
func (a *Application) Start(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			serverConfig := a.config.GetServer()

			a.server = &http.Server{
				Addr:         fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port),
				Handler:      a.router,
				ReadTimeout:  serverConfig.ReadTimeout,
				WriteTimeout: serverConfig.WriteTimeout,
			}

			// Démarrage serveur en goroutine
			go func() {
				fmt.Printf("[SERVER] 🚀 Démarrage serveur HTTP sur %s:%d\n", serverConfig.Host, serverConfig.Port)
				if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					fmt.Printf("[SERVER] ❌ Échec démarrage serveur: %v\n", err)
				}
			}()

			fmt.Printf("[SERVER] ✅ Serveur HTTP initialisé (env: %s)\n", a.config.Environment)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Printf("[SERVER] 🛑 Arrêt serveur HTTP\n")

			// Timeout pour arrêt graceful
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			if err := a.server.Shutdown(shutdownCtx); err != nil {
				fmt.Printf("[SERVER] ⚠️ Arrêt forcé: %v\n", err)
				return err
			}

			fmt.Printf("[SERVER] ✅ Serveur arrêté proprement\n")
			return nil
		},
	})
}

// GetConfig retourne la configuration pour accès externe
func (a *Application) GetConfig() *config.Config {
	return a.config
}

// IsDocker indique si l'application est en mode docker (production/staging)
func (a *Application) IsDocker() bool {
	return a.config.Environment == "docker"
}

// IsDevelopment indique si l'application est en mode développement
func (a *Application) IsDevelopment() bool {
	return a.config.Environment == "development"
}

// IsProduction alias pour IsDocker (rétrocompatibilité)
func (a *Application) IsProduction() bool {
	return a.IsDocker()
}
