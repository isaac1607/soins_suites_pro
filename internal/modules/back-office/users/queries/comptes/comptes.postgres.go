package comptes

var ComptesQueries = struct {
	CheckUserExists          string
	CreateUser              string
	GetModulesByIds         string
	GetRubriquesByIds       string
	ValidateModulesInLicense string
	CreateUserProfil        string
	CreateUserModule        string
	CreateUserModuleRubrique string
	GetBackOfficeModules    string
	GetFrontOfficeModules   string
	GetUsersWithPermissionsSummary string
	CountUsersWithFilters   string
	GetUserDetails          string
	GetUserProfils          string
	GetUserModulesComplets  string
	GetUserModulesPartiels  string
	GetUserStatistiques     string
	// Permissions modification
	ValidateUserExists       string
	ValidateProfilExists     string
	ValidateModuleExists     string
	ValidateRubriqueExists   string
	AddUserProfil           string
	RemoveUserProfil        string
	AddUserModuleComplet    string
	RemoveUserModuleComplet string
	AddUserModulePartiel    string
	RemoveUserModulePartiel string
	UpdateUserModulePartiel string
	CheckUserPermissionExists string
}{
	/**
	 * Vérifie si un utilisateur existe déjà avec cet identifiant
	 * Paramètres: $1 = etablissement_id, $2 = identifiant
	 */
	CheckUserExists: `
		SELECT EXISTS(
			SELECT 1 FROM user_utilisateur
			WHERE etablissement_id = $1
				AND identifiant = $2
				AND statut != 'archive'
		)
	`,

	/**
	 * Crée un nouvel utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = identifiant, $4 = nom, $5 = prenoms,
	 *            $6 = telephone, $7 = password_hash, $8 = salt,
	 *            $9 = must_change_password, $10 = est_admin, $11 = type_admin,
	 *            $12 = est_medecin, $13 = role_metier, $14 = est_temporaire,
	 *            $15 = date_expiration, $16 = statut, $17 = created_by
	 */
	CreateUser: `
		INSERT INTO user_utilisateur (
			id, etablissement_id, identifiant, nom, prenoms, telephone,
			password_hash, salt, must_change_password, est_admin, type_admin,
			est_medecin, role_metier, est_temporaire, date_expiration,
			statut, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, NOW(), NOW()
		)
		RETURNING id, identifiant
	`,

	/**
	 * Récupère les modules par leurs IDs
	 * Paramètres: $1 = array des module_ids
	 */
	GetModulesByIds: `
		SELECT id, code_module, nom_standard, est_module_back_office, est_actif
		FROM base_module
		WHERE id = ANY($1) AND est_actif = TRUE
	`,

	/**
	 * Récupère les rubriques par leurs IDs
	 * Paramètres: $1 = array des rubrique_ids
	 */
	GetRubriquesByIds: `
		SELECT r.id, r.code_rubrique, r.nom, r.module_id, m.code_module
		FROM base_rubrique r
		JOIN base_module m ON r.module_id = m.id
		WHERE r.id = ANY($1) AND r.est_actif = TRUE AND m.est_actif = TRUE
	`,

	/**
	 * Valide que les modules sont autorisés dans la licence
	 * Paramètres: $1 = etablissement_id, $2 = array des codes_modules
	 */
	ValidateModulesInLicense: `
		SELECT 
			CASE 
				WHEN modules_autorises->'modules' @> to_jsonb($2::text[])
				THEN TRUE
				ELSE FALSE
			END as modules_valides
		FROM base_licence
		WHERE etablissement_id = $1 
			AND statut = 'actif'
		LIMIT 1
	`,

	/**
	 * Attribue un profil à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, $4 = profil_template_id,
	 *            $5 = attribue_par
	 */
	CreateUserProfil: `
		INSERT INTO user_profil_utilisateurs (
			id, etablissement_id, utilisateur_id, profil_template_id,
			attribue_par, date_attribution, est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), TRUE, NOW(), NOW()
		)
	`,

	/**
	 * Attribue un module complet à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, $4 = module_id,
	 *            $5 = attribue_par
	 */
	CreateUserModule: `
		INSERT INTO user_modules (
			id, etablissement_id, utilisateur_id, module_id,
			acces_toutes_rubriques, source_attribution, attribue_par,
			date_attribution, est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, TRUE, 'individuelle', $5, NOW(), TRUE, NOW(), NOW()
		)
	`,

	/**
	 * Attribue une rubrique spécifique à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, $4 = module_id,
	 *            $5 = rubrique_id, $6 = attribue_par
	 */
	CreateUserModuleRubrique: `
		INSERT INTO user_modules_rubriques (
			id, etablissement_id, utilisateur_id, module_id, rubrique_id,
			source_attribution, attribue_par, date_attribution,
			est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'individuelle', $6, NOW(), TRUE, NOW(), NOW()
		)
	`,

	/**
	 * Récupère les modules du back-office
	 */
	GetBackOfficeModules: `
		SELECT id, code_module, nom_standard
		FROM base_module
		WHERE est_module_back_office = TRUE AND est_actif = TRUE
	`,

	/**
	 * Récupère les modules du front-office
	 */
	GetFrontOfficeModules: `
		SELECT id, code_module, nom_standard
		FROM base_module
		WHERE est_module_back_office = FALSE AND est_actif = TRUE
	`,

	/**
	 * Requête optimisée avec CTEs et agrégations pour la liste des utilisateurs
	 * Paramètres: $1 = etablissement_id, $2 = statut_filter, $3 = search_pattern,
	 *            $4 = est_admin_filter, $5 = est_medecin_filter, $6 = profil_id_filter,
	 *            $7 = module_code_filter, $8 = sort_column, $9 = sort_direction,
	 *            $10 = include_archived, $11 = limit, $12 = offset
	 */
	GetUsersWithPermissionsSummary: `
		WITH user_profils AS (
			SELECT
				pu.utilisateur_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pt.id,
						'nom_profil', pt.nom_profil,
						'code_profil', pt.code_profil
					) ORDER BY pt.nom_profil
				) as profils
			FROM user_profil_utilisateurs pu
			JOIN user_profil_template pt ON pu.profil_template_id = pt.id
			WHERE pu.est_actif = TRUE AND pt.etablissement_id = $1
			GROUP BY pu.utilisateur_id
		),
		user_permissions_count AS (
			SELECT
				u.id as user_id,
				CASE 
					-- Si c'est un super admin, il a accès à tous les modules back-office
					WHEN u.est_admin = TRUE AND u.type_admin = 'super_admin' THEN (
						SELECT COUNT(*) FROM base_module WHERE est_module_back_office = TRUE AND est_actif = TRUE
					)
					ELSE COUNT(DISTINCT CASE
						WHEN um.acces_toutes_rubriques OR pm.acces_toutes_rubriques
						THEN COALESCE(um.module_id, pm.module_id)
					END)
				END as modules_complets,
				CASE 
					-- Si c'est un super admin, il n'a pas de modules partiels (tout est complet)
					WHEN u.est_admin = TRUE AND u.type_admin = 'super_admin' THEN 0
					ELSE COUNT(DISTINCT CASE
						WHEN NOT COALESCE(um.acces_toutes_rubriques, pm.acces_toutes_rubriques, FALSE)
						THEN COALESCE(um.module_id, pm.module_id)
					END)
				END as modules_partiels,
				CASE 
					-- Si c'est un super admin, il a accès à toutes les rubriques des modules back-office
					WHEN u.est_admin = TRUE AND u.type_admin = 'super_admin' THEN (
						SELECT COUNT(*) 
						FROM base_rubrique br 
						JOIN base_module bm ON br.module_id = bm.id 
						WHERE bm.est_module_back_office = TRUE AND bm.est_actif = TRUE AND br.est_actif = TRUE
					)
					ELSE COUNT(DISTINCT COALESCE(umr.rubrique_id, pr.rubrique_id))
				END as total_rubriques
			FROM user_utilisateur u
			LEFT JOIN user_modules um ON u.id = um.utilisateur_id AND um.est_actif = TRUE
			LEFT JOIN user_modules_rubriques umr ON u.id = umr.utilisateur_id AND umr.est_actif = TRUE
			LEFT JOIN user_profil_utilisateurs pu ON u.id = pu.utilisateur_id AND pu.est_actif = TRUE
			LEFT JOIN user_profil_modules pm ON pu.profil_template_id = pm.profil_template_id AND pm.est_actif = TRUE
			LEFT JOIN user_profil_rubriques pr ON pu.profil_template_id = pr.profil_template_id AND pr.est_actif = TRUE
			WHERE u.etablissement_id = $1
			GROUP BY u.id, u.est_admin, u.type_admin
		)
		SELECT
			u.id,
			u.identifiant,
			u.nom,
			u.prenoms,
			u.telephone,
			u.est_admin,
			u.type_admin,
			u.est_medecin,
			u.role_metier,
			u.est_temporaire,
			u.date_expiration,
			u.statut,
			u.last_login_at,
			u.created_at,
			COALESCE(up.profils, '[]'::jsonb) as profils,
			jsonb_build_object(
				'total_modules', COALESCE(upc.modules_complets, 0) + COALESCE(upc.modules_partiels, 0),
				'modules_complets', COALESCE(upc.modules_complets, 0),
				'modules_partiels', COALESCE(upc.modules_partiels, 0),
				'total_rubriques', COALESCE(upc.total_rubriques, 0)
			) as permissions_resume
		FROM user_utilisateur u
		LEFT JOIN user_profils up ON u.id = up.utilisateur_id
		LEFT JOIN user_permissions_count upc ON u.id = upc.user_id
		WHERE u.etablissement_id = $1
			AND ($2::text IS NULL OR $2 = 'tous' OR u.statut = $2)
			AND ($10::boolean = TRUE OR u.statut != 'archive')
			AND ($3::text IS NULL OR u.identifiant ILIKE '%' || $3 || '%'
				OR u.nom ILIKE '%' || $3 || '%'
				OR u.prenoms ILIKE '%' || $3 || '%')
			AND ($4::boolean IS NULL OR u.est_admin = $4)
			AND ($5::boolean IS NULL OR u.est_medecin = $5)
			AND ($6::uuid IS NULL OR EXISTS (
				SELECT 1 FROM user_profil_utilisateurs
				WHERE utilisateur_id = u.id
				AND profil_template_id = $6
				AND est_actif = TRUE
			))
			AND ($7::text IS NULL OR EXISTS (
				SELECT 1 FROM user_modules um2
				JOIN base_module bm ON um2.module_id = bm.id
				WHERE um2.utilisateur_id = u.id
				AND bm.code_module = $7
				AND um2.est_actif = TRUE
			))
		ORDER BY 
			CASE WHEN $8 = 'nom' AND $9 = 'asc' THEN u.nom END ASC,
			CASE WHEN $8 = 'nom' AND $9 = 'desc' THEN u.nom END DESC,
			CASE WHEN $8 = 'identifiant' AND $9 = 'asc' THEN u.identifiant END ASC,
			CASE WHEN $8 = 'identifiant' AND $9 = 'desc' THEN u.identifiant END DESC,
			CASE WHEN $8 = 'last_login_at' AND $9 = 'asc' THEN u.last_login_at END ASC,
			CASE WHEN $8 = 'last_login_at' AND $9 = 'desc' THEN u.last_login_at END DESC,
			CASE WHEN $8 = 'created_at' AND $9 = 'asc' THEN u.created_at END ASC,
			CASE WHEN $8 = 'created_at' AND $9 = 'desc' THEN u.created_at END DESC,
			u.created_at DESC
		LIMIT $11 OFFSET $12
	`,

	/**
	 * Compte le nombre total d'utilisateurs avec les filtres appliqués
	 * Paramètres: $1 = etablissement_id, $2 = statut_filter, $3 = search_pattern,
	 *            $4 = est_admin_filter, $5 = est_medecin_filter, $6 = profil_id_filter,
	 *            $7 = module_code_filter, $8 = include_archived
	 */
	CountUsersWithFilters: `
		SELECT COUNT(DISTINCT u.id)
		FROM user_utilisateur u
		LEFT JOIN user_profil_utilisateurs pu ON u.id = pu.utilisateur_id AND pu.est_actif = TRUE
		LEFT JOIN user_modules um ON u.id = um.utilisateur_id AND um.est_actif = TRUE
		LEFT JOIN base_module bm ON um.module_id = bm.id
		WHERE u.etablissement_id = $1
			AND ($2::text IS NULL OR $2 = 'tous' OR u.statut = $2)
			AND ($8::boolean = TRUE OR u.statut != 'archive')
			AND ($3::text IS NULL OR u.identifiant ILIKE '%' || $3 || '%'
				OR u.nom ILIKE '%' || $3 || '%'
				OR u.prenoms ILIKE '%' || $3 || '%')
			AND ($4::boolean IS NULL OR u.est_admin = $4)
			AND ($5::boolean IS NULL OR u.est_medecin = $5)
			AND ($6::uuid IS NULL OR EXISTS (
				SELECT 1 FROM user_profil_utilisateurs
				WHERE utilisateur_id = u.id
				AND profil_template_id = $6
				AND est_actif = TRUE
			))
			AND ($7::text IS NULL OR EXISTS (
				SELECT 1 FROM user_modules um2
				JOIN base_module bm2 ON um2.module_id = bm2.id
				WHERE um2.utilisateur_id = u.id
				AND bm2.code_module = $7
				AND um2.est_actif = TRUE
			))
	`,

	/**
	 * Récupère les détails complets d'un utilisateur
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	GetUserDetails: `
		SELECT 
			u.id,
			u.identifiant,
			u.nom,
			u.prenoms,
			u.telephone,
			u.est_admin,
			u.type_admin,
			u.est_admin_tir,
			u.est_medecin,
			u.role_metier,
			u.photo_url,
			u.est_temporaire,
			u.date_expiration,
			u.statut,
			u.motif_desactivation,
			u.must_change_password,
			u.password_changed_at,
			u.last_login_at,
			u.created_at,
			u.updated_at,
			cb.id as created_by_id,
			cb.nom as created_by_nom,
			cb.prenoms as created_by_prenoms,
			ub.id as updated_by_id,
			ub.nom as updated_by_nom,
			ub.prenoms as updated_by_prenoms
		FROM user_utilisateur u
		LEFT JOIN user_utilisateur cb ON u.created_by = cb.id
		LEFT JOIN user_utilisateur ub ON u.updated_by = ub.id
		WHERE u.etablissement_id = $1 
			AND u.id = $2
			AND u.statut != 'archive'
	`,

	/**
	 * Récupère les profils d'un utilisateur avec détails
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	GetUserProfils: `
		SELECT 
			pt.id,
			pt.nom_profil,
			pt.code_profil,
			pt.description,
			pu.date_attribution,
			pu.est_actif,
			a.id as attribue_par_id,
			a.nom as attribue_par_nom,
			a.prenoms as attribue_par_prenoms
		FROM user_profil_utilisateurs pu
		JOIN user_profil_template pt ON pu.profil_template_id = pt.id
		LEFT JOIN user_utilisateur a ON pu.attribue_par = a.id
		WHERE pu.etablissement_id = $1 
			AND pu.utilisateur_id = $2
			AND pu.est_actif = TRUE
		ORDER BY pt.nom_profil
	`,

	/**
	 * Récupère les modules complets d'un utilisateur avec source
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	GetUserModulesComplets: `
		WITH user_modules_individuels AS (
			SELECT 
				bm.code_module,
				bm.nom_standard,
				bm.nom_personnalise,
				bm.description,
				'individuelle' as source,
				NULL as profil_source,
				um.date_attribution,
				CONCAT(a.prenoms, ' ', a.nom) as attribue_par
			FROM user_modules um
			JOIN base_module bm ON um.module_id = bm.id
			LEFT JOIN user_utilisateur a ON um.attribue_par = a.id
			WHERE um.etablissement_id = $1 
				AND um.utilisateur_id = $2
				AND um.est_actif = TRUE
				AND um.acces_toutes_rubriques = TRUE
		),
		user_modules_profils AS (
			SELECT DISTINCT
				bm.code_module,
				bm.nom_standard,
				bm.nom_personnalise,
				bm.description,
				'profil' as source,
				pt.nom_profil as profil_source,
				pu.date_attribution,
				CONCAT(a.prenoms, ' ', a.nom) as attribue_par
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pu.profil_template_id = pm.profil_template_id
			JOIN base_module bm ON pm.module_id = bm.id
			JOIN user_profil_template pt ON pu.profil_template_id = pt.id
			LEFT JOIN user_utilisateur a ON pu.attribue_par = a.id
			WHERE pu.etablissement_id = $1 
				AND pu.utilisateur_id = $2
				AND pu.est_actif = TRUE
				AND pm.est_actif = TRUE
				AND pm.acces_toutes_rubriques = TRUE
		)
		SELECT * FROM user_modules_individuels
		UNION ALL
		SELECT * FROM user_modules_profils
		ORDER BY code_module, source
	`,

	/**
	 * Récupère les modules partiels d'un utilisateur avec rubriques
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	GetUserModulesPartiels: `
		WITH user_rubriques_individuelles AS (
			SELECT 
				bm.code_module,
				bm.nom_standard,
				bm.nom_personnalise,
				bm.description,
				'individuelle' as source,
				umr.date_attribution,
				CONCAT(a.prenoms, ' ', a.nom) as attribue_par,
				br.code_rubrique,
				br.nom as rubrique_nom,
				br.description as rubrique_description
			FROM user_modules_rubriques umr
			JOIN base_module bm ON umr.module_id = bm.id
			JOIN base_rubrique br ON umr.rubrique_id = br.id
			LEFT JOIN user_utilisateur a ON umr.attribue_par = a.id
			WHERE umr.etablissement_id = $1 
				AND umr.utilisateur_id = $2
				AND umr.est_actif = TRUE
		),
		user_rubriques_profils AS (
			SELECT 
				bm.code_module,
				bm.nom_standard,
				bm.nom_personnalise,
				bm.description,
				'profil' as source,
				pu.date_attribution,
				CONCAT(a.prenoms, ' ', a.nom) as attribue_par,
				br.code_rubrique,
				br.nom as rubrique_nom,
				br.description as rubrique_description
			FROM user_profil_utilisateurs pu
			JOIN user_profil_rubriques pr ON pu.profil_template_id = pr.profil_template_id
			JOIN base_module bm ON pr.module_id = bm.id
			JOIN base_rubrique br ON pr.rubrique_id = br.id
			LEFT JOIN user_utilisateur a ON pu.attribue_par = a.id
			WHERE pu.etablissement_id = $1 
				AND pu.utilisateur_id = $2
				AND pu.est_actif = TRUE
				AND pr.est_actif = TRUE
		)
		SELECT * FROM user_rubriques_individuelles
		UNION ALL
		SELECT * FROM user_rubriques_profils
		ORDER BY code_module, source, code_rubrique
	`,

	/**
	 * Récupère les statistiques d'un utilisateur
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	GetUserStatistiques: `
		WITH login_stats AS (
			SELECT 
				COUNT(*) as connexions_30j
			FROM user_login_attempts ula
			WHERE ula.etablissement_id = $1
				AND EXISTS (
					SELECT 1 FROM user_utilisateur u2 
					WHERE u2.id = $2 AND u2.identifiant = ula.identifiant
				)
				AND ula.success = TRUE
				AND ula.attempted_at >= NOW() - INTERVAL '30 days'
		),
		session_stats AS (
			SELECT COUNT(*) as sessions_actives
			FROM user_session us
			WHERE us.etablissement_id = $1
				AND us.user_id = $2
				AND us.expires_at > NOW()
		),
		permission_stats AS (
			SELECT 
				COUNT(DISTINCT pu.profil_template_id) as permissions_via_profils,
				COUNT(DISTINCT um.id) + COUNT(DISTINCT umr.id) as permissions_individuelles
			FROM user_utilisateur u
			LEFT JOIN user_profil_utilisateurs pu ON u.id = pu.utilisateur_id AND pu.est_actif = TRUE
			LEFT JOIN user_modules um ON u.id = um.utilisateur_id AND um.est_actif = TRUE
			LEFT JOIN user_modules_rubriques umr ON u.id = umr.utilisateur_id AND umr.est_actif = TRUE
			WHERE u.etablissement_id = $1 AND u.id = $2
		)
		SELECT 
			COALESCE(ls.connexions_30j, 0) as nombre_connexions_30j,
			u.last_login_at as derniere_activite,
			COALESCE(ss.sessions_actives, 0) as nombre_sessions_actives,
			COALESCE(ps.permissions_via_profils, 0) as permissions_via_profils,
			COALESCE(ps.permissions_individuelles, 0) as permissions_individuelles
		FROM user_utilisateur u
		CROSS JOIN login_stats ls
		CROSS JOIN session_stats ss  
		CROSS JOIN permission_stats ps
		WHERE u.etablissement_id = $1 AND u.id = $2
	`,

	/**
	 * Valide qu'un utilisateur existe et n'est pas archivé
	 * Paramètres: $1 = etablissement_id, $2 = user_id
	 */
	ValidateUserExists: `
		SELECT EXISTS(
			SELECT 1 FROM user_utilisateur
			WHERE etablissement_id = $1
				AND id = $2
				AND statut != 'archive'
		)
	`,

	/**
	 * Valide qu'un profil existe dans l'établissement
	 * Paramètres: $1 = etablissement_id, $2 = profil_id
	 */
	ValidateProfilExists: `
		SELECT EXISTS(
			SELECT 1 FROM user_profil_template
			WHERE etablissement_id = $1
				AND id = $2
				AND est_actif = TRUE
		)
	`,

	/**
	 * Valide qu'un module existe
	 * Paramètres: $1 = module_id, $2 = est_admin (pour vérifier cohérence back/front office)
	 */
	ValidateModuleExists: `
		SELECT 
			EXISTS(
				SELECT 1 FROM base_module
				WHERE id = $1 AND est_actif = TRUE
			) as module_exists,
			CASE 
				WHEN $2::boolean = TRUE THEN est_module_back_office
				ELSE NOT est_module_back_office
			END as module_type_valid,
			code_module
		FROM base_module
		WHERE id = $1 AND est_actif = TRUE
	`,

	/**
	 * Valide qu'une rubrique existe pour un module donné
	 * Paramètres: $1 = rubrique_id, $2 = module_id
	 */
	ValidateRubriqueExists: `
		SELECT EXISTS(
			SELECT 1 FROM base_rubrique r
			JOIN base_module m ON r.module_id = m.id
			WHERE r.id = $1
				AND r.module_id = $2
				AND r.est_actif = TRUE
				AND m.est_actif = TRUE
		)
	`,

	/**
	 * Ajoute un profil à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, 
	 *            $4 = profil_template_id, $5 = attribue_par
	 */
	AddUserProfil: `
		INSERT INTO user_profil_utilisateurs (
			id, etablissement_id, utilisateur_id, profil_template_id,
			attribue_par, date_attribution, est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), TRUE, NOW(), NOW()
		)
		ON CONFLICT (etablissement_id, utilisateur_id, profil_template_id, est_actif)
		WHERE est_actif = TRUE
		DO NOTHING
	`,

	/**
	 * Retire un profil d'un utilisateur (soft delete)
	 * Paramètres: $1 = etablissement_id, $2 = utilisateur_id, $3 = profil_template_id
	 */
	RemoveUserProfil: `
		UPDATE user_profil_utilisateurs
		SET est_actif = FALSE, updated_at = NOW()
		WHERE etablissement_id = $1
			AND utilisateur_id = $2
			AND profil_template_id = $3
			AND est_actif = TRUE
	`,

	/**
	 * Ajoute un module complet à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, 
	 *            $4 = module_id, $5 = attribue_par
	 */
	AddUserModuleComplet: `
		INSERT INTO user_modules (
			id, etablissement_id, utilisateur_id, module_id,
			acces_toutes_rubriques, source_attribution, attribue_par,
			date_attribution, est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, TRUE, 'individuelle', $5, NOW(), TRUE, NOW(), NOW()
		)
		ON CONFLICT (utilisateur_id, module_id, est_actif)
		WHERE est_actif = TRUE
		DO NOTHING
	`,

	/**
	 * Retire un module complet d'un utilisateur (soft delete)
	 * Paramètres: $1 = etablissement_id, $2 = utilisateur_id, $3 = module_id
	 */
	RemoveUserModuleComplet: `
		UPDATE user_modules
		SET est_actif = FALSE, updated_at = NOW()
		WHERE etablissement_id = $1
			AND utilisateur_id = $2
			AND module_id = $3
			AND est_actif = TRUE
			AND acces_toutes_rubriques = TRUE
	`,

	/**
	 * Ajoute une rubrique spécifique à un utilisateur
	 * Paramètres: $1 = id, $2 = etablissement_id, $3 = utilisateur_id, 
	 *            $4 = module_id, $5 = rubrique_id, $6 = attribue_par
	 */
	AddUserModulePartiel: `
		INSERT INTO user_modules_rubriques (
			id, etablissement_id, utilisateur_id, module_id, rubrique_id,
			source_attribution, attribue_par, date_attribution,
			est_actif, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'individuelle', $6, NOW(), TRUE, NOW(), NOW()
		)
		ON CONFLICT (utilisateur_id, rubrique_id, est_actif)
		WHERE est_actif = TRUE
		DO NOTHING
	`,

	/**
	 * Retire toutes les rubriques d'un module partiel pour un utilisateur
	 * Paramètres: $1 = etablissement_id, $2 = utilisateur_id, $3 = module_id
	 */
	RemoveUserModulePartiel: `
		UPDATE user_modules_rubriques
		SET est_actif = FALSE, updated_at = NOW()
		WHERE etablissement_id = $1
			AND utilisateur_id = $2
			AND module_id = $3
			AND est_actif = TRUE
	`,

	/**
	 * Met à jour les rubriques d'un module partiel (retire d'abord toutes, puis ajoute les nouvelles)
	 * Cette requête fait partie d'une transaction avec RemoveUserModulePartiel et AddUserModulePartiel
	 */
	UpdateUserModulePartiel: `
		-- Cette requête est utilisée dans une transaction avec les autres
		SELECT 1
	`,

	/**
	 * Vérifie si une permission existe déjà pour éviter les doublons
	 * Paramètres: $1 = etablissement_id, $2 = utilisateur_id, $3 = type (profil/module/rubrique), 
	 *            $4 = entity_id, $5 = module_id (optionnel pour rubriques)
	 */
	CheckUserPermissionExists: `
		SELECT 
			CASE 
				WHEN $3 = 'profil' THEN EXISTS(
					SELECT 1 FROM user_profil_utilisateurs
					WHERE etablissement_id = $1
						AND utilisateur_id = $2
						AND profil_template_id = $4::uuid
						AND est_actif = TRUE
				)
				WHEN $3 = 'module' THEN EXISTS(
					SELECT 1 FROM user_modules
					WHERE etablissement_id = $1
						AND utilisateur_id = $2
						AND module_id = $4::uuid
						AND est_actif = TRUE
				)
				WHEN $3 = 'rubrique' THEN EXISTS(
					SELECT 1 FROM user_modules_rubriques
					WHERE etablissement_id = $1
						AND utilisateur_id = $2
						AND rubrique_id = $4::uuid
						AND est_actif = TRUE
				)
				ELSE FALSE
			END as permission_exists
	`,
}
