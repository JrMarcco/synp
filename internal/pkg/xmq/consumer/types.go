package consumer

import (
	"context"

	"github.com/JrMarcco/synp/internal/pkg/xmq"
)

type Consumer interface {
	Consume(ctx context.Context) (*xmq.Message, error)
	ConsumeChan(ctx context.Context) (<-chan *xmq.Message, error)
	Close() error
}
