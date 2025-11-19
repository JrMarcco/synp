package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestJsonCodec(t *testing.T) {
	t.Parallel()

	jsonCodec := NewJSONCodec()
	assert.Equal(t, "json", jsonCodec.Name())

	suite.Run(t, &CodecSuite{codec: jsonCodec})
}
