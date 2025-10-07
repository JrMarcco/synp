package ioc

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var RedisFxOpt = fx.Module("redis", fx.Provide(InitRedis))

type redisFxParams struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func InitRedis(params redisFxParams) redis.Cmdable {
	type config struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("redis", &cfg); err != nil {
		panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 测试 redis 连接。
			if err := rdb.WithContext(ctx).Ping().Err(); err != nil {
				params.Logger.Error("[synp-ioc] failed to ping redis", zap.Error(err))
				return fmt.Errorf("failed to ping redis: %w", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 关闭 redis 连接。
			if err := rdb.Close(); err != nil {
				params.Logger.Error("[synp-ioc] failed to close redis", zap.Error(err))
				return fmt.Errorf("failed to close redis: %w", err)
			}

			params.Logger.Info("[synp-ioc] redis closed")
			return nil
		},
	})

	return rdb
}
