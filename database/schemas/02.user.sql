-- ======================================================
-- SOINS SUITE - Schémas PostgreSQL - Domaine User
-- ======================================================
-- Description : Tables utilisateurs avec système de permissions complet
-- Domaine : user_*
-- Version : 2.0
-- ======================================================

-- =====================================
-- TABLE : USER_PROFIL_TEMPLATE
-- =====================================
-- Description : Modèles de profils prédéfinis (Médecin, Infirmier, etc.)
CREATE TABLE user_profil_template (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL,

  -- Identification profil
  code_profil VARCHAR(50) NOT NULL,
  nom_profil VARCHAR(255) NOT NULL,
  description TEXT,

  -- Configuration profil
  est_predefinit BOOLEAN DEFAULT FALSE,  -- Profil système non modifiable
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID,

  -- Contraintes
  CONSTRAINT UQ_user_profil_template_etablissement_code UNIQUE (etablissement_id, code_profil),
  CONSTRAINT FK_user_profil_template_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_profil_template_created_by FOREIGN KEY (created_by) REFERENCES user_utilisateur(id)
);

-- =====================================
-- TABLE : USER_PROFIL_MODULES
-- =====================================
-- Description : Modules associés aux profils templates
CREATE TABLE user_profil_modules (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL,
  
  -- Relations
  profil_template_id UUID NOT NULL,
  module_id UUID NOT NULL,

  -- Accès modules
  acces_toutes_rubriques BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  created_by UUID,

  -- Contraintes
  CONSTRAINT UQ_user_profil_modules_profil_module UNIQUE (profil_template_id, module_id, etablissement_id),
  CONSTRAINT FK_user_profil_modules_profil FOREIGN KEY (profil_template_id) REFERENCES user_profil_template(id) ON DELETE CASCADE,
  CONSTRAINT FK_user_profil_modules_module FOREIGN KEY (module_id) REFERENCES base_module(id)
  CONSTRAINT FK_user_profil_modules_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_profil_modules_created_by FOREIGN KEY (created_by) REFERENCES user_utilisateur(id)
);

-- =====================================
-- TABLE : USER_PROFIL_RUBRIQUES
-- =====================================
-- Description : Rubriques spécifiques pour les profils (mode granulaire)
CREATE TABLE user_profil_rubriques (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL,

  -- Relations
  profil_template_id UUID NOT NULL,
  module_id UUID NOT NULL,
  rubrique_id UUID NOT NULL,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  created_by UUID,

  -- Contraintes
  CONSTRAINT UQ_user_profil_rubriques_profil_rubrique UNIQUE (profil_template_id, rubrique_id, etablissement_id),
  CONSTRAINT FK_user_profil_rubriques_profil FOREIGN KEY (profil_template_id) REFERENCES user_profil_template(id) ON DELETE CASCADE,
  CONSTRAINT FK_user_profil_rubriques_module FOREIGN KEY (module_id) REFERENCES base_module(id),
  CONSTRAINT FK_user_profil_rubriques_rubrique FOREIGN KEY (rubrique_id) REFERENCES base_rubrique(id),
  CONSTRAINT FK_user_profil_rubriques_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_profil_rubriques_created_by FOREIGN KEY (created_by) REFERENCES user_utilisateur(id)
);

-- =====================================
-- TABLE : USER_PROFIL_UTILISATEURS
-- =====================================
-- Description : Association utilisateurs aux profils templates
CREATE TABLE user_profil_utilisateurs (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  -- Multi-tenant : Isolation par établissement
  etablissement_id UUID NOT NULL,

  -- Relations
  utilisateur_id UUID NOT NULL,
  profil_template_id UUID NOT NULL,
  
  -- Traçabilité
  attribue_par UUID NOT NULL,
  date_attribution TIMESTAMP DEFAULT NOW(),
  date_fin TIMESTAMP,
  est_actif BOOLEAN DEFAULT TRUE,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT UQ_user_profil_utilisateurs_user_profil_actif UNIQUE (etablissement_id,utilisateur_id, profil_template_id, est_actif),
  CONSTRAINT FK_user_profil_utilisateurs_user FOREIGN KEY (utilisateur_id) REFERENCES user_utilisateur(id),
  CONSTRAINT FK_user_profil_utilisateurs_profil FOREIGN KEY (profil_template_id) REFERENCES user_profil_template(id),
  CONSTRAINT FK_user_profil_utilisateurs_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_profil_utilisateurs_attribue_par FOREIGN KEY (attribue_par) REFERENCES user_utilisateur(id)
);

