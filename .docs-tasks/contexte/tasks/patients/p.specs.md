# SPÉCIFICATIONS TECHNIQUES - MODULE CORE-SERVICE PATIENT

## 📋 Informations Générales

- **Module :** Core-Service Patient (Services Métier Centralisés)
- **Version :** 1.0 MVP
- **Priorité :** Must Have (Core Infrastructure)
- **Architecture :** Core Services → Queries → DB (PostgreSQL + Redis)

## 🎯 Objectif

Développer un référentiel patient unique et centralisé permettant :

- **Identification unique** : Un patient = Un code unique valable dans tous les établissements
- **Zéro duplication** : Éviter la recréation de fiches patients entre établissements
- **Portabilité** : Le patient conserve son identifiant médical à vie
- **Performance** : Cache intelligent Redis pour accès ultra-rapide
- **Anti-doublon** : Détection intelligente des patients similaires

## 🏗️ Architecture Technique

```
internal/modules/core-services/patient/
├── patient-core.module.go              # Module Fx Core Services
├── services/
│   ├── patient-creation.service.go     # Création avec anti-doublon
│   ├── patient-search.service.go       # Recherche multi-critères
│   ├── patient-validation.service.go   # Validations métier
│   ├── patient-cache.service.go        # Gestion cache Redis
│   └── patient-code-generator.service.go # Génération codes uniques
├── dto/
│   ├── patient-core.dto.go
│   ├── patient-search.dto.go
│   └── patient-creation.dto.go
└── queries/
    ├── patient-core.postgres.go
    ├── patient-search.postgres.go
    └── patient-code-generation.postgres.go
```

---

## 🔧 CS-P-001 : Génération Code Patient (PRÉREQUIS CRITIQUE)

### **Service : GeneratePatientCode**

#### **Règles Métier**

1. **Format standardisé** : `{ETABLISSEMENT}-{YYYY}-{NNN}-{LLL}`
2. **Unicité garantie** : Lock Redis + PostgreSQL pour éviter doublons
3. **Performance optimale** : Génération Redis (< 5ms) avec fallback DB
4. **Capacité massive** : 17.5M codes par établissement/année
5. **Reset annuel** : Retour à 001-AAA chaque 1er janvier
6. **Atomicité** : Transaction PostgreSQL + Redis pour cohérence

#### **Signature Service**

```go
func (s *PatientCodeGeneratorService) GeneratePatientCode(
    ctx context.Context,
    etablissementCode string,
) (string, error)
```

#### **Format de Code**

```
CENTREA-2025-001-AAA
│       │    │   │
│       │    │   └── Suffixe alphabétique (AAA à ZZZ)
│       │    └────── Numéro séquentiel (001 à 999)
│       └────────── Année courante
└────────────────── Code établissement
```

#### **Capacité par Établissement/Année**

- **Par bloc** : 999 codes (001 à 999)
- **Nombre de blocs** : 17 576 (AAA à ZZZ)
- **Total** : **17 558 424 codes** par établissement/année

#### **Algorithme de Génération**

```go
type PatientCodeGenerator struct {
    db    *postgres.Client
    redis *redis.Client
    mu    sync.Map  // Lock en mémoire par établissement
}

// GeneratePatientCode génère un code patient unique atomiquement
func (g *PatientCodeGenerator) GeneratePatientCode(
    ctx context.Context,
    etablissementCode string,
) (string, error) {
    year := time.Now().Year()

    // 1. Tentative rapide via Redis (99% des cas)
    if code, err := g.generateFromRedis(ctx, etablissementCode, year); err == nil {
        return code, nil
    }

    // 2. Fallback PostgreSQL si Redis indisponible
    return g.generateFromPostgres(ctx, etablissementCode, year)
}
```

#### **Tables PostgreSQL Requises**

```sql
-- Table légère pour stocker l'état des séquences
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

-- Index pour performance
CREATE INDEX IDX_patients_sequences_lookup
    ON patients_code_sequences (etablissement_code, annee);
```

