package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestProtoCodec(t *testing.T) {
	t.Parallel()

	protoCodec := NewProtoCodec()
	assert.Equal(t, "proto", protoCodec.Name())

	suite.Run(t, &CodecSuite{codec: protoCodec})
}