-- =====================================
-- TABLE : USER_MODULES
-- =====================================
-- Description : Attribution de modules complets à un utilisateur (permissions additionnelles)
CREATE TABLE user_modules (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant et relations
  etablissement_id UUID NOT NULL,
  utilisateur_id UUID NOT NULL,
  module_id UUID NOT NULL,

  -- Accès aux rubriques
  acces_toutes_rubriques BOOLEAN DEFAULT TRUE,

  -- Traçabilité
  source_attribution VARCHAR(30) DEFAULT 'individuelle',
  profil_template_id UUID,  -- Si attribution via profil
  attribue_par UUID NOT NULL,
  date_attribution TIMESTAMP DEFAULT NOW(),
  date_fin TIMESTAMP,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_user_modules_user_module_actif UNIQUE (utilisateur_id, module_id, est_actif),
  CONSTRAINT FK_user_modules_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_modules_utilisateur FOREIGN KEY (utilisateur_id) REFERENCES user_utilisateur(id),
  CONSTRAINT FK_user_modules_module FOREIGN KEY (module_id) REFERENCES base_module(id),
  CONSTRAINT FK_user_modules_profil FOREIGN KEY (profil_template_id) REFERENCES user_profil_template(id),
  CONSTRAINT FK_user_modules_attribue_par FOREIGN KEY (attribue_par) REFERENCES user_utilisateur(id),
  CONSTRAINT CK_user_modules_source CHECK (source_attribution IN ('individuelle', 'profil', 'duplication'))
);

-- =====================================
-- TABLE : USER_MODULES_RUBRIQUES  
-- =====================================
-- Description : Attribution de rubriques spécifiques (permissions additionnelles granulaires)
CREATE TABLE user_modules_rubriques (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant et relations
  etablissement_id UUID NOT NULL,
  utilisateur_id UUID NOT NULL,
  module_id UUID NOT NULL,
  rubrique_id UUID NOT NULL,

  -- Traçabilité
  source_attribution VARCHAR(30) DEFAULT 'individuelle',
  profil_template_id UUID,  -- Si attribution via profil
  attribue_par UUID NOT NULL,
  date_attribution TIMESTAMP DEFAULT NOW(),
  date_fin TIMESTAMP,
  est_actif BOOLEAN DEFAULT TRUE,

  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Contraintes
  CONSTRAINT UQ_user_modules_rubriques_user_rubrique_actif UNIQUE (utilisateur_id, rubrique_id, est_actif),
  CONSTRAINT FK_user_modules_rubriques_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_modules_rubriques_utilisateur FOREIGN KEY (utilisateur_id) REFERENCES user_utilisateur(id),
  CONSTRAINT FK_user_modules_rubriques_module FOREIGN KEY (module_id) REFERENCES base_module(id),
  CONSTRAINT FK_user_modules_rubriques_rubrique FOREIGN KEY (rubrique_id) REFERENCES base_rubrique(id),
  CONSTRAINT FK_user_modules_rubriques_profil FOREIGN KEY (profil_template_id) REFERENCES user_profil_template(id),
  CONSTRAINT FK_user_modules_rubriques_attribue_par FOREIGN KEY (attribue_par) REFERENCES user_utilisateur(id),
  CONSTRAINT CK_user_modules_rubriques_source CHECK (source_attribution IN ('individuelle', 'profil', 'duplication'))
);

