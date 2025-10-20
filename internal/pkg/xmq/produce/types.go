package produce

import (
	"context"

	"github.com/JrMarcco/synp/internal/pkg/xmq"
)

type Producer interface {
	Produce(ctx context.Context, msg *xmq.Message) error
}
