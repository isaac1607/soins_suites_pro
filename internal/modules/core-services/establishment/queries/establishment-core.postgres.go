package queries

// EstablishmentQueries regroupe toutes les requêtes SQL pour le core-service Établissement
var EstablishmentQueries = struct {
	Create                  string
	GetByID                 string
	GetByCode               string
	GetByAppInstance        string
	UpdateByAdminTir        string
	UpdateByUser            string
	CheckCodeExists         string
	CheckAppInstanceExists  string
	GetHealthInfo           string
	GetHealthInfoList       string
	GetHealthInfoByCode     string
}{
	/**
	 * Crée un nouvel établissement
	 * Paramètres: $1 = code_etablissement, $2 = nom, $3 = nom_court, $4 = adresse_complete,
	 *            $5 = telephone_principal, $6 = ville, $7 = commune, $8 = email,
	 *            $9 = created_by (tir_admin_global)
	 */
	Create: `
		INSERT INTO base_etablissement (
			code_etablissement,
			nom,
			nom_court,
			adresse_complete,
			telephone_principal,
			ville,
			commune,
			email,
			created_by,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()
		)
		RETURNING 
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			adresse_complete,
			telephone_principal,
			ville,
			commune,
			email,
			statut,
			created_at
	`,

	/**
	 * Récupère un établissement par ID
	 * Paramètres: $1 = establishment_id
	 */
	GetByID: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			adresse_complete,
			telephone_principal,
			ville,
			commune,
			email,
			second_telephone,
			rccm,
			cnps,
			logo_principal_url,
			logo_documents_url,
			duree_validite_ticket_jours,
			nb_souches_par_caisse,
			garde_heure_debut,
			garde_heure_fin,
			statut,
			created_at,
			updated_at_admin_tir,
			updated_at_user,
			created_by,
			updated_by_admin_tir,
			updated_by_user
		FROM base_etablissement
		WHERE id = $1
	`,

	/**
	 * Récupère un établissement par code
	 * Paramètres: $1 = code_etablissement
	 */
	GetByCode: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			statut,
			created_at
		FROM base_etablissement
		WHERE code_etablissement = $1
	`,

	/**
	 * Récupère un établissement par app_instance
	 * Paramètres: $1 = app_instance
	 */
	GetByAppInstance: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			statut,
			created_at
		FROM base_etablissement
		WHERE app_instance = $1
	`,

	/**
	 * Met à jour un établissement par admin TIR
	 * Paramètres: $1 = establishment_id, $2 = nom, $3 = nom_court, $4 = adresse_complete,
	 *            $5 = telephone_principal, $6 = ville, $7 = commune, $8 = email,
	 *            $9 = updated_by_admin_tir
	 */
	UpdateByAdminTir: `
		UPDATE base_etablissement 
		SET 
			nom = $2,
			nom_court = $3,
			adresse_complete = $4,
			telephone_principal = $5,
			ville = $6,
			commune = $7,
			email = $8,
			updated_at_admin_tir = NOW(),
			updated_by_admin_tir = $9,
			updated_by_user = NULL,
			updated_at_user = NULL
		WHERE id = $1
		RETURNING 
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			updated_at_admin_tir
	`,

	/**
	 * Met à jour un établissement par utilisateur établissement
	 * Paramètres: $1 = establishment_id, $2 = nom, $3 = nom_court, $4 = adresse_complete,
	 *            $5 = telephone_principal, $6 = ville, $7 = commune, $8 = email,
	 *            $9 = updated_by_user
	 */
	UpdateByUser: `
		UPDATE base_etablissement 
		SET 
			nom = $2,
			nom_court = $3,
			adresse_complete = $4,
			telephone_principal = $5,
			ville = $6,
			commune = $7,
			email = $8,
			updated_at_user = NOW(),
			updated_by_user = $9,
			updated_by_admin_tir = NULL,
			updated_at_admin_tir = NULL
		WHERE id = $1
		RETURNING 
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			updated_at_user
	`,

	/**
	 * Vérifie si un code établissement existe déjà
	 * Paramètres: $1 = code_etablissement
	 */
	CheckCodeExists: `
		SELECT COUNT(*) 
		FROM base_etablissement 
		WHERE code_etablissement = $1
	`,

	/**
	 * Vérifie si un app_instance existe déjà
	 * Paramètres: $1 = app_instance
	 */
	CheckAppInstanceExists: `
		SELECT COUNT(*) 
		FROM base_etablissement 
		WHERE app_instance = $1
	`,

	/**
	 * Récupère les informations sanitaires complètes d'un établissement par ID
	 * Paramètres: $1 = establishment_id
	 */
	GetHealthInfo: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			statut,
			adresse_complete,
			telephone_principal,
			second_telephone,
			email,
			ville,
			commune,
			rccm,
			cnps,
			duree_validite_ticket_jours,
			nb_souches_par_caisse,
			garde_heure_debut,
			garde_heure_fin,
			logo_principal_url,
			logo_documents_url,
			created_at,
			updated_at_admin_tir,
			updated_at_user,
			CASE 
				WHEN updated_at_admin_tir IS NOT NULL AND updated_at_user IS NOT NULL THEN
					CASE WHEN updated_at_admin_tir > updated_at_user THEN 'admin_tir' ELSE 'user' END
				WHEN updated_at_admin_tir IS NOT NULL THEN 'admin_tir'
				WHEN updated_at_user IS NOT NULL THEN 'user'
				ELSE NULL
			END as last_modified_by,
			CASE 
				WHEN updated_at_admin_tir IS NOT NULL AND updated_at_user IS NOT NULL THEN
					CASE WHEN updated_at_admin_tir > updated_at_user THEN updated_at_admin_tir ELSE updated_at_user END
				WHEN updated_at_admin_tir IS NOT NULL THEN updated_at_admin_tir
				WHEN updated_at_user IS NOT NULL THEN updated_at_user
				ELSE NULL
			END as last_modified_at
		FROM base_etablissement
		WHERE id = $1
	`,

	/**
	 * Récupère la liste des informations sanitaires d'établissements avec pagination
	 * Paramètres: $1 = limit, $2 = offset
	 */
	GetHealthInfoList: `
		SELECT
			id,
			code_etablissement,
			nom,
			nom_court,
			ville,
			commune,
			statut,
			telephone_principal,
			email,
			created_at,
			COUNT(*) OVER() as total_count
		FROM base_etablissement
		WHERE statut != 'archive'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`,

	/**
	 * Récupère les informations sanitaires d'un établissement par code
	 * Paramètres: $1 = code_etablissement
	 */
	GetHealthInfoByCode: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			nom_court,
			statut,
			adresse_complete,
			telephone_principal,
			second_telephone,
			email,
			ville,
			commune,
			rccm,
			cnps,
			duree_validite_ticket_jours,
			nb_souches_par_caisse,
			garde_heure_debut,
			garde_heure_fin,
			logo_principal_url,
			logo_documents_url,
			created_at,
			updated_at_admin_tir,
			updated_at_user,
			CASE 
				WHEN updated_at_admin_tir IS NOT NULL AND updated_at_user IS NOT NULL THEN
					CASE WHEN updated_at_admin_tir > updated_at_user THEN 'admin_tir' ELSE 'user' END
				WHEN updated_at_admin_tir IS NOT NULL THEN 'admin_tir'
				WHEN updated_at_user IS NOT NULL THEN 'user'
				ELSE NULL
			END as last_modified_by,
			CASE 
				WHEN updated_at_admin_tir IS NOT NULL AND updated_at_user IS NOT NULL THEN
					CASE WHEN updated_at_admin_tir > updated_at_user THEN updated_at_admin_tir ELSE updated_at_user END
				WHEN updated_at_admin_tir IS NOT NULL THEN updated_at_admin_tir
				WHEN updated_at_user IS NOT NULL THEN updated_at_user
				ELSE NULL
			END as last_modified_at
		FROM base_etablissement
		WHERE code_etablissement = $1
	`,
}

// LicenseQueries regroupe toutes les requêtes SQL pour la gestion des licences
var LicenseQueries = struct {
	CheckActiveLicense         string
	GetAvailableFrontOfficeModules string
	CreateLicense             string
	CreateLicenseHistory      string
	GetLicenseByID           string
	GetLicenseByEstablishment string
	GetLicenseDetailedByID    string
	GetLicenseDetailedByEstablishment string
	GetLicenseHistory         string
	GetLicenseListByEstablishment string
}{
	/**
	 * Vérifie s'il existe une licence active pour un établissement
	 * Paramètres: $1 = etablissement_id
	 */
	CheckActiveLicense: `
		SELECT 
			id,
			etablissement_id,
			mode_deploiement,
			type_licence,
			statut,
			date_activation,
			date_expiration
		FROM base_licence
		WHERE etablissement_id = $1 
			AND statut = 'actif'
		LIMIT 1
	`,

	/**
	 * Récupère tous les modules front-office disponibles (est_module_back_office = FALSE)
	 * Paramètres: aucun
	 */
	GetAvailableFrontOfficeModules: `
		SELECT
			id,
			code_module,
			nom_standard,
			description,
			est_actif
		FROM base_module
		WHERE est_module_back_office = FALSE
			AND est_actif = TRUE
		ORDER BY nom_standard ASC
	`,

	/**
	 * Crée une nouvelle licence
	 * Paramètres: $1 = etablissement_id, $2 = mode_deploiement, $3 = type_licence, 
	 *            $4 = modules_autorises (JSONB), $5 = date_expiration, $6 = created_by
	 */
	CreateLicense: `
		INSERT INTO base_licence (
			etablissement_id,
			mode_deploiement,
			type_licence,
			modules_autorises,
			date_activation,
			date_expiration,
			statut,
			sync_initial_complete,
			created_at,
			updated_at,
			created_by
		) VALUES (
			$1, $2, $3, $4, NOW(), $5, 'actif', FALSE, NOW(), NOW(), $6
		)
		RETURNING 
			id,
			etablissement_id,
			mode_deploiement,
			type_licence,
			modules_autorises,
			date_activation,
			date_expiration,
			statut,
			sync_initial_complete,
			created_at,
			updated_at,
			created_by
	`,

	/**
	 * Crée un enregistrement dans l'historique des licences
	 * Paramètres: $1 = etablissement_id, $2 = licence_id, $3 = type_evenement,
	 *            $4 = statut_precedent, $5 = statut_nouveau, $6 = motif_changement,
	 *            $7 = utilisateur_action, $8 = ip_action
	 */
	CreateLicenseHistory: `
		INSERT INTO base_licence_historique (
			etablissement_id,
			licence_id,
			type_evenement,
			statut_precedent,
			statut_nouveau,
			motif_changement,
			utilisateur_action,
			ip_action,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
		)
		RETURNING id, created_at
	`,

	/**
	 * Récupère une licence par ID
	 * Paramètres: $1 = licence_id
	 */
	GetLicenseByID: `
		SELECT
			id,
			etablissement_id,
			mode_deploiement,
			type_licence,
			modules_autorises,
			date_activation,
			date_expiration,
			statut,
			sync_initial_complete,
			date_sync_initial,
			created_at,
			updated_at,
			created_by
		FROM base_licence
		WHERE id = $1
	`,

	/**
	 * Récupère les licences d'un établissement
	 * Paramètres: $1 = etablissement_id
	 */
	GetLicenseByEstablishment: `
		SELECT
			id,
			etablissement_id,
			mode_deploiement,
			type_licence,
			modules_autorises,
			date_activation,
			date_expiration,
			statut,
			sync_initial_complete,
			date_sync_initial,
			created_at,
			updated_at,
			created_by
		FROM base_licence
		WHERE etablissement_id = $1
		ORDER BY created_at DESC
	`,

	/**
	 * Récupère une licence avec informations détaillées et établissement par ID de licence
	 * Paramètres: $1 = licence_id
	 */
	GetLicenseDetailedByID: `
		SELECT
			l.id,
			l.etablissement_id,
			l.mode_deploiement,
			l.type_licence,
			l.modules_autorises,
			l.date_activation,
			l.date_expiration,
			l.statut,
			l.sync_initial_complete,
			l.date_sync_initial,
			l.created_at,
			l.updated_at,
			l.created_by,
			e.nom as etablissement_nom,
			e.code_etablissement as etablissement_code,
			e.statut as etablissement_statut,
			-- Calculs statut et expiration
			CASE 
				WHEN l.statut != 'actif' THEN l.statut
				WHEN l.date_expiration IS NULL THEN 'actif'
				WHEN l.date_expiration < NOW() THEN 'expire'
				WHEN l.date_expiration < NOW() + INTERVAL '30 days' THEN 'bientot_expire'
				ELSE 'actif'
			END as statut_calcule,
			CASE 
				WHEN l.date_expiration IS NULL THEN NULL
				ELSE EXTRACT(days FROM l.date_expiration - NOW())::INTEGER
			END as jours_avant_expiration,
			CASE 
				WHEN l.date_expiration IS NULL THEN FALSE
				ELSE l.date_expiration < NOW()
			END as est_expire,
			CASE 
				WHEN l.date_expiration IS NULL THEN FALSE
				ELSE l.date_expiration < NOW() + INTERVAL '30 days' AND l.date_expiration >= NOW()
			END as est_bientot_expire
		FROM base_licence l
		INNER JOIN base_etablissement e ON l.etablissement_id = e.id
		WHERE l.id = $1
	`,

	/**
	 * Récupère la licence active d'un établissement avec informations détaillées
	 * Paramètres: $1 = etablissement_id
	 */
	GetLicenseDetailedByEstablishment: `
		SELECT
			l.id,
			l.etablissement_id,
			l.mode_deploiement,
			l.type_licence,
			l.modules_autorises,
			l.date_activation,
			l.date_expiration,
			l.statut,
			l.sync_initial_complete,
			l.date_sync_initial,
			l.created_at,
			l.updated_at,
			l.created_by,
			e.nom as etablissement_nom,
			e.code_etablissement as etablissement_code,
			e.statut as etablissement_statut,
			-- Calculs statut et expiration
			CASE 
				WHEN l.statut != 'actif' THEN l.statut
				WHEN l.date_expiration IS NULL THEN 'actif'
				WHEN l.date_expiration < NOW() THEN 'expire'
				WHEN l.date_expiration < NOW() + INTERVAL '30 days' THEN 'bientot_expire'
				ELSE 'actif'
			END as statut_calcule,
			CASE 
				WHEN l.date_expiration IS NULL THEN NULL
				ELSE EXTRACT(days FROM l.date_expiration - NOW())::INTEGER
			END as jours_avant_expiration,
			CASE 
				WHEN l.date_expiration IS NULL THEN FALSE
				ELSE l.date_expiration < NOW()
			END as est_expire,
			CASE 
				WHEN l.date_expiration IS NULL THEN FALSE
				ELSE l.date_expiration < NOW() + INTERVAL '30 days' AND l.date_expiration >= NOW()
			END as est_bientot_expire
		FROM base_licence l
		INNER JOIN base_etablissement e ON l.etablissement_id = e.id
		WHERE l.etablissement_id = $1 
			AND l.statut = 'actif'
		ORDER BY l.created_at DESC
		LIMIT 1
	`,

	/**
	 * Récupère l'historique d'une licence
	 * Paramètres: $1 = licence_id
	 */
	GetLicenseHistory: `
		SELECT
			id,
			type_evenement,
			statut_precedent,
			statut_nouveau,
			motif_changement,
			utilisateur_action,
			ip_action,
			created_at
		FROM base_licence_historique
		WHERE licence_id = $1
		ORDER BY created_at DESC
	`,

	/**
	 * Récupère la liste des licences d'un établissement avec informations de base
	 * Paramètres: $1 = etablissement_id
	 */
	GetLicenseListByEstablishment: `
		SELECT
			l.id,
			l.mode_deploiement,
			l.type_licence,
			l.modules_autorises,
			l.date_activation,
			l.date_expiration,
			l.statut,
			l.created_at,
			-- Calculs statut et expiration
			CASE 
				WHEN l.statut != 'actif' THEN l.statut
				WHEN l.date_expiration IS NULL THEN 'actif'
				WHEN l.date_expiration < NOW() THEN 'expire'
				WHEN l.date_expiration < NOW() + INTERVAL '30 days' THEN 'bientot_expire'
				ELSE 'actif'
			END as statut_calcule,
			CASE 
				WHEN l.date_expiration IS NULL THEN NULL
				ELSE EXTRACT(days FROM l.date_expiration - NOW())::INTEGER
			END as jours_avant_expiration,
			CASE 
				WHEN l.date_expiration IS NULL THEN FALSE
				ELSE l.date_expiration < NOW()
			END as est_expire
		FROM base_licence l
		WHERE l.etablissement_id = $1
		ORDER BY l.created_at DESC
	`,
}