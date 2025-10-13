package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

var _ Codec = (*ProtoCodec)(nil)

type ProtoCodec struct{}

func (c *ProtoCodec) Name() string {
	return "proto"
}

func (c *ProtoCodec) Marshal(val any) ([]byte, error) {
	protoMsg, ok := val.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to marshal message: invalid message type, expected proto.Message, got %T", val)
	}
	return proto.Marshal(protoMsg)
}

func (c *ProtoCodec) Unmarshal(data []byte, val any) error {
	protoMsg, ok := val.(proto.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal message: invalid message type, expected proto.Message, got %T", val)
	}
	return proto.Unmarshal(data, protoMsg)
}

func NewProtoCodec() *ProtoCodec {
	return &ProtoCodec{}
}
