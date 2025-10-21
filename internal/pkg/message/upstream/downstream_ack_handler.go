package upstream

import (
	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"go.uber.org/zap"
)

var _ UMsgHandler = (*DownstreamAckHandler)(nil)

type DownstreamAckHandler struct {
	retransmitManager *retransmit.Manager
	logger            *zap.Logger
}

func (h *DownstreamAckHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	// 停止向前端推送 downstream 消息的重试。
	h.retransmitManager.Stop(msg.MessageId)

	h.logger.Debug(
		"[synp-downstream-ack-handler] received downstream ack message",
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
	)
	return nil
}

func (h *DownstreamAckHandler) CmdType() commonv1.CommandType {
	return commonv1.CommandType_COMMAND_TYPE_DOWNSTREAM_ACK
}

func NewDownstreamAckHandler(retransmitManager *retransmit.Manager, logger *zap.Logger) *DownstreamAckHandler {
	return &DownstreamAckHandler{
		retransmitManager: retransmitManager,
		logger:            logger,
	}
}
