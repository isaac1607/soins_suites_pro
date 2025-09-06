package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"
	"soins-suite-core/internal/modules/auth/dto"
	"soins-suite-core/internal/modules/auth/queries"
)

type PermissionService struct {
	db          *postgres.Client
	redisClient *redis.Client
}

// NewPermissionService crée une nouvelle instance du service de permissions
func NewPermissionService(db *postgres.Client, redisClient *redis.Client) *PermissionService {
	return &PermissionService{
		db:          db,
		redisClient: redisClient,
	}
}

// GetUserPermissions récupère toutes les permissions d'un utilisateur avec mise en cache
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID, establishmentID, establishmentCode string) ([]dto.Permission, error) {
	// Essayer de récupérer depuis le cache Redis
	if permissions, found := s.getPermissionsFromCache(ctx, establishmentCode, userID); found {
		return permissions, nil
	}

	// Récupérer depuis PostgreSQL
	permissions, err := s.getPermissionsFromDB(ctx, userID, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des permissions: %w", err)
	}

	// Mettre en cache
	go s.cacheUserPermissions(ctx, establishmentCode, userID, permissions)

	return permissions, nil
}

// CheckPermission vérifie si un utilisateur a une permission spécifique
func (s *PermissionService) CheckPermission(ctx context.Context, userID, establishmentCode, module, rubrique string) (bool, error) {
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)

	// Vérifier l'accès complet au module
	moduleAccess := fmt.Sprintf("module:%s", module)
	hasModule, err := s.redisClient.Client().SIsMember(ctx, permissionsKey, moduleAccess).Result()
	if err == nil && hasModule {
		return true, nil
	}

	// Vérifier l'accès à la rubrique spécifique
	if rubrique != "" {
		rubriqueAccess := fmt.Sprintf("rubrique:%s:%s", module, rubrique)
		hasRubrique, err := s.redisClient.Client().SIsMember(ctx, permissionsKey, rubriqueAccess).Result()
		if err == nil && hasRubrique {
			return true, nil
		}
	}

	// Si Redis n'est pas disponible, fallback base de données
	if err != nil {
		return s.checkPermissionFromDB(ctx, userID, module, rubrique)
	}

	return false, nil
}

// CacheUserPermissions met en cache les permissions d'un utilisateur
func (s *PermissionService) CacheUserPermissions(ctx context.Context, establishmentCode, userID string, permissions []dto.Permission) error {
	return s.cacheUserPermissions(ctx, establishmentCode, userID, permissions)
}

// InvalidateUserPermissions invalide le cache des permissions d'un utilisateur
func (s *PermissionService) InvalidateUserPermissions(ctx context.Context, establishmentCode, userID string) error {
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
	return s.redisClient.Del(ctx, permissionsKey)
}

