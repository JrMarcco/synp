package message

import (
	"fmt"

	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"go.uber.org/zap"
)

var _ MsgHandler = (*UpstreamMsgHandler)(nil)

// UpstreamMsgHandler 是上游消息处理器的实现，用于处理前端（业务客户端）发送的消息。
type UpstreamMsgHandler struct {
	codec  codec.Codec
	logger *zap.Logger
}

func (h *UpstreamMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	// 更新连接活跃时间。
	conn.UpdateActivityTime()

	bizId := conn.Session().UserInfo().Bid

	// 转发消息到业务服务端。
	_, err := h.forwardToBackend(bizId, msg)
	if err != nil {
		h.logger.Error(
			"[synp-conn-message-handler] failed to forward message to backend",
			zap.String("step", "upstream_message_handle"),
			zap.Error(err),
		)
		return err
	}

	//TODO: 发送消息到业务服务端。

	//TODO: 发送 ack 消息，通知前端消息已收到。

	return nil
}

// forwardToBackend 转发消息到业务服务端。
// 通信方式为推送消息到 kafka，由业务服务端订阅并处理。
func (h *UpstreamMsgHandler) forwardToBackend(bizId uint64, msg *messagev1.Message) (*messagev1.Message, error) {
	//TODO: not implemented
	panic("not implemented")
}

// messageAck 发送 ack 消息，通知前端消息已收到。
func (h *UpstreamMsgHandler) messageAck(conn synp.Conn, oriMsg *messagev1.Message) error {
	//TODO: body 需要放入消息转发结果
	msg := &messagev1.Message{
		Cmd:    commonv1.CommandType_COMMAND_TYPE_UPSTREAM_ACK,
		BizKey: oriMsg.GetBizKey(),
	}

	payload, err := h.codec.Marshal(msg)
	if err != nil {
		h.logger.Error(
			"[synp-conn-message-handler] failed to marshal message",
			zap.String("step", "upstream_message_ack"),
			zap.String("codec_name", h.codec.Name()),
			zap.String("message", msg.String()),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", ErrMarshalMessage, err)
	}

	if err = conn.Send(payload); err != nil {
		h.logger.Error(
			"[synp-conn-message-handler] failed to send message",
			zap.String("step", "upstream_message_ack"),
			zap.String("connection_id", conn.Id()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
