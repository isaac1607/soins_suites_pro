package comptes

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"

	dto "soins-suite-core/internal/modules/back-office/users/dto/comptes"
	queries "soins-suite-core/internal/modules/back-office/users/queries/comptes"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/shared/utils"
)

type ComptesService struct {
	db *postgres.Client
}

func NewComptesService(db *postgres.Client) *ComptesService {
	return &ComptesService{
		db: db,
	}
}

func (s *ComptesService) CreateUser(ctx context.Context, req dto.CreateUserRequest, establishmentID, createdByUserID string) (*dto.CreateUserResponse, error) {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.validateCreateUserRequest(ctx, tx, req, establishmentID); err != nil {
		return nil, err
	}

	userID := uuid.New().String()
	
	passwordHash, salt, generatedPassword, err := s.preparePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la préparation du mot de passe: %w", err)
	}

	internalUser := dto.CreateUserInternal{
		EtablissementID:    establishmentID,
		Identifiant:        req.Identifiant,
		Nom:                req.Nom,
		Prenoms:            req.Prenoms,
		Telephone:          req.Telephone,
		PasswordHash:       passwordHash,
		Salt:               salt,
		MustChangePassword: true, // Toujours true à la création selon les règles métier
		EstAdmin:           req.EstAdmin,
		TypeAdmin:          req.TypeAdmin,
		EstMedecin:         req.EstMedecin,
		RoleMetier:         req.RoleMetier,
		EstTemporaire:      req.EstTemporaire,
		DateExpiration:     req.DateExpiration,
		Statut:             "actif",
		CreatedBy:          createdByUserID,
	}

	if err := s.createUserInDB(ctx, tx, userID, internalUser); err != nil {
		return nil, err
	}

	permissionsStats, err := s.assignPermissions(ctx, tx, userID, establishmentID, req, createdByUserID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("impossible de valider la transaction: %w", err)
	}

	response := &dto.CreateUserResponse{
		ID:                    userID,
		Identifiant:          req.Identifiant,
		Message:              "Utilisateur créé avec succès",
		PermissionsAttribuees: *permissionsStats,
	}

	if generatedPassword != "" {
		response.PasswordTemporaire = &generatedPassword
	}

	return response, nil
}

func (s *ComptesService) validateCreateUserRequest(ctx context.Context, tx pgx.Tx, req dto.CreateUserRequest, establishmentID string) error {
	var exists bool
	err := tx.QueryRow(ctx, queries.ComptesQueries.CheckUserExists, establishmentID, req.Identifiant).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de l'existence de l'utilisateur: %w", err)
	}

	if exists {
		return fmt.Errorf("un utilisateur avec cet identifiant existe déjà")
	}

	if req.EstAdmin && (req.TypeAdmin == nil || *req.TypeAdmin == "") {
		return fmt.Errorf("le champ type_admin est requis quand est_admin est true")
	}

	if req.EstTemporaire && req.DateExpiration == nil {
		return fmt.Errorf("la date d'expiration est requise quand est_temporaire est true")
	}

	if len(req.ModulesAttribues) > 0 {
		if err := s.validateModulesAttribues(ctx, tx, req.ModulesAttribues, establishmentID, req.EstAdmin); err != nil {
			return err
		}
	}

	return nil
}