// getPermissionsFromDB récupère les permissions depuis PostgreSQL
func (s *PermissionService) getPermissionsFromDB(ctx context.Context, userID, establishmentID string) ([]dto.Permission, error) {
	rows, err := s.db.Query(ctx, queries.UserQueries.GetUserPermissions, userID, establishmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []dto.Permission
	for rows.Next() {
		var permission dto.Permission
		var rubriquesJSON []byte

		err := rows.Scan(
			&permission.CodeModule,
			&permission.NomStandard,
			&permission.NomPersonnalise,
			&permission.Description,
			&rubriquesJSON,
		)
		if err != nil {
			continue
		}

		// Parser les rubriques JSON
		var rubriques []dto.Rubrique
		if len(rubriquesJSON) > 0 && string(rubriquesJSON) != "[]" {
			json.Unmarshal(rubriquesJSON, &rubriques)
		}
		permission.Rubriques = rubriques

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// getPermissionsFromCache récupère les permissions depuis Redis
func (s *PermissionService) getPermissionsFromCache(ctx context.Context, establishmentCode, userID string) ([]dto.Permission, bool) {
	// Vérifier si le cache des permissions existe
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
	
	exists, err := s.redisClient.Exists(ctx, permissionsKey)
	if err != nil || !exists {
		return nil, false
	}

	// Le cache existe, essayer de récupérer les permissions détaillées
	detailKey := fmt.Sprintf("soins_suite_%s_auth_permissions_detail:%s", establishmentCode, userID)
	permissionsJSON, err := s.redisClient.Get(ctx, detailKey)
	if err != nil {
		return nil, false
	}

	var permissions []dto.Permission
	if err := json.Unmarshal([]byte(permissionsJSON), &permissions); err != nil {
		// Cache corrompu, le supprimer
		s.redisClient.Del(ctx, permissionsKey)
		s.redisClient.Del(ctx, detailKey)
		return nil, false
	}

	return permissions, true
}

// cacheUserPermissions met en cache les permissions d'un utilisateur
func (s *PermissionService) cacheUserPermissions(ctx context.Context, establishmentCode, userID string, permissions []dto.Permission) error {
	pipe := s.redisClient.Client().Pipeline()

	// 1. Cache pour vérifications rapides (SET)
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
	
	// Supprimer l'ancien cache
	pipe.Del(ctx, permissionsKey)

	// Ajouter les permissions
	for _, perm := range permissions {
		if len(perm.Rubriques) == 0 {
			// Accès complet au module
			pipe.SAdd(ctx, permissionsKey, fmt.Sprintf("module:%s", perm.CodeModule))
		} else {
			// Accès aux rubriques spécifiques
			for _, rubrique := range perm.Rubriques {
				pipe.SAdd(ctx, permissionsKey, fmt.Sprintf("rubrique:%s:%s", perm.CodeModule, rubrique.CodeRubrique))
			}
		}
	}

	// TTL pour le SET
	pipe.Expire(ctx, permissionsKey, time.Hour)

	// 2. Cache détaillé pour récupération complète (JSON)
	detailKey := fmt.Sprintf("soins_suite_%s_auth_permissions_detail:%s", establishmentCode, userID)
	permissionsJSON, err := json.Marshal(permissions)
	if err == nil {
		pipe.Set(ctx, detailKey, string(permissionsJSON), time.Hour)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// checkPermissionFromDB vérifie une permission depuis PostgreSQL (fallback)
func (s *PermissionService) checkPermissionFromDB(ctx context.Context, userID, module, rubrique string) (bool, error) {
	// Récupérer toutes les permissions (plus coûteux mais nécessaire pour fallback)
	permissions, err := s.getPermissionsFromDB(ctx, userID, "")
	if err != nil {
		return false, err
	}

	// Vérifier la permission
	for _, perm := range permissions {
		if perm.CodeModule == module {
			// Si pas de rubriques spécifiées, accès complet
			if len(perm.Rubriques) == 0 {
				return true, nil
			}

			// Vérifier la rubrique spécifique
			if rubrique != "" {
				for _, r := range perm.Rubriques {
					if r.CodeRubrique == rubrique {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// GetUserPermissionsList retourne la liste des permissions sous forme de strings (pour middleware)
func (s *PermissionService) GetUserPermissionsList(ctx context.Context, establishmentCode, userID string) ([]string, error) {
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)
	
	members, err := s.redisClient.Client().SMembers(ctx, permissionsKey).Result()
	if err != nil {
		return nil, err
	}

	return members, nil
}

// HasModuleAccess vérifie l'accès à un module complet
func (s *PermissionService) HasModuleAccess(ctx context.Context, userID, establishmentCode, module string) (bool, error) {
	return s.CheckPermission(ctx, userID, establishmentCode, module, "")
}

// HasRubriqueAccess vérifie l'accès à une rubrique spécifique
func (s *PermissionService) HasRubriqueAccess(ctx context.Context, userID, establishmentCode, module, rubrique string) (bool, error) {
	return s.CheckPermission(ctx, userID, establishmentCode, module, rubrique)
}

// RefreshUserPermissions force le rafraîchissement des permissions depuis la DB
func (s *PermissionService) RefreshUserPermissions(ctx context.Context, userID, establishmentID, establishmentCode string) ([]dto.Permission, error) {
	// Invalider le cache
	s.InvalidateUserPermissions(ctx, establishmentCode, userID)

	// Récupérer depuis la DB et remettre en cache
	return s.GetUserPermissions(ctx, userID, establishmentID, establishmentCode)
}