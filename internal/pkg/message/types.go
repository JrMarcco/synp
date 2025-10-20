package message

import (
	"errors"

	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"go.uber.org/zap"
)

var (
	ErrMarshalMessage = errors.New("failed to marshal message")
)

type MessagePushFunc func(conn synp.Conn, msg *messagev1.Message) error

// DefaultMessagePushFunc 创建默认推送消息到前端 ( 业务客户端 ) 的函数的默认实现，用于将结构化消息通过连接发送。
// 该函数将消息编码后通过 Conn.Send 发送，适用于 retransmit.Manager 的 taskFunc 参数。
//
// 参数：
//   - codec: 消息编解码器
//   - logger: 日志记录器
//
// 返回：
//   - retransmit.TaskFunc: 可用于发送消息和重传的函数
func DefaultMessagePushFunc(codec codec.Codec, logger *zap.Logger) MessagePushFunc {
	return func(conn synp.Conn, msg *messagev1.Message) error {
		payload, err := codec.Marshal(msg)
		if err != nil {
			logger.Error(
				"[synp] failed to marshal message",
				zap.String("codec_name", codec.Name()),
				zap.String("message", msg.String()),
				zap.Error(err),
			)
			return err
		}

		if err = conn.Send(payload); err != nil {
			logger.Error(
				"[synp] failed to send message",
				zap.String("connection_id", conn.Id()),
				zap.Error(err),
			)
			return err
		}
		return nil
	}
}
