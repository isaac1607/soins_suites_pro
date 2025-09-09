-- ======================================================
-- SOINS SUITE - Schémas PostgreSQL - Domaine Base
-- ======================================================
-- Description : Tables base/organisation avec 18 tables du domaine métier
-- Domaine : base_*
-- Version : 1.0
-- ======================================================

-- Extension UUID requise pour uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================
-- TABLE : BASE_HEURE_OUVERTURE ET BASE_JOUR_FERIE
-- =====================================
CREATE TABLE base_heure_ouverture (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  
  -- Horaires normaux
  jour_semaine INTEGER,
  heure_debut TIME NOT NULL,
  heure_fin TIME NOT NULL,
  
  -- Métadonnées
  est_actif BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT CK_base_heure_ouverture_jour_semaine CHECK (jour_semaine BETWEEN 1 AND 7),
  CONSTRAINT CK_base_heure_ouverture_coherence CHECK (heure_fin > heure_debut),
  CONSTRAINT UQ_base_heure_ouverture_etablissement_jour UNIQUE (etablissement_id, jour_semaine)
);

CREATE TABLE base_jour_ferie (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  
  date_ferie DATE NOT NULL,
  libelle VARCHAR(255) NOT NULL,
  type_ferie VARCHAR(20),
  
  est_actif BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT CK_base_jour_ferie_type_ferie CHECK (type_ferie IN ('national', 'religieux', 'local', 'exceptionnel')),
  CONSTRAINT UQ_base_jour_ferie_etablissement_date UNIQUE (etablissement_id, date_ferie)
);

-- =====================================
-- TABLE : BASE_TYPE_PRESTATION
-- =====================================
-- Description : Types de prestations médicales (indépendants des modules)
CREATE TABLE base_type_prestation (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),

  -- Identification type prestation
  code_type VARCHAR(50) NOT NULL,
  libelle VARCHAR(255) NOT NULL,
  description TEXT,

  -- Configuration
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID REFERENCES user_utilisateur(id),
  updated_by UUID REFERENCES user_utilisateur(id),

  -- Contraintes
  CONSTRAINT UQ_base_type_prestation_etablissement_code UNIQUE (etablissement_id, code_type)
  -- Note: Contrainte d'unicité pour libelle remplacée par index insensible à la casse UQ_base_type_prestation_libelle_ci
);

-- =====================================
-- TABLE : BASE_TYPE_PRESTATION_MODULE
-- =====================================
-- Description : Table d'association Many-to-Many entre types de prestations et modules (peut_prendre_ticket = true uniquement)
CREATE TABLE base_type_prestation_module (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis type_prestation et module
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  type_prestation_id UUID NOT NULL,
  module_id UUID NOT NULL,

  -- Configuration spécifique à l'association
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID REFERENCES user_utilisateur(id),

  -- Contraintes
  CONSTRAINT FK_base_type_prestation_module_type_prestation_id FOREIGN KEY (type_prestation_id) REFERENCES base_type_prestation(id),
  CONSTRAINT FK_base_type_prestation_module_module_id FOREIGN KEY (module_id) REFERENCES base_module(id),
  CONSTRAINT UQ_base_type_prestation_module_etablissement_association UNIQUE (etablissement_id, type_prestation_id, module_id)
);

