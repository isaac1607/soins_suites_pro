package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool *pgxpool.Pool
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

func NewClient(config *DatabaseConfig) (*Client, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	// Configuration du pool optimisée
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 30 * time.Second

	// Configuration des connexions
	poolConfig.ConnConfig.ConnectTimeout = 30 * time.Second
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = "30s"
	poolConfig.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "60s"

	// Optimisations PostgreSQL (paramètres session uniquement)
	poolConfig.ConnConfig.RuntimeParams["log_statement"] = "none"
	poolConfig.ConnConfig.RuntimeParams["log_min_duration_statement"] = "1000"

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	client := &Client{
		pool: pool,
	}

	// Test de connexion initial
	if err := client.Ping(context.Background()); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for ping: %w", err)
	}
	defer conn.Release()

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

func (c *Client) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

func (c *Client) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.pool.QueryRow(ctx, sql, args...)
}

func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.pool.Exec(ctx, sql, args...)
	return err
}

func (c *Client) Stats() *pgxpool.Stat {
	return c.pool.Stat()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	stats := c.Stats()
	
	if stats.TotalConns() == 0 {
		return fmt.Errorf("no database connections available")
	}

	if stats.IdleConns() == 0 && stats.AcquiredConns() >= stats.MaxConns() {
		return fmt.Errorf("database connection pool exhausted")
	}

	return c.Ping(ctx)
}

func (c *Client) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return c.pool.Begin(ctx)
}