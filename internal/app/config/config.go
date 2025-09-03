package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"soins-suite-core/internal/infrastructure/database/atlas"
	"soins-suite-core/internal/infrastructure/database/mongodb"
	"soins-suite-core/internal/infrastructure/database/postgres"
	"soins-suite-core/internal/infrastructure/database/redis"

	"github.com/joho/godotenv"
)

// Uniquement variables d'environnement

// Config structure unifiée
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	MongoDB     MongoConfig
	Atlas       AtlasConfig
	System      SystemConfig
	Logging     LoggingConfig
	CORS        CORSConfig
}

// ServerConfig configuration serveur HTTP
type ServerConfig struct {
	Host         string        `env:"SERVER_HOST"`
	Port         int           `env:"SERVER_PORT"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT"`
}

// DatabaseConfig configuration PostgreSQL
type DatabaseConfig struct {
	Host           string        `env:"DB_HOST"`
	Port           int           `env:"DB_PORT"`
	Database       string        `env:"DB_NAME"`
	Username       string        `env:"DB_USERNAME"`
	Password       string        `env:"DB_PASSWORD"`
	MaxConnections int           `env:"DB_MAX_CONNECTIONS"`
	ConnectionTTL  time.Duration `env:"DB_CONNECTION_TTL"`
	QueryTimeout   time.Duration `env:"DB_QUERY_TIMEOUT"`
	SSLMode        string        `env:"DB_SSL_MODE"`
}

// RedisConfig configuration Redis
type RedisConfig struct {
	Host        string        `env:"REDIS_HOST"`
	Port        int           `env:"REDIS_PORT"`
	Password    string        `env:"REDIS_PASSWORD"`
	Database    int           `env:"REDIS_DATABASE"`
	MaxRetries  int           `env:"REDIS_MAX_RETRIES"`
	PoolSize    int           `env:"REDIS_POOL_SIZE"`
	PoolTimeout time.Duration `env:"REDIS_POOL_TIMEOUT"`
}

// MongoConfig configuration MongoDB
type MongoConfig struct {
	URI            string        `env:"MONGODB_URI"`
	Database       string        `env:"MONGODB_DATABASE"`
	ConnectTimeout time.Duration `env:"MONGODB_CONNECT_TIMEOUT"`
	MaxPoolSize    int           `env:"MONGODB_MAX_POOL_SIZE"`
}

// AtlasConfig configuration Atlas migrations
type AtlasConfig struct {
	Enabled          bool          `env:"ATLAS_ENABLED"`
	WorkingDir       string        `env:"ATLAS_WORKING_DIR"`
	ConfigPath       string        `env:"ATLAS_CONFIG_PATH"`
	Environment      string        `env:"ATLAS_ENVIRONMENT"`
	SchemasPath      string        `env:"ATLAS_SCHEMAS_PATH"`
	MigrationsPath   string        `env:"ATLAS_MIGRATIONS_PATH"`
	DatabaseURL      string        `env:"DATABASE_URL"`
	DevDatabaseURL   string        `env:"ATLAS_DEV_DATABASE_URL"`
	AutoMigrate      bool          `env:"ATLAS_AUTO_MIGRATE"`
	DefaultTimeout   time.Duration `env:"ATLAS_DEFAULT_TIMEOUT"`
	MigrationTimeout time.Duration `env:"ATLAS_MIGRATION_TIMEOUT"`
}

// SystemConfig configuration système
type SystemConfig struct {
	MaxActivationAttempts int  `env:"SYSTEM_MAX_ACTIVATION_ATTEMPTS"`
	ActivationTimeout     int  `env:"SYSTEM_ACTIVATION_TIMEOUT"`
	CacheTTLActive        int  `env:"SYSTEM_CACHE_TTL_ACTIVE"`
	CacheTTLExpired       int  `env:"SYSTEM_CACHE_TTL_EXPIRED"`
	LicenseKeyLength      int  `env:"SYSTEM_LICENSE_KEY_LENGTH"`
	EnableMiddleware      bool `env:"SYSTEM_ENABLE_MIDDLEWARE"`

	// Variables sécurisées (obligatoires en production)
	LicenseServerURL   string `env:"LICENSE_SERVER_URL"`
	AppInstanceID      string `env:"APP_INSTANCE_ID"`
	EstablishmentToken string `env:"ESTABLISHMENT_TOKEN"`
	AdminTIRPassword   string `env:"ADMIN_TIR_DEFAULT_PASSWORD"`
}