#### **Clés Redis Utilisées**

```redis
# Séquence courante
soins_suite_patient_sequence:{etablissement_code}:{year}
Type: STRING
Value: "999:AAB"  # Format: {numero}:{suffixe}
TTL: Fin d'année (calculé dynamiquement)

# Lock anti-concurrence
soins_suite_patient_sequence_lock:{etablissement_code}:{year}
Type: STRING
Value: "1"
TTL: 5 secondes
```

#### **Règles Métier Spécifiques**

- **Lock distribué** : Redis SETNX 5 secondes pour éviter concurrence
- **Progression logique** : 001→002→...→999 puis AAA→AAB→AAC→...→ZZZ
- **Fallback robuste** : Basculement PostgreSQL si Redis indisponible
- **TTL dynamique** : Cache Redis expire automatiquement le 31/12 à 23h59
- **Monitoring** : Métriques Prometheus pour generation_duration et success_rate

---

## 🔧 CS-P-002 : Création d'un Patient

### **Service : CreatePatient**

#### **Règles Métier**

1. **Code patient unique** : Utilisation du service PatientCodeGeneratorService (CS-P-001)
2. **Anti-doublon intelligent** : Vérification sur nom+prénom+date_naissance avec scoring
3. **Transaction atomique** : Patient + assurances + cache warming en une transaction
4. **Validation stricte** : Téléphone, email, dates avec regex Côte d'Ivoire
5. **Cache immédiat** : Warming automatique Redis après création

#### **Signature Service**

```go
func (s *PatientCreationService) CreatePatient(
    ctx context.Context,
    etablissementCode string,
    req *dto.CreatePatientRequest,
) (*dto.PatientResponse, error)
```

#### **Input DTO**

```go
type CreatePatientRequest struct {
    // IDENTITÉ (Obligatoires)
    Nom                     string    `json:"nom" validate:"required,min=2,max=255"`
    Prenoms                 string    `json:"prenoms" validate:"required,min=2,max=255"`
    DateNaissance           time.Time `json:"date_naissance" validate:"required"`
    EstDateSupposee         bool      `json:"est_date_supposee"`
    Sexe                    string    `json:"sexe" validate:"required,oneof=M F"`
    NationaliteID           uuid.UUID `json:"nationalite_id" validate:"required"`
    SituationMatrimonialeID uuid.UUID `json:"situation_matrimoniale_id" validate:"required"`

    // PIÈCE D'IDENTITÉ (Optionnels)
    TypePieceIdentiteID  *uuid.UUID `json:"type_piece_identite_id"`
    CniNni              *string    `json:"cni_nni"`
    NumeroPieceIdentite *string    `json:"numero_piece_identite"`
    LieuNaissance       *string    `json:"lieu_naissance"`
    NomJeuneFille       *string    `json:"nom_jeune_fille"`

    // CONTACT (Requis)
    TelephonePrincipal   string  `json:"telephone_principal" validate:"required,e164"`
    TelephoneSecondaire  *string `json:"telephone_secondaire"`
    Email               *string `json:"email" validate:"omitempty,email"`

    // LOCALISATION
    AdresseComplete string  `json:"adresse_complete" validate:"required"`
    Quartier       *string `json:"quartier"`
    Ville          *string `json:"ville"`
    Commune        *string `json:"commune"`
    PaysResidence  string  `json:"pays_residence" default:"Côte d'Ivoire"`

    // SOCIO-PROFESSIONNEL
    ProfessionID *uuid.UUID `json:"profession_id"`

    // PERSONNES À CONTACTER
    PersonnesAContacter []PersonneContact `json:"personnes_a_contacter"`

    // ASSURANCE
    EstAssure bool                    `json:"est_assure"`
    Assurances []CreateAssuranceData `json:"assurances" validate:"required_if=EstAssure true"`
}

type PersonneContact struct {
    NomPrenoms           string     `json:"nom_prenoms" validate:"required"`
    Telephone           string     `json:"telephone" validate:"required,e164"`
    TelephoneSecondaire *string    `json:"telephone_secondaire"`
    AffiliationID       uuid.UUID  `json:"affiliation_id" validate:"required"`
}

type CreateAssuranceData struct {
    AssuranceID              uuid.UUID `json:"assurance_id" validate:"required"`
    NumeroAssure            string    `json:"numero_assure" validate:"required"`
    TypeBeneficiaire        string    `json:"type_beneficiaire" validate:"oneof=principal ayant_droit" default:"principal"`
    NumeroAssurePrincipal   *string   `json:"numero_assure_principal"`
    LienAvecPrincipal       *string   `json:"lien_avec_principal" validate:"required_if=TypeBeneficiaire ayant_droit"`
}
```

