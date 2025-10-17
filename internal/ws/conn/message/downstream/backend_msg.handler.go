package downstream

import (
	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"go.uber.org/zap"
)

var _ DMsgHandler = (*BackendMsgHandler)(nil)

// BackendMsgHandler 是 backend 消息处理器的实现，用于处理后端推送的消息。
type BackendMsgHandler struct {
	codec  codec.Codec
	logger *zap.Logger
}

func (h *BackendMsgHandler) Handle(conn synp.Conn, pushMsg *messagev1.PushMessage) error {
	// 设置重试。
	// 当前端返回 ack 消息后，停止重试。
	defer func() {
		//TODO: 设置重试
	}()

	downstreamMsg := &messagev1.Message{
		Cmd:       commonv1.CommandType_COMMAND_TYPE_DOWNSTREAM,
		MessageId: pushMsg.GetMessageId(),
		Body:      pushMsg.GetBody(),
	}

	if err := h.sendDownstreamMessage(conn, downstreamMsg); err != nil {
		return err
	}

	// 成功发送消息到前端，更新连接活跃时间。
	conn.UpdateActivityTime()
	return nil
}

func (h *BackendMsgHandler) sendDownstreamMessage(conn synp.Conn, msg *messagev1.Message) error {
	payload, err := h.codec.Marshal(msg)
	if err != nil {
		h.logger.Error(
			"[synp-backend-msg-handler] failed to marshal message",
			zap.String("codec_name", h.codec.Name()),
			zap.String("message", msg.String()),
			zap.Error(err),
		)
	}

	if err = conn.Send(payload); err != nil {
		h.logger.Error(
			"[synp-backend-msg-handler] failed to send message",
			zap.String("connection_id", conn.Id()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
