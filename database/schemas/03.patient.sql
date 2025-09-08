-- ======================================================
-- SOINS SUITE - Schémas PostgreSQL - Domaine Patient
-- ======================================================
-- Description : Référentiel patient unique avec conventions conformes
-- Domaine : patients_* et ref_*
-- Version : 1.0 MVP
-- ======================================================

-- Extension trigram requise pour recherche textuelle optimisée
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =====================================
-- TABLES DE RÉFÉRENCE (NOMENCLATURES)
-- =====================================
-- Description : Tables partagées entre tous les établissements

-- Nationalités
CREATE TABLE ref_nationalite (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(3) NOT NULL,  -- Code ISO Alpha-3 (CIV, FRA, etc.)
    nom VARCHAR(100) NOT NULL,
    est_actif BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_ref_nationalite_code UNIQUE (code),
    CONSTRAINT UQ_ref_nationalite_nom UNIQUE (nom)
);

-- Situations matrimoniales
CREATE TABLE ref_situation_matrimoniale (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) NOT NULL,  -- CELIBATAIRE, MARIE, DIVORCE, VEUF
    nom VARCHAR(50) NOT NULL,
    est_actif BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_ref_situation_matrimoniale_code UNIQUE (code),
    CONSTRAINT UQ_ref_situation_matrimoniale_nom UNIQUE (nom)
);

-- Types de pièces d'identité
CREATE TABLE ref_type_piece_identite (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) NOT NULL,  -- CNI, PASSPORT, ATTESTATION, etc.
    nom VARCHAR(100) NOT NULL,
    est_actif BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_ref_type_piece_identite_code UNIQUE (code),
    CONSTRAINT UQ_ref_type_piece_identite_nom UNIQUE (nom)
);

-- Professions
CREATE TABLE ref_profession (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) NOT NULL,
    nom VARCHAR(150) NOT NULL,
    categorie VARCHAR(50),  -- SANTE, EDUCATION, COMMERCE, etc.
    est_actif BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_ref_profession_code UNIQUE (code),
    CONSTRAINT UQ_ref_profession_nom UNIQUE (nom)
);

-- Liens de parenté / Affiliations
CREATE TABLE ref_affiliation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) NOT NULL,  -- PERE, MERE, CONJOINT, ENFANT, TUTEUR
    nom VARCHAR(50) NOT NULL,
    ordre_affichage INTEGER DEFAULT 0,
    est_actif BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_ref_affiliation_code UNIQUE (code),
    CONSTRAINT UQ_ref_affiliation_nom UNIQUE (nom)
);

-- =====================================
-- TABLE : PATIENTS_CODE_SEQUENCES
-- =====================================
-- Description : Gestion des séquences pour génération codes patient uniques

CREATE TABLE patients_code_sequences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    etablissement_code VARCHAR(20) NOT NULL,
    annee INTEGER NOT NULL,
    dernier_numero INTEGER DEFAULT 0,      -- 001 à 999
    dernier_suffixe VARCHAR(3) DEFAULT 'AAA', -- AAA à ZZZ
    nombre_generes BIGINT DEFAULT 0,       -- Statistique

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT UQ_patients_sequences_etablissement_annee
        UNIQUE (etablissement_code, annee)
);

-- Index pour performance lookup rapide
CREATE INDEX IDX_patients_sequences_lookup
    ON patients_code_sequences (etablissement_code, annee);

-- Trigger pour updated_at
CREATE TRIGGER trigger_patients_code_sequences_updated_at
    BEFORE UPDATE ON patients_code_sequences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Fonction PostgreSQL pour calculer le suffixe suivant
CREATE OR REPLACE FUNCTION next_alpha_suffix(current_suffix VARCHAR(3))
RETURNS VARCHAR(3) AS $$
DECLARE
    chars CHAR(3)[];
    i INTEGER;
BEGIN
    IF current_suffix = 'ZZZ' THEN
        RAISE EXCEPTION 'Capacité maximale atteinte pour l''année';
    END IF;

    chars := string_to_array(current_suffix, NULL);

    -- Incrémenter de droite à gauche
    FOR i IN REVERSE 3..1 LOOP
        IF chars[i] < 'Z' THEN
            chars[i] := chr(ascii(chars[i]) + 1);
            EXIT;
        ELSE
            chars[i] := 'A';
        END IF;
    END LOOP;

    RETURN array_to_string(chars, '');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- =====================================
