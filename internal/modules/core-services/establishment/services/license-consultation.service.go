package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/modules/core-services/establishment/dto"
	"soins-suite-core/internal/modules/core-services/establishment/queries"
)

// LicenseConsultationService - Service métier pour la consultation de licences
type LicenseConsultationService struct {
	db *postgres.Client
}

// NewLicenseConsultationService - Constructeur du service de consultation de licences
func NewLicenseConsultationService(db *postgres.Client) *LicenseConsultationService {
	return &LicenseConsultationService{
		db: db,
	}
}

// GetLicenseByID - Récupère une licence spécifique par son ID avec informations détaillées
func (s *LicenseConsultationService) GetLicenseByID(
	ctx context.Context, 
	licenseID uuid.UUID, 
	includeHistory bool,
) (*dto.LicenseConsultationResult, error) {
	
	// 1. Récupérer la licence détaillée
	license, err := s.getLicenseDetailedByID(ctx, licenseID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{
				Type:    "not_found",
				Message: "Licence non trouvée",
				Details: map[string]interface{}{
					"license_id": licenseID,
				},
			}
		}
		return nil, fmt.Errorf("erreur lors de la récupération de la licence: %w", err)
	}

	// 2. Récupérer l'historique si demandé
	var history []dto.LicenseHistoryEntry
	if includeHistory {
		history, err = s.getLicenseHistory(ctx, licenseID)
		if err != nil {
			// Log l'erreur mais ne fait pas échouer la consultation
			fmt.Printf("[WARNING] Échec de récupération de l'historique pour licence %s: %v\n", licenseID, err)
			history = []dto.LicenseHistoryEntry{} // Historique vide en cas d'erreur
		}
	}

	// 3. Construire le message informatif
	message := s.buildLicenseMessage(license)

	return &dto.LicenseConsultationResult{
		License: license,
		History: history,
		Message: message,
	}, nil
}

// GetActiveLicenseByEstablishment - Récupère la licence active d'un établissement
func (s *LicenseConsultationService) GetActiveLicenseByEstablishment(
	ctx context.Context, 
	etablissementID uuid.UUID, 
	includeHistory bool,
) (*dto.LicenseConsultationResult, error) {
	
	// 1. Récupérer la licence active de l'établissement
	license, err := s.getLicenseDetailedByEstablishment(ctx, etablissementID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{
				Type:    "not_found",
				Message: "Aucune licence active trouvée pour cet établissement",
				Details: map[string]interface{}{
					"etablissement_id": etablissementID,
				},
			}
		}
		return nil, fmt.Errorf("erreur lors de la récupération de la licence active: %w", err)
	}

	// 2. Récupérer l'historique si demandé
	var history []dto.LicenseHistoryEntry
	if includeHistory {
		history, err = s.getLicenseHistory(ctx, license.ID)
		if err != nil {
			// Log l'erreur mais ne fait pas échouer la consultation
			fmt.Printf("[WARNING] Échec de récupération de l'historique pour licence %s: %v\n", license.ID, err)
			history = []dto.LicenseHistoryEntry{}
		}
	}

	// 3. Construire le message informatif
	message := s.buildLicenseMessage(license)

	return &dto.LicenseConsultationResult{
		License: license,
		History: history,
		Message: message,
	}, nil
}

// GetLicenseListByEstablishment - Récupère toutes les licences d'un établissement (résumé)
func (s *LicenseConsultationService) GetLicenseListByEstablishment(
	ctx context.Context, 
	etablissementID uuid.UUID,
) (*dto.LicenseListResponse, error) {
	
	rows, err := s.db.Query(ctx, queries.LicenseQueries.GetLicenseListByEstablishment, etablissementID)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des licences: %w", err)
	}
	defer rows.Close()

	var licenses []dto.LicenseSummary
	for rows.Next() {
		var license dto.LicenseSummary
		var modulesJSON []byte

		err := rows.Scan(
			&license.ID,
			&license.ModeDeploiement,
			&license.TypeLicence,
			&modulesJSON,
			&license.DateActivation,
			&license.DateExpiration,
			&license.Statut,
			&license.CreatedAt,
			&license.StatutCalcule,
			&license.JoursAvantExpiration,
			&license.EstExpire,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan de la licence: %w", err)
		}

		// Calculer le nombre de modules
		var modulesData struct {
			Modules []dto.ModuleInfo `json:"modules"`
		}
		if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
			return nil, fmt.Errorf("erreur lors du parsing des modules pour licence %s: %w", license.ID, err)
		}

		license.NombreModules = len(modulesData.Modules)
		licenses = append(licenses, license)
	}

	return &dto.LicenseListResponse{
		Licenses: licenses,
		Total:    len(licenses),
	}, nil
}

// getLicenseDetailedByID - Récupère une licence avec détails par ID
func (s *LicenseConsultationService) getLicenseDetailedByID(ctx context.Context, licenseID uuid.UUID) (*dto.LicenseDetailedResponse, error) {
	var license dto.LicenseDetailedResponse
	var modulesJSON []byte

	err := s.db.QueryRow(ctx, queries.LicenseQueries.GetLicenseDetailedByID, licenseID).Scan(
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
		&license.EtablissementNom,
		&license.EtablissementCode,
		&license.EtablissementStatut,
		&license.StatutCalcule,
		&license.JoursAvantExpiration,
		&license.EstExpire,
		&license.EstBientotExpire,
	)

	if err != nil {
		return nil, err
	}

	// Parser les modules autorisés
	var modulesData struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}
	if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des modules autorisés: %w", err)
	}

	license.ModulesAutorises = modulesData.Modules
	license.NombreModules = len(modulesData.Modules)

	return &license, nil
}

