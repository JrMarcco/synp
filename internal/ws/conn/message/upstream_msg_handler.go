package message

import (
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"go.uber.org/zap"
)

var _ MsgHandler = (*UpstreamMsgHandler)(nil)

// UpstreamMsgHandler 是上游消息处理器的实现，用于处理前端（业务客户端）发送的消息。
type UpstreamMsgHandler struct {
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
			zap.String("step", "handle_upstream_message"),
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
func (h *UpstreamMsgHandler) messageAck(conn synp.Conn, msg *messagev1.Message) error {
	//TODO: not implemented
	panic("not implemented")
}
