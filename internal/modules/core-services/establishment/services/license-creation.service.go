package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/establishment/dto"
	"soins-suite-core/internal/modules/core-services/establishment/queries"
)

// LicenseCreationService - Service métier pour la création de licences
type LicenseCreationService struct {
	db *postgres.Client
}

// NewLicenseCreationService - Constructeur du service de création de licences
func NewLicenseCreationService(db *postgres.Client) *LicenseCreationService {
	return &LicenseCreationService{
		db: db,
	}
}


// CreateLicense - Crée une nouvelle licence pour un établissement
func (s *LicenseCreationService) CreateLicense(
	ctx context.Context, 
	req dto.CreateLicenseRequest, 
	createdBy uuid.UUID,
	userIP *net.IP,
) (*dto.LicenseCreationResult, error) {
	
	// 1. Vérifier qu'aucune licence active n'existe pour cet établissement
	existingLicense, err := s.checkActiveLicense(ctx, req.EtablissementID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la vérification de licence existante: %w", err)
	}
	
	if existingLicense != nil {
		return nil, &ServiceError{
			Type:    "conflict",
			Message: "Une licence active existe déjà pour cet établissement",
			Details: map[string]interface{}{
				"existing_license_id": existingLicense.ID,
				"statut": existingLicense.Statut,
				"date_activation": existingLicense.DateActivation,
			},
		}
	}

	// 2. Récupérer les modules front-office disponibles
	availableModules, err := s.getAvailableFrontOfficeModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des modules disponibles: %w", err)
	}

	if len(availableModules) == 0 {
		return nil, &ServiceError{
			Type:    "validation",
			Message: "Aucun module front-office n'est disponible pour créer une licence",
			Details: map[string]interface{}{
				"available_modules_count": 0,
			},
		}
	}

	// 3. Déterminer les modules autorisés et la configuration selon les règles métier
	var modulesAutorises []dto.ModuleInfo
	var dateExpiration *time.Time
	var typeLicence string

	if req.ModeDeploiement == "local" {
		// Mode local: type premium, tous les modules, licence à vie
		typeLicence = "premium"
		modulesAutorises = make([]dto.ModuleInfo, len(availableModules))
		for i, module := range availableModules {
			modulesAutorises[i] = dto.ModuleInfo{
				ID:   module.ID,
				Code: module.CodeModule,
			}
		}
		dateExpiration = nil // Pas d'expiration pour premium local
	} else {
		// Mode online: type selon la requête, date d'expiration obligatoire
		typeLicence = req.TypeLicence
		
		// Calculer la date d'expiration si non fournie
		if req.DateExpiration == nil {
			expirationDate := s.calculateExpirationDate(typeLicence)
			dateExpiration = &expirationDate
		} else {
			dateExpiration = req.DateExpiration
		}

		// Déterminer les modules autorisés
		if req.ModulesAutorises != nil && len(*req.ModulesAutorises) > 0 {
			// Modules spécifiés dans la requête
			modulesAutorises, err = s.validateAndGetRequestedModules(ctx, *req.ModulesAutorises, availableModules)
			if err != nil {
				return nil, fmt.Errorf("erreur lors de la validation des modules demandés: %w", err)
			}
		} else {
			// Tous les modules front-office par défaut
			modulesAutorises = make([]dto.ModuleInfo, len(availableModules))
			for i, module := range availableModules {
				modulesAutorises[i] = dto.ModuleInfo{
					ID:   module.ID,
					Code: module.CodeModule,
				}
			}
		}
	}

	// 4. Créer la licence en base de données
	licenseResponse, err := s.createLicenseInDatabase(
		ctx,
		req.EtablissementID,
		req.ModeDeploiement,
		typeLicence,
		modulesAutorises,
		dateExpiration,
		createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la licence: %w", err)
	}

	// 5. Historiser la création de la licence
	err = s.createLicenseHistory(
		ctx,
		req.EtablissementID,
		licenseResponse.ID,
		"activation_initiale",
		nil, // statut_precedent
		"actif",
		"Création initiale de la licence",
		createdBy,
		userIP,
	)
	if err != nil {
		// Log l'erreur mais ne fait pas échouer la création
		fmt.Printf("[WARNING] Échec de l'historisation de la licence %s: %v\n", licenseResponse.ID, err)
	}

	// 6. Construire le message de succès
	var message string
	if req.ModeDeploiement == "local" {
		message = fmt.Sprintf("Licence premium locale créée avec succès pour l'établissement. %d modules autorisés, licence à vie.", len(modulesAutorises))
	} else {
		message = fmt.Sprintf("Licence %s créée avec succès pour l'établissement. %d modules autorisés, expiration le %s.", 
			typeLicence, len(modulesAutorises), dateExpiration.Format("02/01/2006"))
	}

	return &dto.LicenseCreationResult{
		License: licenseResponse,
		Message: message,
	}, nil
}

