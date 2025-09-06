package queries

// EstablishmentQueries regroupe toutes les requêtes SQL pour la validation d'établissement
var EstablishmentQueries = struct {
	GetByCode string
}{
	/**
	 * Récupère les informations critiques d'un établissement par son code
	 * Optimisé pour cache middleware - données immuables uniquement
	 * Paramètres: $1 = establishment_code
	 */
	GetByCode: `
		SELECT 
			id,
			app_instance,
			code_etablissement,
			statut
		FROM base_etablissement 
		WHERE code_etablissement = $1
		  AND statut IN ('actif', 'suspendu')
	`,
}
