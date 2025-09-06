package redis

import (
	"context"
	"time"

	"go.uber.org/fx"
)

func NewRedisClient(config *RedisConfig, keyGenerator *RedisKeyGenerator) (*Client, error) {
	return NewClient(config, keyGenerator)
}

var Module = fx.Options(
	fx.Provide(NewRedisKeyGenerator),
	fx.Provide(NewRedisClient),
	fx.Invoke(RegisterLifecycle),
)

func RegisterLifecycle(lc fx.Lifecycle, client *Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := client.Ping(timeoutCtx); err != nil {
				return err
			}

			return client.HealthCheck(timeoutCtx)
		},
		OnStop: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	})
}
