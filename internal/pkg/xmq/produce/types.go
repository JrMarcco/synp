package produce

import (
	"context"

	"github.com/cespare/xxhash/v2"
)

type Producer[T any] interface {
	Produce(ctx context.Context, msg T) error
}

func partitionFromMsg(messageId string, partitions int32) int32 {
	return int32(xxhash.Sum64String(messageId) % uint64(partitions))
}
