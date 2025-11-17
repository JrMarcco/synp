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

		// 初始化 zap.Logger。
		ioc.LoggerFxOpt,

		// 初始化 etcd.Client。
		// ioc.EtcdFxOpt,

		// 初始化 redis.Cmdable。
		ioc.RedisFxOpt,

		// 初始化 kafka。
		ioc.KafkaFxOpt,
		ioc.KafkaConsumerFxOpt,
		ioc.KafkaProducerFxOpt,

		// 初始化 token validator。
		ioc.ValidatorFxOpt,

		// 初始化 codec。
		ioc.CodecFxOpt,

		// 初始化 upgrader。
		ioc.UpgraderFxOpt,

		// 初始化 message push func。
		ioc.MessagePushFxOpt,

		// 初始化 retransmit manager。
		ioc.RetransmitManagerFxOpt,

		// 初始化 message handler。
		ioc.MessageHandlerFxOpt,

		// 初始化 conn handler 和 conn manager。
		ioc.ConnFxOpt,

		// 初始化 app。
		ioc.AppFxOpt,
	).Run()
}

func initViper() {
	configFile := pflag.String("config", "etc/config.yaml", "path to config file")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}
