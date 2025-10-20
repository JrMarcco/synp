package codec_test

import (
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type CodecSuite struct {
	suite.Suite
	codec codec.Codec
}

func (c *CodecSuite) TestNormalMessage() {
	t := c.T()

	expectedBody := wrapperspb.String("jrmarcco")
	body, err := protojson.Marshal(wrapperspb.String("jrmarcco"))
	assert.NoError(t, err)

	sendMsg := &messagev1.Message{
		MessageId: "test_message_id",
		Cmd:       commonv1.CommandType_COMMAND_TYPE_UPSTREAM,
		Body:      body,
	}

	bytes, err := c.codec.Marshal(sendMsg)
	assert.NoError(t, err)

	receivedMsg := &messagev1.Message{}
	err = c.codec.Unmarshal(bytes, receivedMsg)
	assert.NoError(t, err)

	assert.Equal(t, sendMsg.String(), receivedMsg.String())
	assert.True(t, proto.Equal(sendMsg, receivedMsg))

	actualBody := &wrapperspb.StringValue{}
	err = protojson.Unmarshal(receivedMsg.GetBody(), actualBody)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody.String(), actualBody.String())
	assert.True(t, proto.Equal(expectedBody, actualBody))
}

func (c *CodecSuite) TestHeartbeatMessage() {
	t := c.T()

	sendMsg := &messagev1.Message{
		MessageId: "test_message_id",
		Cmd:       commonv1.CommandType_COMMAND_TYPE_HEARTBEAT,
		Body:      nil,
	}

	bytes, err := c.codec.Marshal(sendMsg)
	assert.NoError(t, err)

	receivedMsg := &messagev1.Message{}
	err = c.codec.Unmarshal(bytes, receivedMsg)
	assert.NoError(t, err)

	assert.Equal(t, sendMsg.String(), receivedMsg.String())
	assert.True(t, proto.Equal(sendMsg, receivedMsg))
}

func (c *CodecSuite) TestInvalidMessage() {
	t := c.T()

	msg := "invalid message"
	payload, err := c.codec.Marshal(msg)
	assert.Error(t, err)
	assert.Nil(t, payload)
}