#### **Output DTO**

```go
type PatientResponse struct {
    ID           uuid.UUID `json:"id"`
    CodePatient  string    `json:"code_patient"`

    // Identité
    Nom                     string     `json:"nom"`
    Prenoms                 string     `json:"prenoms"`
    DateNaissance           time.Time  `json:"date_naissance"`
    EstDateSupposee         bool       `json:"est_date_supposee"`
    Sexe                    string     `json:"sexe"`

    // Contact
    TelephonePrincipal      string     `json:"telephone_principal"`
    TelephoneSecondaire     *string    `json:"telephone_secondaire"`
    Email                   *string    `json:"email"`
    AdresseComplete         string     `json:"adresse_complete"`

    // Assurance
    EstAssure               bool                  `json:"est_assure"`
    Assurances             []AssuranceResponse   `json:"assurances,omitempty"`

    // Métadonnées
    EtablissementCreateurID uuid.UUID `json:"etablissement_createur_id"`
    Statut                 string     `json:"statut"`
    CreatedAt              time.Time  `json:"created_at"`
    CreatedBy              *UserInfo  `json:"created_by,omitempty"`
}

type AssuranceResponse struct {
    ID                    uuid.UUID `json:"id"`
    AssuranceNom         string    `json:"assurance_nom"`
    NumeroAssure         string    `json:"numero_assure"`
    TypeBeneficiaire     string    `json:"type_beneficiaire"`
    EstActif             bool      `json:"est_actif"`
}
```

#### **Règles Métier Spécifiques**

- **Code unique** : Format `CENTREA-2025-001-AAA` avec incrémentation automatique
- **Anti-doublon** : Score > 85% = blocage, 70-85% = alerte, < 70% = création
- **Validation téléphone** : Regex `^(\+225|00225)?[0-9]{10}$` pour Côte d'Ivoire
- **Cache warming** : Patient mis en Redis immédiatement après création
- **Audit trail** : Enregistrement complet dans logs avec utilisateur créateur

---

## 🔧 CS-P-002 : Recherche de Patients

### **Service : SearchPatients**

#### **Règles Métier**

1. **Stratégie cache-first** : Recherche par code_patient via Redis (< 5ms)
2. **Full-text PostgreSQL** : Utilisation search_vector pour recherche textuelle
3. **Pagination obligatoire** : Max 50 résultats par recherche
4. **Tri intelligent** : Score de pertinence en priorité
5. **Filtres cumulatifs** : Combinaison AND de tous les critères

#### **Signature Service**

```go
func (s *PatientSearchService) SearchPatients(
    ctx context.Context,
    req *dto.SearchPatientRequest,
) (*dto.SearchPatientResponse, error)
```

#### **Input DTO**

