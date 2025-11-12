package codec

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _ Codec = (*JSONCodec)(nil)

type JSONCodec struct{}

func (c *JSONCodec) Name() string {
	return "json"
}

func (c *JSONCodec) Marshal(val any) ([]byte, error) {
	protoMsg, ok := val.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to marshal message: invalid message type, expected proto.Message, got %T", val)
	}
	return protojson.Marshal(protoMsg)
}

func (c *JSONCodec) Unmarshal(data []byte, val any) error {
	protoMsg, ok := val.(proto.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal message: invalid message type, expected proto.Message, got %T", val)
	}
	return protojson.Unmarshal(data, protoMsg)
}

func NewJSONCodec() *JSONCodec {
	return &JSONCodec{}
}
