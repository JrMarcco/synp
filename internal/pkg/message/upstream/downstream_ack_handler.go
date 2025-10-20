package upstream

import (
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"go.uber.org/zap"
)

type DownstreamAckHandler struct {
	retransmitManager *retransmit.Manager
	logger            *zap.Logger
}

func (h *DownstreamAckHandler) Handle(conn synp.Conn, msg *messagev1.AckMessage) error {
	// 停止向前端推送 downstream 消息的重试。
	h.retransmitManager.Stop(msg.MessageId)

	h.logger.Debug(
		"[synp-downstream-ack-handler] received downstream ack message",
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
	)
	return nil
}

func NewDownstreamAckHandler(retransmitManager *retransmit.Manager, logger *zap.Logger) *DownstreamAckHandler {
	return &DownstreamAckHandler{
		retransmitManager: retransmitManager,
		logger:            logger,
	}
}