```go
type SearchPatientRequest struct {
    // RECHERCHE DIRECTE
    CodePatient string `json:"code_patient"`

    // RECHERCHE TEXTUELLE
    SearchTerm string `json:"search_term"` // Nom, prénom, téléphone, CNI

    // FILTRES PRÉCIS
    Nom                    *string    `json:"nom"`
    Prenoms               *string    `json:"prenoms"`
    TelephonePrincipal    *string    `json:"telephone_principal"`
    DateNaissance         *time.Time `json:"date_naissance"`
    DateNaissanceDebut    *time.Time `json:"date_naissance_debut"`
    DateNaissanceFin      *time.Time `json:"date_naissance_fin"`
    Sexe                  *string    `json:"sexe" validate:"omitempty,oneof=M F"`
    CniNni                *string    `json:"cni_nni"`

    // FILTRES STATUT
    Statut                []string   `json:"statut"` // actif, inactif, decede, archive
    EstAssure             *bool      `json:"est_assure"`
    EtablissementCreateur *uuid.UUID `json:"etablissement_createur_id"`

    // PAGINATION & TRI
    Page       int    `json:"page" validate:"min=1" default:"1"`
    Limit      int    `json:"limit" validate:"min=1,max=50" default:"20"`
    SortBy     string `json:"sort_by" validate:"oneof=nom created_at score" default:"score"`
    SortOrder  string `json:"sort_order" validate:"oneof=asc desc" default:"desc"`

    // OPTIONS
    IncludeAssurances bool `json:"include_assurances"`
}
```

#### **Output DTO**

```go
type SearchPatientResponse struct {
    Patients   []PatientSearchResult `json:"patients"`
    Pagination PaginationInfo        `json:"pagination"`
    SearchInfo SearchMetadata        `json:"search_info"`
}

type PatientSearchResult struct {
    ID               uuid.UUID `json:"id"`
    CodePatient      string    `json:"code_patient"`
    Nom              string    `json:"nom"`
    Prenoms          string    `json:"prenoms"`
    DateNaissance    time.Time `json:"date_naissance"`
    Sexe             string    `json:"sexe"`
    TelephonePrincipal string  `json:"telephone_principal"`
    AdresseComplete   string   `json:"adresse_complete"`
    EstAssure        bool      `json:"est_assure"`
    Statut           string    `json:"statut"`

    // MÉTADONNÉES RECHERCHE
    Score            *float64             `json:"score,omitempty"` // Score pertinence full-text
    Assurances       []AssuranceResponse  `json:"assurances,omitempty"`

    CreatedAt        time.Time `json:"created_at"`
}

type SearchMetadata struct {
    SearchType       string        `json:"search_type"` // "direct_code", "full_text", "criteria"
    ExecutionTimeMs  int           `json:"execution_time_ms"`
    CacheHit         bool          `json:"cache_hit"`
    TotalResults     int           `json:"total_results"`
    AppliedFilters   []string      `json:"applied_filters"`
}
```

#### **Règles Métier Spécifiques**

- **Cache Redis prioritaire** : Si code_patient fourni, accès direct cache
- **Search vector** : PostgreSQL GIN index pour recherche textuelle performante
- **Score pertinence** : Calcul automatique avec ts_rank pour full-text
- **Limite performance** : Max 50 résultats, pagination obligatoire
- **Filtrage intelligent** : Combinaison optimisée des critères SQL

---

## 🔧 CS-P-003 : Récupération Patient par Code

### **Service : GetPatientByCode**

#### **Règles Métier**

1. **Cache-first strategy** : Redis en priorité (< 1ms)
2. **Fallback PostgreSQL** : Si cache miss, chargement DB + warming
3. **Données complètes** : Patient + assurances + personnes à contacter
4. **Statut validation** : Exclusion automatique patients archivés
5. **Audit access** : Log consultation pour patients sensibles

#### **Signature Service**

```go
func (s *PatientCacheService) GetPatientByCode(
    ctx context.Context,
    codePatient string,
) (*dto.PatientDetailResponse, error)
```

#### **Output DTO**

