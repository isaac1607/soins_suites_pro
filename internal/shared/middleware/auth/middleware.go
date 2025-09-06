package auth

import (
	"go.uber.org/fx"
	"github.com/gin-gonic/gin"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/shared/middleware/authentication"
)

// AuthMiddlewares contient tous les middlewares d'authentification
type AuthMiddlewares struct {
	Session    *SessionMiddleware
	Permission *PermissionMiddleware
}

// NewAuthMiddlewares crée une nouvelle instance des middlewares d'authentification
func NewAuthMiddlewares(db *postgres.Client, redisClient *redis.Client) *AuthMiddlewares {
	return &AuthMiddlewares{
		Session:    NewSessionMiddleware(db, redisClient),
		Permission: NewPermissionMiddleware(db, redisClient),
	}
}

// AuthMiddlewareStack représente une pile de middlewares d'authentification
type AuthMiddlewareStack struct {
	EstablishmentMiddleware *authentication.EstablishmentMiddleware
	SessionMiddleware       *SessionMiddleware
	PermissionMiddleware    *PermissionMiddleware
}

// NewAuthMiddlewareStack crée une nouvelle pile de middlewares
func NewAuthMiddlewareStack(
	establishmentMiddleware *authentication.EstablishmentMiddleware,
	authMiddlewares *AuthMiddlewares,
) *AuthMiddlewareStack {
	return &AuthMiddlewareStack{
		EstablishmentMiddleware: establishmentMiddleware,
		SessionMiddleware:       authMiddlewares.Session,
		PermissionMiddleware:    authMiddlewares.Permission,
	}
}

// ApplyBasicAuth applique la chaîne de middlewares de base (establishment + session)
func (stack *AuthMiddlewareStack) ApplyBasicAuth() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		stack.EstablishmentMiddleware.Handler(),
		stack.SessionMiddleware.Handler(),
	}
}

// ApplyFullAuth applique tous les middlewares d'authentification pour une permission donnée
func (stack *AuthMiddlewareStack) ApplyFullAuth(permission string) []gin.HandlerFunc {
	middlewares := stack.ApplyBasicAuth()
	middlewares = append(middlewares, stack.PermissionMiddleware.RequirePermission(permission))
	return middlewares
}

// ApplyModuleAuth applique l'authentification pour un module spécifique
func (stack *AuthMiddlewareStack) ApplyModuleAuth(moduleCode string) []gin.HandlerFunc {
	middlewares := stack.ApplyBasicAuth()
	middlewares = append(middlewares, stack.PermissionMiddleware.RequireModule(moduleCode))
	return middlewares
}

// ApplyRubriqueAuth applique l'authentification pour une rubrique spécifique
func (stack *AuthMiddlewareStack) ApplyRubriqueAuth(moduleCode, rubriqueCode string) []gin.HandlerFunc {
	middlewares := stack.ApplyBasicAuth()
	middlewares = append(middlewares, stack.PermissionMiddleware.RequireRubrique(moduleCode, rubriqueCode))
	return middlewares
}

// ApplyAdminAuth applique l'authentification pour les administrateurs uniquement
func (stack *AuthMiddlewareStack) ApplyAdminAuth() []gin.HandlerFunc {
	middlewares := stack.ApplyBasicAuth()
	middlewares = append(middlewares, stack.PermissionMiddleware.RequireAdmin())
	return middlewares
}

// ApplyClientTypeAuth applique l'authentification pour un type de client spécifique
func (stack *AuthMiddlewareStack) ApplyClientTypeAuth(clientType string) []gin.HandlerFunc {
	middlewares := stack.ApplyBasicAuth()
	middlewares = append(middlewares, stack.PermissionMiddleware.RequireClientType(clientType))
	return middlewares
}

// Module Fx pour l'injection de dépendances
var AuthMiddlewareModule = fx.Options(
	fx.Provide(NewAuthMiddlewares),
	fx.Provide(NewAuthMiddlewareStack),
)

// Helpers pour les routes courantes

// Protected applique l'authentification de base (establishment + session)
func Protected(stack *AuthMiddlewareStack) []gin.HandlerFunc {
	return stack.ApplyBasicAuth()
}

// RequireModule crée un middleware pour un module spécifique
func RequireModule(stack *AuthMiddlewareStack, moduleCode string) []gin.HandlerFunc {
	return stack.ApplyModuleAuth(moduleCode)
}

// RequireRubrique crée un middleware pour une rubrique spécifique
func RequireRubrique(stack *AuthMiddlewareStack, moduleCode, rubriqueCode string) []gin.HandlerFunc {
	return stack.ApplyRubriqueAuth(moduleCode, rubriqueCode)
}

// RequireAdmin crée un middleware pour administrateurs
func RequireAdmin(stack *AuthMiddlewareStack) []gin.HandlerFunc {
	return stack.ApplyAdminAuth()
}

// RequireBackOffice crée un middleware pour le back-office
func RequireBackOffice(stack *AuthMiddlewareStack) []gin.HandlerFunc {
	return stack.ApplyClientTypeAuth("back-office")
}

// RequireFrontOffice crée un middleware pour le front-office
func RequireFrontOffice(stack *AuthMiddlewareStack) []gin.HandlerFunc {
	return stack.ApplyClientTypeAuth("front-office")
}