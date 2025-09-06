-- ============================================================================================================
-- SOINS SUITE - Schémas PostgreSQL - TABLES QUI DOIVENT  ETRE DANS  PLUSIEURS FICHIERS
-- ATLAS FONCTIONNENT PAR ORDRE D'ANALYSE
-- Extension UUID requise pour uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================
-- FONCTION TRIGGER POUR UPDATED_AT
-- =====================================
-- Description : Fonction qui met à jour automatiquement le champ updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- ============================================================================================================

-- =====================================
-- TABLE : BASE_ETABLISSEMENT
-- =====================================
-- Description : Table principale de l'établissement

CREATE TABLE base_etablissement (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Code établissement
  app_instance UUID NOT NULL DEFAULT uuid_generate_v4(),
  code_etablissement VARCHAR(20) NOT NULL DEFAULT '',

  -- Identification établissement
  nom VARCHAR(255) NOT NULL,
  nom_court VARCHAR(100) NOT NULL,
  adresse_complete TEXT NOT NULL,
  telephone_principal VARCHAR(20) NOT NULL,
  ville VARCHAR(20) NOT NULL,
  commune VARCHAR(20) NOT NULL,
  email VARCHAR(255),
  second_telephone VARCHAR(255),

  -- Informations légales
  rccm VARCHAR(50),
  cnps VARCHAR(50),

  -- Logos et visuels
  logo_principal_url VARCHAR(500),
  logo_documents_url VARCHAR(500),

  -- Configuration administrative
  duree_validite_ticket_jours INTEGER DEFAULT 15,
  nb_souches_par_caisse INTEGER DEFAULT 100,

  -- Config Back Office
  setup_est_termine BOOLEAN NOT NULL DEFAULT FALSE,
  setup_etape INTEGER NOT NULL DEFAULT 0,

  -- Horaires de garde
  garde_heure_debut TIME,
  garde_heure_fin TIME,

  -- État
  statut VARCHAR(20) NOT NULL DEFAULT 'actif',

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Ajout du champ updated_by manquant
  updated_by UUID,

  -- Contraintes métier
  CONSTRAINT UQ_base_code_etablissement_etablissement UNIQUE (code_etablissement),
  CONSTRAINT UQ_base_etablissement_app_instance UNIQUE (app_instance),
  CONSTRAINT CK_base_etablissement_statut CHECK (statut IN ('actif', 'suspendu','archive'))
);


-- =====================================
-- TABLE : USER_UTILISATEUR
-- =====================================
-- Description : Comptes utilisateurs avec authentification et permissions
CREATE TABLE user_utilisateur (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant
  etablissement_id UUID NOT NULL,

  -- Identification utilisateur
  identifiant VARCHAR(255) NOT NULL,
  nom VARCHAR(255) NOT NULL,
  prenoms VARCHAR(255) NOT NULL,
  telephone VARCHAR(20) NOT NULL,

  -- Authentification
  password_hash VARCHAR(255) NOT NULL,
  salt VARCHAR(100) NOT NULL,
  must_change_password BOOLEAN DEFAULT TRUE,
  password_changed_at TIMESTAMP,

  -- Profil métier
  est_medecin BOOLEAN DEFAULT FALSE,
  role_metier VARCHAR(100),
  photo_url VARCHAR(500),

  -- Différenciation admin/utilisateur
  est_admin BOOLEAN DEFAULT FALSE,
  type_admin VARCHAR(20) DEFAULT NULL,
  est_admin_tir BOOLEAN DEFAULT FALSE,

  -- Gestion temporaire
  est_temporaire BOOLEAN DEFAULT FALSE,
  date_expiration TIMESTAMP,

  -- État utilisateur
  statut VARCHAR(20) DEFAULT 'actif',
  motif_desactivation TEXT,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  last_login_at TIMESTAMP,
  created_by UUID,
  updated_by UUID,

  -- Contraintes métier

  -- Contraintes métier Unique
  CONSTRAINT UQ_user_utilisateur_etablissement_identifiant UNIQUE (etablissement_id, identifiant),
  -- Contraintes métier Check
  CONSTRAINT CK_user_utilisateur_type_admin CHECK (type_admin IN ('super_admin', 'admin_simple') OR type_admin IS NULL),
  CONSTRAINT CK_user_utilisateur_statut CHECK (statut IN ('actif', 'suspendu', 'expire','archive')),
  CONSTRAINT CK_user_utilisateur_admin_coherence CHECK (
    (est_admin = FALSE AND type_admin IS NULL) OR
    (est_admin = TRUE AND type_admin IN ('super_admin', 'admin_simple'))
  ),
  -- Contraintes métier Foreign Key
  CONSTRAINT FK_user_utilisateur_etablissement_id FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_utilisateur_created_by FOREIGN KEY (created_by) REFERENCES user_utilisateur(id),
  CONSTRAINT FK_user_utilisateur_updated_by FOREIGN KEY (updated_by) REFERENCES user_utilisateur(id)
);

-- =====================================
-- AJOUT DES FOREIGN KEYS APRÈS CRÉATION DES TABLES
-- =====================================
-- Maintenant que les deux tables existent, on peut ajouter les FK manquantes

-- FK de base_etablissement vers user_utilisateur
ALTER TABLE base_etablissement 
ADD CONSTRAINT FK_base_etablissement_updated_by 
FOREIGN KEY (updated_by) REFERENCES user_utilisateur(id);