-- TABLE PRINCIPALE : PATIENTS_PATIENT
-- =====================================
-- Description : Référentiel patient unique (mono-tenant pour l'identification)

CREATE TABLE patients_patient (
    -- Clé primaire
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- IDENTIFICATION UNIQUE
    code_patient VARCHAR(20) NOT NULL,  -- Unique
    etablissement_createur_id UUID NOT NULL,  -- Traçabilité création

    -- ==========================================
    -- SECTION 1: IDENTITÉ (Obligatoires)
    -- ==========================================
    nom VARCHAR(255) NOT NULL,
    prenoms VARCHAR(255) NOT NULL,
    date_naissance DATE NOT NULL,
    est_date_supposee BOOLEAN DEFAULT FALSE NOT NULL,
    sexe CHAR(1) NOT NULL,
    nationalite_id UUID NOT NULL,
    situation_matrimoniale_id UUID NOT NULL,

    -- Pièce d'identité (Optionnels mais recommandés)
    type_piece_identite_id UUID,
    cni_nni VARCHAR(100),
    numero_piece_identite VARCHAR(100),

    -- Complément identité (Optionnels)
    lieu_naissance VARCHAR(255),
    nom_jeune_fille VARCHAR(255),  -- Pour les femmes mariées

    -- ==========================================
    -- SECTION 2: CONTACT & LOCALISATION
    -- ==========================================
    telephone_principal VARCHAR(20) NOT NULL,
    telephone_secondaire VARCHAR(20),
    email VARCHAR(255),

    -- Adresse
    adresse_complete TEXT NOT NULL,
    quartier VARCHAR(100),
    ville VARCHAR(100),
    commune VARCHAR(100),
    pays_residence VARCHAR(100) DEFAULT 'Côte d''Ivoire',

    -- Informations socio-professionnelles
    profession_id UUID,

    -- ==========================================
    -- SECTION 3: PERSONNES À CONTACTER
    -- ==========================================
    personnes_a_contacter JSONB DEFAULT '[]'::jsonb NOT NULL,
    /* Structure JSON validée:
    [
        {
            "nom_prenoms": "Marie KOUADIO",
            "telephone": "+225 07 98 76 54 32",
            "telephone_secondaire": "+225 01 23 45 67 89",
            "affiliation_id": "uuid-ref-affiliation",
        }
    ]
    */

    -- ==========================================
    -- SECTION 4: ASSURANCE & COUVERTURE
    -- ==========================================
    est_assure BOOLEAN DEFAULT FALSE NOT NULL,
    -- Les détails d'assurance sont dans patients_patient_assurance

    -- ==========================================
    -- STATUT & MÉTADONNÉES
    -- ==========================================
    statut VARCHAR(20) DEFAULT 'actif' NOT NULL,

    -- Flags spéciaux
    est_decede BOOLEAN DEFAULT FALSE,
    date_deces DATE,

    -- Recherche & matching
    search_vector tsvector,  -- Pour recherche full-text PostgreSQL

    -- Métadonnées standards
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() NOT NULL,
    created_by UUID,
    updated_by UUID,

    -- ==========================================
    -- CONTRAINTES UNIQUE
    -- ==========================================
    CONSTRAINT UQ_patients_patient_code_patient UNIQUE (code_patient),

    -- ==========================================
    -- CONTRAINTES FOREIGN KEY
    -- ==========================================
    CONSTRAINT FK_patients_patient_etablissement_createur FOREIGN KEY (etablissement_createur_id)
        REFERENCES base_etablissement(id),
    CONSTRAINT FK_patients_patient_nationalite FOREIGN KEY (nationalite_id)
        REFERENCES ref_nationalite(id),
    CONSTRAINT FK_patients_patient_situation_matrimoniale FOREIGN KEY (situation_matrimoniale_id)
        REFERENCES ref_situation_matrimoniale(id),
    CONSTRAINT FK_patients_patient_type_piece_identite FOREIGN KEY (type_piece_identite_id)
        REFERENCES ref_type_piece_identite(id),
    CONSTRAINT FK_patients_patient_profession FOREIGN KEY (profession_id)
        REFERENCES ref_profession(id),
    CONSTRAINT FK_patients_patient_created_by FOREIGN KEY (created_by)
        REFERENCES user_utilisateur(id),
    CONSTRAINT FK_patients_patient_updated_by FOREIGN KEY (updated_by)
        REFERENCES user_utilisateur(id),

    -- ==========================================
    -- CONTRAINTES CHECK
    -- ==========================================
    CONSTRAINT CK_patients_patient_sexe CHECK (sexe IN ('M', 'F')),
    CONSTRAINT CK_patients_patient_statut CHECK (statut IN ('actif', 'inactif', 'decede', 'archive')),
    CONSTRAINT CK_patients_patient_date_naissance CHECK (date_naissance <= CURRENT_DATE),
    CONSTRAINT CK_patients_patient_date_naissance_realiste CHECK (
        date_naissance >= '1900-01-01' AND
        date_naissance <= CURRENT_DATE
    ),
    CONSTRAINT CK_patients_patient_telephone_format CHECK (
        telephone_principal ~ '^[+0-9 ()-]+$'
    ),
    CONSTRAINT CK_patients_patient_email_format CHECK (
        email IS NULL OR email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'
    ),
    CONSTRAINT CK_patients_patient_coherence_deces CHECK (
        (est_decede = FALSE AND date_deces IS NULL) OR
        (est_decede = TRUE AND date_deces IS NOT NULL)
    )
);

-- =====================================
-- TABLE : PATIENTS_PATIENT_ASSURANCE
-- =====================================
-- Description : Association patient-assurance (plusieurs assurances possibles)

CREATE TABLE patients_patient_assurance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    patient_id UUID NOT NULL,
    assurance_id UUID NOT NULL,

    -- Numéro d'assuré du patient actuel
    numero_assure VARCHAR(100) NOT NULL,  -- Ex: "MUGEF-2024-12345"

    -- Type de bénéficiaire
    type_beneficiaire VARCHAR(30) DEFAULT 'principal', -- principal, ayant_droit

    -- Si ayant_droit, référence vers l'assuré principal
    numero_assure_principal VARCHAR(100),  -- Ex: "MUGEF-2024-00001" (le parent/conjoint/enfant,autre)
    lien_avec_principal VARCHAR(50),       -- conjoint, enfant, parent, autre

    -- Statut
    est_actif BOOLEAN DEFAULT TRUE,
    motif_inactivation VARCHAR(255),

    -- Métadonnées standards
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,

    -- ==========================================
    -- CONTRAINTES UNIQUE
    -- ==========================================
    CONSTRAINT UQ_patients_patient_assurance_patient_assurance_actif
        UNIQUE (patient_id, assurance_id, est_actif),

    -- ==========================================
    -- CONTRAINTES FOREIGN KEY
    -- ==========================================
    CONSTRAINT FK_patients_patient_assurance_patient FOREIGN KEY (patient_id)
        REFERENCES patients_patient(id) ON DELETE CASCADE,
    CONSTRAINT FK_patients_patient_assurance_assurance FOREIGN KEY (assurance_id)
        REFERENCES base_assurance(id),
    CONSTRAINT FK_patients_patient_assurance_created_by FOREIGN KEY (created_by)
        REFERENCES user_utilisateur(id),

    -- ==========================================
    -- CONTRAINTES CHECK
    -- ==========================================
    CONSTRAINT CK_patients_patient_assurance_type_beneficiaire
        CHECK (type_beneficiaire IN ('principal', 'ayant_droit')),
    CONSTRAINT CK_patients_patient_assurance_lien_avec_principal
        CHECK (lien_avec_principal IN ('conjoint', 'enfant', 'parent', 'autre')),
    CONSTRAINT CK_patients_patient_assurance_coherence_ayant_droit
        CHECK (
            (type_beneficiaire = 'principal' AND numero_assure_principal IS NULL AND lien_avec_principal IS NULL) OR
            (type_beneficiaire = 'ayant_droit' AND numero_assure_principal IS NOT NULL AND lien_avec_principal IS NOT NULL)
        )
);

