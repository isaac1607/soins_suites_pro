# Configuration Atlas pour Soins Suite - Mode Schema-Driven
# Variables d'environnement requises pour chaque environnement

# Variables globales
variable "database_url" {
  type = string
  description = "URL de connexion PostgreSQL"
}

variable "dev_database_url" {
  type = string
  description = "URL de connexion PostgreSQL pour base temporaire Atlas"
}

variable "schema_name" {
  type = string
  default = "public"
  description = "Nom du schéma PostgreSQL"
}

# Environnement de développement
env "development" {
  # Source : connexion à la base de données de développement
  src = var.database_url
  
  # Base temporaire pour calculs Atlas (utilise variable d'environnement)
  dev = var.dev_database_url
  
  # Configuration des migrations
  migration {
    # Répertoire des fichiers de migration
    dir = "file://migrations/postgresql"
  }
}