// LoggingConfig configuration logging
type LoggingConfig struct {
	Level string `env:"LOG_LEVEL"`
}

// CORSConfig configuration CORS
type CORSConfig struct {
	AllowedOrigins   []string `env:"CORS_ALLOWED_ORIGINS"`
	AllowedMethods   []string `env:"CORS_ALLOWED_METHODS"`
	AllowedHeaders   []string `env:"CORS_ALLOWED_HEADERS"`
	AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS"`
	MaxAge           int      `env:"CORS_MAX_AGE"`
}

// NewConfig charge la configuration depuis les variables d'environnement uniquement
func NewConfig() (*Config, error) {
	// Charger le fichier .env (optionnel)
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("[CONFIG] Warning: Fichier .env non trouvé: %v\n", err)
	}

	config := &Config{}

	// Déterminer environnement
	config.Environment = getEnv("APP_ENV", "development")

	// Charger configuration serveur
	config.Server = ServerConfig{
		Host:         getEnv("SERVER_HOST", "localhost"),
		Port:         getEnvInt("SERVER_PORT", 4000),
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30) * time.Second,
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30) * time.Second,
	}

	// Charger configuration database
	config.Database = DatabaseConfig{
		Host:           getEnv("DB_HOST", "localhost"),
		Port:           getEnvInt("DB_PORT", 5432),
		Database:       getEnv("DB_NAME", "soins_suite"),
		Username:       getEnv("DB_USERNAME", "postgres"),
		Password:       getEnv("DB_PASSWORD", ""),
		MaxConnections: getEnvInt("DB_MAX_CONNECTIONS", 100),
		ConnectionTTL:  getEnvDuration("DB_CONNECTION_TTL", 300) * time.Second,
		QueryTimeout:   getEnvDuration("DB_QUERY_TIMEOUT", 30) * time.Second,
		SSLMode:        getEnv("DB_SSL_MODE", "disable"),
	}

	// Charger configuration Redis
	config.Redis = RedisConfig{
		Host:        getEnv("REDIS_HOST", "localhost"),
		Port:        getEnvInt("REDIS_PORT", 6379),
		Password:    getEnv("REDIS_PASSWORD", ""),
		Database:    getEnvInt("REDIS_DATABASE", 0),
		MaxRetries:  getEnvInt("REDIS_MAX_RETRIES", 3),
		PoolSize:    getEnvInt("REDIS_POOL_SIZE", 10),
		PoolTimeout: getEnvDuration("REDIS_POOL_TIMEOUT", 30) * time.Second,
	}

	// Charger configuration MongoDB
	defaultMongoURI := ""
	if config.Environment == "development" {
		defaultMongoURI = "mongodb://localhost:27011"
	}

	mongoURI := getEnv("MONGODB_URI", defaultMongoURI)
	fmt.Printf("[CONFIG] Debug MongoDB: URI from env='%s', default='%s', final='%s'\n",
		os.Getenv("MONGODB_URI"), defaultMongoURI, mongoURI)

	config.MongoDB = MongoConfig{
		URI:            mongoURI,
		Database:       getEnv("MONGODB_DATABASE", "soins_suite_dynamic"),
		ConnectTimeout: getEnvDuration("MONGODB_CONNECT_TIMEOUT", 10) * time.Second,
		MaxPoolSize:    getEnvInt("MONGODB_MAX_POOL_SIZE", 100),
	}

	// Charger configuration Atlas
	config.Atlas = AtlasConfig{
		Enabled:          getEnvBool("ATLAS_ENABLED", true),
		WorkingDir:       getEnv("ATLAS_WORKING_DIR", "."),
		ConfigPath:       getEnv("ATLAS_CONFIG_PATH", "database/atlas.hcl"),
		Environment:      getEnv("ATLAS_ENVIRONMENT", "development"),
		SchemasPath:      getEnv("ATLAS_SCHEMAS_PATH", "database/schemas"),
		MigrationsPath:   getEnv("ATLAS_MIGRATIONS_PATH", "database/migrations/postgresql"),
		DatabaseURL:      getEnv("ATLAS_DATABASE_URL", ""),
		DevDatabaseURL:   getEnv("ATLAS_DEV_DATABASE_URL", ""),
		AutoMigrate:      getEnvBool("ATLAS_AUTO_MIGRATE", false),
		DefaultTimeout:   getEnvDuration("ATLAS_DEFAULT_TIMEOUT", 30) * time.Second,
		MigrationTimeout: getEnvDuration("ATLAS_MIGRATION_TIMEOUT", 60) * time.Second,
	}

	// Auto-générer URLs database si manquantes
	if config.Atlas.DatabaseURL == "" {
		config.Atlas.DatabaseURL = generateDatabaseURL(config.Database)
	}
	if config.Atlas.DevDatabaseURL == "" {
		config.Atlas.DevDatabaseURL = generateDevDatabaseURL(config.Database)
	}

	// Charger configuration système
	config.System = SystemConfig{
		MaxActivationAttempts: getEnvInt("SYSTEM_MAX_ACTIVATION_ATTEMPTS", 3),
		ActivationTimeout:     getEnvInt("SYSTEM_ACTIVATION_TIMEOUT", 30),
		CacheTTLActive:        getEnvInt("SYSTEM_CACHE_TTL_ACTIVE", 1800),
		CacheTTLExpired:       getEnvInt("SYSTEM_CACHE_TTL_EXPIRED", 300),
		LicenseKeyLength:      getEnvInt("SYSTEM_LICENSE_KEY_LENGTH", 20),
		EnableMiddleware:      getEnvBool("SYSTEM_ENABLE_MIDDLEWARE", true),
		LicenseServerURL:      getEnv("LICENSE_SERVER_URL", ""),
		AppInstanceID:         getEnv("APP_INSTANCE_ID", ""),
		EstablishmentToken:    getEnv("ESTABLISHMENT_TOKEN", ""),
		AdminTIRPassword:      getEnv("ADMIN_TIR_DEFAULT_PASSWORD", ""),
	}

	// Charger configuration logging
	config.Logging = LoggingConfig{
		Level: getEnv("LOG_LEVEL", "debug"),
	}

	// Charger configuration CORS
	config.CORS = CORSConfig{
		AllowedOrigins:   getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		AllowedMethods:   getEnvStringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		AllowedHeaders:   getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization"}),
		AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getEnvInt("CORS_MAX_AGE", 3600),
	}

	// Validation configuration critique
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("validation configuration échouée: %w", err)
	}

	fmt.Printf("[CONFIG] ✅ Configuration chargée pour environnement: %s\n", config.Environment)
	return config, nil
}

