package upstream

import (
	"fmt"

	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/ws/conn/message"
	"go.uber.org/zap"
)

var _ UMsgHandler = (*HeartbeatMsgHandler)(nil)

// HeartbeatMsgHandler 是心跳消息处理器的实现，用于处理心跳消息。
type HeartbeatMsgHandler struct {
	codec  codec.Codec
	logger *zap.Logger
}

func (h *HeartbeatMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	h.logger.Debug(
		"[synp-heartbeat-msg-handler] received heartbeat message",
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
	)

	// 将心跳包直接返回给前端 ( ping & pong )。
	payload, err := h.codec.Marshal(msg)
	if err != nil {
		h.logger.Error(
			"[synp-heartbeat-msg-handler] failed to marshal message",
			zap.String("connection_id", conn.Id()),
			zap.String("codec_name", h.codec.Name()),
			zap.String("message", msg.String()),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", message.ErrMarshalMessage, err)
	}

	if err = conn.Send(payload); err != nil {
		h.logger.Error(
			"[synp-heartbeat-msg-handler] failed to send message",
			zap.String("connection_id", conn.Id()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