```go
type PatientDetailResponse struct {
    // PATIENT PRINCIPAL
    Patient PatientResponse `json:"patient"`

    // RÉFÉRENCES ENRICHIES
    Nationalite         ReferenceInfo `json:"nationalite"`
    SituationMatrimoniale ReferenceInfo `json:"situation_matrimoniale"`
    TypePieceIdentite   *ReferenceInfo `json:"type_piece_identite,omitempty"`
    Profession          *ReferenceInfo `json:"profession,omitempty"`

    // PERSONNES À CONTACTER
    PersonnesAContacter []PersonneContactDetail `json:"personnes_a_contacter"`

    // ASSURANCES DÉTAILLÉES
    AssurancesDetails []AssuranceDetail `json:"assurances_details"`

    // MÉTADONNÉES
    LoadedFrom   string    `json:"loaded_from"` // "cache" ou "database"
    LoadTime     int       `json:"load_time_ms"`
    LastUpdated  time.Time `json:"last_updated"`
}

type PersonneContactDetail struct {
    NomPrenoms           string        `json:"nom_prenoms"`
    Telephone           string        `json:"telephone"`
    TelephoneSecondaire *string       `json:"telephone_secondaire"`
    Affiliation         ReferenceInfo `json:"affiliation"`
}

type AssuranceDetail struct {
    ID                    uuid.UUID     `json:"id"`
    Assurance            ReferenceInfo `json:"assurance"`
    NumeroAssure         string        `json:"numero_assure"`
    TypeBeneficiaire     string        `json:"type_beneficiaire"`
    NumeroAssurePrincipal *string       `json:"numero_assure_principal"`
    LienAvecPrincipal    *string       `json:"lien_avec_principal"`
    EstActif             bool          `json:"est_actif"`
    CreatedAt            time.Time     `json:"created_at"`
}

type ReferenceInfo struct {
    ID   uuid.UUID `json:"id"`
    Code string    `json:"code"`
    Nom  string    `json:"nom"`
}
```

#### **Règles Métier Spécifiques**

- **TTL Redis** : 1 heure pour données patient
- **Lazy loading** : Chargement assurances uniquement si est_assure = true
- **Enrichissement** : Résolution automatique des références (nationalité, etc.)
- **Performance** : < 1ms cache hit, < 50ms cache miss avec warming
- **Exclusion** : Patients avec statut "archive" retournent 404

---

## 🔧 CS-P-004 : Mise à Jour Patient

### **Service : UpdatePatient**

#### **Règles Métier**

1. **Modification partielle** : Seuls les champs fournis sont modifiés
2. **Code immutable** : Le code_patient ne peut jamais être changé
3. **Validation cohérence** : Mêmes règles que création
4. **Cache invalidation** : Suppression Redis immédiate après modification
5. **Historique préservé** : Enregistrement updated_by et updated_at

#### **Signature Service**

```go
func (s *PatientValidationService) UpdatePatient(
    ctx context.Context,
    codePatient string,
    req *dto.UpdatePatientRequest,
) (*dto.PatientResponse, error)
```

#### **Input DTO**

```go
type UpdatePatientRequest struct {
    // IDENTITÉ (Optionnels - seuls les fournis sont modifiés)
    Nom                     *string    `json:"nom,omitempty"`
    Prenoms                 *string    `json:"prenoms,omitempty"`
    DateNaissance           *time.Time `json:"date_naissance,omitempty"`
    EstDateSupposee         *bool      `json:"est_date_supposee,omitempty"`
    Sexe                    *string    `json:"sexe,omitempty" validate:"omitempty,oneof=M F"`
    NationaliteID           *uuid.UUID `json:"nationalite_id,omitempty"`
    SituationMatrimonialeID *uuid.UUID `json:"situation_matrimoniale_id,omitempty"`

    // CONTACT
    TelephonePrincipal   *string `json:"telephone_principal,omitempty" validate:"omitempty,e164"`
    TelephoneSecondaire  *string `json:"telephone_secondaire,omitempty"`
    Email               *string `json:"email,omitempty" validate:"omitempty,email"`

    // LOCALISATION
    AdresseComplete *string `json:"adresse_complete,omitempty"`
    Quartier       *string `json:"quartier,omitempty"`
    Ville          *string `json:"ville,omitempty"`
    Commune        *string `json:"commune,omitempty"`

    // STATUT & MÉTADONNÉES
    Statut            *string `json:"statut,omitempty" validate:"omitempty,oneof=actif inactif decede archive"`
    EstDecede         *bool   `json:"est_decede,omitempty"`
    DateDeces         *time.Time `json:"date_deces,omitempty"`

    // PERSONNES À CONTACTER (remplacement complet si fourni)
    PersonnesAContacter *[]PersonneContact `json:"personnes_a_contacter,omitempty"`
}
```

