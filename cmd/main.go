package main

import (
	"github.com/JrMarcco/synp/internal/ioc"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	initViper()

	fx.New(
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),

		// 初始化 zap.Logger
		ioc.LoggerFxOpt,

		// 初始化 etcd.Client
		ioc.EtcdFxOpt,

		// 初始化 redis.Cmdable
		ioc.RedisFxOpt,

		// 初始化 token validator
		ioc.ValidatorFxOpt,
	).Run()
}

func initViper() {
	configFile := pflag.String("config", "config.yaml", "path to config file")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}
