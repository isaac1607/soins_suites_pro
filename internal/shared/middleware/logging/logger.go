package logging

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLoggerConfig configuration pour le middleware logger Gin
type GinLoggerConfig struct {
	Environment string
	SkipPaths   []string
}

// NewGinLogger retourne le middleware logger Gin par défaut configuré
func NewGinLogger(config *GinLoggerConfig) gin.HandlerFunc {
	// En mode production, utiliser le logger Gin par défaut optimisé
	if config.Environment == "production" {
		return gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: config.SkipPaths,
		})
	}

	// En mode développement, utiliser le logger Gin par défaut avec format étendu
	return gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: config.SkipPaths,
		Formatter: ginCustomFormatter,
	})
}

// ginCustomFormatter format personnalisé pour le développement
func ginCustomFormatter(param gin.LogFormatterParams) string {
	var statusColor, methodColor, resetColor string
	
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}

	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	)
}

// DefaultSkipPaths chemins à ignorer par le logger
func DefaultSkipPaths() []string {
	return []string{
		"/health",
		"/ready",
		"/favicon.ico",
	}
}

// NewGinLoggerWithDefaults retourne le middleware logger avec configuration par défaut
func NewGinLoggerWithDefaults(environment string) gin.HandlerFunc {
	config := &GinLoggerConfig{
		Environment: environment,
		SkipPaths:   DefaultSkipPaths(),
	}
	
	return NewGinLogger(config)
}