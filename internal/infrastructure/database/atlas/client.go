package atlas

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Client wrapper Go pour Atlas CLI avec support rollback
type Client struct {
	workingDir  string
	configPath  string
	timeout     time.Duration
	environment string
}

// NewClient crée une nouvelle instance du client Atlas
func NewClient(workingDir, configPath, environment string) *Client {
	return &Client{
		workingDir:  workingDir,
		configPath:  configPath,
		timeout:     30 * time.Second,
		environment: environment,
	}
}

// SetTimeout configure le timeout pour les commandes Atlas
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// IsInstalled vérifie si Atlas CLI est installé et accessible
func (c *Client) IsInstalled() bool {
	cmd := exec.Command("atlas", "version")
	return cmd.Run() == nil
}

// GetVersion retourne la version d'Atlas CLI installée
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	// Utiliser commande simple sans config pour version
	cmd := exec.CommandContext(ctx, "atlas", "version")
	cmd.Dir = c.workingDir
	cmd.Env = os.Environ()
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("impossible de récupérer la version Atlas: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	if strings.Contains(version, "atlas version") {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			return parts[2], nil
		}
	}
	
	return version, nil
}

// Ping vérifie la connectivité à la base de données via Atlas
func (c *Client) Ping(ctx context.Context) error {
	cmd := c.buildCommand(ctx, "schema", "inspect", "--env", c.environment)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("échec de connexion via Atlas: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetStatus retourne le statut actuel des migrations
func (c *Client) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	cmd := c.buildCommand(ctx, "migrate", "status", "--env", c.environment)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer le statut des migrations: %w", err)
	}
	return parseMigrationStatus(string(output))
}

// ValidateConfig valide la configuration Atlas
func (c *Client) ValidateConfig(ctx context.Context) error {
	cmd := c.buildCommand(ctx, "migrate", "validate", "--env", c.environment)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("configuration Atlas invalide: %w (output: %s)", err, string(output))
	}
	return nil
}

// Close libère les ressources du client (pas de ressources à libérer pour le CLI)
func (c *Client) Close() error {
	return nil
}

// buildCommand construit une commande Atlas avec les paramètres de base
func (c *Client) buildCommand(ctx context.Context, args ...string) *exec.Cmd {
	baseArgs := []string{}
	
	if c.configPath != "" {
		baseArgs = append(baseArgs, "--config", fmt.Sprintf("file://%s", c.configPath))
	}
	
	baseArgs = append(baseArgs, args...)
	
	cmd := exec.CommandContext(ctx, "atlas", baseArgs...)
	cmd.Dir = c.workingDir
	
	// Construire environnement avec variables Atlas
	env := os.Environ()
	// Les variables seront ajoutées par le SchemaManager via buildEnv()
	cmd.Env = env
	
	return cmd
}

// MigrationStatus représente le statut d'une migration
type MigrationStatus struct {
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Applied     bool      `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// parseMigrationStatus parse la sortie du statut des migrations Atlas
func parseMigrationStatus(output string) ([]MigrationStatus, error) {
	var statuses []MigrationStatus
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Migration") || strings.HasPrefix(line, "---") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		
		status := MigrationStatus{
			Version:     parts[0],
			Description: strings.Join(parts[2:], " "),
			Type:        "baseline",
		}
		
		if parts[1] == "APPLIED" {
			status.Applied = true
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}