#### **Règles Métier Spécifiques**

- **Validation conditionnelle** : EstDecede = true nécessite DateDeces
- **Cache invalidation** : DEL Redis automatique après succès transaction
- **Atomicité** : Transaction PostgreSQL pour cohérence
- **Audit** : Enregistrement utilisateur et timestamp modification
- **Immutabilité** : Code patient, ID, dates création non modifiables

---

## 🔧 CS-P-005 : Validation Anti-Doublon

### **Service : CheckPatientDuplicate**

#### **Règles Métier**

1. **Algorithme intelligent** : Scoring basé sur nom+prénom+date_naissance
2. **Seuils configurables** : Score > 85% = blocage, 70-85% = alerte
3. **Normalisation** : Suppression accents, espaces, casse pour comparaison
4. **Exclusion logique** : Ignore patients archivés ou décédés
5. **Performance** : Utilisation index trigram PostgreSQL

#### **Signature Service**

```go
func (s *PatientValidationService) CheckPatientDuplicate(
    ctx context.Context,
    req *dto.DuplicateCheckRequest,
) (*dto.DuplicateCheckResponse, error)
```

#### **Input DTO**

```go
type DuplicateCheckRequest struct {
    Nom             string    `json:"nom" validate:"required"`
    Prenoms         string    `json:"prenoms" validate:"required"`
    DateNaissance   time.Time `json:"date_naissance" validate:"required"`
    TelephonePrincipal *string `json:"telephone_principal"`

    // OPTIONS
    ScoreMinimum    int  `json:"score_minimum" default:"70"`
    LimiteResultats int  `json:"limite_resultats" default:"5"`
}
```

#### **Output DTO**

```go
type DuplicateCheckResponse struct {
    HasDuplicates    bool                    `json:"has_duplicates"`
    HighestScore     int                     `json:"highest_score"`
    Recommendation   string                  `json:"recommendation"` // "BLOCK", "WARN", "ALLOW"
    PotentialMatches []PotentialDuplicate    `json:"potential_matches"`
    CheckExecutedAt  time.Time               `json:"check_executed_at"`
}

type PotentialDuplicate struct {
    Patient       PatientSearchResult `json:"patient"`
    Score         int                `json:"score"`
    MatchDetails  MatchDetail        `json:"match_details"`
}

type MatchDetail struct {
    NomMatch          int  `json:"nom_match"`           // Score 0-100
    PrenomsMatch      int  `json:"prenoms_match"`       // Score 0-100
    DateNaissanceMatch int `json:"date_naissance_match"` // Score 0-100
    TelephoneMatch    int  `json:"telephone_match"`     // Score 0-100
    ScoreGlobal      int  `json:"score_global"`        // Score 0-100
}
```

#### **Règles Métier Spécifiques**

- **Algorithme de scoring** : Levenshtein distance + pondération champs
- **Normalisation intelligente** : Suppression accents avec unaccent PostgreSQL
- **Exclusion patients** : WHERE statut NOT IN ('archive', 'decede')
- **Performance optimisée** : Index GIN trigram sur colonnes nom/prenoms
- **Seuils recommandés** : 85%+ = blocage, 70-84% = alerte, <70% = autorisation

---

## 📊 Requêtes SQL Optimisées

### CreatePatientWithValidation

