package produce

import (
	"context"

	"github.com/JrMarcco/synp/internal/pkg/xmq"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var _ Producer = (*KafkaProducer)(nil)

type KafkaProducer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func (p *KafkaProducer) Produce(ctx context.Context, msg *xmq.Message) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: msg.Topic,
		Key:   msg.Key,
		Value: msg.Val,
	})

	if err != nil {
		return err
	}

	p.logger.Debug(
		"[synp-xmq-producer] successfully produced message to kafka",
		zap.String("topic", msg.Topic),
		zap.Int64("offset", msg.Offset),
		zap.String("message", string(msg.Val)),
	)

	return nil
}

func NewKafkaProducer(writer *kafka.Writer, logger *zap.Logger) *KafkaProducer {
	return &KafkaProducer{
		writer: writer,
		logger: logger,
	}
}
