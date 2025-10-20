package codec_test

import (
	"testing"

	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestProtoCodec(t *testing.T) {
	t.Parallel()

	protoCodec := codec.NewProtoCodec()
	assert.Equal(t, "proto", protoCodec.Name())

	suite.Run(t, &CodecSuite{codec: protoCodec})
}