// checkActiveLicense - Vérifie s'il existe une licence active pour l'établissement
func (s *LicenseCreationService) checkActiveLicense(ctx context.Context, etablissementID uuid.UUID) (*dto.LicenseResponse, error) {
	var license dto.LicenseResponse
	var modulesJSON []byte

	err := s.db.QueryRow(ctx, queries.LicenseQueries.CheckActiveLicense, etablissementID).Scan(
		&license.ID,
		&license.EtablissementID,
		&license.ModeDeploiement,
		&license.TypeLicence,
		&license.Statut,
		&license.DateActivation,
		&license.DateExpiration,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // Pas de licence active trouvée
	}
	if err != nil {
		return nil, err
	}

	// Parser les modules autorisés depuis JSON
	var modulesData struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}
	if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des modules autorisés: %w", err)
	}

	license.ModulesAutorises = modulesData.Modules
	return &license, nil
}

// getAvailableFrontOfficeModules - Récupère tous les modules front-office disponibles
func (s *LicenseCreationService) getAvailableFrontOfficeModules(ctx context.Context) ([]dto.AvailableModule, error) {
	rows, err := s.db.Query(ctx, queries.LicenseQueries.GetAvailableFrontOfficeModules)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []dto.AvailableModule
	for rows.Next() {
		var module dto.AvailableModule
		err := rows.Scan(
			&module.ID,
			&module.CodeModule,
			&module.Nom,
			&module.Description,
			&module.EstActif,
		)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// validateAndGetRequestedModules - Valide les modules demandés et retourne les infos complètes
func (s *LicenseCreationService) validateAndGetRequestedModules(
	ctx context.Context, 
	requestedCodes []string, 
	availableModules []dto.AvailableModule,
) ([]dto.ModuleInfo, error) {
	
	// Créer une map des modules disponibles par code
	availableMap := make(map[string]dto.AvailableModule)
	for _, module := range availableModules {
		availableMap[module.CodeModule] = module
	}

	var validModules []dto.ModuleInfo
	var invalidCodes []string

	// Valider chaque code demandé
	for _, code := range requestedCodes {
		if module, exists := availableMap[code]; exists {
			validModules = append(validModules, dto.ModuleInfo{
				ID:   module.ID,
				Code: module.CodeModule,
			})
		} else {
			invalidCodes = append(invalidCodes, code)
		}
	}

	if len(invalidCodes) > 0 {
		return nil, &ServiceError{
			Type:    "validation",
			Message: "Certains modules demandés ne sont pas disponibles",
			Details: map[string]interface{}{
				"invalid_codes": invalidCodes,
			},
		}
	}

	return validModules, nil
}

// calculateExpirationDate - Calcule la date d'expiration selon le type de licence
func (s *LicenseCreationService) calculateExpirationDate(typeLicence string) time.Time {
	now := time.Now()
	
	switch typeLicence {
	case "evaluation":
		return now.AddDate(0, 1, 0) // 1 mois
	case "standard":
		return now.AddDate(1, 0, 0) // 1 an
	case "premium":
		// En mode online, même premium a une expiration (3 ans par exemple)
		return now.AddDate(3, 0, 0) // 3 ans
	default:
		return now.AddDate(1, 0, 0) // Par défaut 1 an
	}
}

// createLicenseInDatabase - Crée la licence en base de données
func (s *LicenseCreationService) createLicenseInDatabase(
	ctx context.Context,
	etablissementID uuid.UUID,
	modeDeploiement string,
	typeLicence string,
	modulesAutorises []dto.ModuleInfo,
	dateExpiration *time.Time,
	createdBy uuid.UUID,
) (*dto.LicenseResponse, error) {

	// Préparer le JSON des modules autorisés
	modulesData := struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}{
		Modules: modulesAutorises,
	}

	modulesJSON, err := json.Marshal(modulesData)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la sérialisation des modules: %w", err)
	}

	// Créer la licence
	var license dto.LicenseResponse
	var returnedModulesJSON []byte

	err = s.db.QueryRow(
		ctx,
		queries.LicenseQueries.CreateLicense,
		etablissementID,
		modeDeploiement,
		typeLicence,
		string(modulesJSON),
		dateExpiration,
		createdBy,
	).Scan(
		&license.ID,
		&license.EtablissementID,
		&license.ModeDeploiement,
		&license.TypeLicence,
		&returnedModulesJSON,
		&license.DateActivation,
		&license.DateExpiration,
		&license.Statut,
		&license.SyncInitialComplete,
		&license.CreatedAt,
		&license.UpdatedAt,
		&license.CreatedBy,
	)

	if err != nil {
		return nil, err
	}

	// Parser les modules depuis la base
	var returnedModulesData struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}
	if err := json.Unmarshal(returnedModulesJSON, &returnedModulesData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des modules retournés: %w", err)
	}

	license.ModulesAutorises = returnedModulesData.Modules
	return &license, nil
}

