#!/bin/bash

# Script pour supprimer et recréer les bases de données PostgreSQL
# Utilise les paramètres du fichier .env

set -e

echo "🗑️  Suppression des bases de données existantes..."

# Terminer toutes les connexions actives sur soins_suite
psql -U postgres -h localhost -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'soins_suite' AND pid <> pg_backend_pid();" 2>/dev/null || true

# Terminer toutes les connexions actives sur soins_suite_atlas
psql -U postgres -h localhost -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'soins_suite_atlas' AND pid <> pg_backend_pid();" 2>/dev/null || true

# Suppression de la base soins_suite
psql -U postgres -h localhost -c "DROP DATABASE IF EXISTS soins_suite;" 

# Suppression de la base soins_suite_atlas
psql -U postgres -h localhost -c "DROP DATABASE IF EXISTS soins_suite_atlas;"

echo "✅ Bases de données supprimées"

echo "🔧 Création des nouvelles bases de données..."

# Création de la base soins_suite
psql -U postgres -h localhost -c "CREATE DATABASE soins_suite;" 

# Création de la base soins_suite_atlas
psql -U postgres -h localhost -c "CREATE DATABASE soins_suite_atlas;"

echo "✅ Bases de données créées avec succès"
echo "📋 Bases disponibles:"
echo "   - soins_suite (principale)"
echo "   - soins_suite_atlas (migrations Atlas)"