// Getters pour compatibilité avec l'ancien code
func (c *Config) GetDatabase() DatabaseConfig { return c.Database }
func (c *Config) GetRedis() RedisConfig       { return c.Redis }
func (c *Config) GetMongoDB() MongoConfig     { return c.MongoDB }
func (c *Config) GetAtlas() AtlasConfig       { return c.Atlas }
func (c *Config) GetSystem() SystemConfig     { return c.System }
func (c *Config) GetServer() ServerConfig     { return c.Server }
func (c *Config) GetLogging() LoggingConfig   { return c.Logging }
func (c *Config) GetCORS() CORSConfig         { return c.CORS }

// Providers pour database ConfigProvider (compatibilité)
func NewDatabaseConfigProvider(config *Config) *DatabaseConfigProvider {
	return &DatabaseConfigProvider{
		Database: DatabaseConfig(config.Database),
		Redis:    RedisConfig(config.Redis),
		MongoDB:  MongoConfig(config.MongoDB),
	}
}

type DatabaseConfigProvider struct {
	Database DatabaseConfig
	Redis    RedisConfig
	MongoDB  MongoConfig
}

// Convertisseur vers configurations infrastructure
func NewAtlasConfigFromApp(config *Config) *atlas.AtlasConfig {
	atlasConfig := atlas.NewAtlasConfig()
	atlasConfig.Enabled = config.Atlas.Enabled
	atlasConfig.WorkingDir = config.Atlas.WorkingDir
	atlasConfig.ConfigPath = config.Atlas.ConfigPath
	atlasConfig.Environment = config.Atlas.Environment
	atlasConfig.SchemasPath = config.Atlas.SchemasPath
	atlasConfig.MigrationsPath = config.Atlas.MigrationsPath
	atlasConfig.DatabaseURL = config.Atlas.DatabaseURL
	atlasConfig.DevDatabaseURL = config.Atlas.DevDatabaseURL
	atlasConfig.AutoMigrate = config.Atlas.AutoMigrate
	atlasConfig.DefaultTimeout = config.Atlas.DefaultTimeout
	atlasConfig.MigrationTimeout = config.Atlas.MigrationTimeout
	return atlasConfig
}

