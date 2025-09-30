package codec_test

import (
	"testing"

	"github.com/JrMarcco/synp/pkg/codec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CodecSuite struct {
	suite.Suite
	c codec.Codec
}

func (c *CodecSuite) TestMarshal() {
}

func TestJsonCodec(t *testing.T) {
	t.Parallel()

	jsonCodec := codec.NewJsonCodec()
	assert.Equal(t, "json", jsonCodec.Name())

	suite.Run(t, &CodecSuite{c: jsonCodec})
}

func TestProtoCodec(t *testing.T) {
	t.Parallel()

	protoCodec := codec.NewProtoCodec()
	assert.Equal(t, "proto", protoCodec.Name())

	suite.Run(t, &CodecSuite{c: protoCodec})
}
