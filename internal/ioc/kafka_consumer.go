package ioc

import (
	pkgconsumer "github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/JrMarcco/synp/internal/ws/gateway"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var KafkaConsumerFxOpt = fx.Module("kafka_consumer", fx.Provide(
	// kafka consumer factory。
	InitKafkaConsumerFactory,
	fx.Annotate(
		InitPushMessageConsumer,
		fx.ResultTags(`group:"gateway_consumer"`),
	),
))

func InitKafkaConsumerFactory(readerFactory pkgconsumer.KafkaReaderFactory) pkgconsumer.ConsumerFactory {
	return pkgconsumer.NewKafkaConsumerFactory(readerFactory)
}

type consumerConfig struct {
	Topic      string `mapstructure:"topic"`
	GroupId    string `mapstructure:"group_id"`
	Partitions int32  `mapstructure:"partitions"`
}

type consumerFxParams struct {
	fx.In

	ConsumerFactory pkgconsumer.ConsumerFactory
	Logger          *zap.Logger
}

func InitPushMessageConsumer(params consumerFxParams) *gateway.Consumer {
	cfg := consumerConfig{}
	if err := viper.UnmarshalKey("gateway.consumer.push_message", &cfg); err != nil {
		panic(err)
	}

	return gateway.NewConsumer(
		params.ConsumerFactory,
		cfg.Topic,
		cfg.GroupId,
		cfg.Partitions,
		params.Logger,
	)
}
