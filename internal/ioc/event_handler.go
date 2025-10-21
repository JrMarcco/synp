package ioc

import (
	"time"

	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message/downstream"
	"github.com/JrMarcco/synp/internal/pkg/message/upstream"
	"github.com/JrMarcco/synp/internal/ws/conn/event"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var EventHandlerFxOpt = fx.Module("event_handler", fx.Provide(InitEventHandler))

type eventHandlerFxParams struct {
	fx.In

	Rdb redis.Cmdable

	Codec codec.Codec

	UMsgHandlers []upstream.UMsgHandler
	DMsgHandler  downstream.DMsgHandler

	Logger *zap.Logger
}

func InitEventHandler(params eventHandlerFxParams) synp.ConnEventHandler {
	type config struct {
		CacheRequestTimeout int `mapstructure:"cache_request_timeout"`
		CacheExpiration     int `mapstructure:"cache_expiration"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("synp.handler.event", &cfg); err != nil {
		panic(err)
	}

	return event.NewEventHandler(
		params.Rdb,
		time.Duration(cfg.CacheRequestTimeout)*time.Millisecond,
		time.Duration(cfg.CacheExpiration)*time.Millisecond,
		params.Codec,
		params.UMsgHandlers,
		params.DMsgHandler,
		params.Logger,
	)

}