// getLicenseDetailedByEstablishment - Récupère la licence active d'un établissement avec détails
func (s *LicenseConsultationService) getLicenseDetailedByEstablishment(ctx context.Context, etablissementID uuid.UUID) (*dto.LicenseDetailedResponse, error) {
	var license dto.LicenseDetailedResponse
	var modulesJSON []byte

	err := s.db.QueryRow(ctx, queries.LicenseQueries.GetLicenseDetailedByEstablishment, etablissementID).Scan(
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
		&license.EtablissementNom,
		&license.EtablissementCode,
		&license.EtablissementStatut,
		&license.StatutCalcule,
		&license.JoursAvantExpiration,
		&license.EstExpire,
		&license.EstBientotExpire,
	)

	if err != nil {
		return nil, err
	}

	// Parser les modules autorisés
	var modulesData struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}
	if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des modules autorisés: %w", err)
	}

	license.ModulesAutorises = modulesData.Modules
	license.NombreModules = len(modulesData.Modules)

	return &license, nil
}

// getLicenseHistory - Récupère l'historique d'une licence
func (s *LicenseConsultationService) getLicenseHistory(ctx context.Context, licenseID uuid.UUID) ([]dto.LicenseHistoryEntry, error) {
	rows, err := s.db.Query(ctx, queries.LicenseQueries.GetLicenseHistory, licenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []dto.LicenseHistoryEntry
	for rows.Next() {
		var entry dto.LicenseHistoryEntry
		
		err := rows.Scan(
			&entry.ID,
			&entry.TypeEvenement,
			&entry.StatutPrecedent,
			&entry.StatutNouveau,
			&entry.MotifChangement,
			&entry.UtilisateurAction,
			&entry.IPAction,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		history = append(history, entry)
	}

	return history, nil
}

// buildLicenseMessage - Construit un message informatif sur l'état de la licence
func (s *LicenseConsultationService) buildLicenseMessage(license *dto.LicenseDetailedResponse) string {
	switch license.StatutCalcule {
	case "actif":
		if license.DateExpiration == nil {
			return fmt.Sprintf("Licence %s active à vie pour l'établissement %s (%s). %d modules autorisés.", 
				license.TypeLicence, license.EtablissementNom, license.EtablissementCode, license.NombreModules)
		} else if license.EstBientotExpire {
			return fmt.Sprintf("⚠️ Licence %s active mais expire dans %d jours (%s). %d modules autorisés.", 
				license.TypeLicence, *license.JoursAvantExpiration, license.DateExpiration.Format("02/01/2006"), license.NombreModules)
		} else {
			return fmt.Sprintf("Licence %s active jusqu'au %s. %d modules autorisés.", 
				license.TypeLicence, license.DateExpiration.Format("02/01/2006"), license.NombreModules)
		}
	
	case "expire":
		return fmt.Sprintf("❌ Licence %s expirée depuis %d jours. Renouvellement requis.", 
			license.TypeLicence, -*license.JoursAvantExpiration)
	
	case "bientot_expire":
		return fmt.Sprintf("⚠️ Licence %s expire dans %d jours. Renouvellement recommandé.", 
			license.TypeLicence, *license.JoursAvantExpiration)
	
	case "revoquee":
		return fmt.Sprintf("❌ Licence %s révoquée. Contact administrateur requis.", license.TypeLicence)
	
	default:
		return fmt.Sprintf("Licence %s en statut %s pour l'établissement %s.", 
			license.TypeLicence, license.StatutCalcule, license.EtablissementNom)
	}
}

// CheckEstablishmentHasActiveLicense - Vérifie simplement si un établissement a une licence active
func (s *LicenseConsultationService) CheckEstablishmentHasActiveLicense(ctx context.Context, etablissementID uuid.UUID) (bool, error) {
	var count int
	err := s.db.QueryRow(
		ctx, 
		`SELECT COUNT(*) FROM base_licence WHERE etablissement_id = $1 AND statut = 'actif'`, 
		etablissementID,
	).Scan(&count)
	
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// GetLicenseStatusByEstablishment - Récupère uniquement le statut de licence d'un établissement (léger)
func (s *LicenseConsultationService) GetLicenseStatusByEstablishment(ctx context.Context, etablissementID uuid.UUID) (*dto.LicenseSummary, error) {
	var license dto.LicenseSummary
	var modulesJSON []byte

	err := s.db.QueryRow(
		ctx,
		queries.LicenseQueries.GetLicenseListByEstablishment+" LIMIT 1",
		etablissementID,
	).Scan(
		&license.ID,
		&license.ModeDeploiement,
		&license.TypeLicence,
		&modulesJSON,
		&license.DateActivation,
		&license.DateExpiration,
		&license.Statut,
		&license.CreatedAt,
		&license.StatutCalcule,
		&license.JoursAvantExpiration,
		&license.EstExpire,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &ServiceError{
				Type:    "not_found",
				Message: "Aucune licence trouvée pour cet établissement",
				Details: map[string]interface{}{
					"etablissement_id": etablissementID,
				},
			}
		}
		return nil, err
	}

	// Calculer le nombre de modules
	var modulesData struct {
		Modules []dto.ModuleInfo `json:"modules"`
	}
	if err := json.Unmarshal(modulesJSON, &modulesData); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing des modules: %w", err)
	}

	license.NombreModules = len(modulesData.Modules)
	
	return &license, nil
}