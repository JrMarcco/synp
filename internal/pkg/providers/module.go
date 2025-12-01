package providers

import (
	"github.com/JrMarcco/synp/internal/pkg/auth"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"github.com/JrMarcco/synp/internal/pkg/message/downstream"
	"github.com/JrMarcco/synp/internal/pkg/message/upstream"
	pkgconsumer "github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/JrMarcco/synp/internal/pkg/xmq/produce"
	"go.uber.org/fx"
)

var (
	ZapLoggerFxModule       = fx.Module("zap-logger", fx.Provide(newLogger))
	RedisFxModule           = fx.Module("redis", fx.Provide(newRedisCmdable))
	KafkaFxModule           = fx.Module("kafka", fx.Provide(newKafkaClient))
	CodecFxModule           = fx.Module("codec", fx.Provide(newCodec))
	MessagePushFuncFxModule = fx.Module("message-push-func", fx.Provide(message.DefaultPushFunc))
	RetransmitFxModule      = fx.Module("retransmit", fx.Provide(newRetransmitManager))
)

var (
	KafkaProducerFxModule = fx.Module(
		"kafka-producer",
		fx.Provide(
			fx.Annotate(
				newKafkaProducer,
				fx.As(new(produce.Producer)),
			),
		),
	)

	KafkaConsumerFxModule = fx.Module(
		"kafka-consumer",
		fx.Provide(
			fx.Annotate(
				pkgconsumer.NewKafkaConsumerFactory,
				fx.As(new(pkgconsumer.ConsumerFactory)),
			),
			newKafkaConsumers,
		),
	)

	ValidatorFxModule = fx.Module(
		"validator",
		fx.Provide(
			fx.Annotate(
				newValidator,
				fx.As(new(auth.Validator)),
			),
		),
	)

	MessageHandlerFxModule = fx.Module(
		"message-handler",
		fx.Provide(
			// 心跳消息处理器。
			fx.Annotate(
				upstream.NewHeartbeatMsgHandler,
				fx.As(new(upstream.UMsgHandler)),
				fx.ResultTags(`group:"upstream-message-handler"`),
			),

			// 前端消息处理器。
			fx.Annotate(
				newFrontendMsgHandler,
				fx.As(new(upstream.UMsgHandler)),
				fx.ResultTags(`group:"upstream-message-handler"`),
			),

			// 下行消息 ack 处理器。
			fx.Annotate(
				upstream.NewDownstreamAckHandler,
				fx.As(new(upstream.UMsgHandler)),
				fx.ResultTags(`group:"upstream-message-handler"`),
			),

			// 后端消息处理器。
			fx.Annotate(
				newBackendMsgHandler,
				fx.As(new(downstream.DMsgHandler)),
			),
		))
)