-- =====================================
-- TABLE : BASE_PRESTATION_MEDICALE
-- =====================================
-- Description : Catalogue détaillé des prestations médicales avec triple référencement
CREATE TABLE base_prestation_medicale (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  
  -- Référence principale (intégrité référentielle stricte)
  type_prestation_module_id UUID NOT NULL,
  
  -- Références dénormalisées (performance + accessibilité)
  type_prestation_id UUID NOT NULL,
  module_id UUID NOT NULL,

  -- Identification prestation
  code_prestation VARCHAR(50) NOT NULL,
  libelle VARCHAR(500) NOT NULL,
  description TEXT,

  -- Conformité nomenclature
  code_nomenclature_nationale VARCHAR(50),
  version_nomenclature VARCHAR(20),

  -- Configuration
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID REFERENCES user_utilisateur(id),
  updated_by UUID REFERENCES user_utilisateur(id),

  -- Contraintes
  CONSTRAINT UQ_base_prestation_medicale_etablissement_code UNIQUE (etablissement_id, code_prestation),
  -- Note: Contrainte d'unicité pour (type_prestation_id, module_id, libelle) remplacée par index insensible à la casse UQ_base_prestation_medicale_libelle_ci
  
  -- Références d'intégrité
  CONSTRAINT FK_base_prestation_medicale_type_prestation_module_id FOREIGN KEY (type_prestation_module_id) REFERENCES base_type_prestation_module(id),
  CONSTRAINT FK_base_prestation_medicale_type_prestation_id FOREIGN KEY (type_prestation_id) REFERENCES base_type_prestation(id),
  CONSTRAINT FK_base_prestation_medicale_module_id FOREIGN KEY (module_id) REFERENCES base_module(id)
  
  -- Note: Contrainte de cohérence entre les 3 références sera gérée par un trigger
  -- pour vérifier que type_prestation_module_id correspond bien à (type_prestation_id, module_id)
);

-- =====================================
-- TABLE : BASE_TARIF_PRESTATION
-- =====================================
-- Description : Historique tarifaire complet des prestations médicales
CREATE TABLE base_tarif_prestation (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis prestation_medicale
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  prestation_medicale_id UUID NOT NULL,

  -- Tarification standard
  tarif_unitaire INTEGER NOT NULL,
  unite_tarification VARCHAR(50) DEFAULT 'FCFA',

  -- Tarification différenciée selon contexte temporel
  tarif_unitaire_periode_garde INTEGER,
  tarif_unitaire_jour_ferie INTEGER,

  -- Contexte du changement
  motif_changement VARCHAR(255),
  type_changement VARCHAR(20),

  -- Validité tarif
  date_debut_validite TIMESTAMP NOT NULL DEFAULT NOW(),
  date_fin_validite TIMESTAMP,

  -- Audit renforcé
  created_at TIMESTAMP DEFAULT NOW(),
  created_by UUID NOT NULL,

  -- Contraintes métier
  CONSTRAINT FK_base_tarif_prestation_prestation_medicale_id FOREIGN KEY (prestation_medicale_id) REFERENCES base_prestation_medicale(id),
  CONSTRAINT CK_base_tarif_prestation_type_changement CHECK (type_changement IN ('creation', 'modification', 'correction')),
  CONSTRAINT CK_base_tarif_prestation_validite CHECK (date_fin_validite IS NULL OR date_fin_validite > date_debut_validite),
  CONSTRAINT CK_base_tarif_prestation_positif CHECK (tarif_unitaire > 0),
  CONSTRAINT CK_base_tarif_prestation_garde_positif CHECK (tarif_unitaire_periode_garde IS NULL OR tarif_unitaire_periode_garde > 0),
  CONSTRAINT CK_base_tarif_prestation_ferie_positif CHECK (tarif_unitaire_jour_ferie IS NULL OR tarif_unitaire_jour_ferie > 0)
);

-- ======================================================
-- CIRCUITS PATIENTS - Workflow Flexible 
-- ======================================================

