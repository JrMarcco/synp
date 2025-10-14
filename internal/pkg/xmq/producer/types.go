package producer

import "context"

type Producer[T any] interface {
	Produce(ctx context.Context, msg T) error
}
