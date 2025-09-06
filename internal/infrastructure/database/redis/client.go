package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb          *redis.Client
	keyGenerator *RedisKeyGenerator
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

func NewClient(config *RedisConfig, keyGenerator *RedisKeyGenerator) (*Client, error) {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.Database,
		MaxRetries:   3,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MinIdleConns: 2,
	}

	rdb := redis.NewClient(opts)

	client := &Client{
		rdb:          rdb,
		keyGenerator: keyGenerator,
	}

	// Test connexion
	if err := client.Ping(context.Background()); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.rdb == nil {
		return fmt.Errorf("Redis client is nil")
	}

	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (c *Client) Close() {
	if c.rdb != nil {
		c.rdb.Close()
	}
}

func (c *Client) Client() *redis.Client {
	return c.rdb
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	result := c.rdb.Get(ctx, key)
	if result.Err() == redis.Nil {
		return "", redis.Nil // Conserver l'erreur redis.Nil native
	}
	return result.Val(), result.Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result := c.rdb.Exists(ctx, key)
	return result.Val() > 0, result.Err()
}

func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	result := c.rdb.HGet(ctx, key, field)
	if result.Err() == redis.Nil {
		return "", redis.Nil // Conserver l'erreur redis.Nil native
	}
	return result.Val(), result.Err()
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.rdb.HDel(ctx, key, fields...).Err()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.Ping(ctx); err != nil {
		return err
	}

	// Vérifier les statistiques de pool
	stats := c.rdb.PoolStats()
	if stats.TotalConns == 0 {
		return fmt.Errorf("no Redis connections available")
	}

	return nil
}

func (c *Client) Stats() *redis.PoolStats {
	return c.rdb.PoolStats()
}

// ============================================
// MÉTHODES AVEC GÉNÉRATION AUTOMATIQUE DE CLÉS
// ============================================

// SetWithPattern sauvegarde une valeur avec un pattern standardisé selon les conventions
func (c *Client) SetWithPattern(ctx context.Context, patternName, establishmentCode string, value interface{}, identifier ...string) error {
	key, err := c.keyGenerator.GenerateKey(patternName, establishmentCode, identifier...)
	if err != nil {
		return fmt.Errorf("erreur génération clé: %w", err)
	}

	// Récupérer le TTL du pattern avec logique spéciale pour cache_middleware
	ttl, err := c.keyGenerator.GetTTL(patternName)
	if err != nil {
		return fmt.Errorf("erreur récupération TTL: %w", err)
	}

	// Logique spéciale pour cache_middleware selon l'identifier
	if patternName == "cache_middleware" && len(identifier) > 0 {
		switch identifier[0] {
		case "license":
			ttl = 86400 // 24h pour les licences (données semi-statiques)
		case "establishment":
			ttl = 0 // Infini pour établissement (données immuables)
		}
	}

	var duration time.Duration
	if ttl == 0 {
		duration = 0 // Pas d'expiration
	} else {
		duration = time.Duration(ttl) * time.Second
	}

	return c.rdb.Set(ctx, key, value, duration).Err()
}

// GetWithPattern récupère une valeur avec un pattern standardisé selon les conventions
func (c *Client) GetWithPattern(ctx context.Context, patternName, establishmentCode string, identifier ...string) (string, error) {
	key, err := c.keyGenerator.GenerateKey(patternName, establishmentCode, identifier...)
	if err != nil {
		return "", fmt.Errorf("erreur génération clé: %w", err)
	}

	result := c.rdb.Get(ctx, key)
	if result.Err() == redis.Nil {
		return "", redis.Nil
	}
	return result.Val(), result.Err()
}

// DelWithPattern supprime une valeur avec un pattern standardisé selon les conventions
func (c *Client) DelWithPattern(ctx context.Context, patternName, establishmentCode string, identifier ...string) error {
	key, err := c.keyGenerator.GenerateKey(patternName, establishmentCode, identifier...)
	if err != nil {
		return fmt.Errorf("erreur génération clé: %w", err)
	}

	return c.rdb.Del(ctx, key).Err()
}

// ExistsWithPattern vérifie l'existence avec un pattern standardisé selon les conventions
func (c *Client) ExistsWithPattern(ctx context.Context, patternName, establishmentCode string, identifier ...string) (bool, error) {
	key, err := c.keyGenerator.GenerateKey(patternName, establishmentCode, identifier...)
	if err != nil {
		return false, fmt.Errorf("erreur génération clé: %w", err)
	}

	result := c.rdb.Exists(ctx, key)
	return result.Val() > 0, result.Err()
}

// GenerateKey expose la génération de clé pour usage direct selon les conventions
func (c *Client) GenerateKey(patternName, establishmentCode string, identifier ...string) (string, error) {
	return c.keyGenerator.GenerateKey(patternName, establishmentCode, identifier...)
}

// ValidateKey valide une clé selon les standards
func (c *Client) ValidateKey(key string) error {
	return c.keyGenerator.ValidateKey(key)
}

// InvalidateModuleCache invalide toutes les clés d'un domaine/context pour un établissement
func (c *Client) InvalidateModuleCache(ctx context.Context, establishmentCode, domain, context string) error {
	pattern := c.keyGenerator.GenerateWildcardPattern(establishmentCode, domain, context)

	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("erreur récupération clés pattern: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	return c.rdb.Del(ctx, keys...).Err()
}
