package core

import (
	"log/slog"
	"runtime"

	"github.com/gin-gonic/gin"
)

// RecoveryHandler type spécifique pour Fx
type RecoveryHandler gin.HandlerFunc

// RecoveryMiddleware capture les panics et retourne une réponse d'erreur propre
func RecoveryMiddleware() RecoveryHandler {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capturer la stack trace
				stack := make([]byte, 4096)
				n := runtime.Stack(stack, false)

				// Logger avec slog structuré
				slog.Error("panic recovered",
					"error", err,
					"stack", string(stack[:n]),
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"client_ip", c.ClientIP(),
					"request_id", c.GetString("request_id"),
				)

				// Répondre avec erreur standardisée
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Une erreur interne s'est produite",
					"details": map[string]interface{}{
						"code":       "INTERNAL_ERROR",
						"request_id": c.GetString("request_id"),
					},
				})
			}
		}()
		c.Next()
	}
}