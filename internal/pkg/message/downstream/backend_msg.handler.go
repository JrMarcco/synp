package downstream

import (
	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"go.uber.org/zap"
)

var _ DMsgHandler = (*BackendMsgHandler)(nil)

// BackendMsgHandler 是 backend 消息处理器的实现，用于处理后端推送的消息。
type BackendMsgHandler struct {
	pushFunc          message.PushFunc
	retransmitManager *retransmit.Manager
	logger            *zap.Logger
}

func (h *BackendMsgHandler) Handle(conns []synp.Conn, pushMsg *messagev1.PushMessage) error {
	downstreamMsg := &messagev1.Message{
		Cmd:       commonv1.CommandType_COMMAND_TYPE_DOWNSTREAM,
		MessageId: pushMsg.GetMessageId(),
		Body:      pushMsg.GetBody(),
	}

	// 设置重试。
	// 当前端返回 ack 消息后，停止重试。
	defer h.retransmitManager.Start(conns, downstreamMsg)

	for _, conn := range conns {
		if err := h.pushFunc(conn, downstreamMsg); err != nil {
			return err
		}

		// 成功发送消息到前端，更新连接活跃时间。
		conn.UpdateActivityTime()
	}
	return nil
}

func NewBackendMsgHandler(
	pushFunc message.PushFunc,
	retransmitManager *retransmit.Manager,
	logger *zap.Logger,
) *BackendMsgHandler {
	return &BackendMsgHandler{
		pushFunc:          pushFunc,
		retransmitManager: retransmitManager,
		logger:            logger,
	}
}
