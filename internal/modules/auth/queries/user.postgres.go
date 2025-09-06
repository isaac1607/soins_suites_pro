package queries

// UserQueries regroupe toutes les requêtes SQL pour la gestion des utilisateurs
var UserQueries = struct {
	GetByIdentifiant           string
	GetUserPermissions         string
	GetSetupState              string
	CreateSession              string
	GetSessionByToken          string
	DeleteSession              string
	GetActiveSessionsByUserID  string
	CleanExpiredSessions       string
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
		WITH user_modules_access AS (
			-- Modules via profils templates
			SELECT DISTINCT
				m.id, m.code_module, m.nom_standard, m.nom_personnalise,
				m.description, pm.acces_toutes_rubriques, 'profil' as source
			FROM user_profil_utilisateurs pu
			JOIN user_profil_modules pm ON pm.profil_template_id = pu.profil_template_id
			JOIN base_module m ON m.id = pm.module_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND m.est_actif = TRUE
		
			UNION
		
			-- Modules directs utilisateur
			SELECT DISTINCT
				m.id, m.code_module, m.nom_standard, m.nom_personnalise,
				m.description, um.acces_toutes_rubriques, 'direct' as source
			FROM user_modules um
			JOIN base_module m ON m.id = um.module_id
			WHERE um.utilisateur_id = $1 
			  AND um.etablissement_id = $2
			  AND um.est_actif = TRUE
			  AND m.est_actif = TRUE
		),
		user_rubriques_access AS (
			-- Rubriques via profils templates (quand acces_toutes_rubriques = FALSE)
			SELECT pr.module_id, r.id, r.code_rubrique, r.nom,
				   r.description, r.ordre_affichage
			FROM user_profil_utilisateurs pu
			JOIN user_profil_rubriques pr ON pr.profil_template_id = pu.profil_template_id
			JOIN base_rubrique r ON r.id = pr.rubrique_id
			WHERE pu.utilisateur_id = $1 
			  AND pu.etablissement_id = $2
			  AND pu.est_actif = TRUE
			  AND r.est_actif = TRUE
		
			UNION
		
			-- Rubriques directes utilisateur
			SELECT umr.module_id, r.id, r.code_rubrique, r.nom,
				   r.description, r.ordre_affichage
			FROM user_modules_rubriques umr
			JOIN base_rubrique r ON r.id = umr.rubrique_id
			WHERE umr.utilisateur_id = $1 
			  AND umr.etablissement_id = $2
			  AND umr.est_actif = TRUE
			  AND r.est_actif = TRUE
		)
		SELECT
			m.code_module,
			m.nom_standard,
			m.nom_personnalise,
			m.description,
			CASE
				-- Si au moins un accès donne toutes les rubriques, alors accès complet
				WHEN bool_or(m.acces_toutes_rubriques) THEN '[]'::jsonb
				-- Sinon, agréger les rubriques spécifiques
				ELSE COALESCE(
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
				)
			END as rubriques
		FROM user_modules_access m
		LEFT JOIN user_rubriques_access r ON r.module_id = m.id
			AND NOT bool_or(m.acces_toutes_rubriques) -- Joint les rubriques seulement si pas d'accès complet
		GROUP BY m.code_module, m.nom_standard, m.nom_personnalise, m.description
		ORDER BY m.code_module
	`,

	/**
	 * Récupère l'état du setup pour un établissement (back-office uniquement)
	 * Paramètres: $1 = etablissement_id
	 */
	GetSetupState: `
		SELECT 
			setup_est_termine,
			setup_etape
		FROM base_etablissement
		WHERE id = $1
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