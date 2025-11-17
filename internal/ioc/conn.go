package ioc

import (
	"time"

	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message/downstream"
	"github.com/JrMarcco/synp/internal/pkg/message/upstream"
	"github.com/JrMarcco/synp/internal/ws/conn"
	"github.com/JrMarcco/synp/internal/ws/conn/lifecycle"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ConnFxOpt = fx.Module(
	"conn",
	fx.Provide(
		fx.Annotate(
			initConnHandler,
			fx.As(new(synp.Handler)),
		),
		fx.Annotate(
			initConnManager,
			fx.As(new(synp.ConnManager)),
		),
	),
)

type connHandlerFxParams struct {
	fx.In

	Rdb redis.Cmdable

	Codec codec.Codec

	UMsgHandlers []upstream.UMsgHandler `group:"upstream_message_handler"`
	DMsgHandler  downstream.DMsgHandler

	Logger *zap.Logger
}

func initConnHandler(params connHandlerFxParams) *lifecycle.Handler {
	type config struct {
		CacheRequestTimeout time.Duration `mapstructure:"cache_request_timeout"`
		CacheExpiration     time.Duration `mapstructure:"cache_expiration"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("synp.conn.handler", &cfg); err != nil {
		panic(err)
	}

	return lifecycle.NewHandler(
		params.Rdb,
		cfg.CacheRequestTimeout,
		cfg.CacheExpiration,
		params.Codec,
		params.UMsgHandlers,
		params.DMsgHandler,
		params.Logger,
	)
}

type connManagerFxParams struct {
	fx.In

	Logger *zap.Logger
}

func initConnManager(params connManagerFxParams) *conn.ConnManager {
	type config = struct {
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`

		InitRetryInterval time.Duration `mapstructure:"init_retry_interval"`
		MaxRetryInterval  time.Duration `mapstructure:"max_retry_interval"`
		MaxRetryCount     int32         `mapstructure:"max_retry_count"`

		SendBufferSize    int `mapstructure:"send_buffer_size"`
		ReceiveBufferSize int `mapstructure:"receive_buffer_size"`

		CloseTimeout time.Duration `mapstructure:"close_timeout"`
		RateLimit    int           `mapstructure:"rate_limit"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("synp.conn.manager", &cfg); err != nil {
		panic(err)
	}

	return conn.NewConnManager(params.Logger, conn.ConnManagerWithConfig(&conn.ConnConfig{
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		InitRetryInterval: cfg.InitRetryInterval,
		MaxRetryInterval:  cfg.MaxRetryInterval,
		MaxRetryCount:     cfg.MaxRetryCount,
		SendBufferSize:    cfg.SendBufferSize,
		ReceiveBufferSize: cfg.ReceiveBufferSize,
		CloseTimeout:      cfg.CloseTimeout,
		RateLimit:         cfg.RateLimit,
	}))
}
