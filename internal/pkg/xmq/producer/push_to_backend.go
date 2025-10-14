package producer

import (
	"context"

	"github.com/IBM/sarama"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ Producer[*messagev1.Message] = (*PushToBackendProducer)(nil)

type PushToBackendProducer struct {
	topic string

	producer sarama.SyncProducer
	logger   *zap.Logger
}

func (p *PushToBackendProducer) Produce(ctx context.Context, msg *messagev1.Message) error {
	val, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	partition, offset, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(val),
	})
	if err != nil {
		return err
	}

	p.logger.Debug(
		"[synp-xmq-producer] successfully produced message",
		zap.String("topic", p.topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)
	return nil
}