-- =====================================
-- TABLE : BASE_CIRCUIT_PATIENT
-- =====================================
-- Description : Circuit patient principal avec types d'application
CREATE TABLE base_circuit_patient (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  
  -- Identification circuit
  nom_circuit VARCHAR(255) NOT NULL,
  description TEXT,
  
  -- Type d'application avec hiérarchie
  type_application VARCHAR(20) NOT NULL,
  
  -- Références spécifiques (exclusives selon type)
  type_prestation_id UUID REFERENCES base_type_prestation(id),
  type_prestation_module_id UUID REFERENCES base_type_prestation_module(id),
  
  -- Point d'entrée du circuit
  module_entree_id UUID NOT NULL REFERENCES base_module(id),
  
  -- Configuration
  est_actif BOOLEAN DEFAULT TRUE,
  
  -- Période d'activité (unicité temporelle)
  date_debut_activite TIMESTAMP NOT NULL DEFAULT NOW(),
  date_fin_activite TIMESTAMP,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID REFERENCES user_utilisateur(id),
  
  -- Contraintes métier
  CONSTRAINT CK_base_circuit_patient_type_application CHECK (type_application IN ('defaut', 'type_prestation', 'type_module')),
  CONSTRAINT CK_circuit_type_application CHECK (
    (type_application = 'defaut' AND type_prestation_id IS NULL AND type_prestation_module_id IS NULL) OR
    (type_application = 'type_prestation' AND type_prestation_id IS NOT NULL AND type_prestation_module_id IS NULL) OR
    (type_application = 'type_module' AND type_prestation_module_id IS NOT NULL AND type_prestation_id IS NULL)
  ),
  CONSTRAINT CK_circuit_periode_coherence CHECK (
    date_fin_activite IS NULL OR date_fin_activite > date_debut_activite
  )
  -- Note: La contrainte de validation pour peut_prendre_ticket sera gérée par un trigger
  -- car PostgreSQL n'autorise pas les sous-requêtes dans les contraintes CHECK
);

-- =====================================
-- TABLE : BASE_CIRCUIT_PATIENT_PARCOURS
-- =====================================
-- Description : Parcours possibles dans un circuit
CREATE TABLE base_circuit_patient_parcours (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis circuit
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  circuit_id UUID NOT NULL REFERENCES base_circuit_patient(id) ON DELETE CASCADE,
  
  -- Identification parcours
  nom_parcours VARCHAR(255) NOT NULL,
  description TEXT,
  
  -- Chemin principal du parcours (JSONB array des UUID modules)
  chemin_principal JSONB NOT NULL,
  
  -- Métadonnées parcours
  ordre_affichage INTEGER DEFAULT 0,
  est_actif BOOLEAN DEFAULT TRUE,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT CK_circuit_chemin_non_vide CHECK (jsonb_array_length(chemin_principal) > 0)
);

-- =====================================
-- TABLE : BASE_CIRCUIT_PATIENT_SOUS_CHEMIN
-- =====================================
-- Description : Sous-chemins à partir d'un parcours principal
CREATE TABLE base_circuit_patient_sous_chemin (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis parcours
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  parcours_id UUID NOT NULL REFERENCES base_circuit_patient_parcours(id) ON DELETE CASCADE,
  
  -- Identification sous-chemin
  nom_sous_chemin VARCHAR(255) NOT NULL,
  description TEXT,
  
  -- Suite du chemin à partir de la fin du parcours principal
  suite_chemin JSONB NOT NULL,
  
  -- Métadonnées sous-chemin
  ordre_affichage INTEGER DEFAULT 0,
  est_actif BOOLEAN DEFAULT TRUE,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT CK_circuit_suite_non_vide CHECK (jsonb_array_length(suite_chemin) > 0)
);

-- =====================================
-- INDEX ESSENTIELS UNIQUEMENT
-- =====================================

-- INDEX INSENSIBLES À LA CASSE pour les libellés
-- Remplacent les contraintes d'unicité standard pour éviter les doublons de casse

-- Index unique insensible à la casse pour base_type_prestation.libelle
CREATE UNIQUE INDEX UQ_base_type_prestation_libelle_ci 
ON base_type_prestation (etablissement_id, LOWER(libelle));

-- Index unique insensible à la casse pour base_prestation_medicale.libelle
-- Respecte la contrainte métier (etablissement_id, type_prestation_id, module_id, libelle)
CREATE UNIQUE INDEX UQ_base_prestation_medicale_libelle_ci 
ON base_prestation_medicale (etablissement_id, type_prestation_id, module_id, LOWER(libelle));

