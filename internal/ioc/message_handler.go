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
		initHeartbeatMsgHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 前端消息处理器。
	fx.Annotate(
		initFrontendMsgHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 下行消息 ack 处理器。
	fx.Annotate(
		initDownstreamAckHandler,
		fx.As(new(upstream.UMsgHandler)),
		fx.ResultTags(`group:"upstream_message_handler"`),
	),

	// 后端消息处理器。
	fx.Annotate(
		initBackendMsgHandler,
		fx.As(new(downstream.DMsgHandler)),
	),
))

type heartbeatMsgHandlerFxParams struct {
	fx.In

	PushFunc message.PushFunc
}

func initHeartbeatMsgHandler(params heartbeatMsgHandlerFxParams) *upstream.HeartbeatMsgHandler {
	return upstream.NewHeartbeatMsgHandler(params.PushFunc)
}

type frontendMsgHandlerFxParams struct {
	fx.In

	Codec    codec.Codec
	Producer produce.Producer
	PushFunc message.PushFunc
}

func initFrontendMsgHandler(params frontendMsgHandlerFxParams) *upstream.FrontendMsgHandler {
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

func initDownstreamAckHandler(params downstreamAckHandlerFxParams) *upstream.DownstreamAckHandler {
	return upstream.NewDownstreamAckHandler(params.RetransmitManager)
}

type backendMsgHandlerFxParams struct {
	fx.In

	PushFunc          message.PushFunc
	RetransmitManager *retransmit.Manager
}

func initBackendMsgHandler(params backendMsgHandlerFxParams) *downstream.BackendMsgHandler {
	return downstream.NewBackendMsgHandler(params.PushFunc, params.RetransmitManager)
}
