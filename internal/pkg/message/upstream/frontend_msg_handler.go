package upstream

import (
	"context"
	"fmt"
	"time"

	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/xmq"
	"github.com/JrMarcco/synp/internal/pkg/xmq/produce"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ UMsgHandler = (*FrontendMsgHandler)(nil)

// FrontendMsgHandler 是前端消息处理器的实现，用于处理前端 ( 业务客户端 ) 发送的消息。
type FrontendMsgHandler struct {
	mqTopic          string        // 消息队列 topic
	onReceiveTimeout time.Duration // 接收消息超时时间

	codec    codec.Codec
	producer produce.Producer
	pushFunc message.PushFunc

	logger *zap.Logger
}

func (h *FrontendMsgHandler) Handle(conn synp.Conn, msg *messagev1.Message) error {
	// 接收到前端消息，更新连接活跃时间。
	conn.UpdateActivityTime()

	ackPayload := &messagev1.AckPayload{
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	}

	// 转发消息到业务服务端。
	if err := h.forwardToBackend(msg); err != nil {
		ackPayload.Success = false
		ackPayload.ErrorMessage = err.Error()
	}

	// 发送消息到业务服务端。
	body, err := protojson.Marshal(ackPayload)
	if err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to marshal ack payload",
			zap.String("step", "frontend_msg_handle"),
			zap.String("ack_payload", ackPayload.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal ack payload: %w", err)
	}

	return h.pushFunc(conn, &messagev1.Message{
		MessageId: msg.GetMessageId(),
		Cmd:       commonv1.CommandType_COMMAND_TYPE_UPSTREAM_ACK,
		Body:      body,
	})
}

// forwardToBackend 转发消息到业务服务端。
// 通信方式为推送消息到 kafka，由业务服务端订阅并处理。
func (h *FrontendMsgHandler) forwardToBackend(msg *messagev1.Message) error {
	val, err := protojson.Marshal(msg)
	if err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to marshal message",
			zap.String("step", "frontend_msg_handle"),
			zap.String("message", msg.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	mqMsg := &xmq.Message{
		Topic: h.mqTopic,
		Key:   []byte(msg.GetMessageId()),
		Val:   val,
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.onReceiveTimeout)
	defer cancel()

	if err := h.producer.Produce(ctx, mqMsg); err != nil {
		h.logger.Error(
			"[synp-frontend-msg-handler] failed to forward message to backend with messsage queue",
			zap.String("step", "frontend_msg_handle"),
			zap.Error(err),
		)
		return fmt.Errorf("failed to forward message: %w", err)
	}

	return nil
}

func (h *FrontendMsgHandler) CmdType() commonv1.CommandType {
	return commonv1.CommandType_COMMAND_TYPE_UPSTREAM
}

func NewFrontendMsgHandler(
	mqTopic string,
	onReceiveTimeout time.Duration,
	codec codec.Codec,
	producer produce.Producer,
	pushFunc message.PushFunc,
	logger *zap.Logger,
) *FrontendMsgHandler {
	return &FrontendMsgHandler{
		mqTopic:          mqTopic,
		onReceiveTimeout: onReceiveTimeout,

		codec:    codec,
		producer: producer,
		pushFunc: pushFunc,

		logger: logger,
	}
}
