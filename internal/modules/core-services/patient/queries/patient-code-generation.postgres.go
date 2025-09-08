package queries

// PatientCodeGenerationQueries regroupe toutes les requêtes SQL pour la génération de codes patient
var PatientCodeGenerationQueries = struct {
	GetSequenceState                 string
	GenerateNextCodeFromPostgres     string
	InitializeYearlySequence        string
	UpdateSequenceAfterGeneration   string
}{
	/**
	 * Récupère l'état actuel de la séquence pour un établissement/année
	 * Paramètres: $1 = etablissement_code, $2 = annee
	 * Retour: dernier_numero, dernier_suffixe, nombre_generes
	 */
	GetSequenceState: `
		SELECT 
			dernier_numero,
			dernier_suffixe,
			nombre_generes
		FROM patients_code_sequences 
		WHERE etablissement_code = $1 AND annee = $2
	`,

	/**
	 * Génère le prochain code patient de manière atomique avec PostgreSQL
	 * Utilise UPSERT pour gérer l'initialisation + incrémentation en une seule opération
	 * Paramètres: $1 = etablissement_code, $2 = annee
	 * Retour: nouveau_numero, nouveau_suffixe, nombre_total_generes
	 */
	GenerateNextCodeFromPostgres: `
		INSERT INTO patients_code_sequences (etablissement_code, annee, dernier_numero, dernier_suffixe, nombre_generes)
		VALUES ($1, $2, 1, 'AAA', 1)
		ON CONFLICT (etablissement_code, annee)
		DO UPDATE SET
			dernier_numero = CASE
				WHEN patients_code_sequences.dernier_numero = 999 THEN 1
				ELSE patients_code_sequences.dernier_numero + 1
			END,
			dernier_suffixe = CASE
				WHEN patients_code_sequences.dernier_numero = 999
				THEN next_alpha_suffix(patients_code_sequences.dernier_suffixe)
				ELSE patients_code_sequences.dernier_suffixe
			END,
			nombre_generes = patients_code_sequences.nombre_generes + 1,
			updated_at = NOW()
		RETURNING dernier_numero, dernier_suffixe, nombre_generes
	`,

	/**
	 * Initialise une nouvelle séquence pour une année donnée
	 * Paramètres: $1 = etablissement_code, $2 = annee
	 * Retour: confirmation d'insertion
	 */
	InitializeYearlySequence: `
		INSERT INTO patients_code_sequences (etablissement_code, annee, dernier_numero, dernier_suffixe, nombre_generes)
		VALUES ($1, $2, 0, 'AAA', 0)
		ON CONFLICT (etablissement_code, annee) DO NOTHING
		RETURNING id, etablissement_code, annee
	`,

	/**
	 * Met à jour la séquence après génération réussie (utilisé pour synchronisation Redis)
	 * Paramètres: $1 = etablissement_code, $2 = annee, $3 = nouveau_numero, $4 = nouveau_suffixe
	 * Retour: confirmation mise à jour
	 */
	UpdateSequenceAfterGeneration: `
		UPDATE patients_code_sequences 
		SET 
			dernier_numero = $3,
			dernier_suffixe = $4,
			nombre_generes = nombre_generes + 1,
			updated_at = NOW()
		WHERE etablissement_code = $1 AND annee = $2
		RETURNING dernier_numero, dernier_suffixe, nombre_generes
	`,
}