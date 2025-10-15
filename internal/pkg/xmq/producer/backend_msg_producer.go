package producer

import (
	"context"

	"github.com/IBM/sarama"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ Producer[*messagev1.Message] = (*BackendMsgProducer)(nil)

// BackendMsgProducer 是后端消息生产者。
// 负责将前端发送到网关的消息转发到 kafka，由后端订阅并处理。
// frontend -> gateway -> kafka -> backend
type BackendMsgProducer struct {
	topic      string
	partitions int32

	producer sarama.SyncProducer
	logger   *zap.Logger
}

func (p *BackendMsgProducer) Produce(ctx context.Context, msg *messagev1.Message) error {
	val, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	// 根据 message_id 选择 kafka partition。
	partition := partitionFromMsg(msg.GetMessageId(), p.partitions)
	rtnPartition, offset, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     p.topic,
		Partition: partition,
		Value:     sarama.ByteEncoder(val),
	})
	if err != nil {
		return err
	}

	p.logger.Debug(
		"[synp-xmq-producer] successfully produced message to backend",
		zap.String("topic", p.topic),
		zap.Int32("partition", rtnPartition),
		zap.String("message", string(val)),
		zap.Int64("offset", offset),
	)
	return nil
}
