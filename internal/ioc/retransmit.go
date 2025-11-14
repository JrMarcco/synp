package ioc

import (
	"context"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var RetransmitManagerFxOpt = fx.Module("retransmit_manager", fx.Provide(InitRetransmitManager))

type retransmitManagerFxParams struct {
	fx.In

	PushFunc  message.PushFunc
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
	)

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			manager.Close()
			return nil
		},
	})

	return manager
}