```sql
-- Transaction complète création patient avec vérifications
BEGIN;

-- 1. Vérification anti-doublon
WITH patient_scores AS (
    SELECT
        id,
        code_patient,
        nom,
        prenoms,
        date_naissance,
        telephone_principal,
        GREATEST(
            similarity(unaccent(LOWER(nom)), unaccent(LOWER($1))),
            similarity(unaccent(LOWER(prenoms)), unaccent(LOWER($2)))
        ) * 100 as score_textuel,
        CASE
            WHEN date_naissance = $3 THEN 100
            WHEN ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 7 THEN 90
            WHEN ABS(EXTRACT(DAYS FROM date_naissance - $3::date)) <= 30 THEN 70
            ELSE 0
        END as score_date,
        CASE
            WHEN telephone_principal = $4 THEN 100
            ELSE 0
        END as score_telephone
    FROM patients_patient
    WHERE statut NOT IN ('archive', 'decede')
    AND (
        similarity(unaccent(LOWER(nom)), unaccent(LOWER($1))) > 0.3 OR
        similarity(unaccent(LOWER(prenoms)), unaccent(LOWER($2))) > 0.3 OR
        date_naissance = $3 OR
        telephone_principal = $4
    )
)
SELECT
    id,
    code_patient,
    nom,
    prenoms,
    ((score_textuel * 0.4) + (score_date * 0.4) + (score_telephone * 0.2)) as score_global
FROM patient_scores
WHERE ((score_textuel * 0.4) + (score_date * 0.4) + (score_telephone * 0.2)) >= 70
ORDER BY score_global DESC
LIMIT 5;

-- 2. Si score < 85%, insertion patient
INSERT INTO patients_patient (
    id, code_patient, etablissement_createur_id,
    nom, prenoms, date_naissance, est_date_supposee, sexe,
    nationalite_id, situation_matrimoniale_id,
    telephone_principal, telephone_secondaire, email,
    adresse_complete, quartier, ville, commune, pays_residence,
    profession_id, personnes_a_contacter,
    est_assure, statut, created_by, created_at, updated_at
) VALUES (
    $5, $6, $7, $1, $2, $3, $8, $9, $10, $11, $4, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21, 'actif', $22, NOW(), NOW()
) RETURNING *;

-- 3. Insertion assurances si est_assure = true
INSERT INTO patients_patient_assurance (
    patient_id, assurance_id, numero_assure, type_beneficiaire,
    numero_assure_principal, lien_avec_principal, created_by
)
SELECT $5, unnest($23::uuid[]), unnest($24::text[]), unnest($25::text[]),
       unnest($26::text[]), unnest($27::text[]), $22
WHERE $21 = true;

COMMIT;
```

### SearchPatientsFullText

```sql
-- Recherche full-text optimisée avec scoring
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
        ELSE NULL
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
    AND ($9::boolean IS NULL OR p.est_assure = $9)
    AND ($10::uuid IS NULL OR p.etablissement_createur_id = $10)
ORDER BY
    CASE
        WHEN $11 = 'score' AND $1 != '' THEN ts_rank(p.search_vector, plainto_tsquery('french', $1))
        WHEN $11 = 'nom' THEN p.nom
        WHEN $11 = 'created_at' THEN extract(epoch from p.created_at)
        ELSE extract(epoch from p.created_at)
    END DESC
LIMIT $12 OFFSET $13;
```

### GetPatientByCodeWithAssurances

```sql
-- Chargement patient complet avec références enrichies
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
    pd.*,
    COALESCE(pa.assurances, '[]'::jsonb) as assurances_details
FROM patient_data pd
LEFT JOIN patient_assurances pa ON pd.id = pa.patient_id;
```

---

## 🔴 Schéma Redis - Cache Optimisé

### Clés Redis Utilisées

```
# 1. Cache patient complet (PRINCIPALE)
soins_suite_patient:{code_patient}
Type: HASH
TTL: 3600s (1 heure)

# 2. Séquences génération code (GÉNÉRATION)
soins_suite_patient_sequence:{etablissement_code}:{year}
Type: STRING
TTL: Fin d'année dynamique

# 3. Lock génération code (ANTI-CONCURRENCE)
soins_suite_patient_sequence_lock:{etablissement_code}:{year}
Type: STRING
TTL: 5 secondes
```

### Stratégie Cache