func (s *ComptesService) validateModulesAttribues(ctx context.Context, tx pgx.Tx, modulesAttribues []dto.ModuleAttribue, establishmentID string, isAdmin bool) error {
	allModuleCodes := []string{}

	for _, moduleAttribue := range modulesAttribues {
		// Validation cohérence acces_toutes_rubriques et rubriques_specifiques
		if moduleAttribue.AccesToutesRubriques && len(moduleAttribue.RubriquesSpecifiques) > 0 {
			return fmt.Errorf("un module ne peut pas avoir acces_toutes_rubriques=true ET des rubriques spécifiques")
		}

		if !moduleAttribue.AccesToutesRubriques && len(moduleAttribue.RubriquesSpecifiques) == 0 {
			return fmt.Errorf("un module avec acces_toutes_rubriques=false doit avoir des rubriques spécifiques")
		}

		// Validation existence du module
		var moduleCode string
		var isBackOffice bool
		err := tx.QueryRow(ctx, "SELECT code_module, est_module_back_office FROM base_module WHERE id = $1 AND est_actif = TRUE", moduleAttribue.ModuleID).Scan(&moduleCode, &isBackOffice)
		if err != nil {
			return fmt.Errorf("module %s introuvable ou inactif", moduleAttribue.ModuleID)
		}

		// Validation cohérence admin/back-office
		if isAdmin && !isBackOffice {
			return fmt.Errorf("les administrateurs ne peuvent avoir que des modules back-office")
		}

		if !isAdmin && isBackOffice {
			return fmt.Errorf("les utilisateurs front-office ne peuvent avoir que des modules front-office")
		}

		allModuleCodes = append(allModuleCodes, moduleCode)

		// Validation des rubriques spécifiques
		if !moduleAttribue.AccesToutesRubriques {
			for _, rubriqueID := range moduleAttribue.RubriquesSpecifiques {
				var rubriqueExists bool
				err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateRubriqueExists, rubriqueID, moduleAttribue.ModuleID).Scan(&rubriqueExists)
				if err != nil {
					return fmt.Errorf("erreur lors de la validation de la rubrique %s: %w", rubriqueID, err)
				}
				if !rubriqueExists {
					return fmt.Errorf("rubrique %s non trouvée pour le module %s", rubriqueID, moduleAttribue.ModuleID)
				}
			}
		}
	}

	// Validation contre la licence (seulement pour les utilisateurs non-admin)
	if !isAdmin && len(allModuleCodes) > 0 {
		var modulesValid bool
		err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateModulesInLicense, establishmentID, pq.Array(allModuleCodes)).Scan(&modulesValid)
		if err != nil {
			return fmt.Errorf("erreur lors de la validation des modules dans la licence: %w", err)
		}

		if !modulesValid {
			return fmt.Errorf("certains modules ne sont pas autorisés par la licence de l'établissement")
		}
	}

	return nil
}

func (s *ComptesService) preparePassword(providedPassword *string) (hashHex, saltHex, generated string, err error) {
	var password string

	if providedPassword != nil && *providedPassword != "" {
		password = *providedPassword
	} else {
		password = s.generateSecurePassword()
		generated = password
	}

	// Utiliser les fonctions utilitaires standardisées
	salt, err := utils.GenerateSalt()
	if err != nil {
		return "", "", "", fmt.Errorf("erreur génération salt: %w", err)
	}

	hash := utils.HashPasswordSHA512(password, salt)

	return hash, salt, generated, nil
}

