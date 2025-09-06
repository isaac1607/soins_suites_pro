# Configuration Atlas pour Soins Suite
# Utilise des variables d'environnement pour éviter d'exposer les informations sensibles

# Variables pour les URLs de base de données
# Ces variables doivent être définies dans l'environnement ou dans .env (non versionné)
variable "database_url" {
  type = string
  default = getenv("ATLAS_DATABASE_URL")
  description = "URL de connexion à la base de données principale"
}

variable "dev_database_url" {
  type = string
  default = getenv("ATLAS_DEV_DATABASE_URL")
  description = "URL de connexion à la base de données de développement/test"
}

# Environnement de développement
env "development" {
  # URLs récupérées depuis les variables d'environnement
  url = var.database_url
  dev-url = var.dev_database_url
  
  # Configuration des migrations
  migration {
    dir = "file://database/migrations/postgresql"
  }
}

# Environnement Docker/Production
env "docker" {
  # Pour Docker, on peut utiliser des URLs spécifiques ou les mêmes variables
  url = getenv("ATLAS_DATABASE_URL_DOCKER") != "" ? getenv("ATLAS_DATABASE_URL_DOCKER") : var.database_url
  dev-url = getenv("ATLAS_DEV_DATABASE_URL_DOCKER") != "" ? getenv("ATLAS_DEV_DATABASE_URL_DOCKER") : "docker://postgres/15/dev?search_path=public"
  
  migration {
    dir = "file://database/migrations/postgresql"
  }
}

# Environnement de staging (optionnel)
env "staging" {
  url = getenv("ATLAS_DATABASE_URL_STAGING") != "" ? getenv("ATLAS_DATABASE_URL_STAGING") : var.database_url
  dev-url = getenv("ATLAS_DEV_DATABASE_URL_STAGING") != "" ? getenv("ATLAS_DEV_DATABASE_URL_STAGING") : var.dev_database_url
  
  migration {
    dir = "file://database/migrations/postgresql"
  }
}