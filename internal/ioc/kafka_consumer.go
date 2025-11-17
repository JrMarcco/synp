package ioc

import (
	pkgconsumer "github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/JrMarcco/synp/internal/ws/gateway"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var KafkaConsumerFxOpt = fx.Module("kafka_consumer", fx.Provide(
	initKafkaConsumerFactory,
	initKafkaConsumers,
))

func initKafkaConsumerFactory(readerFactory pkgconsumer.KafkaReaderFactory) pkgconsumer.ConsumerFactory {
	return pkgconsumer.NewKafkaConsumerFactory(readerFactory)
}

type consumersFxParams struct {
	fx.In

	ConsumerFactory pkgconsumer.ConsumerFactory
	Logger          *zap.Logger
}

func initKafkaConsumers(params consumersFxParams) map[string]*gateway.Consumer {
	consumers := make(map[string]*gateway.Consumer)

	consumers[gateway.EventPushMessage] = pushMessageConsumer(params.ConsumerFactory, params.Logger)

	return consumers
}

type consumerConfig struct {
	Topic      string `mapstructure:"topic"`
	GroupID    string `mapstructure:"group_id"`
	Partitions int32  `mapstructure:"partitions"`
}

func pushMessageConsumer(consumerFactory pkgconsumer.ConsumerFactory, logger *zap.Logger) *gateway.Consumer {
	cfg := consumerConfig{}
	if err := viper.UnmarshalKey("synp.gateway.consumer.event_message_downstream", &cfg); err != nil {
		panic(err)
	}

	return gateway.NewConsumer(
		consumerFactory,
		cfg.Topic,
		cfg.GroupID,
		cfg.Partitions,
		logger,
	)
}