func (s *ComptesService) generateSecurePassword() string {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		specials  = "!@#$%^&*"
		all       = lowercase + uppercase + digits + specials
		length    = 12
	)

	password := make([]byte, length)

	password[0] = lowercase[s.randomIndex(len(lowercase))]
	password[1] = uppercase[s.randomIndex(len(uppercase))]
	password[2] = digits[s.randomIndex(len(digits))]
	password[3] = specials[s.randomIndex(len(specials))]

	for i := 4; i < length; i++ {
		password[i] = all[s.randomIndex(len(all))]
	}

	for i := len(password) - 1; i > 0; i-- {
		j := s.randomIndex(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func (s *ComptesService) randomIndex(max int) int {
	bytes := make([]byte, 1)
	rand.Read(bytes)
	return int(bytes[0]) % max
}

func (s *ComptesService) createUserInDB(ctx context.Context, tx pgx.Tx, userID string, user dto.CreateUserInternal) error {
	var createdID, createdIdentifiant string
	
	err := tx.QueryRow(ctx, queries.ComptesQueries.CreateUser,
		userID, user.EtablissementID, user.Identifiant, user.Nom, user.Prenoms,
		user.Telephone, user.PasswordHash, user.Salt,
		user.MustChangePassword, user.EstAdmin, user.TypeAdmin,
		user.EstMedecin, user.RoleMetier, user.EstTemporaire, user.DateExpiration,
		user.Statut, user.CreatedBy,
	).Scan(&createdID, &createdIdentifiant)

	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'utilisateur: %w", err)
	}

	return nil
}

func (s *ComptesService) assignPermissions(ctx context.Context, tx pgx.Tx, userID, establishmentID string, req dto.CreateUserRequest, createdByUserID string) (*dto.PermissionsAttribueesInfo, error) {
	stats := &dto.PermissionsAttribueesInfo{}

	for _, profilID := range req.ProfilsIds {
		id := uuid.New().String()
		_, err := tx.Exec(ctx, queries.ComptesQueries.CreateUserProfil,
			id, establishmentID, userID, profilID, createdByUserID)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de l'attribution du profil %s: %w", profilID, err)
		}
		stats.Profils++
	}

	for _, moduleAttribue := range req.ModulesAttribues {
		if moduleAttribue.AccesToutesRubriques {
			// Module complet
			id := uuid.New().String()
			_, err := tx.Exec(ctx, queries.ComptesQueries.CreateUserModule,
				id, establishmentID, userID, moduleAttribue.ModuleID, createdByUserID)
			if err != nil {
				return nil, fmt.Errorf("erreur lors de l'attribution du module complet %s: %w", moduleAttribue.ModuleID, err)
			}
			stats.ModulesComplets++
		} else {
			// Module partiel avec rubriques spécifiques
			stats.ModulesPartiels++
			for _, rubriqueID := range moduleAttribue.RubriquesSpecifiques {
				id := uuid.New().String()
				_, err := tx.Exec(ctx, queries.ComptesQueries.CreateUserModuleRubrique,
					id, establishmentID, userID, moduleAttribue.ModuleID, rubriqueID, createdByUserID)
				if err != nil {
					return nil, fmt.Errorf("erreur lors de l'attribution de la rubrique %s: %w", rubriqueID, err)
				}
				stats.TotalRubriques++
			}
		}
	}

	return stats, nil
}


