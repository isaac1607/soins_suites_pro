package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"soins-suite-core/internal/app/config"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/system/dto"
	"soins-suite-core/internal/modules/system/queries"
)

type SystemService struct {
	db     *postgres.Client
	config *config.Config
}

func NewSystemService(db *postgres.Client, config *config.Config) *SystemService {
	return &SystemService{
		db:     db,
		config: config,
	}
}

// ServiceError représente une erreur métier avec type et détails
type ServiceError struct {
	Type    string                 `json:"type"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

func (e *ServiceError) Error() string {
	return e.Message
}

// GetSystemInfo récupère toutes les informations système pour l'établissement
func (s *SystemService) GetSystemInfo(ctx context.Context, establishmentID string) (*dto.SystemInfoResponse, error) {
	response := &dto.SystemInfoResponse{}

	// Récupération des informations établissement
	establishment, err := s.getEstablishmentInfo(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération établissement: %w", err)
	}
	response.Etablissement = *establishment

	// Récupération des informations licence (peut être null/vide)
	license, err := s.getLicenseInfo(ctx, establishmentID)
	if err != nil {
		// Si pas de licence trouvée, on continue avec une licence vide
		if errors.Is(err, sql.ErrNoRows) {
			// Licence par défaut pour établissement sans licence
			license = &dto.LicenseInfoDTO{
				ID:               "",
				Type:             "",
				ModeDeploiement:  "",
				Statut:           "non_configuree",
				ModulesAutorises: []string{},
			}
		} else {
			return nil, fmt.Errorf("erreur récupération licence: %w", err)
		}
	}
	response.Licence = *license

	// Récupération des modules autorisés
	modules, err := s.getAuthorizedModules(ctx, establishmentID, license.ModulesAutorises)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération modules: %w", err)
	}
	response.ModulesDisponibles = modules

	// Récupération de la configuration
	config, err := s.getSystemConfiguration(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération configuration: %w", err)
	}
	response.Configuration = *config

	return response, nil
}

// getEstablishmentInfo récupère les informations de l'établissement
func (s *SystemService) getEstablishmentInfo(ctx context.Context, establishmentID string) (*dto.EstablishmentInfoDTO, error) {
	var establishment dto.EstablishmentInfoDTO

	err := s.db.QueryRow(ctx, queries.SystemQueries.GetEstablishmentInfo, establishmentID).Scan(
		&establishment.ID,
		&establishment.Code,
		&establishment.Nom,
		&establishment.NomCourt,
		&establishment.Ville,
		&establishment.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("établissement non trouvé: %s", establishmentID)
		}
		return nil, fmt.Errorf("erreur base de données: %w", err)
	}

	return &establishment, nil
}

// getLicenseInfo récupère les informations de licence
func (s *SystemService) getLicenseInfo(ctx context.Context, establishmentID string) (*dto.LicenseInfoDTO, error) {
	var license dto.LicenseInfoDTO
	var modulesJSON []byte
	var dateActivation time.Time

	err := s.db.QueryRow(ctx, queries.SystemQueries.GetLicenseInfo, establishmentID).Scan(
		&license.ID,
		&license.Type,
		&license.ModeDeploiement,
		&license.Statut,
		&dateActivation,
		&license.DateExpiration,
		&modulesJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Pas de licence trouvée - Retourner sql.ErrNoRows pour gestion dans GetSystemInfo
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("erreur base de données: %w", err)
	}

	// Assigner la date d'activation
	license.DateActivation = &dateActivation

	// Désérialisation des modules autorisés
	// Le format peut être soit un tableau direct, soit un objet avec une clé "modules"
	var modulesData interface{}
	err = json.Unmarshal(modulesJSON, &modulesData)
	if err != nil {
		return nil, fmt.Errorf("erreur désérialisation modules autorisés: %w", err)
	}

	// Vérifier le format des données
	switch v := modulesData.(type) {
	case []interface{}:
		// Format tableau direct: ["ACCUEIL", "CAISSE"]
		license.ModulesAutorises = make([]string, len(v))
		for i, module := range v {
			if str, ok := module.(string); ok {
				license.ModulesAutorises[i] = str
			}
		}
	case map[string]interface{}:
		// Format objet: {"modules": ["ACCUEIL", "CAISSE"]}
		if modules, exists := v["modules"]; exists {
			if moduleArray, ok := modules.([]interface{}); ok {
				license.ModulesAutorises = make([]string, len(moduleArray))
				for i, module := range moduleArray {
					if str, ok := module.(string); ok {
						license.ModulesAutorises[i] = str
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("format modules autorisés invalide")
	}

	// Calcul des jours restants si expiration définie
	if license.DateExpiration != nil {
		now := time.Now()
		if license.DateExpiration.After(now) {
			jours := int(license.DateExpiration.Sub(now).Hours() / 24)
			license.JoursRestants = &jours
		} else {
			jours := 0
			license.JoursRestants = &jours
		}
	}

	return &license, nil
}

// getAuthorizedModules récupère la liste des modules avec leur statut d'autorisation
func (s *SystemService) getAuthorizedModules(ctx context.Context, establishmentID string, modulesAutorises []string) ([]dto.ModuleInfoDTO, error) {
	rows, err := s.db.Query(ctx, queries.SystemQueries.GetAuthorizedModules)
	if err != nil {
		return nil, fmt.Errorf("erreur requête modules: %w", err)
	}
	defer rows.Close()

	var modules []dto.ModuleInfoDTO
	for rows.Next() {
		var module dto.ModuleInfoDTO
		err := rows.Scan(
			&module.CodeModule,
			&module.NomStandard,
			&module.EstMedical,
			&module.PeutPrendreTicket,
			&module.EstActif,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur scan module: %w", err)
		}
		modules = append(modules, module)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur parcours modules: %w", err)
	}

	return modules, nil
}

// getSystemConfiguration récupère la configuration système
func (s *SystemService) getSystemConfiguration(ctx context.Context, establishmentID string) (*dto.SystemConfigurationDTO, error) {
	var config dto.SystemConfigurationDTO

	err := s.db.QueryRow(ctx, queries.SystemQueries.GetSystemConfiguration, establishmentID).Scan(
		&config.DureeValiditeTicketJours,
		&config.NbSouchesParCaisse,
		&config.GardeActive,
		&config.GardeHeureDebut,
		&config.GardeHeureFin,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration non trouvée pour l'établissement: %s", establishmentID)
		}
		return nil, fmt.Errorf("erreur base de données: %w", err)
	}

	return &config, nil
}

// GenerateAlertes génère les alertes système appropriées
func (s *SystemService) GenerateAlertes(license *dto.LicenseInfoDTO) []dto.AlerteDTO {
	var alertes []dto.AlerteDTO

	// Alerte licence non configurée
	if license.Statut == "non_configuree" {
		alertes = append(alertes, dto.AlerteDTO{
			Type:    "error",
			Code:    "LICENSE_NOT_CONFIGURED",
			Message: "Licence non configurée pour cet établissement",
			Details: map[string]interface{}{
				"required_action": "Configurer une licence depuis le back-office",
			},
		})
		return alertes // Pas d'autres alertes si licence non configurée
	}

	// Alerte expiration proche (< 30 jours)
	if license.JoursRestants != nil && *license.JoursRestants < 30 && *license.JoursRestants > 0 {
		alertes = append(alertes, dto.AlerteDTO{
			Type:    "warning",
			Code:    "LICENSE_EXPIRING_SOON",
			Message: fmt.Sprintf("Licence expire dans %d jours", *license.JoursRestants),
			Details: map[string]interface{}{
				"date_expiration": license.DateExpiration,
				"jours_restants":  *license.JoursRestants,
			},
		})
	}

	// Alerte licence expirée
	if license.JoursRestants != nil && *license.JoursRestants <= 0 {
		alertes = append(alertes, dto.AlerteDTO{
			Type:    "error",
			Code:    "LICENSE_EXPIRED",
			Message: "Licence expirée",
			Details: map[string]interface{}{
				"date_expiration": license.DateExpiration,
				"required_action": "Renouveler la licence",
			},
		})
	}

	return alertes
}

// GetAuthorizedModulesOnly récupère uniquement les modules autorisés par la licence
func (s *SystemService) GetAuthorizedModulesOnly(ctx context.Context, establishmentID string) (*dto.AuthorizedModulesResponse, error) {
	rows, err := s.db.Query(ctx, queries.SystemQueries.GetAuthorizedModulesOnly, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur requête modules autorisés: %w", err)
	}
	defer rows.Close()

	var modules []dto.AuthorizedModuleDTO
	for rows.Next() {
		var module dto.AuthorizedModuleDTO
		err := rows.Scan(
			&module.CodeModule,
			&module.NomStandard,
			&module.EstMedical,
			&module.PeutPrendreTicket,
			&module.EstModuleBackOffice,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur scan module autorisé: %w", err)
		}
		modules = append(modules, module)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur parcours modules autorisés: %w", err)
	}

	return &dto.AuthorizedModulesResponse{
		ModulesAutorises: modules,
	}, nil
}

// ActivateOfflineSystem déclenche la synchronisation avec le serveur central
func (s *SystemService) ActivateOfflineSystem(ctx context.Context, req dto.ActivateOfflineRequest) (*dto.ActivateOfflineResponse, error) {
	// Récupérer l'APP_INSTANCE depuis la config
	appInstance := s.config.System.AppInstance
	if appInstance == "" {
		return nil, &ServiceError{
			Type:    "configuration",
			Code:    "APP_INSTANCE_NOT_CONFIGURED",
			Message: "APP_INSTANCE non configuré",
			Details: map[string]interface{}{
				"required_env": "APP_INSTANCE_ID",
			},
		}
	}

	// Construire l'URL du serveur central
	centralServerURL := s.config.System.CentralServerURL
	if centralServerURL == "" {
		return nil, &ServiceError{
			Type:    "configuration",
			Code:    "CENTRAL_SERVER_NOT_CONFIGURED",
			Message: "Serveur central non configuré",
			Details: map[string]interface{}{
				"required_env": "CENTRAL_SERVER_URL",
			},
		}
	}

	// Préparer la requête pour le serveur central
	syncRequest := dto.SyncOfflineRequest{
		AppInstance: appInstance,
	}

	jsonData, err := json.Marshal(syncRequest)
	if err != nil {
		return nil, &ServiceError{
			Type:    "internal",
			Code:    "JSON_MARSHAL_ERROR",
			Message: "Erreur sérialisation requête",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Effectuer la requête HTTP vers le serveur central
	url := fmt.Sprintf("%s/api/v1/system/offline/synchronised", centralServerURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, &ServiceError{
			Type:    "http",
			Code:    "HTTP_REQUEST_ERROR",
			Message: "Erreur création requête HTTP",
			Details: map[string]interface{}{
				"url":   url,
				"error": err.Error(),
			},
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Establishment-Code", req.CodeEtablissement)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &ServiceError{
			Type:    "network",
			Code:    "NETWORK_ERROR",
			Message: "Impossible de contacter le serveur central",
			Details: map[string]interface{}{
				"url":   url,
				"error": err.Error(),
			},
		}
	}
	defer resp.Body.Close()

	// Vérifier le statut de la réponse
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)

		if resp.StatusCode == http.StatusNotFound {
			return nil, &ServiceError{
				Type:    "not_found",
				Code:    "ESTABLISHMENT_OR_INSTANCE_NOT_FOUND",
				Message: "Établissement ou instance non trouvé(e)",
				Details: map[string]interface{}{
					"app_instance":         appInstance,
					"code_etablissement":   req.CodeEtablissement,
					"central_server_error": errorResp,
				},
			}
		}

		if resp.StatusCode == http.StatusConflict {
			return nil, &ServiceError{
				Type:    "conflict",
				Code:    "ESTABLISHMENT_ALREADY_SYNCHRONIZED",
				Message: "Établissement déjà synchronisé",
				Details: map[string]interface{}{
					"app_instance":         appInstance,
					"code_etablissement":   req.CodeEtablissement,
					"central_server_error": errorResp,
				},
			}
		}

		return nil, &ServiceError{
			Type:    "central_server",
			Code:    "CENTRAL_SERVER_ERROR",
			Message: fmt.Sprintf("Erreur serveur central (status %d)", resp.StatusCode),
			Details: map[string]interface{}{
				"status_code":          resp.StatusCode,
				"central_server_error": errorResp,
			},
		}
	}

	// Décoder la réponse du serveur central
	var syncResponse struct {
		Success bool                    `json:"success"`
		Data    dto.SyncOfflineResponse `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&syncResponse); err != nil {
		return nil, &ServiceError{
			Type:    "parsing",
			Code:    "RESPONSE_PARSING_ERROR",
			Message: "Erreur analyse réponse serveur central",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	if !syncResponse.Success {
		return nil, &ServiceError{
			Type:    "central_server",
			Code:    "SYNC_FAILED",
			Message: "Synchronisation échouée",
			Details: map[string]interface{}{
				"server_response": syncResponse,
			},
		}
	}

	// Insérer les données dans la base locale
	err = s.insertSyncData(ctx, &syncResponse.Data)
	if err != nil {
		return nil, &ServiceError{
			Type:    "database",
			Code:    "DATA_INSERTION_ERROR",
			Message: "Erreur insertion données synchronisées",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	return &dto.ActivateOfflineResponse{
		SynchronisationEffectuee: true,
		EtablissementExiste:      true,
		Message:                  "Synchronisation des données depuis le serveur central effectuée",
	}, nil
}

// SynchronizeOfflineData endpoint pour serveur central
func (s *SystemService) SynchronizeOfflineData(ctx context.Context, req dto.SyncOfflineRequest, establishmentCode string) (*dto.SyncOfflineResponse, error) {
	// Vérifier que l'app_instance existe et correspond au code établissement
	var establishmentID string
	err := s.db.QueryRow(ctx, queries.SystemQueries.GetEstablishmentByAppInstance, req.AppInstance, establishmentCode).Scan(
		&establishmentID, nil, nil, nil, nil,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &ServiceError{
				Type:    "not_found",
				Code:    "ESTABLISHMENT_OR_INSTANCE_NOT_FOUND",
				Message: "Établissement ou instance non trouvé(e)",
				Details: map[string]interface{}{
					"app_instance":       req.AppInstance,
					"code_etablissement": establishmentCode,
				},
			}
		}
		return nil, fmt.Errorf("erreur vérification établissement: %w", err)
	}

	// Vérifier le statut de synchronisation
	var syncComplete bool
	var syncDate sql.NullTime
	err = s.db.QueryRow(ctx, queries.SystemQueries.CheckSyncStatus, establishmentID).Scan(
		&syncComplete, &syncDate,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("erreur vérification sync status: %w", err)
	}

	if syncComplete {
		return nil, &ServiceError{
			Type:    "conflict",
			Code:    "ESTABLISHMENT_ALREADY_SYNCHRONIZED",
			Message: "Établissement déjà synchronisé",
			Details: map[string]interface{}{
				"app_instance":       req.AppInstance,
				"code_etablissement": establishmentCode,
				"date_derniere_sync": syncDate.Time.Format(time.RFC3339),
			},
		}
	}

	// Récupérer les données complètes de l'établissement
	response, err := s.buildCompleteEstablishmentData(ctx, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur construction données établissement: %w", err)
	}

	// Mettre à jour le statut de synchronisation
	err = s.db.Exec(ctx, queries.SystemQueries.UpdateSyncStatus, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("erreur mise à jour sync status: %w", err)
	}

	return response, nil
}

// insertSyncData insère toutes les données reçues lors de la synchronisation
func (s *SystemService) insertSyncData(ctx context.Context, data *dto.SyncOfflineResponse) error {
	// Commencer une transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("erreur début transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insérer l'établissement
	_, err = tx.Exec(ctx, queries.SystemQueries.InsertEstablishment,
		data.Etablissement.ID, data.Etablissement.AppInstance, data.Etablissement.EtablissementCode,
		data.Etablissement.Nom, data.Etablissement.NomCourt, data.Etablissement.AdresseComplete,
		data.Etablissement.Ville, data.Etablissement.Commune, data.Etablissement.TelephonePrincipal,
		data.Etablissement.Email, data.Etablissement.DureeValiditeTicketJours, data.Etablissement.NbSouchesParCaisse,
		data.Etablissement.GardeHeureDebut, data.Etablissement.GardeHeureFin, data.Etablissement.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur insertion établissement: %w", err)
	}

	// Insérer la licence
	modulesJSON, _ := json.Marshal(data.Licence.ModulesAutorises)
	_, err = tx.Exec(ctx, queries.SystemQueries.InsertLicense,
		data.Licence.ID, data.Licence.EtablissementID, data.Licence.ModeDeploiement,
		data.Licence.TypeLicence, string(modulesJSON), data.Licence.DateActivation,
		data.Licence.DateExpiration, data.Licence.Statut, data.Licence.SyncInitialComplete,
		data.Licence.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur insertion licence: %w", err)
	}

	// Insérer les modules et leurs rubriques
	for _, module := range data.Modules {
		// Insérer le module
		var moduleID string
		err = tx.QueryRow(ctx, queries.SystemQueries.InsertModule,
			module.ID, module.CodeModule, module.NomStandard, module.Description,
			module.PeutPrendreTicket, module.EstMedical, module.EstObligatoire,
			module.EstActif, module.EstModuleBackOffice, time.Now(),
		).Scan(&moduleID)
		if err != nil {
			return fmt.Errorf("erreur insertion module %s: %w", module.CodeModule, err)
		}

		// Insérer les rubriques du module
		for _, rubrique := range module.Rubriques {
			_, err = tx.Exec(ctx, queries.SystemQueries.InsertRubrique,
				rubrique.ID, moduleID, rubrique.CodeRubrique, rubrique.Nom,
				rubrique.Description, rubrique.OrdreAffichage, rubrique.EstObligatoire,
				rubrique.EstActif, time.Now(),
			)
			if err != nil {
				return fmt.Errorf("erreur insertion rubrique %s: %w", rubrique.CodeRubrique, err)
			}
		}
	}

	// Insérer le super admin
	_, err = tx.Exec(ctx, queries.SystemQueries.InsertSuperAdmin,
		data.SuperAdmin.ID, data.SuperAdmin.EtablissementID, data.SuperAdmin.Identifiant,
		data.SuperAdmin.Nom, data.SuperAdmin.Prenoms, data.SuperAdmin.Telephone,
		data.SuperAdmin.PasswordHash, data.SuperAdmin.Salt, data.SuperAdmin.MustChangePassword,
		data.SuperAdmin.EstAdmin, data.SuperAdmin.TypeAdmin, data.SuperAdmin.EstAdminTir,
		data.SuperAdmin.EstTemporaire, data.SuperAdmin.Statut, data.SuperAdmin.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur insertion super admin: %w", err)
	}

	return tx.Commit(ctx)
}

// buildCompleteEstablishmentData construit la réponse complète pour synchronisation
func (s *SystemService) buildCompleteEstablishmentData(ctx context.Context, establishmentID string) (*dto.SyncOfflineResponse, error) {
	response := &dto.SyncOfflineResponse{}

	// Récupérer les données établissement + licence
	var establishment dto.EstablissementSyncDTO
	var licence dto.LicenceSyncDTO
	var modulesJSON []byte

	err := s.db.QueryRow(ctx, queries.SystemQueries.GetCompleteEstablishmentData, establishmentID).Scan(
		&establishment.ID, &establishment.AppInstance, &establishment.EtablissementCode,
		&establishment.Nom, &establishment.NomCourt, &establishment.AdresseComplete,
		&establishment.Ville, &establishment.Commune, &establishment.TelephonePrincipal,
		&establishment.Email, &establishment.DureeValiditeTicketJours, &establishment.NbSouchesParCaisse,
		&establishment.GardeHeureDebut, &establishment.GardeHeureFin, &establishment.CreatedAt,
		&licence.ID, &licence.ModeDeploiement, &licence.TypeLicence, &modulesJSON,
		&licence.DateActivation, &licence.DateExpiration, &licence.Statut,
		&licence.SyncInitialComplete, &licence.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération données établissement: %w", err)
	}

	licence.EtablissementID = establishment.ID

	// Désérialiser les modules autorisés
	json.Unmarshal(modulesJSON, &licence.ModulesAutorises)

	response.Etablissement = establishment
	response.Licence = licence

	// Récupérer tous les modules avec rubriques
	modules, err := s.getCompleteModulesData(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération modules: %w", err)
	}
	response.Modules = modules

	// Générer le super admin
	superAdmin, err := s.generateSuperAdmin(ctx, establishment.ID)
	if err != nil {
		return nil, fmt.Errorf("erreur génération super admin: %w", err)
	}
	response.SuperAdmin = *superAdmin

	// Métadonnées
	response.SyncMetadata = dto.SyncMetadataDTO{
		SyncTimestamp:  time.Now().Format(time.RFC3339),
		TotalModules:   len(modules),
		TotalRubriques: s.countRubriques(modules),
		VersionDonnees: "1.0",
	}

	return response, nil
}

// getCompleteModulesData récupère tous les modules avec leurs rubriques
func (s *SystemService) getCompleteModulesData(ctx context.Context) ([]dto.ModuleSyncDTO, error) {
	rows, err := s.db.Query(ctx, queries.SystemQueries.GetCompleteModulesData)
	if err != nil {
		return nil, fmt.Errorf("erreur requête modules complets: %w", err)
	}
	defer rows.Close()

	modulesMap := make(map[string]*dto.ModuleSyncDTO)

	for rows.Next() {
		var moduleID, codeModule, nomStandard, description string
		var numeroModule int
		var estMedical, estObligatoire, estActif, estModuleBackOffice, peutPrendreTicket bool
		var rubriqueID, codeRubrique, rubriqueNom, rubriqueDescription sql.NullString
		var ordreAffichage sql.NullInt32
		var rubriqueEstObligatoire, rubriqueEstActif sql.NullBool

		err := rows.Scan(
			&moduleID, &numeroModule, &codeModule, &nomStandard, &description,
			&estMedical, &estObligatoire, &estActif, &estModuleBackOffice, &peutPrendreTicket,
			&rubriqueID, &codeRubrique, &rubriqueNom, &rubriqueDescription,
			&ordreAffichage, &rubriqueEstObligatoire, &rubriqueEstActif,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur scan module complet: %w", err)
		}

		// Créer le module s'il n'existe pas
		if _, exists := modulesMap[codeModule]; !exists {
			modulesMap[codeModule] = &dto.ModuleSyncDTO{
				ID:                  moduleID,
				NumeroModule:        numeroModule,
				CodeModule:          codeModule,
				NomStandard:         nomStandard,
				Description:         description,
				EstMedical:          estMedical,
				EstObligatoire:      estObligatoire,
				EstActif:            estActif,
				EstModuleBackOffice: estModuleBackOffice,
				PeutPrendreTicket:   peutPrendreTicket,
				Rubriques:           []dto.RubriqueDTO{},
			}
		}

		// Ajouter la rubrique si elle existe
		if rubriqueID.Valid {
			rubrique := dto.RubriqueDTO{
				ID:             rubriqueID.String,
				CodeRubrique:   codeRubrique.String,
				Nom:            rubriqueNom.String,
				Description:    rubriqueDescription.String,
				OrdreAffichage: int(ordreAffichage.Int32),
				EstObligatoire: rubriqueEstObligatoire.Bool,
				EstActif:       rubriqueEstActif.Bool,
			}
			modulesMap[codeModule].Rubriques = append(modulesMap[codeModule].Rubriques, rubrique)
		}
	}

	// Convertir en slice
	var modules []dto.ModuleSyncDTO
	for _, module := range modulesMap {
		modules = append(modules, *module)
	}

	return modules, nil
}

// generateSuperAdmin génère les données du super admin TIR
func (s *SystemService) generateSuperAdmin(ctx context.Context, establishmentID string) (*dto.SuperAdminSyncDTO, error) {
	// Générer un salt aléatoire
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("erreur génération salt: %w", err)
	}
	saltString := fmt.Sprintf("%x", salt)

	// Password par défaut depuis la config
	defaultPassword := s.config.System.AdminTIRPassword
	if defaultPassword == "" {
		defaultPassword = "AdminTIR2024!"
	}

	// Hacher le password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword+saltString), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("erreur hachage password: %w", err)
	}

	return &dto.SuperAdminSyncDTO{
		ID:                 fmt.Sprintf("admin-tir-%s", establishmentID),
		EtablissementID:    establishmentID,
		Identifiant:        "admin.tir",
		Nom:                "Admin",
		Prenoms:            "TIR",
		Telephone:          "+225 00 00 00 00",
		PasswordHash:       string(hashedPassword),
		Salt:               saltString,
		MustChangePassword: true,
		EstAdmin:           true,
		TypeAdmin:          "super_admin",
		EstAdminTir:        true,
		EstTemporaire:      false,
		Statut:             "actif",
		CreatedAt:          time.Now().Format(time.RFC3339),
	}, nil
}

// countRubriques compte le nombre total de rubriques
func (s *SystemService) countRubriques(modules []dto.ModuleSyncDTO) int {
	total := 0
	for _, module := range modules {
		total += len(module.Rubriques)
	}
	return total
}
