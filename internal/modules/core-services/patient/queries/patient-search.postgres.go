package queries

// PatientSearchQueries contient toutes les requêtes SQL pour la recherche de patients
var PatientSearchQueries = struct {
	SearchPatientsFullText        string
	SearchPatientsByCriteria      string
	GetPatientByCodeFromCache     string
	CountPatientsFullText         string
	CountPatientsByCriteria       string
	SearchPatientsWithAssurances  string
}{
	// SearchPatientsFullText - Recherche full-text optimisée avec scoring
	SearchPatientsFullText: `
		SELECT
			p.id,
			p.code_patient,
			p.nom,
			p.prenoms,
			p.date_naissance,
			p.sexe,
			p.telephone_principal,
			p.adresse_complete,
			p.est_assure,
			p.statut,
			p.created_at,
			CASE
				WHEN $1 != '' THEN ts_rank(p.search_vector, plainto_tsquery('french', $1))
				ELSE 0
			END as score
		FROM patients_patient p
		WHERE p.statut = ANY($2::text[])  -- Statuts autorisés
			AND ($1 = '' OR p.search_vector @@ plainto_tsquery('french', $1))
			AND ($3::text IS NULL OR p.nom ILIKE '%' || $3 || '%')
			AND ($4::text IS NULL OR p.prenoms ILIKE '%' || $4 || '%')
			AND ($5::text IS NULL OR p.telephone_principal = $5)
			AND ($6::date IS NULL OR p.date_naissance >= $6)
			AND ($7::date IS NULL OR p.date_naissance <= $7)
			AND ($8::char IS NULL OR p.sexe = $8)
			AND ($9::text IS NULL OR p.cni_nni = $9)
			AND ($10::boolean IS NULL OR p.est_assure = $10)
			AND ($11::uuid IS NULL OR p.etablissement_createur_id = $11)
		ORDER BY
			CASE
				WHEN $12 = 'score' AND $1 != '' THEN ts_rank(p.search_vector, plainto_tsquery('french', $1))
				WHEN $12 = 'nom' THEN 
					CASE WHEN $13 = 'asc' THEN NULL ELSE NULL END
				WHEN $12 = 'created_at' THEN 
					CASE WHEN $13 = 'asc' THEN NULL ELSE NULL END
				ELSE ts_rank(p.search_vector, plainto_tsquery('french', $1))
			END DESC,
			CASE
				WHEN $12 = 'nom' AND $13 = 'asc' THEN p.nom
				ELSE NULL
			END ASC,
			CASE
				WHEN $12 = 'nom' AND $13 = 'desc' THEN p.nom
				ELSE NULL
			END DESC,
			CASE
				WHEN $12 = 'created_at' AND $13 = 'asc' THEN p.created_at
				ELSE NULL
			END ASC,
			CASE
				WHEN $12 = 'created_at' AND $13 = 'desc' THEN p.created_at
				ELSE NULL
			END DESC,
			p.created_at DESC  -- Tri par défaut
		LIMIT $14 OFFSET $15;
	`,

	// SearchPatientsByCriteria - Recherche par critères spécifiques (sans full-text)
	SearchPatientsByCriteria: `
		SELECT
			p.id,
			p.code_patient,
			p.nom,
			p.prenoms,
			p.date_naissance,
			p.sexe,
			p.telephone_principal,
			p.adresse_complete,
			p.est_assure,
			p.statut,
			p.created_at,
			NULL as score  -- Pas de score pour recherche par critères
		FROM patients_patient p
		WHERE p.statut = ANY($1::text[])  -- Statuts autorisés
			AND ($2::text IS NULL OR p.nom ILIKE '%' || $2 || '%')
			AND ($3::text IS NULL OR p.prenoms ILIKE '%' || $3 || '%')
			AND ($4::text IS NULL OR p.telephone_principal = $4)
			AND ($5::date IS NULL OR p.date_naissance >= $5)
			AND ($6::date IS NULL OR p.date_naissance <= $6)
			AND ($7::date IS NULL OR p.date_naissance = $7)
			AND ($8::char IS NULL OR p.sexe = $8)
			AND ($9::text IS NULL OR p.cni_nni = $9)
			AND ($10::boolean IS NULL OR p.est_assure = $10)
			AND ($11::uuid IS NULL OR p.etablissement_createur_id = $11)
		ORDER BY
			CASE
				WHEN $12 = 'nom' AND $13 = 'asc' THEN p.nom
				ELSE NULL
			END ASC,
			CASE
				WHEN $12 = 'nom' AND $13 = 'desc' THEN p.nom
				ELSE NULL
			END DESC,
			CASE
				WHEN $12 = 'created_at' AND $13 = 'asc' THEN p.created_at
				ELSE NULL
			END ASC,
			CASE
				WHEN $12 = 'created_at' AND $13 = 'desc' THEN p.created_at
				ELSE NULL
			END DESC,
			p.created_at DESC  -- Tri par défaut
		LIMIT $14 OFFSET $15;
	`,

	// GetPatientByCodeFromCache - Recherche directe par code patient (priorité cache)
	GetPatientByCodeFromCache: `
		SELECT
			p.id,
			p.code_patient,
			p.nom,
			p.prenoms,
			p.date_naissance,
			p.sexe,
			p.telephone_principal,
			p.adresse_complete,
			p.est_assure,
			p.statut,
			p.created_at,
			1.0 as score  -- Score maximum pour recherche exacte
		FROM patients_patient p
		WHERE p.code_patient = $1 
			AND p.statut != 'archive';
	`,

	// CountPatientsFullText - Compte le nombre total de résultats pour recherche full-text
	CountPatientsFullText: `
		SELECT COUNT(*) as total
		FROM patients_patient p
		WHERE p.statut = ANY($1::text[])  -- Statuts autorisés
			AND ($2 = '' OR p.search_vector @@ plainto_tsquery('french', $2))
			AND ($3::text IS NULL OR p.nom ILIKE '%' || $3 || '%')
			AND ($4::text IS NULL OR p.prenoms ILIKE '%' || $4 || '%')
			AND ($5::text IS NULL OR p.telephone_principal = $5)
			AND ($6::date IS NULL OR p.date_naissance >= $6)
			AND ($7::date IS NULL OR p.date_naissance <= $7)
			AND ($8::char IS NULL OR p.sexe = $8)
			AND ($9::text IS NULL OR p.cni_nni = $9)
			AND ($10::boolean IS NULL OR p.est_assure = $10)
			AND ($11::uuid IS NULL OR p.etablissement_createur_id = $11);
	`,

	// CountPatientsByCriteria - Compte le nombre total de résultats pour recherche par critères
	CountPatientsByCriteria: `
		SELECT COUNT(*) as total
		FROM patients_patient p
		WHERE p.statut = ANY($1::text[])  -- Statuts autorisés
			AND ($2::text IS NULL OR p.nom ILIKE '%' || $2 || '%')
			AND ($3::text IS NULL OR p.prenoms ILIKE '%' || $3 || '%')
			AND ($4::text IS NULL OR p.telephone_principal = $4)
			AND ($5::date IS NULL OR p.date_naissance >= $5)
			AND ($6::date IS NULL OR p.date_naissance <= $6)
			AND ($7::date IS NULL OR p.date_naissance = $7)
			AND ($8::char IS NULL OR p.sexe = $8)
			AND ($9::text IS NULL OR p.cni_nni = $9)
			AND ($10::boolean IS NULL OR p.est_assure = $10)
			AND ($11::uuid IS NULL OR p.etablissement_createur_id = $11);
	`,

	// SearchPatientsWithAssurances - Recherche avec inclusion des assurances
	SearchPatientsWithAssurances: `
		WITH patient_results AS (
			SELECT
				p.id,
				p.code_patient,
				p.nom,
				p.prenoms,
				p.date_naissance,
				p.sexe,
				p.telephone_principal,
				p.adresse_complete,
				p.est_assure,
				p.statut,
				p.created_at,
				CASE
					WHEN $1 != '' THEN ts_rank(p.search_vector, plainto_tsquery('french', $1))
					ELSE 0
				END as score
			FROM patients_patient p
			WHERE p.statut = ANY($2::text[])  -- Statuts autorisés
				AND ($1 = '' OR p.search_vector @@ plainto_tsquery('french', $1))
				AND ($3::text IS NULL OR p.nom ILIKE '%' || $3 || '%')
				AND ($4::text IS NULL OR p.prenoms ILIKE '%' || $4 || '%')
				AND ($5::text IS NULL OR p.telephone_principal = $5)
				AND ($6::date IS NULL OR p.date_naissance >= $6)
				AND ($7::date IS NULL OR p.date_naissance <= $7)
				AND ($8::char IS NULL OR p.sexe = $8)
				AND ($9::text IS NULL OR p.cni_nni = $9)
				AND ($10::boolean IS NULL OR p.est_assure = $10)
				AND ($11::uuid IS NULL OR p.etablissement_createur_id = $11)
			ORDER BY
				CASE
					WHEN $12 = 'score' AND $1 != '' THEN ts_rank(p.search_vector, plainto_tsquery('french', $1))
					WHEN $12 = 'nom' AND $13 = 'asc' THEN p.nom
					WHEN $12 = 'nom' AND $13 = 'desc' THEN p.nom
					WHEN $12 = 'created_at' AND $13 = 'asc' THEN extract(epoch from p.created_at)
					WHEN $12 = 'created_at' AND $13 = 'desc' THEN extract(epoch from p.created_at)
					ELSE extract(epoch from p.created_at)
				END DESC
			LIMIT $14 OFFSET $15
		),
		patient_assurances AS (
			SELECT
				pa.patient_id,
				jsonb_agg(
					jsonb_build_object(
						'id', pa.id,
						'assurance_nom', COALESCE(a.nom_assurance, 'Assurance inconnue'),
						'numero_assure', pa.numero_assure,
						'type_beneficiaire', pa.type_beneficiaire,
						'est_actif', pa.est_actif
					) ORDER BY pa.created_at
				) as assurances
			FROM patients_patient_assurance pa
			LEFT JOIN base_assurance a ON pa.assurance_id = a.id
			WHERE pa.patient_id IN (SELECT id FROM patient_results)
			AND pa.est_actif = true
			GROUP BY pa.patient_id
		)
		SELECT
			pr.*,
			COALESCE(pa.assurances, '[]'::jsonb) as assurances_details
		FROM patient_results pr
		LEFT JOIN patient_assurances pa ON pr.id = pa.patient_id
		ORDER BY pr.score DESC, pr.created_at DESC;
	`,
}