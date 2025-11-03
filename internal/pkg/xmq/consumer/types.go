package consumer

import (
	"context"

	"github.com/JrMarcco/synp/internal/pkg/xmq"
)

// ConsumerFactory 是创建消费者的工厂函数。
type ConsumerFactory interface {
	NewConsumer(topic, groupId string) (Consumer, error)
}

type Consumer interface {
	Consume(ctx context.Context) (*xmq.Message, error)
	ConsumeChan(ctx context.Context) (<-chan *xmq.Message, error)
	Close() error
}
