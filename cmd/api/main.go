package main

import (
	"context"
	"log"

	"soins-suite-core/internal/app"

	"go.uber.org/fx"
)

func main() {

	fx.New(
		app.AppModule,
		fx.Invoke(func(lifecycle fx.Lifecycle) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Println("Soins Suite API starting...")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Println("Soins Suite API stopping...")
					return nil
				},
			})
		}),
	).Run()
}
