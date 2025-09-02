✅ Conventions Base De Données - POSTGRESQL

## NOMENCLATURE DES CONTRAINTES

### **1. FOREIGN KEYS (FK)**

```sql
CONSTRAINT FK_[table]_[colonne] FOREIGN KEY ([colonne]) REFERENCES [table_ref]([colonne_ref])
```

**Format :**

- **Préfixe** : `FK_`
- **Table source** : Nom de la table qui contient la clé étrangère
- **Colonne** : Nom de la colonne clé étrangère (sans `_id` si évident)

**Exemples :**

```sql
-- Table base_rubrique avec module_id → base_module(id)
CONSTRAINT FK_base_rubrique_module_id FOREIGN KEY (module_id) REFERENCES base_module(id)

-- Table user_modules avec utilisateur_id → user_utilisateur(id)
CONSTRAINT FK_user_modules_utilisateur_id FOREIGN KEY (utilisateur_id) REFERENCES user_utilisateur(id)

-- Table base_prestation_medicale avec type_prestation_id → base_type_prestation(id)
CONSTRAINT FK_base_prestation_medicale_type_prestation_id FOREIGN KEY (type_prestation_id) REFERENCES base_type_prestation(id)
```

### **2. UNIQUE CONSTRAINTS (UQ)**

```sql
CONSTRAINT UQ_[table]_[colonne(s)] UNIQUE ([colonne(s)])
```

**Format :**

- **Préfixe** : `UQ_`
- **Table** : Nom de la table
- **Colonne(s)** : Nom de la/les colonne(s) concernée(s)

**Exemples simples :**

```sql
-- Unicité sur une seule colonne
CONSTRAINT UQ_base_licence_cle_licence UNIQUE (cle_licence)
CONSTRAINT UQ_user_session_token UNIQUE (token)
CONSTRAINT UQ_base_assurance_code_organisme UNIQUE (code_organisme)
```

**Exemples composites :**

```sql
-- Unicité sur plusieurs colonnes
CONSTRAINT UQ_user_modules_utilisateur_module UNIQUE (utilisateur_id, module_id)
CONSTRAINT UQ_base_prestation_medicale_libelle_type UNIQUE (type_prestation_id, libelle)
CONSTRAINT UQ_user_profil_modules_profil_module UNIQUE (profil_template_id, module_id)
```

### **3. CHECK CONSTRAINTS (CK)**

```sql
CONSTRAINT CK_[table]_[description] CHECK ([condition])
```

**Format :**

- **Préfixe** : `CK_`
- **Table** : Nom de la table
- **Description** : Description fonctionnelle de la contrainte

**Exemples simples :**

```sql
-- Validation enum/valeurs autorisées
CONSTRAINT CK_base_licence_type_licence CHECK (type_licence IN ('premium', 'standard', 'evaluation'))
CONSTRAINT CK_user_session_client_type CHECK (client_type IN ('front-office', 'back-office'))

-- Validation valeurs positives
CONSTRAINT CK_base_tarif_prestation_positif CHECK (tarif_unitaire > 0)
CONSTRAINT CK_base_heure_ouverture_jour_semaine CHECK (jour_semaine BETWEEN 1 AND 7)
```

**Exemples complexes :**

```sql
-- Validation cohérence temporelle
CONSTRAINT CK_base_heure_ouverture_coherence CHECK (heure_fin > heure_debut)
CONSTRAINT CK_circuit_periode_coherence CHECK (
  date_fin_activite IS NULL OR date_fin_activite > date_debut_activite
)

-- Validation logique métier
CONSTRAINT CK_base_licence_expiration_coherence CHECK (
  (type_licence = 'premium' AND date_expiration IS NULL) OR
  (type_licence IN ('standard', 'evaluation') AND date_expiration IS NOT NULL)
)

-- Validation exclusive
CONSTRAINT CK_circuit_type_application CHECK (
  (type_application = 'defaut' AND module_service_id IS NULL AND type_prestation_id IS NULL) OR
  (type_application = 'module_service' AND module_service_id IS NOT NULL AND type_prestation_id IS NULL) OR
  (type_application = 'type_prestation' AND type_prestation_id IS NOT NULL AND module_service_id IS NULL)
)
```

### **4. INDEX UNIQUES CONDITIONNELS**

```sql
CREATE UNIQUE INDEX UQ_[description]_actif ON [table] ([colonnes]) WHERE ([conditions])
```

**Format :**

- **Préfixe** : `UQ_`
- **Description** : Description fonctionnelle
- **Suffixe** : Condition principale (ex: `_actif`)

**Exemples :**

```sql
-- Unicité temporelle avec condition
CREATE UNIQUE INDEX UQ_circuit_defaut_actif
ON base_circuit_patient (type_application)
WHERE (type_application = 'defaut' AND est_actif = TRUE AND date_fin_activite IS NULL);

CREATE UNIQUE INDEX UQ_circuit_module_service_actif
ON base_circuit_patient (module_service_id)
WHERE (type_application = 'module_service' AND est_actif = TRUE AND date_fin_activite IS NULL);
```

## RÈGLES GÉNÉRALES

### **❌ ÉVITER (Contraintes implicites)**

```sql
-- JAMAIS utiliser les contraintes implicites
cle_licence VARCHAR(50) UNIQUE NOT NULL,
token VARCHAR(64) NOT NULL UNIQUE,
etablissement_code VARCHAR(10) NOT NULL UNIQUE,
```

### **✅ TOUJOURS UTILISER (Contraintes nommées)**

```sql
-- TOUJOURS nommer explicitement les contraintes
cle_licence VARCHAR(50) NOT NULL,
token VARCHAR(64) NOT NULL,
etablissement_code VARCHAR(10) NOT NULL,

-- Dans la section contraintes
CONSTRAINT UQ_base_licence_cle_licence UNIQUE (cle_licence),
CONSTRAINT UQ_user_session_token UNIQUE (token),
CONSTRAINT UQ_base_etablissement_etablissement_code UNIQUE (etablissement_code),
```

## AVANTAGES

1. **Lisibilité** : Nom de contrainte explicite et prévisible
2. **Maintenance** : Facile d'identifier et modifier les contraintes
3. **Debug** : Messages d'erreur PostgreSQL plus clairs
4. **Documentation** : Auto-documenté par le nom
5. **Cohérence** : Standard uniforme dans toute la base

## ORDRE DE DÉCLARATION

Dans chaque table, respecter cet ordre :

```sql
CREATE TABLE example (
  -- 1. Colonnes avec leurs types
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL,

  -- 2. Métadonnées standards
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- 3. Contraintes nommées (dans l'ordre : UQ, FK, CK)
  CONSTRAINT UQ_example_name UNIQUE (name),
  CONSTRAINT FK_example_parent_id FOREIGN KEY (parent_id) REFERENCES parent(id),
  CONSTRAINT CK_example_status CHECK (status IN ('active', 'inactive'))
);
```