func NewPostgresConfig(config *DatabaseConfigProvider) *postgres.DatabaseConfig {
	return &postgres.DatabaseConfig{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		Database: config.Database.Database,
		Username: config.Database.Username,
		Password: config.Database.Password,
		SSLMode:  config.Database.SSLMode,
	}
}

func NewRedisConfig(config *DatabaseConfigProvider) *redis.RedisConfig {
	return &redis.RedisConfig{
		Host:     config.Redis.Host,
		Port:     config.Redis.Port,
		Password: config.Redis.Password,
		Database: config.Redis.Database,
	}
}

func NewMongoConfig(config *DatabaseConfigProvider) *mongodb.MongoConfig {
	return &mongodb.MongoConfig{
		URI:      config.MongoDB.URI,
		Database: config.MongoDB.Database,
	}
}

// Helpers pour parsing variables d'environnement
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultSeconds int) time.Duration {
	return time.Duration(getEnvInt(key, defaultSeconds))
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// generateDatabaseURL génère l'URL PostgreSQL principale
func generateDatabaseURL(dbConfig DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database, dbConfig.SSLMode)
}

// generateDevDatabaseURL génère l'URL de dev Atlas
func generateDevDatabaseURL(dbConfig DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s_atlas?sslmode=%s&search_path=public",
		dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database, dbConfig.SSLMode)
}

// validateConfig valide la configuration selon l'environnement
func validateConfig(config *Config) error {
	env := config.Environment

	// Validation environnements supportés
	if env != "development" && env != "docker" {
		return fmt.Errorf("environnement non supporté: %s (utilisez 'development' ou 'docker')", env)
	}

	missingVars := []string{}

	// Variables critiques en mode docker (production/staging)
	if env == "docker" {
		if config.Database.Password == "" {
			missingVars = append(missingVars, "DB_PASSWORD")
		}
		if config.System.LicenseServerURL == "" {
			missingVars = append(missingVars, "LICENSE_SERVER_URL")
		}
		if config.System.AppInstanceID == "" {
			missingVars = append(missingVars, "APP_INSTANCE_ID")
		}
		if config.System.EstablishmentToken == "" {
			missingVars = append(missingVars, "ESTABLISHMENT_TOKEN")
		}

		// Warning pour variables recommandées en docker
		if config.Redis.Password == "" {
			fmt.Printf("[CONFIG] ⚠️ REDIS_PASSWORD non défini pour environnement docker\n")
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("variables critiques manquantes pour environnement docker: %v", missingVars)
	}

	return nil
}
