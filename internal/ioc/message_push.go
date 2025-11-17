package ioc

import (
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"go.uber.org/fx"
)

var MessagePushFxOpt = fx.Module("message_push", fx.Provide(initMessagePushFunc))

type messagePushFuncFxParams struct {
	fx.In

	Codec codec.Codec
}

func initMessagePushFunc(params messagePushFuncFxParams) message.PushFunc {
	return message.DefaultPushFunc(params.Codec)
}
