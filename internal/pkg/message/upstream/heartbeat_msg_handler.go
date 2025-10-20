package upstream

import (
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"go.uber.org/zap"
)

var _ UMsgHandler = (*HeartbeatMsgHandler)(nil)

// HeartbeatMsgHandler 是心跳消息处理器的实现，用于处理心跳消息。
type HeartbeatMsgHandler struct {
	pushFunc message.MessagePushFunc
	logger   *zap.Logger
}

func (h *HeartbeatMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	h.logger.Debug(
		"[synp-heartbeat-msg-handler] received heartbeat message",
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
	)

	// 将心跳包直接返回给前端 ( ping & pong )。
	return h.pushFunc(conn, msg)
}

func NewHeartbeatMsgHandler(pushFunc message.MessagePushFunc, logger *zap.Logger) *HeartbeatMsgHandler {
	return &HeartbeatMsgHandler{
		pushFunc: pushFunc,
		logger:   logger,
	}
}
