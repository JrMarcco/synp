package ioc

import (
	"time"

	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/message/downstream"
	"github.com/JrMarcco/synp/internal/pkg/message/upstream"
	"github.com/JrMarcco/synp/internal/pkg/retransmit"
	"github.com/JrMarcco/synp/internal/pkg/xmq/produce"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var MessageHandlerFxOpt = fx.Module("message_handler", fx.Provide(
	// 心跳消息处理器。
	fx.Annotate(
		InitHeartbeatMsgHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 前端消息处理器。
	fx.Annotate(
		InitFrontendMsgHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 下行消息 ack 处理器。
	fx.Annotate(
		InitDownstreamAckHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 后端消息处理器。
	fx.Annotate(
		InitBackendMsgHandler,
		fx.As(new(downstream.DMsgHandler)),
	),
))

type heartbeatMsgHandlerFxParams struct {
	fx.In

	PushFunc message.PushFunc
}

func InitHeartbeatMsgHandler(params heartbeatMsgHandlerFxParams) upstream.UMsgHandler {
	return upstream.NewHeartbeatMsgHandler(params.PushFunc)
}

type frontendMsgHandlerFxParams struct {
	fx.In

	Codec    codec.Codec
	Producer produce.Producer
	PushFunc message.PushFunc
}

func InitFrontendMsgHandler(params frontendMsgHandlerFxParams) upstream.UMsgHandler {
	type config struct {
		Topic            string `mapstructure:"topic"`
		OnReceiveTimeout int    `mapstructure:"on_receive_timeout"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("synp.handler.message.frontend", &cfg); err != nil {
		panic(err)
	}

	return upstream.NewFrontendMsgHandler(
		cfg.Topic,
		time.Duration(cfg.OnReceiveTimeout)*time.Millisecond,
		params.Codec,
		params.Producer,
		params.PushFunc,
	)
}

type downstreamAckHandlerFxParams struct {
	fx.In

	RetransmitManager *retransmit.Manager
}

func InitDownstreamAckHandler(params downstreamAckHandlerFxParams) upstream.UMsgHandler {
	return upstream.NewDownstreamAckHandler(params.RetransmitManager)
}

type backendMsgHandlerFxParams struct {
	fx.In

	PushFunc          message.PushFunc
	RetransmitManager *retransmit.Manager
}

func InitBackendMsgHandler(params backendMsgHandlerFxParams) downstream.DMsgHandler {
	return downstream.NewBackendMsgHandler(params.PushFunc, params.RetransmitManager)
}
