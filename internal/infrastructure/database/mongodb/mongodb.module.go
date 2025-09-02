package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
)

func NewMongoClient(config *MongoConfig) (*Client, error) {
	return NewClient(config)
}

var Module = fx.Options(
	fx.Provide(NewMongoClient),
	fx.Provide(NewCollectionManager),
	fx.Invoke(RegisterLifecycle),
)

func RegisterLifecycle(lc fx.Lifecycle, client *Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := client.Ping(timeoutCtx); err != nil {
				fmt.Printf("[MONGODB] ⚠️  MongoDB non disponible - continuera sans MongoDB: %v\n", err)
				return nil // Ne bloque pas le démarrage
			}

			if err := client.HealthCheck(timeoutCtx); err != nil {
				fmt.Printf("[MONGODB] ⚠️  Health check MongoDB échoué - continuera sans MongoDB: %v\n", err)
				return nil // Ne bloque pas le démarrage
			}

			fmt.Printf("[MONGODB] ✅ MongoDB connecté et opérationnel\n")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return client.Close(ctx)
		},
	})
}