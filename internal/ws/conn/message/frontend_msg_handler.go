package message

import (
	"context"
	"fmt"
	"time"

	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/xmq/producer"
	"go.uber.org/zap"
)

var _ MsgHandler = (*FrontendMsgHandler)(nil)

// FrontendMsgHandler 是前端消息处理器的实现，用于处理前端 ( 业务客户端 ) 发送的消息。
type FrontendMsgHandler struct {
	codec    codec.Codec
	producer producer.Producer[*messagev1.Message]

	logger *zap.Logger
}

func (h *FrontendMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	// 更新连接活跃时间。
	conn.UpdateActivityTime()

	// 转发消息到业务服务端。
	ackMsg := h.forwardToBackend(msg)

	// 发送消息到业务服务端。
	return h.messageAck(conn, ackMsg)
}

// forwardToBackend 转发消息到业务服务端。
// 通信方式为推送消息到 kafka，由业务服务端订阅并处理。
func (h *FrontendMsgHandler) forwardToBackend(msg *messagev1.Message) *messagev1.AckMessage {
	ackMsg := &messagev1.AckMessage{
		MessageId: msg.GetMessageId(),
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := h.producer.Produce(context.TODO(), msg); err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to forward message to backend with messsage queue",
			zap.String("step", "frontend_msg_handle"),
			zap.Error(err),
		)

		ackMsg.Success = false
		ackMsg.ErrMsg = err.Error()
	}

	return ackMsg
}

// messageAck 发送 ack 消息，通知前端消息已收到。
func (h *FrontendMsgHandler) messageAck(conn synp.Conn, ackMsg *messagev1.AckMessage) error {
	payload, err := h.codec.Marshal(ackMsg)
	if err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to marshal message",
			zap.String("codec_name", h.codec.Name()),
			zap.String("message", ackMsg.String()),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", ErrMarshalMessage, err)
	}

	if err = conn.Send(payload); err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to send message",
			zap.String("connection_id", conn.Id()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
