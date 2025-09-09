package queries

// TIRAuthQueries regroupe toutes les requêtes SQL pour l'authentification TIR
var TIRAuthQueries = struct {
	GetAdminByIdentifiant string
	CreateSession         string
	GetSessionByToken     string
	UpdateLastActivity    string
	DeleteSession         string
	CleanupExpiredSessions string
}{
	/**
	 * Récupère un admin TIR par identifiant avec ses permissions
	 * Paramètres: $1 = identifiant
	 */
	GetAdminByIdentifiant: `
		SELECT 
			id, identifiant, nom, prenoms, email, 
			password_hash, salt, niveau_admin,
			peut_gerer_licences, peut_gerer_etablissements,
			peut_acceder_donnees_etablissement, peut_gerer_admins_globaux,
			statut, must_change_password, last_login_at
		FROM tir_admin_global 
		WHERE identifiant = $1 AND statut = 'actif'
	`,

	/**
	 * Crée une nouvelle session admin TIR
	 * Paramètres: $1 = token, $2 = admin_id, $3 = ip_address, $4 = user_agent, $5 = expires_at
	 */
	CreateSession: `
		INSERT INTO tir_admin_session (
			token, admin_id, ip_address, user_agent, last_activity, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, NOW(), $5, NOW(), NOW()
		)
	`,

	/**
	 * Récupère une session par token avec les infos admin
	 * Paramètres: $1 = token
	 */
	GetSessionByToken: `
		SELECT 
			s.token, s.admin_id, s.ip_address, s.user_agent, 
			s.last_activity, s.expires_at, s.created_at,
			a.identifiant, a.niveau_admin,
			a.peut_gerer_licences, a.peut_gerer_etablissements,
			a.peut_acceder_donnees_etablissement, a.peut_gerer_admins_globaux
		FROM tir_admin_session s
		JOIN tir_admin_global a ON s.admin_id = a.id
		WHERE s.token = $1 AND s.expires_at > NOW() AND a.statut = 'actif'
	`,

	/**
	 * Met à jour la dernière activité d'une session
	 * Paramètres: $1 = token
	 */
	UpdateLastActivity: `
		UPDATE tir_admin_session 
		SET last_activity = NOW(), updated_at = NOW()
		WHERE token = $1
	`,

	/**
	 * Supprime une session par token (logout)
	 * Paramètres: $1 = token
	 */
	DeleteSession: `
		DELETE FROM tir_admin_session 
		WHERE token = $1
	`,

	/**
	 * Nettoie les sessions expirées (tâche de maintenance)
	 * Paramètres: aucun
	 */
	CleanupExpiredSessions: `
		DELETE FROM tir_admin_session 
		WHERE expires_at < NOW()
	`,
}