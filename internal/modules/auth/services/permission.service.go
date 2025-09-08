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

// GetUserPermissions récupère toutes les permissions d'un utilisateur TOUJOURS depuis la DB
// SÉCURITÉ CRITIQUE : Les permissions sont toujours récupérées depuis PostgreSQL
// pour éviter les problèmes de cache corrompu/obsolète qui pourraient compromettre la sécurité
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID, establishmentID, establishmentCode string) ([]dto.Permission, error) {
	// TOUJOURS récupérer depuis PostgreSQL (source de vérité)
	permissions, err := s.getPermissionsFromDB(ctx, userID, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des permissions: %w", err)
	}

	// Mettre en cache en arrière-plan pour les vérifications rapides CheckPermission uniquement
	// Le cache ne sert QUE pour les middlewares de vérification, pas pour les réponses API
	go s.cacheUserPermissions(ctx, establishmentCode, userID, permissions)

	return permissions, nil
}

// GetSuperAdminPermissions récupère toutes les permissions back-office pour super admin TOUJOURS depuis la DB
// SÉCURITÉ CRITIQUE : Même pour les super admins, toujours récupérer depuis PostgreSQL
func (s *PermissionService) GetSuperAdminPermissions(ctx context.Context, establishmentCode, userID string) ([]dto.Permission, error) {
	// TOUJOURS récupérer depuis PostgreSQL (source de vérité)
	permissions, err := s.getSuperAdminPermissionsFromDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des permissions super admin: %w", err)
	}

	// Mettre en cache en arrière-plan avec clé spécifique pour les vérifications rapides uniquement
	cacheKey := fmt.Sprintf("super_admin_%s", userID)
	go s.cacheUserPermissions(ctx, establishmentCode, cacheKey, permissions)

	return permissions, nil
}