// createLicenseHistory - Crée un enregistrement dans l'historique des licences
func (s *LicenseCreationService) createLicenseHistory(
	ctx context.Context,
	etablissementID uuid.UUID,
	licenseID uuid.UUID,
	typeEvenement string,
	statutPrecedent *string,
	statutNouveau string,
	motifChangement string,
	utilisateurAction uuid.UUID,
	ipAction *net.IP,
) error {

	var ipString *string
	if ipAction != nil {
		ip := ipAction.String()
		ipString = &ip
	}

	var historyID uuid.UUID
	var createdAt time.Time

	err := s.db.QueryRow(
		ctx,
		queries.LicenseQueries.CreateLicenseHistory,
		etablissementID,
		licenseID,
		typeEvenement,
		statutPrecedent,
		statutNouveau,
		motifChangement,
		utilisateurAction,
		ipString,
	).Scan(&historyID, &createdAt)

	return err
}

// GetAvailableFrontOfficeModules - Récupère la liste des modules front-office disponibles
func (s *LicenseCreationService) GetAvailableFrontOfficeModules(ctx context.Context) (*dto.AvailableModulesResponse, error) {
	modules, err := s.getAvailableFrontOfficeModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des modules: %w", err)
	}

	return &dto.AvailableModulesResponse{
		Modules: modules,
		Total:   len(modules),
	}, nil
}

// GetLicenseByEstablishment - Récupère toutes les licences d'un établissement
func (s *LicenseCreationService) GetLicenseByEstablishment(ctx context.Context, etablissementID uuid.UUID) ([]dto.LicenseResponse, error) {
	rows, err := s.db.Query(ctx, queries.LicenseQueries.GetLicenseByEstablishment, etablissementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []dto.LicenseResponse
	for rows.Next() {
		var license dto.LicenseResponse
		var modulesJSON []byte

		err := rows.Scan(
			&license.ID,
			&license.EtablissementID,
			&license.ModeDeploiement,
			&license.TypeLicence,
			&modulesJSON,
			&license.DateActivation,
			&license.DateExpiration,
			&license.Statut,
			&license.SyncInitialComplete,
			&license.DateSyncInitial,
			&license.CreatedAt,
			&license.UpdatedAt,
			&license.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		// Parser les modules autorisés
		var modulesData struct {
			Modules []dto.ModuleInfo `json:"modules"`
		}
		if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
			return nil, fmt.Errorf("erreur lors du parsing des modules pour la licence %s: %w", license.ID, err)
		}

		license.ModulesAutorises = modulesData.Modules
		licenses = append(licenses, license)
	}

	return licenses, nil
}