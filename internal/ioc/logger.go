package ioc

import (
	"context"
	"log/slog"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

var LoggerFxOpt = fx.Module("logger", fx.Provide(InitLogger))

type loggerFxParams struct {
	fx.In

	Lifecycle fx.Lifecycle
}

func InitLogger(params loggerFxParams) *zap.Logger {
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

	// 初始化 slog
	slog.SetDefault(slog.New(zapslog.NewHandler(logger.Core())))

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			_ = logger.Sync()
			return nil
		},
	})

	return logger
}
