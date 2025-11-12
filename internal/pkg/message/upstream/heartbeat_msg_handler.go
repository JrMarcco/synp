package upstream

import (
	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"go.uber.org/zap"
)

var _ UMsgHandler = (*HeartbeatMsgHandler)(nil)

// HeartbeatMsgHandler 是心跳消息处理器的实现，用于处理心跳消息。
type HeartbeatMsgHandler struct {
	pushFunc message.PushFunc
	logger   *zap.Logger
}

func (h *HeartbeatMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	h.logger.Debug(
		"[synp-heartbeat-msg-handler] received heartbeat message",
		zap.String("connection_id", conn.ID()),
		zap.String("message", msg.String()),
	)

	// 将心跳包直接返回给前端 ( ping & pong )。
	return h.pushFunc(conn, msg)
}

func (h *HeartbeatMsgHandler) CmdType() commonv1.CommandType {
	return commonv1.CommandType_COMMAND_TYPE_HEARTBEAT
}

func NewHeartbeatMsgHandler(pushFunc message.PushFunc, logger *zap.Logger) *HeartbeatMsgHandler {
	return &HeartbeatMsgHandler{
		pushFunc: pushFunc,
		logger:   logger,
	}
}
