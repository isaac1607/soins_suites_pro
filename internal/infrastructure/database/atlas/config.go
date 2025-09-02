package atlas

import (
	"path/filepath"
	"time"
)

// AtlasConfig configuration centralisée pour Atlas
type AtlasConfig struct {
	// Configuration générale
	WorkingDir  string `yaml:"working_dir"`
	ConfigPath  string `yaml:"config_path"`
	Environment string `yaml:"environment"`
	
	// Timeouts
	DefaultTimeout   time.Duration `yaml:"default_timeout"`
	MigrationTimeout time.Duration `yaml:"migration_timeout"`
	
	// Chemins
	SchemasPath    string `yaml:"schemas_path"`
	MigrationsPath string `yaml:"migrations_path"`
	
	// Base de données
	DatabaseURL    string `yaml:"database_url"`
	DevDatabaseURL string `yaml:"dev_database_url"`
	
	// Options
	AutoMigrate bool `yaml:"auto_migrate"`
	Enabled     bool `yaml:"enabled"`
}

// NewAtlasConfig crée une configuration Atlas avec valeurs par défaut
func NewAtlasConfig() *AtlasConfig {
	return &AtlasConfig{
		WorkingDir:       ".",
		ConfigPath:       "database/atlas.hcl",
		Environment:      "development",
		DefaultTimeout:   30 * time.Second,
		MigrationTimeout: 60 * time.Second,
		SchemasPath:      "database/schemas",
		MigrationsPath:   "database/migrations/postgresql",
		AutoMigrate:      false,
		Enabled:          true,
	}
}

// GetAbsoluteSchemasPath retourne le chemin absolu vers les schémas
func (c *AtlasConfig) GetAbsoluteSchemasPath() string {
	if filepath.IsAbs(c.SchemasPath) {
		return c.SchemasPath
	}
	return filepath.Join(c.WorkingDir, c.SchemasPath)
}

// GetAbsoluteMigrationsPath retourne le chemin absolu vers les migrations
func (c *AtlasConfig) GetAbsoluteMigrationsPath() string {
	if filepath.IsAbs(c.MigrationsPath) {
		return c.MigrationsPath
	}
	return filepath.Join(c.WorkingDir, c.MigrationsPath)
}

// GetAbsoluteConfigPath retourne le chemin absolu vers le fichier de config Atlas
func (c *AtlasConfig) GetAbsoluteConfigPath() string {
	if filepath.IsAbs(c.ConfigPath) {
		return c.ConfigPath
	}
	return filepath.Join(c.WorkingDir, c.ConfigPath)
}

// Validate valide la configuration Atlas
func (c *AtlasConfig) Validate() error {
	if c.DatabaseURL == "" {
		return ErrMissingDatabaseURL
	}
	if c.DevDatabaseURL == "" {
		return ErrMissingDevDatabaseURL
	}
	if c.DefaultTimeout <= 0 {
		return ErrInvalidTimeout
	}
	if c.MigrationTimeout <= 0 {
		return ErrInvalidMigrationTimeout
	}
	return nil
}