package queries

// PatientCreationQueries contient toutes les requêtes SQL pour la création de patients
var PatientCreationQueries = struct {
	CreatePatientWithValidation                  string
	CheckDuplicateWithScoring                   string
	GetPatientByCodeWithAssurances              string
	InsertPatientAssurances                     string
	ValidateReferenceDataExists                 string
	GetLastPatientCodeForEstablishment          string
	UpdatePatientSearchVector                   string
}{
	// CreatePatientWithValidation - Création complète d'un patient avec transaction
	CreatePatientWithValidation: `
		INSERT INTO patients_patient (
			id, code_patient, etablissement_createur_id,
			nom, prenoms, date_naissance, est_date_supposee, sexe,
			nationalite_id, situation_matrimoniale_id, type_piece_identite_id,
			cni_nni, numero_piece_identite, lieu_naissance, nom_jeune_fille,
			telephone_principal, telephone_secondaire, email,
			adresse_complete, quartier, ville, commune, pays_residence,
			profession_id, personnes_a_contacter,
			est_assure, statut, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, NOW(), NOW()
		) RETURNING id, code_patient, nom, prenoms, date_naissance, sexe, 
		           telephone_principal, adresse_complete, est_assure, statut, created_at;
	`,

	// CheckDuplicateWithScoring - Vérification intelligente des doublons avec scoring
	CheckDuplicateWithScoring: `
		WITH patient_scores AS (
			SELECT
				id,
				code_patient,
				nom,
				prenoms,
				date_naissance,
				sexe,
				telephone_principal,
				adresse_complete,
				est_assure,
				statut,
				created_at,
				-- Score textuel nom/prenoms avec normalisation
				GREATEST(
					similarity(unaccent(LOWER(nom)), unaccent(LOWER($1))),
					similarity(unaccent(LOWER(prenoms)), unaccent(LOWER($2)))
				) * 100 as score_textuel,
				-- Score date de naissance
				CASE
					WHEN date_naissance = $3::date THEN 100
					WHEN ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 7 THEN 90
					WHEN ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 30 THEN 70
					WHEN ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 365 THEN 30
					ELSE 0
				END as score_date,
				-- Score téléphone
				CASE
					WHEN telephone_principal = $4 THEN 100
					WHEN $4 IS NOT NULL AND LENGTH($4) > 8 AND 
						 telephone_principal LIKE '%' || RIGHT($4, 8) || '%' THEN 80
					ELSE 0
				END as score_telephone,
				-- Détails de matching pour debug
				similarity(unaccent(LOWER(nom)), unaccent(LOWER($1))) * 100 as nom_match,
				similarity(unaccent(LOWER(prenoms)), unaccent(LOWER($2))) * 100 as prenoms_match,
				ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) as jours_ecart_naissance
			FROM patients_patient
			WHERE statut NOT IN ('archive', 'decede')
			AND (
				-- Filtres pour limiter la recherche aux candidats potentiels
				similarity(unaccent(LOWER(nom)), unaccent(LOWER($1))) > 0.3 OR
				similarity(unaccent(LOWER(prenoms)), unaccent(LOWER($2))) > 0.3 OR
				ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 365 OR
				($4 IS NOT NULL AND telephone_principal = $4)
			)
		)
		SELECT
			id,
			code_patient,
			nom,
			prenoms,
			date_naissance,
			sexe,
			telephone_principal,
			adresse_complete,
			est_assure,
			statut,
			created_at,
			-- Score global pondéré
			((score_textuel * 0.4) + (score_date * 0.4) + (score_telephone * 0.2))::integer as score_global,
			-- Détails de matching
			nom_match::integer,
			prenoms_match::integer,
			score_date::integer as date_naissance_match,
			score_telephone::integer as telephone_match
		FROM patient_scores
		WHERE ((score_textuel * 0.4) + (score_date * 0.4) + (score_telephone * 0.2)) >= $5
		ORDER BY score_global DESC
		LIMIT $6;
	`,

	// GetPatientByCodeWithAssurances - Chargement patient complet avec assurances
	GetPatientByCodeWithAssurances: `
		WITH patient_data AS (
			SELECT
				p.*,
				n.code as nationalite_code, n.nom as nationalite_nom,
				sm.code as situation_code, sm.nom as situation_nom,
				tpi.code as piece_code, tpi.nom as piece_nom,
				prof.code as profession_code, prof.nom as profession_nom,
				cb.nom as created_by_nom, cb.prenoms as created_by_prenoms,
				ub.nom as updated_by_nom, ub.prenoms as updated_by_prenoms
			FROM patients_patient p
			LEFT JOIN ref_nationalite n ON p.nationalite_id = n.id
			LEFT JOIN ref_situation_matrimoniale sm ON p.situation_matrimoniale_id = sm.id
			LEFT JOIN ref_type_piece_identite tpi ON p.type_piece_identite_id = tpi.id
			LEFT JOIN ref_profession prof ON p.profession_id = prof.id
			LEFT JOIN user_utilisateur cb ON p.created_by = cb.id
			LEFT JOIN user_utilisateur ub ON p.updated_by = ub.id
			WHERE p.code_patient = $1 AND p.statut != 'archive'
		),
		patient_assurances AS (
			SELECT
				pa.patient_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pa.id,
						'assurance_id', a.id,
						'assurance_nom', a.nom_assurance,
						'numero_assure', pa.numero_assure,
						'type_beneficiaire', pa.type_beneficiaire,
						'numero_assure_principal', pa.numero_assure_principal,
						'lien_avec_principal', pa.lien_avec_principal,
						'est_actif', pa.est_actif,
						'created_at', pa.created_at
					) ORDER BY pa.created_at
				) as assurances
			FROM patients_patient_assurance pa
			JOIN base_assurance a ON pa.assurance_id = a.id
			WHERE pa.patient_id = (SELECT id FROM patient_data)
			AND pa.est_actif = true
			GROUP BY pa.patient_id
		)
		SELECT
			pd.id,
			pd.code_patient,
			pd.etablissement_createur_id,
			pd.nom,
			pd.prenoms,
			pd.date_naissance,
			pd.est_date_supposee,
			pd.sexe,
			pd.nationalite_id,
			pd.nationalite_code,
			pd.nationalite_nom,
			pd.situation_matrimoniale_id,
			pd.situation_code,
			pd.situation_nom,
			pd.type_piece_identite_id,
			pd.piece_code,
			pd.piece_nom,
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
			pd.profession_code,
			pd.profession_nom,
			pd.personnes_a_contacter,
			pd.est_assure,
			pd.statut,
			pd.est_decede,
			pd.date_deces,
			pd.created_at,
			pd.updated_at,
			pd.created_by,
			pd.updated_by,
			pd.created_by_nom,
			pd.created_by_prenoms,
			pd.updated_by_nom,
			pd.updated_by_prenoms,
			COALESCE(pa.assurances, '[]'::jsonb) as assurances_details
		FROM patient_data pd
		LEFT JOIN patient_assurances pa ON pd.id = pa.patient_id;
	`,

	// InsertPatientAssurances - Insertion des assurances d'un patient
	InsertPatientAssurances: `
		INSERT INTO patients_patient_assurance (
			patient_id, assurance_id, numero_assure, type_beneficiaire,
			numero_assure_principal, lien_avec_principal, est_actif, created_by, created_at, updated_at
		)
		SELECT 
			$1::uuid as patient_id,
			unnest($2::uuid[]) as assurance_id,
			unnest($3::text[]) as numero_assure,
			unnest($4::text[]) as type_beneficiaire,
			unnest($5::text[]) as numero_assure_principal,
			unnest($6::text[]) as lien_avec_principal,
			true as est_actif,
			$7::uuid as created_by,
			NOW() as created_at,
			NOW() as updated_at
		RETURNING id, assurance_id, numero_assure, type_beneficiaire, est_actif;
	`,

	// ValidateReferenceDataExists - Validation que les références existent
	ValidateReferenceDataExists: `
		SELECT 
			CASE 
				WHEN EXISTS (SELECT 1 FROM ref_nationalite WHERE id = $1 AND est_actif = true) THEN true 
				ELSE false 
			END as nationalite_exists,
			CASE 
				WHEN EXISTS (SELECT 1 FROM ref_situation_matrimoniale WHERE id = $2 AND est_actif = true) THEN true 
				ELSE false 
			END as situation_matrimoniale_exists,
			CASE 
				WHEN $3::uuid IS NULL OR EXISTS (SELECT 1 FROM ref_type_piece_identite WHERE id = $3 AND est_actif = true) THEN true 
				ELSE false 
			END as type_piece_identite_exists,
			CASE 
				WHEN $4::uuid IS NULL OR EXISTS (SELECT 1 FROM ref_profession WHERE id = $4 AND est_actif = true) THEN true 
				ELSE false 
			END as profession_exists,
			CASE 
				WHEN EXISTS (SELECT 1 FROM base_etablissement WHERE id = $5 AND statut = 'actif') THEN true 
				ELSE false 
			END as etablissement_exists;
	`,

	// GetLastPatientCodeForEstablishment - Récupère le dernier code patient pour un établissement
	GetLastPatientCodeForEstablishment: `
		SELECT code_patient, created_at
		FROM patients_patient
		WHERE etablissement_createur_id = $1
		AND code_patient LIKE $2 || '-%'
		ORDER BY created_at DESC
		LIMIT 1;
	`,

	// UpdatePatientSearchVector - Met à jour le vecteur de recherche d'un patient
	UpdatePatientSearchVector: `
		UPDATE patients_patient
		SET search_vector = 
			setweight(to_tsvector('french', COALESCE(nom, '')), 'A') ||
			setweight(to_tsvector('french', COALESCE(prenoms, '')), 'A') ||
			setweight(to_tsvector('french', COALESCE(code_patient, '')), 'A') ||
			setweight(to_tsvector('french', COALESCE(telephone_principal, '')), 'B') ||
			setweight(to_tsvector('french', COALESCE(cni_nni, '')), 'B') ||
			setweight(to_tsvector('french', COALESCE(numero_piece_identite, '')), 'B'),
			updated_at = NOW()
		WHERE id = $1;
	`,
}