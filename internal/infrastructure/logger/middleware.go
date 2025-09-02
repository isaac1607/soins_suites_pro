package logger

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type LoggerMiddleware struct{}

func (lm *LoggerMiddleware) GinLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: lm.customFormatter,
		SkipPaths: []string{"/health", "/ready"},
	})
}

func (lm *LoggerMiddleware) customFormatter(param gin.LogFormatterParams) string {
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

func (lm *LoggerMiddleware) GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		fmt.Printf("[PANIC] %s Recovered from panic: %v\n", 
			time.Now().Format("15:04:05"), recovered)
		
		c.JSON(500, gin.H{
			"error": "Une erreur interne est survenue.",
		})
	})
}