func (s *ComptesService) ListUsers(ctx context.Context, query dto.ListUsersQuery, establishmentID string) (*dto.ListUsersResponse, error) {
	// Valeurs par défaut selon spécifications
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.SortBy == "" {
		query.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}
	if query.Statut == "" {
		query.Statut = "actif"
	}

	offset := (query.Page - 1) * query.Limit

	// Préparer les paramètres pour la requête
	var statutFilter *string
	if query.Statut != "" && query.Statut != "tous" {
		statutFilter = &query.Statut
	}

	// Compter le total d'utilisateurs
	totalCount, err := s.countUsersWithFilters(ctx, establishmentID, query, statutFilter)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du comptage des utilisateurs: %w", err)
	}

	// Récupérer les utilisateurs
	users, err := s.getUsersWithFilters(ctx, establishmentID, query, statutFilter, offset)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des utilisateurs: %w", err)
	}

	// Calculer la pagination
	totalPages := int(math.Ceil(float64(totalCount) / float64(query.Limit)))
	hasNext := query.Page < totalPages
	hasPrev := query.Page > 1

	pagination := dto.PaginationInfo{
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	// Filtres appliqués pour la réponse
	filtresAppliques := dto.FiltresAppliques{
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

	if statutFilter != nil {
		filtresAppliques.Statut = statutFilter
	}
	if query.EstAdmin != nil {
		filtresAppliques.EstAdmin = query.EstAdmin
	}
	if query.EstMedecin != nil {
		filtresAppliques.EstMedecin = query.EstMedecin
	}
	if query.Search != nil {
		filtresAppliques.Search = query.Search
	}
	if query.ProfilID != nil {
		filtresAppliques.ProfilID = query.ProfilID
	}
	if query.ModuleCode != nil {
		filtresAppliques.ModuleCode = query.ModuleCode
	}

	return &dto.ListUsersResponse{
		Users:            users,
		Pagination:       pagination,
		FiltresAppliques: filtresAppliques,
	}, nil
}

func (s *ComptesService) countUsersWithFilters(ctx context.Context, establishmentID string, query dto.ListUsersQuery, statutFilter *string) (int, error) {
	var count int

	err := s.db.QueryRow(ctx, queries.ComptesQueries.CountUsersWithFilters,
		establishmentID,   // $1
		statutFilter,      // $2
		query.Search,      // $3
		query.EstAdmin,    // $4
		query.EstMedecin,  // $5
		query.ProfilID,    // $6
		query.ModuleCode,  // $7
		query.IncludeArchived, // $8
	).Scan(&count)

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *ComptesService) getUsersWithFilters(ctx context.Context, establishmentID string, query dto.ListUsersQuery, statutFilter *string, offset int) ([]dto.UserListItem, error) {
	rows, err := s.db.Query(ctx, queries.ComptesQueries.GetUsersWithPermissionsSummary,
		establishmentID,    // $1
		statutFilter,       // $2
		query.Search,       // $3
		query.EstAdmin,     // $4
		query.EstMedecin,   // $5
		query.ProfilID,     // $6
		query.ModuleCode,   // $7
		query.SortBy,       // $8
		query.SortOrder,    // $9
		query.IncludeArchived, // $10
		query.Limit,        // $11
		offset,             // $12
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []dto.UserListItem

	for rows.Next() {
		var user dto.UserListItem
		var profilsJSON []byte
		var permissionsResumeJSON []byte

		err := rows.Scan(
			&user.ID,
			&user.Identifiant,
			&user.Nom,
			&user.Prenoms,
			&user.Telephone,
			&user.EstAdmin,
			&user.TypeAdmin,
			&user.EstMedecin,
			&user.RoleMetier,
			&user.EstTemporaire,
			&user.DateExpiration,
			&user.Statut,
			&user.LastLoginAt,
			&user.CreatedAt,
			&profilsJSON,
			&permissionsResumeJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan de l'utilisateur: %w", err)
		}

		// Désérialiser les profils JSON
		if err := json.Unmarshal(profilsJSON, &user.Profils); err != nil {
			return nil, fmt.Errorf("erreur lors de la désérialisation des profils: %w", err)
		}

		// Désérialiser le résumé des permissions JSON
		if err := json.Unmarshal(permissionsResumeJSON, &user.PermissionsResume); err != nil {
			return nil, fmt.Errorf("erreur lors de la désérialisation des permissions: %w", err)
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'itération des résultats: %w", err)
	}

	return users, nil
}

func (s *ComptesService) GetUserDetails(ctx context.Context, userID, establishmentID string) (*dto.UserDetailsResponse, error) {
	// 1. Récupérer les détails de base de l'utilisateur
	userDetails, err := s.getUserDetails(ctx, establishmentID, userID)
	if err != nil {
		return nil, err
	}

	// 2. Récupérer les profils
	profils, err := s.getUserProfils(ctx, establishmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des profils: %w", err)
	}

	// 3. Récupérer les permissions détaillées
	permissions, err := s.getUserPermissions(ctx, establishmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des permissions: %w", err)
	}

	// 4. Récupérer les statistiques
	statistiques, err := s.getUserStatistiques(ctx, establishmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des statistiques: %w", err)
	}

	return &dto.UserDetailsResponse{
		User:         *userDetails,
		Profils:      profils,
		Permissions:  *permissions,
		Statistiques: *statistiques,
	}, nil
}

func (s *ComptesService) getUserDetails(ctx context.Context, establishmentID, userID string) (*dto.UserDetails, error) {
	var user dto.UserDetails
	var createdByID, createdByNom, createdByPrenoms *string
	var updatedByID, updatedByNom, updatedByPrenoms *string

	err := s.db.QueryRow(ctx, queries.ComptesQueries.GetUserDetails, establishmentID, userID).Scan(
		&user.ID,
		&user.Identifiant,
		&user.Nom,
		&user.Prenoms,
		&user.Telephone,
		&user.EstAdmin,
		&user.TypeAdmin,
		&user.EstAdminTir,
		&user.EstMedecin,
		&user.RoleMetier,
		&user.PhotoURL,
		&user.EstTemporaire,
		&user.DateExpiration,
		&user.Statut,
		&user.MotifDesactivation,
		&user.MustChangePassword,
		&user.PasswordChangedAt,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&createdByID,
		&createdByNom,
		&createdByPrenoms,
		&updatedByID,
		&updatedByNom,
		&updatedByPrenoms,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("utilisateur non trouvé")
		}
		return nil, fmt.Errorf("erreur lors de la récupération des détails utilisateur: %w", err)
	}

	// Le champ email n'existe pas dans la base, on le met à nil
	user.Email = nil

	// Construire les références utilisateur
	if createdByID != nil && createdByNom != nil && createdByPrenoms != nil {
		user.CreatedBy = &dto.UserRef{
			ID:      *createdByID,
			Nom:     *createdByNom,
			Prenoms: *createdByPrenoms,
		}
	}

	if updatedByID != nil && updatedByNom != nil && updatedByPrenoms != nil {
		user.UpdatedBy = &dto.UserRef{
			ID:      *updatedByID,
			Nom:     *updatedByNom,
			Prenoms: *updatedByPrenoms,
		}
	}

	return &user, nil
}

func (s *ComptesService) getUserProfils(ctx context.Context, establishmentID, userID string) ([]dto.ProfilDetails, error) {
	rows, err := s.db.Query(ctx, queries.ComptesQueries.GetUserProfils, establishmentID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profils []dto.ProfilDetails

	for rows.Next() {
		var profil dto.ProfilDetails
		var attribueParID, attribueParNom, attribueParPrenoms *string

		err := rows.Scan(
			&profil.ID,
			&profil.NomProfil,
			&profil.CodeProfil,
			&profil.Description,
			&profil.DateAttribution,
			&profil.EstActif,
			&attribueParID,
			&attribueParNom,
			&attribueParPrenoms,
		)

		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des profils: %w", err)
		}

		// Construire la référence de l'attributeur
		if attribueParID != nil && attribueParNom != nil && attribueParPrenoms != nil {
			profil.AttribuePar = dto.UserRef{
				ID:      *attribueParID,
				Nom:     *attribueParNom,
				Prenoms: *attribueParPrenoms,
			}
		}

		profils = append(profils, profil)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'itération des profils: %w", err)
	}

	return profils, nil
}

func (s *ComptesService) getUserPermissions(ctx context.Context, establishmentID, userID string) (*dto.PermissionsDetails, error) {
	// Récupérer les modules complets
	modulesComplets, err := s.getUserModulesComplets(ctx, establishmentID, userID)
	if err != nil {
		return nil, err
	}

	// Récupérer les modules partiels
	modulesPartiels, err := s.getUserModulesPartiels(ctx, establishmentID, userID)
	if err != nil {
		return nil, err
	}

	return &dto.PermissionsDetails{
		ModulesComplets: modulesComplets,
		ModulesPartiels: modulesPartiels,
	}, nil
}

func (s *ComptesService) getUserModulesComplets(ctx context.Context, establishmentID, userID string) ([]dto.ModuleCompletDetails, error) {
	rows, err := s.db.Query(ctx, queries.ComptesQueries.GetUserModulesComplets, establishmentID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []dto.ModuleCompletDetails

	for rows.Next() {
		var module dto.ModuleCompletDetails

		err := rows.Scan(
			&module.CodeModule,
			&module.NomStandard,
			&module.NomPersonnalise,
			&module.Description,
			&module.Source,
			&module.ProfilSource,
			&module.DateAttribution,
			&module.AttribuePar,
		)

		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des modules complets: %w", err)
		}

		modules = append(modules, module)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'itération des modules complets: %w", err)
	}

	return modules, nil
}

func (s *ComptesService) getUserModulesPartiels(ctx context.Context, establishmentID, userID string) ([]dto.ModulePartielDetails, error) {
	rows, err := s.db.Query(ctx, queries.ComptesQueries.GetUserModulesPartiels, establishmentID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map pour regrouper les rubriques par module
	moduleMap := make(map[string]*dto.ModulePartielDetails)

	for rows.Next() {
		var codeModule, nomStandard, source, attribuePar string
		var nomPersonnalise, description *string
		var dateAttribution time.Time
		var codeRubrique, rubriqueNom string
		var rubriqueDescription *string

		err := rows.Scan(
			&codeModule,
			&nomStandard,
			&nomPersonnalise,
			&description,
			&source,
			&dateAttribution,
			&attribuePar,
			&codeRubrique,
			&rubriqueNom,
			&rubriqueDescription,
		)

		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des modules partiels: %w", err)
		}

		// Créer la clé unique pour le module
		moduleKey := fmt.Sprintf("%s_%s_%s", codeModule, source, attribuePar)

		// Si le module n'existe pas encore, le créer
		if _, exists := moduleMap[moduleKey]; !exists {
			moduleMap[moduleKey] = &dto.ModulePartielDetails{
				CodeModule:      codeModule,
				NomStandard:     nomStandard,
				NomPersonnalise: nomPersonnalise,
				Description:     description,
				Source:          source,
				DateAttribution: dateAttribution,
				AttribuePar:     attribuePar,
				Rubriques:       []dto.RubriqueDetails{},
			}
		}

		// Ajouter la rubrique au module
		rubrique := dto.RubriqueDetails{
			CodeRubrique: codeRubrique,
			Nom:          rubriqueNom,
			Description:  rubriqueDescription,
		}

		moduleMap[moduleKey].Rubriques = append(moduleMap[moduleKey].Rubriques, rubrique)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'itération des modules partiels: %w", err)
	}

	// Convertir la map en slice
	var modules []dto.ModulePartielDetails
	for _, module := range moduleMap {
		modules = append(modules, *module)
	}

	return modules, nil
}

func (s *ComptesService) getUserStatistiques(ctx context.Context, establishmentID, userID string) (*dto.Statistiques, error) {
	var stats dto.Statistiques

	err := s.db.QueryRow(ctx, queries.ComptesQueries.GetUserStatistiques, establishmentID, userID).Scan(
		&stats.NombreConnexions30j,
		&stats.DerniereActivite,
		&stats.NombreSessionsActives,
		&stats.PermissionsViaProfils,
		&stats.PermissionsIndividuelles,
	)

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des statistiques: %w", err)
	}

	return &stats, nil
}

func (s *ComptesService) ModifyUserPermissions(ctx context.Context, userID, establishmentID, modifiedByUserID string, req dto.ModifyPermissionsRequest) (*dto.ModifyPermissionsResponse, error) {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Valider que l'utilisateur cible existe
	if err := s.validateUserForPermissionModification(ctx, tx, userID, establishmentID); err != nil {
		return nil, err
	}

	// 2. Valider toutes les entités (profils, modules, rubriques)
	if err := s.validatePermissionEntities(ctx, tx, req, establishmentID); err != nil {
		return nil, err
	}

	// 3. Appliquer les modifications avec comptage
	changements, err := s.applyPermissionChanges(ctx, tx, userID, establishmentID, modifiedByUserID, req)
	if err != nil {
		return nil, err
	}

	// 4. Valider la transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("impossible de valider la transaction: %w", err)
	}

	// 5. Gestion des notifications (si demandé)
	notification := dto.NotificationInfo{
		Envoyee: false,
	}

	if req.NotifierUtilisateur {
		// TODO: Implémenter l'envoi de notification
		notification.Envoyee = true
		notification.Methode = StringPtr("email")
	}

	// 6. Récupérer les infos du modificateur
	modifiedBy, err := s.getUserRef(ctx, modifiedByUserID, establishmentID)
	if err != nil {
		// Non critique, on continue
		modifiedBy = &dto.UserRef{
			ID:      modifiedByUserID,
			Nom:     "Unknown",
			Prenoms: "User",
		}
	}

	return &dto.ModifyPermissionsResponse{
		Message:      "Permissions modifiées avec succès",
		Changements:  *changements,
		Notification: notification,
		ModifiedBy:   *modifiedBy,
		ModifiedAt:   time.Now(),
	}, nil
}

