package queries

// PatientDetailQueries contient toutes les requêtes SQL pour la récupération détaillée de patients
var PatientDetailQueries = struct {
	GetPatientByCodeWithAllDetails     string
	GetPatientReferencesEnriched       string  
	GetPatientAssurancesDetailed       string
	GetPatientPersonnesAContacter      string
	CheckPatientExists                 string
	UpdatePatientLastAccessed          string
}{
	// GetPatientByCodeWithAllDetails - Chargement patient complet avec références enrichies (requête principale)
	GetPatientByCodeWithAllDetails: `
		WITH patient_data AS (
			SELECT
				-- Patient principal
				p.id,
				p.code_patient,
				p.etablissement_createur_id,
				p.nom,
				p.prenoms,
				p.date_naissance,
				p.est_date_supposee,
				p.sexe,
				p.nationalite_id,
				p.situation_matrimoniale_id,
				p.type_piece_identite_id,
				p.cni_nni,
				p.numero_piece_identite,
				p.lieu_naissance,
				p.nom_jeune_fille,
				p.telephone_principal,
				p.telephone_secondaire,
				p.email,
				p.adresse_complete,
				p.quartier,
				p.ville,
				p.commune,
				p.pays_residence,
				p.profession_id,
				p.personnes_a_contacter,
				p.est_assure,
				p.statut,
				p.est_decede,
				p.date_deces,
				p.created_at,
				p.updated_at,
				p.created_by,
				p.updated_by,
				
				-- Références enrichies
				n.id as nationalite_ref_id,
				n.code as nationalite_code,
				n.nom as nationalite_nom,
				sm.id as situation_ref_id,
				sm.code as situation_code,
				sm.nom as situation_nom,
				tpi.id as piece_ref_id,
				tpi.code as piece_code,
				tpi.nom as piece_nom,
				prof.id as profession_ref_id,
				prof.code as profession_code,
				prof.nom as profession_nom,
				
				-- Utilisateurs (créateur/modificateur)
				cb.id as created_by_id,
				cb.nom as created_by_nom,
				cb.prenoms as created_by_prenoms,
				ub.id as updated_by_id,
				ub.nom as updated_by_nom,
				ub.prenoms as updated_by_prenoms
				
			FROM patients_patient p
			LEFT JOIN ref_nationalite n ON p.nationalite_id = n.id
			LEFT JOIN ref_situation_matrimoniale sm ON p.situation_matrimoniale_id = sm.id
			LEFT JOIN ref_type_piece_identite tpi ON p.type_piece_identite_id = tpi.id
			LEFT JOIN ref_profession prof ON p.profession_id = prof.id
			LEFT JOIN user_utilisateur cb ON p.created_by = cb.id
			LEFT JOIN user_utilisateur ub ON p.updated_by = ub.id
			WHERE p.code_patient = $1 
			AND ($2::boolean = true OR p.statut != 'archive')
		),
		patient_assurances AS (
			SELECT
				pa.patient_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pa.id,
						'assurance_id', a.id,
						'assurance_code', COALESCE(a.code_assurance, ''),
						'assurance_nom', COALESCE(a.nom_assurance, 'Assurance inconnue'),
						'numero_assure', pa.numero_assure,
						'type_beneficiaire', pa.type_beneficiaire,
						'numero_assure_principal', pa.numero_assure_principal,
						'lien_avec_principal', pa.lien_avec_principal,
						'est_actif', pa.est_actif,
						'created_at', pa.created_at
					) ORDER BY pa.created_at
				) as assurances_details
			FROM patients_patient_assurance pa
			LEFT JOIN base_assurance a ON pa.assurance_id = a.id
			WHERE pa.patient_id = (SELECT id FROM patient_data)
			AND pa.est_actif = true
			GROUP BY pa.patient_id
		)
		SELECT
			-- Données patient principales
			pd.id,
			pd.code_patient,
			pd.etablissement_createur_id,
			pd.nom,
			pd.prenoms,
			pd.date_naissance,
			pd.est_date_supposee,
			pd.sexe,
			pd.nationalite_id,
			pd.situation_matrimoniale_id,
			pd.type_piece_identite_id,
			pd.cni_nni,
			pd.numero_piece_identite,
			pd.lieu_naissance,
			pd.nom_jeune_fille,
			pd.telephone_principal,
			pd.telephone_secondaire,
			pd.email,
			pd.adresse_complete,
			pd.quartier,
			pd.ville,
			pd.commune,
			pd.pays_residence,
			pd.profession_id,
			pd.personnes_a_contacter,
			pd.est_assure,
			pd.statut,
			pd.est_decede,
			pd.date_deces,
			pd.created_at,
			pd.updated_at,
			pd.created_by,
			pd.updated_by,
			
			-- Références enrichies - Nationalité
			pd.nationalite_ref_id,
			pd.nationalite_code,
			pd.nationalite_nom,
			
			-- Références enrichies - Situation matrimoniale
			pd.situation_ref_id,
			pd.situation_code,
			pd.situation_nom,
			
			-- Références enrichies - Type pièce identité (peut être NULL)
			pd.piece_ref_id,
			pd.piece_code,
			pd.piece_nom,
			
			-- Références enrichies - Profession (peut être NULL)
			pd.profession_ref_id,
			pd.profession_code,
			pd.profession_nom,
			
			-- Utilisateurs
			pd.created_by_id,
			pd.created_by_nom,
			pd.created_by_prenoms,
			pd.updated_by_id,
			pd.updated_by_nom,
			pd.updated_by_prenoms,
			
			-- Assurances JSON
			COALESCE(pa.assurances_details, '[]'::jsonb) as assurances_details
			
		FROM patient_data pd
		LEFT JOIN patient_assurances pa ON pd.id = pa.patient_id;
	`,

	// GetPatientReferencesEnriched - Récupère uniquement les références enrichies pour un patient
	GetPatientReferencesEnriched: `
		SELECT
			-- Nationalité
			n.id as nationalite_id,
			n.code as nationalite_code, 
			n.nom as nationalite_nom,
			
			-- Situation matrimoniale
			sm.id as situation_id,
			sm.code as situation_code,
			sm.nom as situation_nom,
			
			-- Type pièce identité (peut être NULL)
			tpi.id as piece_id,
			tpi.code as piece_code,
			tpi.nom as piece_nom,
			
			-- Profession (peut être NULL)
			prof.id as profession_id,
			prof.code as profession_code,
			prof.nom as profession_nom
			
		FROM patients_patient p
		LEFT JOIN ref_nationalite n ON p.nationalite_id = n.id
		LEFT JOIN ref_situation_matrimoniale sm ON p.situation_matrimoniale_id = sm.id
		LEFT JOIN ref_type_piece_identite tpi ON p.type_piece_identite_id = tpi.id
		LEFT JOIN ref_profession prof ON p.profession_id = prof.id
		WHERE p.code_patient = $1;
	`,

	// GetPatientAssurancesDetailed - Récupère les assurances détaillées d'un patient
	GetPatientAssurancesDetailed: `
		SELECT
			pa.id,
			a.id as assurance_id,
			COALESCE(a.code_assurance, '') as assurance_code,
			COALESCE(a.nom_assurance, 'Assurance inconnue') as assurance_nom,
			pa.numero_assure,
			pa.type_beneficiaire,
			pa.numero_assure_principal,
			pa.lien_avec_principal,
			pa.est_actif,
			pa.created_at
		FROM patients_patient_assurance pa
		LEFT JOIN base_assurance a ON pa.assurance_id = a.id
		JOIN patients_patient p ON pa.patient_id = p.id
		WHERE p.code_patient = $1
		AND pa.est_actif = true
		ORDER BY pa.created_at;
	`,

	// GetPatientPersonnesAContacter - Parse les personnes à contacter avec références
	GetPatientPersonnesAContacter: `
		WITH personnes_data AS (
			SELECT 
				p.code_patient,
				jsonb_array_elements(p.personnes_a_contacter) as personne
			FROM patients_patient p
			WHERE p.code_patient = $1
			AND jsonb_array_length(p.personnes_a_contacter) > 0
		)
		SELECT
			pd.code_patient,
			pd.personne->>'nom_prenoms' as nom_prenoms,
			pd.personne->>'telephone' as telephone,
			pd.personne->>'telephone_secondaire' as telephone_secondaire,
			(pd.personne->>'affiliation_id')::uuid as affiliation_id,
			a.code as affiliation_code,
			a.nom as affiliation_nom
		FROM personnes_data pd
		LEFT JOIN ref_affiliation a ON (pd.personne->>'affiliation_id')::uuid = a.id;
	`,

	// CheckPatientExists - Vérifie l'existence d'un patient avec son statut
	CheckPatientExists: `
		SELECT 
			id,
			statut,
			est_decede,
			updated_at
		FROM patients_patient 
		WHERE code_patient = $1;
	`,

	// UpdatePatientLastAccessed - Met à jour la dernière consultation (audit)
	UpdatePatientLastAccessed: `
		UPDATE patients_patient 
		SET updated_at = NOW() 
		WHERE code_patient = $1 
		RETURNING updated_at;
	`,
}