-- =====================================
-- INDEXES CRITIQUES (Maximum 5)
-- =====================================
-- Choix optimisé selon règle 80/20 pour minimiser les coûts

-- 1. RECHERCHE PAR CODE (95% des accès - CRITIQUE)
CREATE UNIQUE INDEX IDX_patients_patient_code_patient
    ON patients_patient (code_patient);

-- 2. RECHERCHE FULL-TEXT (Recherche intelligente - CRITIQUE)
CREATE INDEX IDX_patients_patient_search_vector
    ON patients_patient USING gin (search_vector);

-- 3. RECHERCHE PAR TÉLÉPHONE (60% urgences - IMPORTANT)
CREATE INDEX IDX_patients_patient_telephone
    ON patients_patient (telephone_principal);

-- 4. PATIENTS ACTIFS (Majorité requêtes - IMPORTANT)
CREATE INDEX IDX_patients_patient_statut_actif
    ON patients_patient (statut) WHERE statut = 'actif';

-- 5. RECHERCHE TEXTUELLE FALLBACK (Si search_vector échoue - FALLBACK)
CREATE INDEX IDX_patients_patient_nom_prenoms_trigram
    ON patients_patient USING gin (nom gin_trgm_ops, prenoms gin_trgm_ops);

-- =====================================
-- TRIGGERS
-- =====================================

