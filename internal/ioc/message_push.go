package ioc

import (
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var MessagePushFxOpt = fx.Module("message_push", fx.Provide(InitMessagePush))

type messagePushFuncFxParams struct {
	fx.In

	Codec  codec.Codec
	Logger *zap.Logger
}

func InitMessagePush(params messagePushFuncFxParams) message.MessagePushFunc {
	return message.DefaultMessagePushFunc(params.Codec, params.Logger)
}
