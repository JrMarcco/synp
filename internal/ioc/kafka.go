package ioc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var KafkaFxOpt = fx.Module("kafka", fx.Provide(InitKafka))

type kafkaFxParams struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

type kafkaFxResult struct {
	fx.Out

	Writer        *kafka.Writer
	ReaderFactory consumer.KafkaReaderFactory
}

type kafkaTlsConfig struct {
	CAFile             string `mapstructure:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

type kafkaSaslConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type kafkaProducerConfig struct {
	// Producer 配置
	RequiredAcks      int           `mapstructure:"required_acks"`      // -1=all, 0=none, 1=leader
	Compression       string        `mapstructure:"compression"`        // none, gzip, snappy, lz4, zstd
	MaxMessageBytes   int           `mapstructure:"max_message_bytes"`  // 最大消息大小，默认 1MB
	RetryMax          int           `mapstructure:"retry_max"`          // 最大重试次数
	BatchSize         int           `mapstructure:"batch_size"`         // 批量大小
	BatchTimeout      time.Duration `mapstructure:"batch_timeout"`      // 批量超时时间（毫秒）
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`      // 写入超时（毫秒）
	IdempotentEnabled bool          `mapstructure:"idempotent_enabled"` // 是否启用幂等性

}

type kafkaConsumerConfig struct {
	// Consumer 配置
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`    // 读取超时（毫秒）
	CommitInterval time.Duration `mapstructure:"commit_interval"` // 提交间隔（毫秒）
	StartOffset    int64         `mapstructure:"start_offset"`    // 起始 offset，-1=newest, -2=oldest
	MinBytes       int           `mapstructure:"min_bytes"`       // 最小字节数
	MaxBytes       int           `mapstructure:"max_bytes"`       // 最大字节数
	MaxWait        time.Duration `mapstructure:"max_wait"`        // 最大等待时间（毫秒）
}

type kafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`

	Producer kafkaProducerConfig `mapstructure:"producer"`
	Consumer kafkaConsumerConfig `mapstructure:"consumer"`

	TLS  kafkaTlsConfig  `mapstructure:"tls"`
	SASL kafkaSaslConfig `mapstructure:"sasl"`
}

// loadKafkaConfig 加载 Kafka 配置。
func loadKafkaConfig(logger *zap.Logger) *kafkaConfig {
	// 设置默认值。
	cfg := &kafkaConfig{
		// Producer 默认值
		Producer: kafkaProducerConfig{
			RequiredAcks:      -1,                    // all
			Compression:       "snappy",              // 使用 snappy 压缩
			MaxMessageBytes:   1024000,               // 1MB
			RetryMax:          3,                     // 最多重试 3 次
			BatchSize:         100,                   // 批量大小 100
			BatchTimeout:      10 * time.Millisecond, // 批量超时 10ms
			WriteTimeout:      10 * time.Second,      // 写入超时 10s
			IdempotentEnabled: true,                  // 启用幂等性
		},
		Consumer: kafkaConsumerConfig{
			// Consumer 默认值
			ReadTimeout:    10 * time.Second,       // 读取超时 10s
			CommitInterval: 1 * time.Second,        // 每秒提交一次
			StartOffset:    -1,                     // 从最新消息开始（kafka.LastOffset）
			MinBytes:       1,                      // 最小 1 字节
			MaxBytes:       10e6,                   // 最大 10MB
			MaxWait:        500 * time.Millisecond, // 最大等待 500ms
		},
	}

	if err := viper.UnmarshalKey("kafka", cfg); err != nil {
		logger.Error("[synp-ioc] failed to unmarshal kafka config", zap.Error(err))
		panic(fmt.Errorf("failed to unmarshal kafka config: %w", err))
	}
	return cfg
}

// configureTLS 配置 TLS。
func configureTLS(tlsCfg kafkaTlsConfig, logger *zap.Logger) (*tls.Config, error) {
	if tlsCfg.CAFile == "" {
		return nil, nil
	}

	tlsConf := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify,
	}

	// 加载 CA 证书（用于验证服务器的证书是否可信）。
	caCert, err := os.ReadFile(tlsCfg.CAFile)
	if err != nil {
		logger.Error(
			"[synp-ioc] failed to load CA file for kafka",
			zap.String("ca_file", tlsCfg.CAFile),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to load CA file for kafka: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		logger.Error("[synp-ioc] failed to append CA certificate to pool for kafka")
		return nil, fmt.Errorf("failed to append CA certificate to pool for kafka")
	}

	tlsConf.RootCAs = caCertPool

	logger.Info(
		"[synp-ioc] successfully configured TLS for kafka",
		zap.String("ca_file", tlsCfg.CAFile),
	)

	return tlsConf, nil
}