-- OBLIGATOIRES : Contraintes d'unicité temporelle (intégrité métier)
CREATE UNIQUE INDEX UQ_circuit_defaut_actif 
ON base_circuit_patient (etablissement_id, type_application) 
WHERE (type_application = 'defaut' AND est_actif = TRUE AND date_fin_activite IS NULL);

CREATE UNIQUE INDEX UQ_circuit_type_prestation_actif 
ON base_circuit_patient (etablissement_id, type_prestation_id) 
WHERE (type_application = 'type_prestation' AND est_actif = TRUE AND date_fin_activite IS NULL);

CREATE UNIQUE INDEX UQ_circuit_type_module_actif 
ON base_circuit_patient (etablissement_id, type_prestation_module_id) 
WHERE (type_application = 'type_module' AND est_actif = TRUE AND date_fin_activite IS NULL);

-- CRITIQUE : Performance résolution hiérarchique
CREATE INDEX idx_circuit_resolution 
ON base_circuit_patient (type_application, est_actif);


-- =====================================
-- TABLE : BASE_ASSURANCE
-- =====================================
-- Description : Organismes payeurs et assurances
CREATE TABLE base_assurance (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),

  -- Identification organisme
  code_organisme VARCHAR(50) NOT NULL,
  nom_officiel VARCHAR(255) NOT NULL,
  nom_court VARCHAR(100),

  -- Catégorisation
  type_organisme VARCHAR(20),

  -- Contacts
  adresse TEXT,
  telephone VARCHAR(20),
  email VARCHAR(255),
  contact_facturation VARCHAR(255),

  -- Configuration
  est_actif BOOLEAN DEFAULT TRUE,
  delai_paiement_jours INTEGER DEFAULT 30,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT CK_base_assurance_type_organisme CHECK (type_organisme IN ('publique', 'privee', 'mutuelle', 'internationale')),
  CONSTRAINT UQ_base_assurance_etablissement_code UNIQUE (etablissement_id, code_organisme)
);

-- =====================================
-- TABLE : BASE_BATIMENT
-- =====================================
-- Description : Infrastructure physique - bâtiments
CREATE TABLE base_batiment (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),

  -- Identification bâtiment
  nom_batiment VARCHAR(255) NOT NULL,
  code_batiment VARCHAR(50),
  description TEXT,

  -- Configuration
  nombre_etages INTEGER DEFAULT 1,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_base_batiment_etablissement_code UNIQUE (etablissement_id, code_batiment)
);


-- =====================================
-- TABLE : BASE_CATEGORIE_CHAMBRE
-- =====================================
-- Description : Catégories de chambres d'hospitalisation
CREATE TABLE base_categorie_chambre (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),

  -- Identification catégorie
  nom_categorie VARCHAR(255) NOT NULL,
  code_categorie VARCHAR(50) NOT NULL,
  description TEXT,

  -- Tarif
  tarif INTEGER NOT NULL,
  
  -- État
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_base_categorie_chambre_etablissement_code UNIQUE (etablissement_id, code_categorie)
);

