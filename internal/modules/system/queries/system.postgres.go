package queries

// SystemQueries regroupe toutes les requêtes SQL pour le module System
var SystemQueries = struct {
	GetEstablishmentInfo          string
	GetLicenseInfo                string
	GetAuthorizedModules          string
	GetAuthorizedModulesOnly      string
	GetSystemConfiguration        string
	GetEstablishmentByAppInstance string
	CheckSyncStatus               string
	UpdateSyncStatus              string
	GetCompleteEstablishmentData  string
	GetCompleteModulesData        string
	InsertEstablishment           string
	InsertLicense                 string
	InsertModule                  string
	InsertRubrique                string
	InsertSuperAdmin              string
}{
	/**
	 * Récupère les informations complètes de l'établissement
	 * Paramètres: $1 = establishment_id
	 */
	GetEstablishmentInfo: `
		SELECT
			id,
			code_etablissement,
			nom,
			nom_court,
			ville,
			created_at
		FROM base_etablissement
		WHERE id = $1
	`,

	/**
	 * Récupère les informations de licence avec modules autorisés
	 * Paramètres: $1 = establishment_id
	 */
	GetLicenseInfo: `
		SELECT
			id,
			type_licence,
			mode_deploiement,
			statut,
			date_activation,
			date_expiration,
			modules_autorises
		FROM base_licence
		WHERE etablissement_id = $1
		AND statut = 'actif'
		ORDER BY created_at DESC
		LIMIT 1
	`,

	/**
	 * Récupère tous les modules avec leur statut d'autorisation
	 * Paramètres: $1 = modules_autorises (JSONB array)
	 */
	GetAuthorizedModules: `
		SELECT
			m.id,
			m.code_module,
			m.nom_standard,
			m.est_medical,
			m.peut_prendre_ticket,
			m.est_actif
		FROM base_module m
		WHERE m.est_actif = true
		ORDER BY 
			CASE WHEN m.est_obligatoire THEN 0 ELSE 1 END,
			m.code_module ASC
	`,

	/**
	 * Récupère uniquement les modules autorisés par la licence pour navigation dynamique
	 * Paramètres: $1 = establishment_id
	 */
	GetAuthorizedModulesOnly: `
		WITH licence_data AS (
			SELECT 
				modules_autorises
			FROM base_licence bl
			WHERE bl.etablissement_id = $1
			AND bl.statut = 'actif'
			ORDER BY bl.created_at DESC
			LIMIT 1
		)
		SELECT
			m.code_module,
			m.nom_standard,
			m.est_medical,
			m.peut_prendre_ticket,
			m.est_module_back_office
		FROM base_module m
		CROSS JOIN licence_data ld
		WHERE m.est_actif = true
		AND (
			-- Si licence trouvée, vérifier si module autorisé
			CASE 
				WHEN ld.modules_autorises IS NOT NULL THEN
					CASE 
						WHEN jsonb_typeof(ld.modules_autorises) = 'array' THEN
							ld.modules_autorises ? m.code_module
						WHEN jsonb_typeof(ld.modules_autorises) = 'object' THEN
							(ld.modules_autorises->'modules') ? m.code_module
						ELSE false
					END
				ELSE false
			END
		)
		ORDER BY 
			CASE WHEN m.est_obligatoire THEN 0 ELSE 1 END,
			m.code_module ASC
	`,

	/**
	 * Récupère la configuration système de l'établissement
	 * Paramètres: $1 = establishment_id
	 */
	GetSystemConfiguration: `
		SELECT
			duree_validite_ticket_jours,
			nb_souches_par_caisse,
			CASE 
				WHEN garde_heure_debut IS NOT NULL AND garde_heure_fin IS NOT NULL 
				THEN true 
				ELSE false 
			END as garde_active,
			garde_heure_debut,
			garde_heure_fin
		FROM base_etablissement
		WHERE id = $1
	`,

	/**
	 * Récupère l'établissement par app_instance et code_etablissement
	 * Paramètres: $1 = app_instance, $2 = code_etablissement
	 */
	GetEstablishmentByAppInstance: `
		SELECT
			id,
			app_instance,
			code_etablissement,
			nom,
			statut
		FROM base_etablissement
		WHERE app_instance = $1 AND code_etablissement = $2
	`,

	/**
	 * Vérifier le statut de synchronisation d'une licence
	 * Paramètres: $1 = establishment_id
	 */
	CheckSyncStatus: `
		SELECT
			sync_initial_complete,
			date_sync_initial
		FROM base_licence
		WHERE etablissement_id = $1
		AND statut = 'actif'
		ORDER BY created_at DESC
		LIMIT 1
	`,

	/**
	 * Mettre à jour le statut de synchronisation
	 * Paramètres: $1 = establishment_id
	 */
	UpdateSyncStatus: `
		UPDATE base_licence
		SET 
			sync_initial_complete = true,
			date_sync_initial = NOW()
		WHERE etablissement_id = $1
		AND statut = 'actif'
	`,

	/**
	 * Récupérer les données complètes d'un établissement pour synchronisation
	 * Paramètres: $1 = establishment_id
	 */
	GetCompleteEstablishmentData: `
		SELECT
			e.id,
			e.app_instance,
			e.code_etablissement,
			e.nom,
			e.nom_court,
			e.adresse_complete,
			e.ville,
			e.commune,
			e.telephone_principal,
			e.email,
			e.duree_validite_ticket_jours,
			e.nb_souches_par_caisse,
			e.garde_heure_debut::text,
			e.garde_heure_fin::text,
			e.created_at,
			l.id as licence_id,
			l.mode_deploiement,
			l.type_licence,
			l.modules_autorises,
			l.date_activation,
			l.date_expiration,
			l.statut as licence_statut,
			l.sync_initial_complete,
			l.created_at as licence_created_at
		FROM base_etablissement e
		LEFT JOIN base_licence l ON e.id = l.etablissement_id AND l.statut = 'actif'
		WHERE e.id = $1
	`,

	/**
	 * Récupérer tous les modules avec leurs rubriques
	 * Paramètres: aucun
	 */
	GetCompleteModulesData: `
		SELECT
			m.id,
			ROW_NUMBER() OVER (ORDER BY m.code_module) as numero_module,
			m.code_module,
			m.nom_standard,
			m.description,
			m.est_medical,
			m.est_obligatoire,
			m.est_actif,
			m.est_module_back_office,
			m.peut_prendre_ticket,
			r.id as rubrique_id,
			r.code_rubrique,
			r.nom as rubrique_nom,
			r.description as rubrique_description,
			r.ordre_affichage,
			r.est_obligatoire as rubrique_est_obligatoire,
			r.est_actif as rubrique_est_actif
		FROM base_module m
		LEFT JOIN base_rubrique r ON m.id = r.module_id
		WHERE m.est_actif = true
		ORDER BY m.code_module, r.ordre_affichage
	`,

	/**
	 * Insérer un établissement lors de la synchronisation
	 * Paramètres: $1 = id, $2 = app_instance, $3 = code_etablissement, etc.
	 */
	InsertEstablishment: `
		INSERT INTO base_etablissement (
			id, app_instance, code_etablissement, nom, nom_court,
			adresse_complete, ville, commune, telephone_principal, email,
			duree_validite_ticket_jours, nb_souches_par_caisse,
			garde_heure_debut, garde_heure_fin, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::time, $14::time, $15::timestamp
		)
		ON CONFLICT (id) DO UPDATE SET
			nom = EXCLUDED.nom,
			nom_court = EXCLUDED.nom_court,
			adresse_complete = EXCLUDED.adresse_complete,
			updated_at = NOW()
	`,

	/**
	 * Insérer une licence lors de la synchronisation
	 * Paramètres: $1 = id, $2 = etablissement_id, etc.
	 */
	InsertLicense: `
		INSERT INTO base_licence (
			id, etablissement_id, mode_deploiement, type_licence,
			modules_autorises, date_activation, date_expiration,
			statut, sync_initial_complete, created_at
		) VALUES (
			$1, $2, $3, $4, $5::jsonb, $6::timestamp, $7::timestamp, $8, $9, $10::timestamp
		)
		ON CONFLICT (id) DO UPDATE SET
			modules_autorises = EXCLUDED.modules_autorises,
			updated_at = NOW()
	`,

	/**
	 * Insérer un module lors de la synchronisation
	 * Paramètres: $1 = id, $2 = code_module, etc.
	 */
	InsertModule: `
		INSERT INTO base_module (
			id, code_module, nom_standard, description,
			peut_prendre_ticket, est_medical, est_obligatoire,
			est_actif, est_module_back_office, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10::timestamp
		)
		ON CONFLICT (code_module) DO UPDATE SET
			nom_standard = EXCLUDED.nom_standard,
			description = EXCLUDED.description,
			updated_at = NOW()
		RETURNING id
	`,

	/**
	 * Insérer une rubrique lors de la synchronisation
	 * Paramètres: $1 = id, $2 = module_id, etc.
	 */
	InsertRubrique: `
		INSERT INTO base_rubrique (
			id, module_id, code_rubrique, nom, description,
			ordre_affichage, est_obligatoire, est_actif, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9::timestamp
		)
		ON CONFLICT (module_id, code_rubrique) DO UPDATE SET
			nom = EXCLUDED.nom,
			description = EXCLUDED.description,
			updated_at = NOW()
	`,

	/**
	 * Insérer le super admin lors de la synchronisation
	 * Paramètres: $1 = id, $2 = etablissement_id, etc.
	 */
	InsertSuperAdmin: `
		INSERT INTO user_utilisateur (
			id, etablissement_id, identifiant, nom, prenoms, telephone,
			password_hash, salt, must_change_password, est_admin, type_admin,
			est_admin_tir, est_temporaire, statut, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::timestamp
		)
		ON CONFLICT (etablissement_id, identifiant) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			updated_at = NOW()
	`,
}
