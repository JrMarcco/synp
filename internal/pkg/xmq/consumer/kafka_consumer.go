package consumer

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/JrMarcco/synp/internal/pkg/xmq"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// KafkaReaderFactory 是创建 kafka Reader 的工厂函数。
type KafkaReaderFactory func(topic string, groupId string) *kafka.Reader

var _ Consumer = (*KafkaConsumer)(nil)

// KafkaConsumer 是 kafka 消费者。
// 负责从 kafka 中消费消息，并转换为 xmq.Message。
type KafkaConsumer struct {
	readerFactory KafkaReaderFactory
	reader        *kafka.Reader

	topic   string
	groupId string

	messageChan chan *xmq.Message

	ctx        context.Context
	cancelFunc context.CancelFunc

	closeOnce sync.Once

	logger *zap.Logger
}

func (c *KafkaConsumer) Consume(ctx context.Context) (*xmq.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-c.messageChan:
		if !ok {
			return nil, xmq.ErrConsumerClosed
		}
		return msg, nil
	}
}

func (c *KafkaConsumer) ConsumeChan(ctx context.Context) (<-chan *xmq.Message, error) {
	if c.ctx.Err() != nil {
		return nil, xmq.ErrConsumerClosed
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return c.messageChan, nil
}

func (c *KafkaConsumer) readMessage() {
	defer func() {
		close(c.messageChan)
	}()

	for {
		kafkaMsg, err := c.reader.ReadMessage(c.ctx)
		if err != nil {
			if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return
			}

			c.logger.Error(
				"[synp-xmq-consumer] failed to read message from kafka",
				zap.String("topic", c.topic),
				zap.String("group_id", c.groupId),
				zap.Error(err),
			)
			continue
		}

		msg := c.convertMessage(&kafkaMsg)
		select {
		case c.messageChan <- msg:
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *KafkaConsumer) convertMessage(kafkaMsg *kafka.Message) *xmq.Message {
	headers := xmq.Headers{}
	for _, header := range kafkaMsg.Headers {
		headers[header.Key] = string(header.Value)
	}

	return &xmq.Message{
		Headers: headers,

		Topic:     kafkaMsg.Topic,
		Partition: kafkaMsg.Partition,
		Offset:    kafkaMsg.Offset,

		Key: kafkaMsg.Key,
		Val: kafkaMsg.Value,
	}
}

func (c *KafkaConsumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.cancelFunc()
		err = c.reader.Close()
	})
	return err
}

func NewConsumer(readerFactory KafkaReaderFactory, topic, groupId string, partitions int32, logger *zap.Logger) *KafkaConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	consumer := &KafkaConsumer{
		readerFactory: readerFactory,

		topic:   topic,
		groupId: groupId,

		messageChan: make(chan *xmq.Message),

		ctx:        ctx,
		cancelFunc: cancel,

		logger: logger,
	}

	go consumer.readMessage()
	return consumer
}