-- Trigger pour updated_at automatique
CREATE TRIGGER trigger_patients_patient_updated_at
    BEFORE UPDATE ON patients_patient
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_patients_patient_assurance_updated_at
    BEFORE UPDATE ON patients_patient_assurance
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_ref_nationalite_updated_at
    BEFORE UPDATE ON ref_nationalite
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_ref_situation_matrimoniale_updated_at
    BEFORE UPDATE ON ref_situation_matrimoniale
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_ref_type_piece_identite_updated_at
    BEFORE UPDATE ON ref_type_piece_identite
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_ref_profession_updated_at
    BEFORE UPDATE ON ref_profession
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_ref_affiliation_updated_at
    BEFORE UPDATE ON ref_affiliation
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger pour search_vector (recherche full-text)
CREATE OR REPLACE FUNCTION update_patient_search_vector() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('french', COALESCE(NEW.nom, '')), 'A') ||
        setweight(to_tsvector('french', COALESCE(NEW.prenoms, '')), 'A') ||
        setweight(to_tsvector('french', COALESCE(NEW.code_patient, '')), 'A') ||
        setweight(to_tsvector('french', COALESCE(NEW.telephone_principal, '')), 'B') ||
        setweight(to_tsvector('french', COALESCE(NEW.cni_nni, '')), 'B') ||
        setweight(to_tsvector('french', COALESCE(NEW.numero_piece_identite, '')), 'B');
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_patients_patient_search_vector
    BEFORE INSERT OR UPDATE ON patients_patient
    FOR EACH ROW
    EXECUTE FUNCTION update_patient_search_vector();

-- =====================================
-- COMMENTAIRES DOCUMENTATION
-- =====================================

COMMENT ON TABLE patients_patient IS 'Référentiel patient unique - Un patient = Un code valable partout';
COMMENT ON COLUMN patients_patient.code_patient IS 'Code unique format: CI-YYYYMMDD-XXXXX';
COMMENT ON COLUMN patients_patient.etablissement_createur_id IS 'Établissement ayant créé le patient (traçabilité)';
COMMENT ON COLUMN patients_patient.search_vector IS 'Vecteur de recherche full-text pour recherche rapide';
COMMENT ON COLUMN patients_patient.personnes_a_contacter IS 'Liste JSON des contacts d''urgence avec priorité';
COMMENT ON COLUMN patients_patient.est_date_supposee IS 'TRUE si date de naissance approximative';

COMMENT ON TABLE patients_patient_assurance IS 'Association patient-assurance avec support multi-assurances';
COMMENT ON COLUMN patients_patient_assurance.type_beneficiaire IS 'principal ou ayant_droit (enfant, conjoint, etc.)';
COMMENT ON COLUMN patients_patient_assurance.numero_assure_principal IS 'Numéro de l''assuré principal si ayant_droit';

COMMENT ON TABLE ref_nationalite IS 'Référentiel des nationalités (codes ISO Alpha-3)';
COMMENT ON TABLE ref_situation_matrimoniale IS 'Référentiel des situations matrimoniales';
COMMENT ON TABLE ref_type_piece_identite IS 'Référentiel des types de pièces d''identité';
COMMENT ON TABLE ref_profession IS 'Référentiel des professions par catégorie';
COMMENT ON TABLE ref_affiliation IS 'Référentiel des liens de parenté pour personnes à contacter';