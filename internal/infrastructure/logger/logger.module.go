package logger

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewMiddleware),
)

func NewMiddleware() *LoggerMiddleware {
	return &LoggerMiddleware{}
}