-- =====================================
-- TABLE : BASE_CHAMBRE
-- =====================================
-- Description : Chambres d'hospitalisation (liées directement aux bâtiments)
CREATE TABLE base_chambre (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis bâtiment et catégorie
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  batiment_id UUID NOT NULL REFERENCES base_batiment(id),
  categorie_chambre_id UUID NOT NULL REFERENCES base_categorie_chambre(id),

  -- Identification chambre
  numero_chambre VARCHAR(50) NOT NULL,
  nom_chambre VARCHAR(255),

  -- Localisation dans le bâtiment (remplace les informations d'espace)
  niveau_etage INTEGER,
  type_espace VARCHAR(20),
  code_espace VARCHAR(50),

  -- Tarif spécial
  tarif_special INTEGER NOT NULL,

  -- État
  est_occupee BOOLEAN DEFAULT FALSE,

  -- Configuration
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT CK_base_chambre_type_espace CHECK (type_espace IN ('consultation', 'administration', 'technique', 'hospitalisation', 'bloc_operatoire'))
);

-- =====================================
-- TABLE : BASE_LIT
-- =====================================
-- Description : Lits individuels
CREATE TABLE base_lit (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : Héritage depuis chambre
  etablissement_id UUID NOT NULL REFERENCES base_etablissement(id),
  chambre_id UUID NOT NULL REFERENCES base_chambre(id),

  -- Identification lit
  numero_lit VARCHAR(50) NOT NULL,
  code_lit VARCHAR(50),

  -- Configuration
  type_lit VARCHAR(20),

  -- État
  est_occupee BOOLEAN DEFAULT FALSE,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT UQ_base_lit_etablissement_code UNIQUE (etablissement_id, code_lit),
  CONSTRAINT CK_base_lit_type_lit CHECK (type_lit IN ('standard', 'medicalise', 'reanimation'))
);

-- ======================================================
-- COMMENTAIRES POUR DOCUMENTATION
-- ======================================================

COMMENT ON TABLE base_etablissement IS 'Table principale établissement avec configuration mono-tenant';
COMMENT ON TABLE base_module IS 'Modules métier avec distinction back-office/front-office';
COMMENT ON TABLE base_licence IS 'Licences avec clés obfusquées 20 caractères et validation intégrité';
COMMENT ON TABLE base_licence_historique IS 'Historique événements licence pour audit et traçabilité';
COMMENT ON TABLE base_prestation_medicale IS 'Catalogue prestations médicales avec nomenclature nationale';

COMMENT ON TABLE base_circuit_patient IS 'Circuits patients avec hiérarchie defaut(3) < type_prestation(2) < type_module(1)';
COMMENT ON TABLE base_circuit_patient_parcours IS 'Parcours possibles avec chemins JSONB ["uuid1", "uuid2", "uuid3"]';
COMMENT ON TABLE base_circuit_patient_sous_chemin IS 'Sous-chemins depuis points de branchement ["uuid4", "uuid5"]';

COMMENT ON COLUMN base_circuit_patient.type_application IS 'Types: defaut, type_prestation, type_module';
COMMENT ON COLUMN base_circuit_patient_parcours.chemin_principal IS 'Array JSONB des UUID modules du parcours';
COMMENT ON COLUMN base_circuit_patient_sous_chemin.suite_chemin IS 'Array JSONB suite du chemin depuis parcours parent';


-- =====================================
-- TRIGGERS POUR UPDATED_AT
-- =====================================

-- TRIGGERS POUR UPDATED_AT
CREATE TRIGGER trigger_heure_ouverture_updated_at
    BEFORE UPDATE ON base_heure_ouverture
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_jour_ferie_updated_at
    BEFORE UPDATE ON base_jour_ferie
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_type_prestation_updated_at
    BEFORE UPDATE ON base_type_prestation
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_type_prestation_module_updated_at
    BEFORE UPDATE ON base_type_prestation_module
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_prestation_medicale_updated_at
    BEFORE UPDATE ON base_prestation_medicale
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_circuit_patient_updated_at
    BEFORE UPDATE ON base_circuit_patient
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_circuit_parcours_updated_at
    BEFORE UPDATE ON base_circuit_patient_parcours
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_circuit_sous_chemin_updated_at
    BEFORE UPDATE ON base_circuit_patient_sous_chemin
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_assurance_updated_at
    BEFORE UPDATE ON base_assurance
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_batiment_updated_at
    BEFORE UPDATE ON base_batiment
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


CREATE TRIGGER trigger_categorie_chambre_updated_at
    BEFORE UPDATE ON base_categorie_chambre
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_chambre_updated_at
    BEFORE UPDATE ON base_chambre
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_lit_updated_at
    BEFORE UPDATE ON base_lit
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
