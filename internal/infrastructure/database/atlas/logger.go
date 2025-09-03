package atlas

import (
	"fmt"
	"time"
)

// AtlasLogger interface unifiée pour tous les managers Atlas
// Compatible avec le logger Gin par défaut imposé
type AtlasLogger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// GinCompatibleLogger implémentation compatible avec le logger Gin par défaut
type GinCompatibleLogger struct{}

// NewGinCompatibleLogger crée un logger compatible avec les standards Gin
func NewGinCompatibleLogger() *GinCompatibleLogger {
	return &GinCompatibleLogger{}
}

func (l *GinCompatibleLogger) Info(msg string, fields ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	if len(fields) > 0 {
		fmt.Printf("[ATLAS INFO] %s %s %v\n", timestamp, msg, fields)
	} else {
		fmt.Printf("[ATLAS INFO] %s %s\n", timestamp, msg)
	}
}

func (l *GinCompatibleLogger) Error(msg string, fields ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	if len(fields) > 0 {
		fmt.Printf("[ATLAS ERROR] %s %s %v\n", timestamp, msg, fields)
	} else {
		fmt.Printf("[ATLAS ERROR] %s %s\n", timestamp, msg)
	}
}

func (l *GinCompatibleLogger) Warn(msg string, fields ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	if len(fields) > 0 {
		fmt.Printf("[ATLAS WARN] %s %s %v\n", timestamp, msg, fields)
	} else {
		fmt.Printf("[ATLAS WARN] %s %s\n", timestamp, msg)
	}
}
