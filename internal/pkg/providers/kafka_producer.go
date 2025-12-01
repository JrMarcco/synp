package providers

import (
	"github.com/JrMarcco/synp/internal/pkg/xmq/produce"
	"github.com/segmentio/kafka-go"
)

func newKafkaProducer(writer *kafka.Writer) *produce.KafkaProducer {
	return produce.NewKafkaProducer(writer)
}