-- =====================================
-- TABLE : BASE_LICENCE
-- =====================================
-- Description : Gestion abonnements et mode déploiement
CREATE TABLE base_licence (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant
  etablissement_id UUID NOT NULL,

  -- Mode déploiement
  mode_deploiement VARCHAR(20) NOT NULL, -- 'local', 'online'
  
  -- Type Licence/licence
  type_licence VARCHAR(20) NOT NULL,     -- 'premium', 'standard', 'evaluation'

  -- Modules autorisés
  modules_autorises JSONB NOT NULL,
  
  -- Validité Licence
  date_activation TIMESTAMP NOT NULL DEFAULT NOW(),
  date_expiration TIMESTAMP,             -- NULL pour Premium, calculée pour autres
  
  -- Statut Licence
  statut VARCHAR(20) DEFAULT 'actif',
  
  -- Synchronisation initiale (pour mode local)
  sync_initial_complete BOOLEAN DEFAULT FALSE,
  date_sync_initial TIMESTAMP,
  
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID,
  
  -- Contraintes métier
  CONSTRAINT CK_base_licence_mode_deploiement CHECK (mode_deploiement IN ('local', 'online')),
  CONSTRAINT CK_base_licence_type_licence CHECK (type_licence IN ('premium', 'standard', 'evaluation')),
  CONSTRAINT CK_base_licence_statut CHECK (statut IN ('actif', 'expiree', 'revoquee')),
  CONSTRAINT FK_base_licence_etablissement_id FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_base_licence_created_by FOREIGN KEY (created_by) REFERENCES user_utilisateur(id),
  
  -- Cohérence type/expiration
  CONSTRAINT CK_base_licence_expiration_coherence CHECK (
    (type_licence = 'premium' AND date_expiration IS NULL) OR
    (type_licence IN ('standard', 'evaluation') AND date_expiration IS NOT NULL)
  )
);

-- =====================================
-- TABLE : BASE_LICENCE_HISTORIQUE
-- =====================================
-- Description : Historique des événements licence
CREATE TABLE base_licence_historique (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant : 
  etablissement_id UUID NOT NULL,
  licence_id UUID NOT NULL,
  
  -- Événement
  type_evenement VARCHAR(30) NOT NULL,
  
  -- État transition
  statut_precedent VARCHAR(20),
  statut_nouveau VARCHAR(20) NOT NULL,
  
  -- Contexte
  motif_changement TEXT NOT NULL,
  utilisateur_action UUID,
  ip_action INET,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT FK_base_licence_historique_etablissement_id FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_base_licence_historique_licence_id FOREIGN KEY (licence_id) REFERENCES base_licence(id),
  CONSTRAINT CK_base_licence_historique_type_evenement CHECK (type_evenement IN (
    'activation_initiale', 'reactivation', 'expiration', 'revocation'
  ))
);

-- =====================================
-- TABLE : BASE_MODULE
-- =====================================
-- Description : Modules métiers du projet
CREATE TABLE base_module (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Identification module
  code_module VARCHAR(50) NOT NULL,
  nom_standard VARCHAR(255) NOT NULL,
  nom_personnalise VARCHAR(255),
  description TEXT,

  -- Configuration métier
  peut_prendre_ticket BOOLEAN DEFAULT FALSE,
  est_medical BOOLEAN DEFAULT TRUE,
  est_obligatoire BOOLEAN DEFAULT FALSE,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Différenciation back office / front office
  est_module_back_office BOOLEAN DEFAULT FALSE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_base_module_code_module UNIQUE (code_module)
);

-- =====================================
-- TABLE : BASE_RUBRIQUE
-- =====================================
-- Description : Rubriques fonctionnelles par module (permissions granulaires)
CREATE TABLE base_rubrique (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Module
  module_id UUID NOT NULL,

  -- Identification rubrique
  code_rubrique VARCHAR(50) NOT NULL,
  nom VARCHAR(255) NOT NULL,
  description TEXT,

  -- Hiérarchie et organisation
  rubrique_parent_id UUID,
  ordre_affichage INTEGER DEFAULT 0,

  -- Configuration
  est_obligatoire BOOLEAN DEFAULT FALSE,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_base_rubrique_module_id_code_rubrique UNIQUE (module_id, code_rubrique),
  CONSTRAINT FK_base_rubrique_module_id FOREIGN KEY (module_id) REFERENCES base_module(id),
  CONSTRAINT FK_base_rubrique_rubrique_parent_id FOREIGN KEY (rubrique_parent_id) REFERENCES base_rubrique(id)
);


COMMENT ON TABLE base_etablissement IS 'Table principale établissement avec configuration mono-tenant';
COMMENT ON TABLE base_module IS 'Modules métier avec distinction back-office/front-office';
COMMENT ON TABLE base_licence IS 'Licences avec clés obfusquées 20 caractères et validation intégrité';
COMMENT ON TABLE base_licence_historique IS 'Historique événements licence pour audit et traçabilité';

-- =====================================
-- TRIGGERS POUR UPDATED_AT
-- =====================================
-- Trigger pour user_utilisateur
CREATE TRIGGER trigger_user_utilisateur_updated_at
    BEFORE UPDATE ON user_utilisateur
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger pour base_etablissement  
CREATE TRIGGER trigger_base_etablissement_updated_at
    BEFORE UPDATE ON base_etablissement
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_licence_updated_at
    BEFORE UPDATE ON base_licence
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_licence_historique_updated_at
    BEFORE UPDATE ON base_licence_historique
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger pour base_module  
CREATE TRIGGER trigger_base_module_updated_at
    BEFORE UPDATE ON base_module
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger pour base_rubrique
CREATE TRIGGER trigger_base_rubrique_updated_at
    BEFORE UPDATE ON base_rubrique
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
