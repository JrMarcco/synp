package ioc

import (
	"github.com/JrMarcco/synp/internal/pkg/xmq/produce"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

var KafkaProducerFxOpt = fx.Module(
	"kafka_producer",
	fx.Provide(
		fx.Annotate(
			initKafkaProducer,
			fx.As(new(produce.Producer)),
		),
	),
)

type kafkaProducerFxParams struct {
	fx.In

	Writer *kafka.Writer
}

func initKafkaProducer(params kafkaProducerFxParams) *produce.KafkaProducer {
	return produce.NewKafkaProducer(params.Writer)
}
