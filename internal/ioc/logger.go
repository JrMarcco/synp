package ioc

import (
	"context"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var LoggerFxOpt = fx.Module("logger", fx.Provide(InitLogger))

type loggerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
}

func InitLogger(params loggerParams) *zap.Logger {
	type config struct {
		Env string `mapstructure:"env"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("profile", &cfg); err != nil {
		panic(err)
	}

	var err error
	var logger *zap.Logger

	switch cfg.Env {
	case "prod":
		logger, err = zap.NewProduction()
	default:
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic(err)
	}

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			_ = logger.Sync()
			return nil
		},
	})

	return logger
}