// configureSASL 配置 SASL/SCRAM-SHA-256 认证。
func configureSASL(saslCfg kafkaSaslConfig, logger *zap.Logger) (sasl.Mechanism, error) {
	if saslCfg.Username == "" || saslCfg.Password == "" {
		return nil, nil
	}

	mechanism, err := scram.Mechanism(scram.SHA256, saslCfg.Username, saslCfg.Password)
	if err != nil {
		logger.Error(
			"[synp-ioc] failed to create SASL mechanism",
			zap.String("username", saslCfg.Username),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create SASL mechanism: %w", err)
	}

	logger.Info(
		"[synp-ioc] successfully configured SASL/SCRAM-SHA-256 for kafka",
		zap.String("username", saslCfg.Username),
	)

	return mechanism, nil
}

// getCompression 获取压缩算法。
func getCompression(compression string) kafka.Compression {
	switch compression {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return kafka.Compression(0) // none
	}
}

// getRequiredAcks 获取 RequiredAcks 配置。
func getRequiredAcks(acks int) kafka.RequiredAcks {
	switch acks {
	case -1:
		return kafka.RequireAll
	case 0:
		return kafka.RequireNone
	case 1:
		return kafka.RequireOne
	default:
		return kafka.RequireAll
	}
}

// createTransport 创建带有 TLS 和 SASL 的 Transport。
func createTransport(tlsConfig *tls.Config, saslMechanism sasl.Mechanism) *kafka.Transport {
	transport := &kafka.Transport{
		TLS:  tlsConfig,
		SASL: saslMechanism,
	}
	return transport
}

func InitKafka(params kafkaFxParams) kafkaFxResult {
	cfg := loadKafkaConfig(params.Logger)

	// 配置 TLS。
	tlsConfig, err := configureTLS(cfg.TLS, params.Logger)
	if err != nil {
		panic(err)
	}

	// 配置 SASL。
	saslMechanism, err := configureSASL(cfg.SASL, params.Logger)
	if err != nil {
		panic(err)
	}

	// 创建 Transport。
	transport := createTransport(tlsConfig, saslMechanism)

	// 创建 Writer（Producer）。
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{}, // 使用 Hash 负载均衡
		Compression:  getCompression(cfg.Producer.Compression),
		MaxAttempts:  cfg.Producer.RetryMax,
		BatchSize:    cfg.Producer.BatchSize,
		BatchTimeout: cfg.Producer.BatchTimeout,
		ReadTimeout:  cfg.Consumer.ReadTimeout, // 从 broker 读取响应的超时
		WriteTimeout: cfg.Producer.WriteTimeout,
		RequiredAcks: getRequiredAcks(cfg.Producer.RequiredAcks),
		Async:        false, // 同步模式
		Transport:    transport,
	}

	// 如果启用幂等性，设置为精确一次语义。
	if cfg.Producer.IdempotentEnabled {
		writer.RequiredAcks = kafka.RequireAll
	}

	params.Logger.Info(
		"[synp-ioc] successfully created kafka writer",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("compression", cfg.Producer.Compression),
		zap.Int("required_acks", cfg.Producer.RequiredAcks),
		zap.Bool("idempotent", cfg.Producer.IdempotentEnabled),
	)

	// 创建 ReaderFactory，用于按需创建 Reader ( Consumer )。
	readerFactory := func(topic string, groupId string) *kafka.Reader {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        groupId,
			MinBytes:       cfg.Consumer.MinBytes,
			MaxBytes:       cfg.Consumer.MaxBytes,
			MaxWait:        cfg.Consumer.MaxWait,
			ReadBackoffMin: 100 * time.Millisecond,
			ReadBackoffMax: 1 * time.Second,
			CommitInterval: cfg.Consumer.CommitInterval,
			StartOffset:    cfg.Consumer.StartOffset,
			Dialer: &kafka.Dialer{
				Timeout:       10 * time.Second,
				DualStack:     true,
				TLS:           tlsConfig,
				SASLMechanism: saslMechanism,
			},
		})

		params.Logger.Info(
			"[synp-ioc] created kafka reader",
			zap.String("topic", topic),
			zap.String("group_id", groupId),
			zap.Strings("brokers", cfg.Brokers),
		)

		return reader
	}

	// 注册生命周期钩子。
	params.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := writer.Close(); err != nil {
				params.Logger.Error("[synp-ioc] failed to close kafka writer", zap.Error(err))
				return fmt.Errorf("failed to close kafka writer: %w", err)
			}
			params.Logger.Info("[synp-ioc] kafka writer closed")
			return nil
		},
	})

	return kafkaFxResult{
		Writer:        writer,
		ReaderFactory: readerFactory,
	}
}
