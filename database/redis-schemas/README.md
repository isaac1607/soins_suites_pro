# Redis Schema Management - Soins Suite

## 🎯 Vue d'Ensemble

Ce dossier centralise **tous les schémas et patterns Redis** pour éviter les incohérences et doublons de clés dans l'application Soins Suite.

**Principe fondamental :** Toute clé Redis DOIT être documentée ici avant d'être implémentée dans le code.

## 📋 Convention Obligatoire

Toutes les clés Redis suivent **strictement** la convention définie dans `conventions.md` :

```
soins_suite_{code_etablissement}_{domain}_{context}:{identifier}
```

## 📁 Structure

```
database/redis-schemas/
├── 00.core-patterns.md        # Conventions de base et patterns génériques
├── 01.middleware-keys.md       # Clés utilisées par les middlewares
├── 02.system-keys.md          # Clés du module System
├── 03.auth-keys.md            # Clés du module Auth
├── XX.zzzz-keys.md            # Clés du module ZZZZ
└── README.md                  # Ce fichier (guide d'usage)
```

## 🚀 Pourquoi Cette Documentation

### ✅ **Problèmes résolus**

- **Conflits de clés** ❌ → **Namespace centralisé multi-tenant** ✅
- **TTL incohérents** ❌ → **TTL standardisés par domaine** ✅
- **Clés non documentées** ❌ → **Documentation obligatoire** ✅
- **Duplication logique** ❌ → **Réutilisation forcée** ✅
- **Maintenance chaotique** ❌ → **Source de vérité unique** ✅

### ✅ **Principe Multi-Tenant**

Chaque clé Redis est **isolée par établissement** selon la convention :

- ✅ `soins_suite_CENTREA_cache_middleware:establishment`
- ✅ `soins_suite_HOPITAL_cache_middleware:establishment`
- ❌ `cache_middleware:establishment` (violation isolation)

## 📋 Workflow Obligatoire pour les Développeurs

### **🚨 AVANT d'écrire du code Redis**

1. **Vérifier** si une clé similaire existe déjà dans ces fichiers
2. **Si oui** → Réutiliser la clé existante (pas de duplication)
3. **Si non** → Documenter la nouvelle clé dans le bon fichier `.md`
4. **Seulement après** → Implémenter dans le code

### **1. Créer une Nouvelle Clé Redis**

**Étapes obligatoires :**

1. **Choisir le fichier** selon le domaine :

   - Middleware → `01.middleware-keys.md`
   - System → `02.system-keys.md`
   - Auth → `03.auth-keys.md`
   - Etc ... → `04.etc-keys.md`

2. **Documenter la clé** avec :

   - Pattern exact selon convention
   - TTL justifié
   - Contenu JSON type
   - Stratégie d'usage

3. **Ajouter au code** uniquement après documentation

### **2. Utiliser une Clé Existante**

```go
// 1. Consulter la doc pour le pattern exact
// 2. Utiliser via client standardisé
func (s *MyService) CacheData(ctx context.Context, code string, data interface{}) error {
    // Pattern documenté dans 01.middleware-keys.md
    return s.redisClient.SetWithPattern(ctx, "cache_middleware", code, data, "establishment")
}
```

## ⚠️ Règles Strictes

### **🚨 OBLIGATOIRE**

1. **Toute clé Redis** DOIT être documentée ici AVANT l'implémentation
2. **Aucune clé manuelle** - Utiliser uniquement les patterns documentés
3. **Convention respectée** - Pattern multi-tenant obligatoire
4. **Pas de duplication** - Vérifier l'existant avant créer du nouveau

### **❌ INTERDIT**

- Créer des clés sans documentation préalable
- Violer la convention multi-tenant
- Dupliquer des logiques existantes
- Hardcoder des clés dans le code

### **✅ RECOMMANDÉ**

- Consulter ces fichiers avant tout développement Redis
- Réutiliser les patterns existants
- Proposer des améliorations via les issues du projet

## 🎯 Objectif Final

**Une seule source de vérité pour toutes les clés Redis**, garantissant :

- 🔒 **Isolation multi-tenant** parfaite
- 📖 **Documentation vivante** et à jour
- 🚫 **Zéro duplication** de logique
- 🛠️ **Maintenance simplifiée** pour toute l'équipe

---

**📍 Règle d'or** : _Si ce n'est pas documenté ici, cela ne doit pas exister dans le code !_
