package upstream

import (
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"go.uber.org/zap"
)

type DownstreamAckHandler struct {
	logger *zap.Logger
}

func (h *DownstreamAckHandler) Handle(conn synp.Conn, msg *messagev1.AckMessage) error {
	//TODO: 停止向前端推送 downstream 消息的重试。

	h.logger.Debug(
		"[synp-downstream-ack-handler] received downstream ack message",
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
	)
	return nil
}
