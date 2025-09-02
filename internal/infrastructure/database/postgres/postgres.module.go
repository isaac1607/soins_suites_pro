package postgres

import (
	"context"
	"time"

	"go.uber.org/fx"
)

func NewPostgresClient(config *DatabaseConfig) (*Client, error) {
	return NewClient(config)
}

var Module = fx.Options(
	fx.Provide(NewPostgresClient),
	fx.Provide(NewTxManager),
	fx.Invoke(RegisterLifecycle),
)

func NewTxManager(client *Client) *TransactionManager {
	return NewTransactionManager(client)
}

func RegisterLifecycle(lc fx.Lifecycle, client *Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Test de connexion avec timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := client.Ping(timeoutCtx); err != nil {
				return err
			}

			// Health check complet
			return client.HealthCheck(timeoutCtx)
		},
		OnStop: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	})
}