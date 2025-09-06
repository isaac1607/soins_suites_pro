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
	workingDir     string
	configPath     string
	timeout        time.Duration
	environment    string
	databaseURL    string
	devDatabaseURL string
}

// NewClient crée une nouvelle instance du client Atlas
func NewClient(workingDir, configPath, environment string) *Client {
	// Récupérer les URLs depuis les variables d'environnement
	databaseURL := os.Getenv("ATLAS_DATABASE_URL")
	if databaseURL == "" {
		// Valeur par défaut pour développement
		databaseURL = "postgres://postgres:idriss@localhost:5432/soins_suite?sslmode=disable"
	}

	devDatabaseURL := os.Getenv("ATLAS_DEV_DATABASE_URL")
	if devDatabaseURL == "" {
		// Valeur par défaut pour développement
		devDatabaseURL = "postgres://postgres:idriss@localhost:5432/soins_suite_atlas?sslmode=disable"
	}

	return &Client{
		workingDir:     workingDir,
		configPath:     configPath,
		timeout:        30 * time.Second,
		environment:    environment,
		databaseURL:    databaseURL,
		devDatabaseURL: devDatabaseURL,
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
	cmd := exec.CommandContext(ctx, "atlas", "version")
	cmd.Dir = c.workingDir

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
	// Test direct avec l'URL sans passer par le fichier config
	cmd := exec.CommandContext(ctx, "atlas", "schema", "inspect",
		"--url", c.databaseURL,
	)

	cmd.Dir = c.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("échec de connexion via Atlas: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetStatus retourne le statut actuel des migrations
func (c *Client) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	// Approche 1 : Essayer avec le fichier de config (qui a les URLs hardcodées maintenant)
	cmd := exec.CommandContext(ctx, "atlas", "migrate", "status",
		"--config", fmt.Sprintf("file://%s", c.configPath),
		"--env", c.environment,
	)
	cmd.Dir = c.workingDir

	// Debug
	fmt.Printf("[ATLAS DEBUG] Tentative avec config file\n")
	fmt.Printf("[ATLAS DEBUG] Commande: %s\n", strings.Join(cmd.Args, " "))
	fmt.Printf("[ATLAS DEBUG] WorkingDir: %s\n", cmd.Dir)

	output, err := cmd.CombinedOutput()

	// Si erreur avec le config file, essayer sans
	if err != nil {
		fmt.Printf("[ATLAS DEBUG] Erreur avec config: %v\n", err)
		fmt.Printf("[ATLAS DEBUG] Output: %s\n", string(output))

		// Approche 2 : Commande directe sans fichier config
		fmt.Printf("[ATLAS DEBUG] Tentative sans config file, URL directe\n")

		cmd = exec.CommandContext(ctx, "atlas", "migrate", "status",
			"--dir", "file://database/migrations/postgresql",
			"--url", c.databaseURL,
		)
		cmd.Dir = c.workingDir

		fmt.Printf("[ATLAS DEBUG] Commande directe: %s\n", strings.Join(cmd.Args, " "))

		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("[ATLAS DEBUG] Erreur commande directe: %v\n", err)
			fmt.Printf("[ATLAS DEBUG] Output: %s\n", string(output))

			// Dernière tentative : vérifier si le répertoire migrations existe
			migrationsPath := "database/migrations/postgresql"
			if _, statErr := os.Stat(migrationsPath); os.IsNotExist(statErr) {
				// Créer le répertoire s'il n'existe pas
				fmt.Printf("[ATLAS DEBUG] Création du répertoire migrations: %s\n", migrationsPath)
				os.MkdirAll(migrationsPath, 0755)

				// Créer un fichier atlas.sum vide si nécessaire
				sumFile := migrationsPath + "/atlas.sum"
				if _, err := os.Stat(sumFile); os.IsNotExist(err) {
					file, _ := os.Create(sumFile)
					file.Close()
				}

				// Réessayer
				output, err = cmd.CombinedOutput()
			}

			if err != nil {
				return nil, fmt.Errorf("impossible de récupérer le statut des migrations: %w (output: %s)", err, string(output))
			}
		}
	}

	return parseMigrationStatus(string(output))
}

// ValidateConfig valide la configuration Atlas
func (c *Client) ValidateConfig(ctx context.Context) error {
	// Valider en utilisant une commande simple
	cmd := exec.CommandContext(ctx, "atlas", "migrate", "validate",
		"--dir", "file://database/migrations/postgresql",
	)
	cmd.Dir = c.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Si le répertoire n'existe pas, le créer
		migrationsPath := "database/migrations/postgresql"
		if strings.Contains(string(output), "no such file") {
			fmt.Printf("[ATLAS] Création du répertoire migrations: %s\n", migrationsPath)
			os.MkdirAll(migrationsPath, 0755)
			return nil // Pas d'erreur si on vient de créer le répertoire
		}
		return fmt.Errorf("configuration Atlas invalide: %w (output: %s)", err, string(output))
	}
	return nil
}

// Close libère les ressources du client
func (c *Client) Close() error {
	return nil
}

// buildCommand construit une commande Atlas basique
func (c *Client) buildCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "atlas", args...)
	cmd.Dir = c.workingDir
	cmd.Env = os.Environ()
	return cmd
}

// MigrationStatus représente le statut d'une migration
type MigrationStatus struct {
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Type        string     `json:"type"`
	Applied     bool       `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// parseMigrationStatus parse la sortie du statut des migrations Atlas
func parseMigrationStatus(output string) ([]MigrationStatus, error) {
	var statuses []MigrationStatus

	// Si le répertoire est vide ou n'a pas de migrations
	if strings.Contains(output, "No migrations found") ||
		strings.Contains(output, "The migration directory is synced with the database") ||
		output == "" {
		return statuses, nil // Retourner une liste vide n'est pas une erreur
	}

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Migration") || strings.HasPrefix(line, "---") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := MigrationStatus{
			Version: parts[0],
			Type:    "migration",
		}

		// Parser le statut
		if len(parts) > 1 {
			if parts[1] == "APPLIED" || parts[1] == "OK" {
				status.Applied = true
			} else if parts[1] == "PENDING" {
				status.Applied = false
			}
		}

		// Parser la description si disponible
		if len(parts) > 2 {
			status.Description = strings.Join(parts[2:], " ")
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
