package consume

import (
	"context"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type ConsumeFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

type Consumer struct {
	client sarama.Client

	name       string
	topic      string
	partitions int32

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

func (c *Consumer) Start(ctx context.Context, consumeFunc ConsumeFunc) error {
	for i := range c.partitions {
		partition := i

		consumer, err := sarama.NewConsumerFromClient(c.client)
		if err != nil {
			c.logger.Error(
				"[synp-xmq-consumer] failed to create consumer from client",
				zap.Int32("partition", partition),
				zap.Error(err),
			)
			return err
		}

		pc, err := consumer.ConsumePartition(c.topic, partition, sarama.OffsetNewest)
		if err != nil {
			c.logger.Error(
				"[synp-xmq-consumer] failed to consume partition",
				zap.Int32("partition", partition),
				zap.Error(err),
			)
			return err
		}

		go c.consumeLoop(ctx, partition, pc, consumeFunc)
	}

	return nil
}

func (c *Consumer) consumeLoop(
	ctx context.Context,
	partition int32,
	pc sarama.PartitionConsumer,
	consumeFunc ConsumeFunc,
) {
	c.logger.Info(
		"[synp-xmq-consumer] consume loop started",
		zap.Int32("partition", partition),
		zap.String("topic", c.topic),
	)

	defer pc.Close()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info(
				"[synp-xmq-consumer] inner context done, consume loop stopped",
				zap.Int32("partition", partition),
				zap.String("topic", c.topic),
			)
			return
		case <-ctx.Done():
			c.logger.Info(
				"[synp-xmq-consumer] context done, consume loop stopped",
				zap.Int32("partition", partition),
				zap.String("topic", c.topic),
			)
			return
		case msg, ok := <-pc.Messages():
			if !ok {
				return
			}

			if err := consumeFunc(ctx, msg); err != nil {
				c.logger.Error(
					"[synp-xmq-consumer] failed to consume message",
					zap.Int32("partition", partition),
					zap.String("topic", c.topic),
					zap.String("message", string(msg.Value)),
					zap.Error(err),
				)
			} else {
				c.logger.Debug(
					"[synp-xmq-consumer] consumed message",
					zap.Int32("partition", partition),
					zap.String("topic", c.topic),
					zap.String("message", string(msg.Value)),
				)
			}
		}

	}
}

func (c *Consumer) Stop() error {
	c.cancelFunc()
	return nil
}

func NewConsumer(client sarama.Client, name, topic string, partitions int32, logger *zap.Logger) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		client: client,

		name:       name,
		topic:      topic,
		partitions: partitions,

		ctx:        ctx,
		cancelFunc: cancel,

		logger: logger,
	}
}
