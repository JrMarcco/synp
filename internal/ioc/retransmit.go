package ioc

import (
	"context"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var RetransmitManagerFxOpt = fx.Module("retransmit_manager", fx.Provide(InitRetransmitManager))

type retransmitManagerFxParams struct {
	fx.In

	PushFunc message.PushFunc
	Logger   *zap.Logger

	Lifecycle fx.Lifecycle
}

func InitRetransmitManager(params retransmitManagerFxParams) *retransmit.Manager {
	type config struct {
		Interval int `mapstructure:"interval"`
		MaxRetry int `mapstructure:"max_retry"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("retransmit", &cfg); err != nil {
		panic(err)
	}

	manager := retransmit.NewManager(
		time.Duration(cfg.Interval)*time.Millisecond,
		int32(cfg.MaxRetry),
		params.PushFunc,
		params.Logger,
	)

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			manager.Close()
			params.Logger.Info("[synp-ioc] retransmit manager closed")
			return nil
		},
	})

	return manager
}