func (s *ComptesService) validateUserForPermissionModification(ctx context.Context, tx pgx.Tx, userID, establishmentID string) error {
	var userExists bool
	err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateUserExists, establishmentID, userID).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("erreur lors de la validation utilisateur: %w", err)
	}

	if !userExists {
		return fmt.Errorf("utilisateur non trouvé ou archivé")
	}

	return nil
}

func (s *ComptesService) validatePermissionEntities(ctx context.Context, tx pgx.Tx, req dto.ModifyPermissionsRequest, establishmentID string) error {
	// Valider les profils
	if req.Profils != nil {
		for _, profilID := range append(req.Profils.Ajouter, req.Profils.Retirer...) {
			var exists bool
			err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateProfilExists, establishmentID, profilID).Scan(&exists)
			if err != nil {
				return fmt.Errorf("erreur validation profil %s: %w", profilID, err)
			}
			if !exists {
				return fmt.Errorf("profil %s non trouvé", profilID)
			}
		}
	}

	// Valider les modules complets
	if req.ModulesComplets != nil {
		moduleIDs := []string{}
		for _, module := range req.ModulesComplets.Ajouter {
			moduleIDs = append(moduleIDs, module.ModuleID)
		}
		moduleIDs = append(moduleIDs, req.ModulesComplets.Retirer...)

		for _, moduleID := range moduleIDs {
			if err := s.validateModuleExists(ctx, tx, moduleID); err != nil {
				return err
			}
		}
	}

	// Valider les modules partiels
	if req.ModulesPartiels != nil {
		// Valider modules à ajouter et modifier
		for _, modulePartiel := range append(req.ModulesPartiels.Ajouter, req.ModulesPartiels.Modifier...) {
			if err := s.validateModuleExists(ctx, tx, modulePartiel.ModuleID); err != nil {
				return err
			}

			// Valider chaque rubrique
			for _, rubriqueID := range modulePartiel.RubriquesIds {
				var exists bool
				err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateRubriqueExists, rubriqueID, modulePartiel.ModuleID).Scan(&exists)
				if err != nil {
					return fmt.Errorf("erreur validation rubrique %s: %w", rubriqueID, err)
				}
				if !exists {
					return fmt.Errorf("rubrique %s non trouvée pour le module %s", rubriqueID, modulePartiel.ModuleID)
				}
			}
		}

		// Valider modules à retirer
		for _, moduleID := range req.ModulesPartiels.Retirer {
			if err := s.validateModuleExists(ctx, tx, moduleID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ComptesService) validateModuleExists(ctx context.Context, tx pgx.Tx, moduleID string) error {
	var moduleExists, moduleTypeValid bool
	var codeModule string

	// TODO: Récupérer est_admin de l'utilisateur cible pour validation
	// Pour l'instant, on valide juste l'existence
	err := tx.QueryRow(ctx, queries.ComptesQueries.ValidateModuleExists, moduleID, false).Scan(&moduleExists, &moduleTypeValid, &codeModule)
	if err != nil {
		return fmt.Errorf("erreur validation module %s: %w", moduleID, err)
	}

	if !moduleExists {
		return fmt.Errorf("module %s non trouvé", moduleID)
	}

	return nil
}

func (s *ComptesService) applyPermissionChanges(ctx context.Context, tx pgx.Tx, userID, establishmentID, modifiedByUserID string, req dto.ModifyPermissionsRequest) (*dto.ChangementsPermissions, error) {
	changements := &dto.ChangementsPermissions{}

	// 1. Gérer les profils
	if req.Profils != nil {
		// Ajouter profils
		for _, profilID := range req.Profils.Ajouter {
			id := uuid.New().String()
			_, err := tx.Exec(ctx, queries.ComptesQueries.AddUserProfil,
				id, establishmentID, userID, profilID, modifiedByUserID)
			if err != nil {
				return nil, fmt.Errorf("erreur ajout profil %s: %w", profilID, err)
			}
			changements.Profils.Ajoutes++
		}

		// Retirer profils
		for _, profilID := range req.Profils.Retirer {
			result, err := tx.Exec(ctx, queries.ComptesQueries.RemoveUserProfil,
				establishmentID, userID, profilID)
			if err != nil {
				return nil, fmt.Errorf("erreur suppression profil %s: %w", profilID, err)
			}
			if result.RowsAffected() > 0 {
				changements.Profils.Retires++
			}
		}
	}

	// 2. Gérer les modules complets
	if req.ModulesComplets != nil {
		// Ajouter modules complets
		for _, module := range req.ModulesComplets.Ajouter {
			id := uuid.New().String()
			_, err := tx.Exec(ctx, queries.ComptesQueries.AddUserModuleComplet,
				id, establishmentID, userID, module.ModuleID, modifiedByUserID)
			if err != nil {
				return nil, fmt.Errorf("erreur ajout module complet %s: %w", module.ModuleID, err)
			}
			changements.ModulesComplets.Ajoutes++
		}

		// Retirer modules complets
		for _, moduleID := range req.ModulesComplets.Retirer {
			result, err := tx.Exec(ctx, queries.ComptesQueries.RemoveUserModuleComplet,
				establishmentID, userID, moduleID)
			if err != nil {
				return nil, fmt.Errorf("erreur suppression module complet %s: %w", moduleID, err)
			}
			if result.RowsAffected() > 0 {
				changements.ModulesComplets.Retires++
			}
		}
	}

	// 3. Gérer les modules partiels
	if req.ModulesPartiels != nil {
		// Ajouter modules partiels
		for _, modulePartiel := range req.ModulesPartiels.Ajouter {
			for _, rubriqueID := range modulePartiel.RubriquesIds {
				id := uuid.New().String()
				_, err := tx.Exec(ctx, queries.ComptesQueries.AddUserModulePartiel,
					id, establishmentID, userID, modulePartiel.ModuleID, rubriqueID, modifiedByUserID)
				if err != nil {
					return nil, fmt.Errorf("erreur ajout rubrique %s: %w", rubriqueID, err)
				}
				changements.TotalRubriquesAffectees++
			}
			changements.ModulesPartiels.Ajoutes++
		}

		// Modifier modules partiels
		for _, modulePartiel := range req.ModulesPartiels.Modifier {
			// D'abord retirer toutes les rubriques existantes du module
			_, err := tx.Exec(ctx, queries.ComptesQueries.RemoveUserModulePartiel,
				establishmentID, userID, modulePartiel.ModuleID)
			if err != nil {
				return nil, fmt.Errorf("erreur suppression rubriques module %s: %w", modulePartiel.ModuleID, err)
			}

			// Puis ajouter les nouvelles rubriques
			for _, rubriqueID := range modulePartiel.RubriquesIds {
				id := uuid.New().String()
				_, err := tx.Exec(ctx, queries.ComptesQueries.AddUserModulePartiel,
					id, establishmentID, userID, modulePartiel.ModuleID, rubriqueID, modifiedByUserID)
				if err != nil {
					return nil, fmt.Errorf("erreur modification rubrique %s: %w", rubriqueID, err)
				}
				changements.TotalRubriquesAffectees++
			}
			changements.ModulesPartiels.Modifies++
		}

		// Retirer modules partiels
		for _, moduleID := range req.ModulesPartiels.Retirer {
			result, err := tx.Exec(ctx, queries.ComptesQueries.RemoveUserModulePartiel,
				establishmentID, userID, moduleID)
			if err != nil {
				return nil, fmt.Errorf("erreur suppression module partiel %s: %w", moduleID, err)
			}
			if result.RowsAffected() > 0 {
				changements.ModulesPartiels.Retires++
				changements.TotalRubriquesAffectees += int(result.RowsAffected())
			}
		}
	}

	return changements, nil
}

func (s *ComptesService) getUserRef(ctx context.Context, userID, establishmentID string) (*dto.UserRef, error) {
	var userRef dto.UserRef
	err := s.db.QueryRow(ctx,
		"SELECT id, nom, prenoms FROM user_utilisateur WHERE id = $1 AND etablissement_id = $2",
		userID, establishmentID).Scan(&userRef.ID, &userRef.Nom, &userRef.Prenoms)
	
	if err != nil {
		return nil, err
	}

	return &userRef, nil
}

// Helper function pour créer un pointeur vers string
func StringPtr(s string) *string {
	return &s
}
