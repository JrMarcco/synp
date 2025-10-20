package codec_test

import (
	"testing"

	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestJsonCodec(t *testing.T) {
	t.Parallel()

	jsonCodec := codec.NewJsonCodec()
	assert.Equal(t, "json", jsonCodec.Name())

	suite.Run(t, &CodecSuite{codec: jsonCodec})
}