-- =====================================
-- TABLE : USER_SESSION
-- =====================================
-- Description : Sessions actives utilisateurs (fallback PostgreSQL)
CREATE TABLE user_session (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Multi-tenant
  etablissement_id UUID NOT NULL,
  
  -- Session data
  token UUID NOT NULL,  -- Token format UUID
  user_id UUID NOT NULL,
  client_type VARCHAR(20) NOT NULL,
  
  -- Metadata
  ip_address INET,
  user_agent TEXT,
  last_activity TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL,
  
  -- Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT UQ_user_session_token UNIQUE (token),
  CONSTRAINT FK_user_session_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id),
  CONSTRAINT FK_user_session_user FOREIGN KEY (user_id) REFERENCES user_utilisateur(id),
  CONSTRAINT CK_user_session_client_type CHECK (client_type IN ('front-office', 'back-office'))
);

-- =====================================
-- TABLE : USER_LOGIN_ATTEMPTS
-- =====================================
-- Description : Tentatives de connexion pour rate limiting
CREATE TABLE user_login_attempts (
  -- Clé primaire
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  
  -- Identification
  etablissement_id UUID NOT NULL,
  identifiant VARCHAR(255) NOT NULL,
  
  -- Tentative info
  ip_address INET NOT NULL,
  user_agent TEXT,
  success BOOLEAN DEFAULT FALSE,
  failure_reason VARCHAR(50),
  
  -- Métadonnées
  attempted_at TIMESTAMP DEFAULT NOW(),
  
  -- Contraintes
  CONSTRAINT FK_user_login_attempts_etablissement FOREIGN KEY (etablissement_id) REFERENCES base_etablissement(id)
);

-- =====================================
-- INDEXES POUR PERFORMANCE
-- =====================================

-- Sessions
CREATE INDEX idx_user_session_token ON user_session(token);
CREATE INDEX idx_user_session_user_id ON user_session(user_id);
CREATE INDEX idx_user_session_expires ON user_session(expires_at);

-- Profils et permissions
CREATE INDEX idx_user_profil_utilisateurs_user ON user_profil_utilisateurs(utilisateur_id) WHERE est_actif = TRUE;
CREATE INDEX idx_user_modules_user ON user_modules(utilisateur_id) WHERE est_actif = TRUE;
CREATE INDEX idx_user_modules_rubriques_user ON user_modules_rubriques(utilisateur_id) WHERE est_actif = TRUE;

-- Login attempts
CREATE INDEX idx_user_login_attempts_identifiant ON user_login_attempts(etablissement_id, identifiant, attempted_at);

-- =====================================
-- COMMENTAIRES POUR DOCUMENTATION
-- =====================================

COMMENT ON TABLE user_profil_template IS 'Profils/groupes métier réutilisables avec permissions prédéfinies';
COMMENT ON TABLE user_profil_modules IS 'Modules autorisés pour chaque profil template';
COMMENT ON TABLE user_profil_rubriques IS 'Rubriques spécifiques pour permissions granulaires des profils';
COMMENT ON TABLE user_profil_utilisateurs IS 'Association utilisateurs aux profils avec traçabilité';
COMMENT ON TABLE user_modules IS 'Permissions additionnelles modules par utilisateur';
COMMENT ON TABLE user_modules_rubriques IS 'Permissions additionnelles rubriques par utilisateur';
COMMENT ON TABLE user_session IS 'Sessions PostgreSQL pour fallback Redis et audit';
COMMENT ON TABLE user_login_attempts IS 'Historique tentatives connexion pour rate limiting';

-- =====================================
-- TRIGGERS POUR UPDATED_AT
-- =====================================

CREATE TRIGGER trigger_user_profil_template_updated_at
    BEFORE UPDATE ON user_profil_template
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_profil_utilisateurs_updated_at
    BEFORE UPDATE ON user_profil_utilisateurs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_modules_updated_at
    BEFORE UPDATE ON user_modules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_modules_rubriques_updated_at
    BEFORE UPDATE ON user_modules_rubriques
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_session_updated_at
    BEFORE UPDATE ON user_session
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();