// CheckPermission vérifie si un utilisateur a une permission spécifique
// STRATÉGIE : Cache Redis first (performance) avec fallback PostgreSQL (fiabilité)
// IMPORTANT : Cette méthode est utilisée pour les middlewares/guards (performance critique)
func (s *PermissionService) CheckPermission(ctx context.Context, userID, establishmentID, establishmentCode, module, rubrique string) (bool, error) {
	// 1. ÉTAPE 1 : Vérifier le cache Redis d'abord (performance optimale)
	hasAccess, cacheHit := s.checkPermissionFromCache(ctx, establishmentCode, userID, module, rubrique)
	if cacheHit {
		return hasAccess, nil
	}

	// 2. ÉTAPE 2 : Cache miss ou Redis indisponible - Fallback PostgreSQL (source de vérité)
	return s.checkPermissionFromDB(ctx, userID, establishmentID, module, rubrique)
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
			&permission.ID,
			&permission.CodeModule,
			&permission.NomStandard,
			&permission.NomPersonnalise,
			&permission.Description,
			&permission.AccesToutesRubriques,
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

// getSuperAdminPermissionsFromDB récupère tous les modules back-office depuis PostgreSQL
func (s *PermissionService) getSuperAdminPermissionsFromDB(ctx context.Context) ([]dto.Permission, error) {
	rows, err := s.db.Query(ctx, queries.UserQueries.GetSuperAdminPermissions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []dto.Permission
	for rows.Next() {
		var permission dto.Permission
		var rubriquesJSON []byte

		err := rows.Scan(
			&permission.ID,
			&permission.CodeModule,
			&permission.NomStandard,
			&permission.NomPersonnalise,
			&permission.Description,
			&permission.AccesToutesRubriques,
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

// cacheUserPermissions met en cache les permissions d'un utilisateur
func (s *PermissionService) cacheUserPermissions(ctx context.Context, establishmentCode, userID string, permissions []dto.Permission) error {
	pipe := s.redisClient.Client().Pipeline()

	// 1. Cache pour vérifications rapides (SET)
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)

	// Supprimer l'ancien cache
	pipe.Del(ctx, permissionsKey)

	// Ajouter les permissions
	for _, perm := range permissions {
		if perm.AccesToutesRubriques {
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

// checkPermissionFromCache vérifie une permission depuis le cache Redis (performance optimale)
// Retourne (hasAccess, cacheHit) - cacheHit=false indique qu'il faut faire le fallback PostgreSQL
func (s *PermissionService) checkPermissionFromCache(ctx context.Context, establishmentCode, userID, module, rubrique string) (bool, bool) {
	permissionsKey := fmt.Sprintf("soins_suite_%s_auth_permissions:%s", establishmentCode, userID)

	// 1. Vérifier d'abord l'accès module complet (plus performant)
	moduleAccess := fmt.Sprintf("module:%s", module)
	hasModule, err := s.redisClient.Client().SIsMember(ctx, permissionsKey, moduleAccess).Result()
	if err != nil {
		// Redis indisponible ou erreur - retourner cache miss pour fallback PostgreSQL
		return false, false
	}
	
	if hasModule {
		// Accès complet au module trouvé dans le cache
		return true, true
	}

	// 2. Si pas d'accès module ET rubrique demandée, vérifier la rubrique spécifique
	if rubrique != "" {
		rubriqueAccess := fmt.Sprintf("rubrique:%s:%s", module, rubrique)
		hasRubrique, err := s.redisClient.Client().SIsMember(ctx, permissionsKey, rubriqueAccess).Result()
		if err != nil {
			// Redis indisponible ou erreur - retourner cache miss pour fallback PostgreSQL
			return false, false
		}
		
		if hasRubrique {
			// Accès à la rubrique spécifique trouvé dans le cache
			return true, true
		}
	}

	// 3. Vérifier si la clé existe dans Redis (pour différencier cache miss vs permission refusée)
	exists, err := s.redisClient.Client().Exists(ctx, permissionsKey).Result()
	if err != nil || exists == 0 {
		// Redis indisponible OU permissions non cachées - fallback PostgreSQL nécessaire
		return false, false
	}

	// 4. Clé existe mais permission non trouvée - accès refusé selon le cache
	return false, true
}

// checkPermissionFromDB vérifie une permission depuis PostgreSQL (source de vérité)
func (s *PermissionService) checkPermissionFromDB(ctx context.Context, userID, establishmentID, module, rubrique string) (bool, error) {
	var hasAccess bool
	
	// Utiliser rubrique vide si non spécifiée pour vérifier seulement l'accès au module
	rubriqueParam := ""
	if rubrique != "" {
		rubriqueParam = rubrique
	}

	row := s.db.QueryRow(ctx, queries.UserQueries.CheckUserPermission, userID, establishmentID, module, rubriqueParam)
	err := row.Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("erreur lors de la vérification des permissions: %w", err)
	}

	return hasAccess, nil
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
func (s *PermissionService) HasModuleAccess(ctx context.Context, userID, establishmentID, establishmentCode, module string) (bool, error) {
	return s.CheckPermission(ctx, userID, establishmentID, establishmentCode, module, "")
}

// HasRubriqueAccess vérifie l'accès à une rubrique spécifique
func (s *PermissionService) HasRubriqueAccess(ctx context.Context, userID, establishmentID, establishmentCode, module, rubrique string) (bool, error) {
	return s.CheckPermission(ctx, userID, establishmentID, establishmentCode, module, rubrique)
}

// RefreshUserPermissions force le rafraîchissement des permissions depuis la DB
func (s *PermissionService) RefreshUserPermissions(ctx context.Context, userID, establishmentID, establishmentCode string) ([]dto.Permission, error) {
	// Invalider le cache
	s.InvalidateUserPermissions(ctx, establishmentCode, userID)

	// Récupérer depuis la DB et remettre en cache
	return s.GetUserPermissions(ctx, userID, establishmentID, establishmentCode)
}
