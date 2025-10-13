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
	"github.com/JrMarcco/synp/internal/ws/conn/message"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrInvalidMessage     = errors.New("invalid message")
	ErrCacheMessage       = errors.New("cache message failed")
	ErrUncacheMessage     = errors.New("uncache message failed")
	ErrMessageDuplicated  = errors.New("message duplicated, ignore it")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrMaxRetryExceeded   = errors.New("max retry exceeded")
)

var _ synp.ConnEventHandler = (*EvtHandler)(nil)

// EvtHandler 是连接事件的处理器。
type EvtHandler struct {
	rdb redis.Cmdable

	cacheRequestTimeout time.Duration
	cacheExpiration     time.Duration

	codec codec.Codec

	msgHandlers map[commonv1.CommandType]message.MsgHandler

	logger *zap.Logger
}

func (h *EvtHandler) OnConnect(conn synp.Conn) error {
	//TODO: not implemented
	panic("not implemented")
}

func (h *EvtHandler) OnDisconnect(conn synp.Conn) error {
	//TODO: not implemented
	panic("not implemented")
}

func (h *EvtHandler) OnReceiveFromFrontend(conn synp.Conn, payload []byte) error {
	// 解析 payload。
	msg, err := h.decodePayload(payload)
	if err != nil {
		h.logger.Error(
			"[synp-conn-event-handler] failed to decode payload",
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
			"[synp-conn-event-handler] failed to cache message",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.Any("user", conn.Session().UserInfo()),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", ErrCacheMessage, err)
	}

	if !ok {
		h.logger.Warn(
			"[synp-conn-event-handler] message duplicated, ignore it",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.String("biz_key", msg.GetBizKey()),
			zap.Any("user", conn.Session().UserInfo()),
		)
		return ErrMessageDuplicated
	}

	h.logger.Info(
		"[synp-conn-event-handler] received message from frontend",
		zap.String("step", "on_receive_from_frontend"),
		zap.String("connection_id", conn.Id()),
		zap.String("message", msg.String()),
		zap.Any("user", conn.Session().UserInfo()),
	)

	// 处理消息。
	msgHandler, ok := h.msgHandlers[msg.GetCmd()]
	if !ok {
		h.logger.Error(
			"[synp-conn-event-handler] unknown message type from frontend",
			zap.String("step", "on_receive_from_frontend"),
			zap.String("connection_id", conn.Id()),
			zap.Any("user", conn.Session().UserInfo()),
		)
		return ErrUnknownMessageType
	}

	if err = msgHandler.Handle(conn, msg); err != nil {
		// 删除消息缓存。
		if h.needUncacheMessage(err) {
			uncacheErr := h.uncacheMessage(user.Bid, msg)
			if uncacheErr != nil {
				h.logger.Error(
					"[synp-conn-event-handler] failed to uncache message",
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

func (h *EvtHandler) decodePayload(payload []byte) (*messagev1.Message, error) {
	msg := &messagev1.Message{}
	if err := h.codec.Unmarshal(payload, msg); err != nil {
		h.logger.Error(
			"[synp-conn-event-handler] failed to unmarshal message",
			zap.String("step", "decode_payload"),
			zap.String("codec_name", h.codec.Name()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%w: unknown message type", ErrInvalidMessage)
	}

	if msg.GetCmd() != commonv1.CommandType_COMMAND_TYPE_HEARTBEAT && msg.GetBizKey() == "" {
		// 非心跳消息，biz_key 不能为空。
		h.logger.Error(
			"[synp-conn-event-handler] message biz_key is empty",
			zap.String("step", "decode_payload"),
			zap.String("message", msg.String()),
		)
		return nil, fmt.Errorf("%w: empty biz_key", ErrInvalidMessage)
	}

	return msg, nil
}

func (h *EvtHandler) cacheMessage(bizId uint64, msg *messagev1.Message) (bool, error) {
	if msg.GetCmd() == commonv1.CommandType_COMMAND_TYPE_HEARTBEAT {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.cacheRequestTimeout)
	defer cancel()

	return h.rdb.SetNX(ctx, h.cacheKey(bizId, msg.GetBizKey()), msg.GetBizKey(), h.cacheExpiration).Result()
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

	if _, err := h.rdb.Del(ctx, h.cacheKey(bizId, msg.GetBizKey())).Result(); err != nil {
		return fmt.Errorf("%w: %w", ErrUncacheMessage, err)
	}

	return nil
}

func (h *EvtHandler) cacheKey(bizId uint64, bizKey string) string {
	return fmt.Sprintf("%d:%s", bizId, bizKey)
}

func (h *EvtHandler) OnReceiveFromBackend(conn synp.Conn, payload []byte) error {
	//TODO: not implemented
	panic("not implemented")
}
