package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message/downstream"
	"github.com/JrMarcco/synp/internal/pkg/message/upstream"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrInvalidMessage     = errors.New("invalid message")
	ErrCacheMessage       = errors.New("cache message failed")
	ErrUncacheMessage     = errors.New("uncache message failed")
	ErrMessageDuplicated  = errors.New("message duplicated, ignore it")
	ErrUnknownMessageType = errors.New("unknown message (command) type")
	ErrMaxRetryExceeded   = errors.New("max retry exceeded")
)

var _ synp.ConnEventHandler = (*EvtHandler)(nil)

// EvtHandler 是连接事件的处理器。
type EvtHandler struct {
	rdb redis.Cmdable

	cacheRequestTimeout time.Duration
	cacheExpiration     time.Duration

	codec codec.Codec

	uMsgHandlers map[commonv1.CommandType]upstream.UMsgHandler
	dMsgHandler  downstream.DMsgHandler

	logger *zap.Logger
}

func (h *EvtHandler) OnConnect(conn synp.Conn) error {
	h.logger.Debug(
		"[synp-conn-evt-handler] connection connected",
		zap.String("connection_id", conn.Id()),
	)
	return nil
}

func (h *EvtHandler) OnDisconnect(conn synp.Conn) error {
	h.logger.Debug(
		"[synp-conn-evt-handler] connection disconnected",
		zap.String("connection_id", conn.Id()),
	)
	return conn.Close()
}

func (h *EvtHandler) OnReceiveFromFrontend(conn synp.Conn, payload []byte) error {
	// 解析 payload。
	msg, err := h.decodePayload(payload)
	if err != nil {
		h.logger.Error(
			"[synp-conn-evt-handler] failed to decode payload",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.Any("user", conn.Session().UserInfo()),
			zap.Error(err),
		)
		return err
	}

	// 消息去重（幂等）。
	user := conn.Session().UserInfo()
	ok, err := h.cacheMessage(user.Bid, msg)
	if err != nil {
		h.logger.Error(
			"[synp-conn-evt-handler] failed to cache message",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.Any("user", conn.Session().UserInfo()),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", ErrCacheMessage, err)
	}

	if !ok {
		h.logger.Warn(
			"[synp-conn-evt-handler] message duplicated, ignore it",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.String("message_id", msg.GetMessageId()),
			zap.Any("user", conn.Session().UserInfo()),
		)
		return ErrMessageDuplicated
	}

	h.logger.Info(
		"[synp-conn-evt-handler] received message from frontend",
		zap.String("step", "on_receive_from_frontend"),
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
		zap.Any("user", conn.Session().UserInfo()),
	)

	// 处理消息。
	uMsgHandler, ok := h.uMsgHandlers[msg.GetCmd()]
	if !ok {
		h.logger.Error(
			"[synp-conn-evt-handler] unknown message type from frontend",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.Any("user", conn.Session().UserInfo()),
		)
		return ErrUnknownMessageType
	}

	if err = uMsgHandler.Handle(conn, msg); err != nil {
		// 删除消息缓存。
		if h.needUncacheMessage(err) {
			uncacheErr := h.uncacheMessage(user.Bid, msg)
			if uncacheErr != nil {
				h.logger.Error(
					"[synp-conn-evt-handler] failed to uncache message",
					zap.String("step", "on_receive_from_frontend"),
					zap.String("connection_id", conn.Id()),
					zap.String("message", msg.String()),
					zap.Error(uncacheErr),
				)
			}
		}
	}
	return err
}

// decodePayload 解析 payload。
// 注：
//
//	这里解析的是整个消息字节流，消息体 ( body 字段 ) 需要另外解析。
func (h *EvtHandler) decodePayload(payload []byte) (*messagev1.Message, error) {
	msg := &messagev1.Message{}
	if err := h.codec.Unmarshal(payload, msg); err != nil {
		return nil, fmt.Errorf("%w: unknown message type", ErrInvalidMessage)
	}

	if msg.GetCmd() != commonv1.CommandType_COMMAND_TYPE_HEARTBEAT && msg.GetMessageId() == "" {
		// 非心跳消息，message_id 不能为空。
		return nil, fmt.Errorf("%w: empty message_id", ErrInvalidMessage)
	}

	return msg, nil
}

func (h *EvtHandler) cacheMessage(bizId uint64, msg *messagev1.Message) (bool, error) {
	if msg.GetCmd() == commonv1.CommandType_COMMAND_TYPE_HEARTBEAT {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.cacheRequestTimeout)
	defer cancel()

	return h.rdb.SetNX(ctx, h.cacheKey(bizId, msg.GetMessageId()), msg.GetMessageId(), h.cacheExpiration).Result()
}

func (h *EvtHandler) needUncacheMessage(err error) bool {
	return errors.Is(err, ErrUnknownMessageType) || errors.Is(err, ErrMaxRetryExceeded)
}

func (h *EvtHandler) uncacheMessage(bizId uint64, msg *messagev1.Message) error {
	if msg.GetCmd() == commonv1.CommandType_COMMAND_TYPE_HEARTBEAT {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.cacheRequestTimeout)
	defer cancel()

	if _, err := h.rdb.Del(ctx, h.cacheKey(bizId, msg.GetMessageId())).Result(); err != nil {
		return fmt.Errorf("%w: %w", ErrUncacheMessage, err)
	}

	return nil
}

func (h *EvtHandler) cacheKey(bizId uint64, messageId string) string {
	return fmt.Sprintf("%d:%s", bizId, messageId)
}

func (h *EvtHandler) OnReceiveFromBackend(conn synp.Conn, msg *messagev1.PushMessage) error {
	if msg.GetMessageId() == "" {
		return fmt.Errorf("%w: empty message_id", ErrInvalidMessage)
	}
	if msg.GetBizId() == 0 {
		return fmt.Errorf("%w: empty biz_id", ErrInvalidMessage)
	}
	if msg.GetReceiverId() == 0 {
		return fmt.Errorf("%w: empty receiver_id", ErrInvalidMessage)
	}

	return h.dMsgHandler.Handle(conn, msg)
}

func NewEventHandler(
	rdb redis.Cmdable,
	cacheRequestTimeout time.Duration,
	cacheExpiration time.Duration,
	codec codec.Codec,
	uMsgHandlers []upstream.UMsgHandler,
	dMsgHandler downstream.DMsgHandler,
	logger *zap.Logger,
) synp.ConnEventHandler {
	m := make(map[commonv1.CommandType]upstream.UMsgHandler)
	for _, handler := range uMsgHandlers {
		m[handler.CmdType()] = handler
	}

	return &EvtHandler{
		rdb:                 rdb,
		cacheRequestTimeout: cacheRequestTimeout,
		cacheExpiration:     cacheExpiration,
		codec:               codec,
		uMsgHandlers:        m,
		dMsgHandler:         dMsgHandler,
		logger:              logger,
	}

}
