package providers

import (
	pkgconsumer "github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/JrMarcco/synp/internal/ws/gateway"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func newKafkaConsumers(consumerFactory pkgconsumer.ConsumerFactory, logger *zap.Logger) map[string]*gateway.Consumer {
	consumers := make(map[string]*gateway.Consumer)

	consumers[gateway.EventPushMessage] = pushMessageConsumer(consumerFactory, logger)

	return consumers
}

func pushMessageConsumer(consumerFactory pkgconsumer.ConsumerFactory, logger *zap.Logger) *gateway.Consumer {
	type consumerConfig struct {
		Topic      string `mapstructure:"topic"`
		GroupID    string `mapstructure:"group_id"`
		Partitions int32  `mapstructure:"partitions"`
	}

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