1. **Lecture** : Cache-first avec fallback PostgreSQL + warming
2. **Écriture** : Write-through (DB puis cache)
3. **Invalidation** : Delete immédiat sur modification
4. **Warming** : Automatic après création/modification
5. **TTL** : 1 heure pour équilibre fraîcheur/performance

---

## 🛡️ Sécurité et Validation

### Validations Critiques

```go
type PatientValidator struct {
    // Formats Côte d'Ivoire
    PhoneRegex    = regexp.MustCompile(`^(\+225|00225)?[0-9]{10}$`)
    EmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    CodeRegex     = regexp.MustCompile(`^[A-Z0-9]+-\d{4}-\d{3}-[A-Z]{3}$`)

    // Limites business
    MaxPersonnesContact = 5
    MaxAssurances      = 3
    MinAge             = 0
    MaxAge             = 150
}
```

### Événements Audit

```json
{
  "event_type": "PATIENT_CREATED",
  "timestamp": "2025-01-15T15:30:00Z",
  "actor": {
    "user_id": "user-uuid",
    "etablissement_code": "CENTREA"
  },
  "patient": {
    "id": "patient-uuid",
    "code_patient": "CENTREA-2025-001-AAA",
    "duplicate_score": 45
  },
  "metadata": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  }
}
```

---

## ⚡ Optimisations Performance

### Index PostgreSQL Critiques

| Index                                             | Justification                  | Gain Estimé   |
| ------------------------------------------------- | ------------------------------ | ------------- |
| `UNIQUE (code_patient)`                           | Recherche principale (95% cas) | 50ms → 1ms    |
| `GIN (search_vector)`                             | Full-text search               | 1000ms → 50ms |
| `GIN (nom gin_trgm_ops, prenoms gin_trgm_ops)`    | Anti-doublon trigram           | 800ms → 30ms  |
| `(telephone_principal)`                           | Identification urgence         | 300ms → 15ms  |
| `(etablissement_createur_id, statut, created_at)` | Filtrage multi-établissement   | 200ms → 20ms  |

### Métriques Performance Cibles

- **Création patient** : < 100ms (avec anti-doublon)
- **Recherche par code** : < 5ms (cache) / < 50ms (DB)
- **Recherche full-text** : < 100ms pour 1M+ patients
- **Génération code** : < 5ms (Redis) / < 50ms (fallback DB)
- **Détection doublons** : < 200ms avec scoring

---

## ✅ Checklist Implémentation

### Services Core (Par Ordre de Priorité)

- [ ] `PatientCodeGeneratorService` - **CRITIQUE** : Génération codes uniques
- [ ] `PatientValidationService` - Validations métier et anti-doublon
- [ ] `PatientCreationService` - Création avec anti-doublon
- [ ] `PatientCacheService` - Cache Redis intelligent
- [ ] `PatientSearchService` - Recherche multi-critères

### Queries PostgreSQL (Par Ordre de Priorité)

- [ ] `GenerateNextPatientCode` - **CRITIQUE** : Séquence atomique
- [ ] `CheckDuplicateWithScoring` - Anti-doublon intelligent
- [ ] `CreatePatientWithValidation` - Transaction complète
- [ ] `GetPatientByCodeWithAssurances` - Chargement enrichi
- [ ] `SearchPatientsFullText` - Recherche avec scoring

### DTOs & Validation

- [ ] `CreatePatientRequest` - Validation stricte création
- [ ] `SearchPatientRequest` - Critères recherche flexibles
- [ ] `PatientDetailResponse` - Données complètes enrichies
- [ ] `DuplicateCheckResponse` - Scoring et recommandations

### Infrastructure

- [ ] Module Fx Core Services complet
- [ ] Cache Redis avec TTL dynamique
- [ ] Index PostgreSQL optimisés
- [ ] Audit trail sécurisé
- [ ] Tests unitaires et intégration

---

**Cette spécification garantit un référentiel patient unique, performant et sécurisé, éliminant les doublons tout en maintenant d'excellentes performances grâce au cache intelligent Redis et aux requêtes PostgreSQL optimisées.**
