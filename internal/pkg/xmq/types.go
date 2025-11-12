package xmq

import "errors"

var ErrConsumerClosed = errors.New("consumer closed")

type Headers map[string]string

type Message struct {
	Headers Headers

	Topic     string
	Partition int
	Offset    int64

	Key []byte
	Val []byte
}
