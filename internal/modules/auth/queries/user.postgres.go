package queries

// UserQueries regroupe toutes les requêtes SQL pour la gestion des utilisateurs
var UserQueries = struct {
	GetByIdentifiant          string
	GetUserPermissions        string
	GetSuperAdminPermissions  string
	CheckUserPermission       string
	GetSetupState             string
	ChangePassword            string
	CreateSession             string
	GetSessionByToken         string
	DeleteSession             string
	GetActiveSessionsByUserID string
	CleanExpiredSessions      string
}{
	/**
	 * Récupère un utilisateur par identifiant et établissement
	 * Paramètres: $1 = identifiant, $2 = etablissement_id
	 */
	GetByIdentifiant: `
		SELECT 
			u.id,
			u.identifiant,
			u.nom,
			u.prenoms,
			u.telephone,
			u.password_hash,
			u.salt,
			u.est_admin,
			u.type_admin,
			u.est_admin_tir,
			u.must_change_password,
			u.est_medecin,
			u.role_metier,
			u.statut,
			e.code_etablissement
		FROM user_utilisateur u
		JOIN base_etablissement e ON u.etablissement_id = e.id
		WHERE u.identifiant = $1 
		  AND u.etablissement_id = $2
		  AND u.statut = 'actif'
	`,

	/**
	 * Récupère toutes les permissions d'un utilisateur avec CTEs optimisées
	 * Combine permissions via profils et permissions directes
	 * Paramètres: $1 = user_id, $2 = etablissement_id
	 */
	GetUserPermissions: `
		WITH modules_via_profils AS (
			-- Modules via profils avec leur niveau d'accès
			SELECT DISTINCT
				m.id, m.code_module, m.nom_standard, m.nom_personnalise, m.description,
				pm.acces_toutes_rubriques
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pm.profil_template_id = pu.profil_template_id
			JOIN base_module m ON m.id = pm.module_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND pm.est_actif = TRUE
			  AND m.est_actif = TRUE
		),
		modules_directs AS (
			-- Modules directs avec leur niveau d'accès
			SELECT DISTINCT
				m.id, m.code_module, m.nom_standard, m.nom_personnalise, m.description,
				um.acces_toutes_rubriques
			FROM user_modules um
			JOIN base_module m ON m.id = um.module_id
			WHERE um.utilisateur_id = $1 
			  AND um.etablissement_id = $2
			  AND um.est_actif = TRUE
			  AND m.est_actif = TRUE
		),
		tous_modules AS (
			-- Union de tous les modules avec consolidation des accès
			SELECT 
				id, code_module, nom_standard, nom_personnalise, description,
				bool_or(acces_toutes_rubriques) as acces_toutes_rubriques
			FROM (
				SELECT * FROM modules_via_profils
				UNION ALL
				SELECT * FROM modules_directs
			) all_modules
			GROUP BY id, code_module, nom_standard, nom_personnalise, description
		),
		rubriques_specifiques AS (
			-- Rubriques spécifiques via profils (seulement si pas d'accès complet)
			SELECT pr.module_id, r.code_rubrique, r.nom, r.description, r.ordre_affichage
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pm.profil_template_id = pu.profil_template_id
			JOIN user_profil_rubriques pr ON pr.profil_template_id = pu.profil_template_id AND pr.module_id = pm.module_id
			JOIN base_rubrique r ON r.id = pr.rubrique_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND pm.est_actif = TRUE
			  AND pr.est_actif = TRUE
			  AND r.est_actif = TRUE
			  AND pm.acces_toutes_rubriques = FALSE
		
			UNION
		
			-- Rubriques spécifiques directes (seulement si pas d'accès complet)
			SELECT umr.module_id, r.code_rubrique, r.nom, r.description, r.ordre_affichage
			FROM user_modules um
			JOIN user_modules_rubriques umr ON umr.utilisateur_id = um.utilisateur_id AND umr.module_id = um.module_id
			JOIN base_rubrique r ON r.id = umr.rubrique_id
			WHERE um.utilisateur_id = $1 
			  AND um.etablissement_id = $2
			  AND um.est_actif = TRUE
			  AND umr.est_actif = TRUE
			  AND r.est_actif = TRUE
			  AND um.acces_toutes_rubriques = FALSE
		)
		SELECT
			m.id::text,
			m.code_module,
			m.nom_standard,
			m.nom_personnalise,
			m.description,
			m.acces_toutes_rubriques,
			CASE
				-- Si accès complet, récupérer TOUTES les rubriques du module
				WHEN m.acces_toutes_rubriques = TRUE THEN (
					SELECT COALESCE(
						jsonb_agg(
							jsonb_build_object(
								'code_rubrique', br.code_rubrique,
								'nom', br.nom,
								'description', br.description,
								'ordre_affichage', br.ordre_affichage
							) ORDER BY br.ordre_affichage
						),
						'[]'::jsonb
					)
					FROM base_rubrique br 
					WHERE br.module_id = m.id AND br.est_actif = TRUE
				)
				-- Sinon, récupérer seulement les rubriques spécifiquement accordées
				ELSE COALESCE(
					(
						SELECT jsonb_agg(
							jsonb_build_object(
								'code_rubrique', rs.code_rubrique,
								'nom', rs.nom,
								'description', rs.description,
								'ordre_affichage', rs.ordre_affichage
							) ORDER BY rs.ordre_affichage
						)
						FROM rubriques_specifiques rs
						WHERE rs.module_id = m.id
					),
					'[]'::jsonb
				)
			END as rubriques
		FROM tous_modules m
		ORDER BY m.code_module
	`,

	/**
	 * Récupère tous les modules back-office pour super admin
	 * Paramètres: aucun
	 */
	GetSuperAdminPermissions: `
		SELECT
			m.id,
			m.code_module,
			m.nom_standard,
			m.nom_personnalise,
			m.description,
			TRUE as acces_toutes_rubriques,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'code_rubrique', r.code_rubrique,
						'nom', r.nom,
						'description', r.description,
						'ordre_affichage', r.ordre_affichage
					) ORDER BY jsonb_build_object(
						'code_rubrique', r.code_rubrique,
						'nom', r.nom,
						'description', r.description,
						'ordre_affichage', r.ordre_affichage
					)
				) FILTER (WHERE r.id IS NOT NULL),
				'[]'::jsonb
			) as rubriques
		FROM base_module m
		LEFT JOIN base_rubrique r ON r.module_id = m.id AND r.est_actif = TRUE
		WHERE m.est_actif = TRUE 
		  AND m.est_module_back_office = TRUE
		GROUP BY m.id, m.code_module, m.nom_standard, m.nom_personnalise, m.description
		ORDER BY m.code_module
	`,

	/**
	 * Vérifie si un utilisateur a accès à un module/rubrique spécifique
	 * Paramètres: $1 = user_id, $2 = etablissement_id, $3 = code_module, $4 = code_rubrique (optionnel)
	 */
	CheckUserPermission: `
		WITH user_modules_access AS (
			-- Modules via profils
			SELECT DISTINCT 
				m.code_module, 
				pm.acces_toutes_rubriques
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pm.profil_template_id = pu.profil_template_id
			JOIN base_module m ON m.id = pm.module_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND pm.est_actif = TRUE
			  AND m.est_actif = TRUE
			  AND m.code_module = $3

			UNION

			-- Modules directs
			SELECT DISTINCT 
				m.code_module, 
				um.acces_toutes_rubriques
			FROM user_modules um
			JOIN base_module m ON m.id = um.module_id
			WHERE um.utilisateur_id = $1 
			  AND um.etablissement_id = $2
			  AND um.est_actif = TRUE
			  AND m.est_actif = TRUE
			  AND m.code_module = $3
		),
		user_rubriques_access AS (
			-- Rubriques via profils (si pas d'accès complet module)
			SELECT DISTINCT r.code_rubrique
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pm.profil_template_id = pu.profil_template_id
			JOIN user_profil_rubriques pr ON pr.profil_template_id = pu.profil_template_id AND pr.module_id = pm.module_id
			JOIN base_module m ON m.id = pm.module_id
			JOIN base_rubrique r ON r.id = pr.rubrique_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND pm.est_actif = TRUE
			  AND pr.est_actif = TRUE
			  AND r.est_actif = TRUE
			  AND m.code_module = $3
			  AND ($4 = '' OR r.code_rubrique = $4)
			  AND pm.acces_toutes_rubriques = FALSE

			UNION

			-- Rubriques directes (si pas d'accès complet module)
			SELECT DISTINCT r.code_rubrique
			FROM user_modules um
			JOIN user_modules_rubriques umr ON umr.utilisateur_id = um.utilisateur_id AND umr.module_id = um.module_id
			JOIN base_module m ON m.id = um.module_id
			JOIN base_rubrique r ON r.id = umr.rubrique_id
			WHERE um.utilisateur_id = $1 
			  AND um.etablissement_id = $2
			  AND um.est_actif = TRUE
			  AND umr.est_actif = TRUE
			  AND r.est_actif = TRUE
			  AND m.code_module = $3
			  AND ($4 = '' OR r.code_rubrique = $4)
			  AND um.acces_toutes_rubriques = FALSE
		)
		SELECT 
			CASE 
				-- Si l'utilisateur a un accès complet au module
				WHEN EXISTS (
					SELECT 1 FROM user_modules_access 
					WHERE code_module = $3 AND acces_toutes_rubriques = TRUE
				) THEN TRUE
				-- Sinon, vérifier si pas de rubrique demandée (accès module seul)
				WHEN $4 = '' THEN FALSE
				-- Sinon, vérifier l'accès à la rubrique spécifique
				ELSE EXISTS (
					SELECT 1 FROM user_rubriques_access 
					WHERE code_rubrique = $4
				)
			END as has_access
	`,

	/**
	 * Change le mot de passe d'un utilisateur
	 * Paramètres: $1 = new_password_hash, $2 = new_salt, $3 = user_id, $4 = etablissement_id
	 */
	ChangePassword: `
		UPDATE user_utilisateur 
		SET 
			password_hash = $1,
			salt = $2,
			must_change_password = FALSE,
			password_changed_at = NOW(),
			updated_at = NOW()
		WHERE id = $3 
		  AND etablissement_id = $4 
		  AND statut = 'actif'
		RETURNING id, must_change_password, password_changed_at
	`,

	/**
	 * Crée une nouvelle session dans PostgreSQL (fallback)
	 * Paramètres: $1 = token, $2 = user_id, $3 = etablissement_id, $4 = client_type,
	 *             $5 = ip_address, $6 = user_agent, $7 = expires_at
	 */
	CreateSession: `
		INSERT INTO user_session (
			token, user_id, etablissement_id, client_type, 
			ip_address, user_agent, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (token) DO UPDATE SET
			last_activity = NOW(),
			updated_at = NOW()
	`,

	/**
	 * Récupère une session par token (fallback PostgreSQL)
	 * Paramètres: $1 = token
	 */
	GetSessionByToken: `
		SELECT 
			s.user_id,
			s.etablissement_id,
			s.client_type,
			s.ip_address,
			s.user_agent,
			s.created_at,
			s.last_activity,
			s.expires_at,
			e.code_etablissement
		FROM user_session s
		JOIN base_etablissement e ON s.etablissement_id = e.id
		WHERE s.token = $1 
		  AND s.expires_at > NOW()
	`,

	/**
	 * Supprime une session par token (fallback PostgreSQL)
	 * Paramètres: $1 = token
	 */
	DeleteSession: `
		DELETE FROM user_session 
		WHERE token = $1
	`,

	/**
	 * Récupère toutes les sessions actives d'un utilisateur
	 * Paramètres: $1 = user_id
	 */
	GetActiveSessionsByUserID: `
		SELECT token 
		FROM user_session 
		WHERE user_id = $1 
		  AND expires_at > NOW()
	`,

	/**
	 * Nettoie les sessions expirées
	 * Paramètres: aucun
	 */
	CleanExpiredSessions: `
		DELETE FROM user_session 
		WHERE expires_at <= NOW()
	`,
}
