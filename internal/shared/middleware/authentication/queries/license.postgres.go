package queries

// LicenseQueries regroupe toutes les requêtes SQL pour la validation de licence
var LicenseQueries = struct {
	GetByEstablishmentID string
}{
	/**
	 * Récupère les informations critiques de licence par ID établissement
	 * Optimisé pour cache middleware - données essentielles uniquement
	 * Paramètres: $1 = etablissement_id
	 */
	GetByEstablishmentID: `
		SELECT 
			id,
			type_licence,
			mode_deploiement,
			statut,
			COALESCE(modules_autorises::text, '[]'),
			date_expiration
		FROM base_licence 
		WHERE etablissement_id = $1
		  AND statut IN ('actif', 'expiree')
		ORDER BY created_at DESC
		LIMIT 1
	`,
}
