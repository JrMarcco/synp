package providers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type kafkaFxResult struct {
	fx.Out

	Writer        *kafka.Writer
	ReaderFactory consumer.KafkaReaderFactory
}

func newKafkaClient(zapLogger *zap.Logger, lifecycle fx.Lifecycle) (kafkaFxResult, error) {
	cfg := loadKafkaConfig()

	// 配置 TLS。
	tlsConfig, err := configureKafkaTLS(cfg.TLS, zapLogger)
	if err != nil {
		return kafkaFxResult{}, err
	}

	// 配置 SASL。
	saslMechanism, err := configureKafkaSasl(cfg.SASL, zapLogger)
	if err != nil {
		return kafkaFxResult{}, err
	}

	// 创建 Transport。
	transport := createKafkaTransport(tlsConfig, saslMechanism)

	// 创建 Writer（Producer）。
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{}, // 使用 Hash 负载均衡
		Compression:  getKafkaCompression(cfg.Producer.Compression),
		MaxAttempts:  cfg.Producer.RetryMax,
		BatchSize:    cfg.Producer.BatchSize,
		BatchTimeout: cfg.Producer.BatchTimeout,
		ReadTimeout:  cfg.Consumer.ReadTimeout, // 从 broker 读取响应的超时
		WriteTimeout: cfg.Producer.WriteTimeout,
		RequiredAcks: getKafkaRequiredAcks(cfg.Producer.RequiredAcks),
		Async:        false, // 同步模式
		Transport:    transport,
	}

	// 如果启用幂等性，设置为精确一次语义。
	if cfg.Producer.IdempotentEnabled {
		writer.RequiredAcks = kafka.RequireAll
	}

	zapLogger.Info(
		"[synp-ioc-kafka] successfully created kafka writer",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("compression", cfg.Producer.Compression),
		zap.Int("required_acks", cfg.Producer.RequiredAcks),
		zap.Bool("idempotent", cfg.Producer.IdempotentEnabled),
	)

	// 创建 ReaderFactory，用于按需创建 Reader ( Consumer )。
	const (
		defaultReadBackoffMin = 100 * time.Millisecond
		defaultReadBackoffMax = 1 * time.Second
		defaultDialTimeout    = 10 * time.Second
	)
	readerFactory := func(topic string, groupId string) *kafka.Reader {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        groupId,
			MinBytes:       cfg.Consumer.MinBytes,
			MaxBytes:       cfg.Consumer.MaxBytes,
			MaxWait:        cfg.Consumer.MaxWait,
			ReadBackoffMin: defaultReadBackoffMin,
			ReadBackoffMax: defaultReadBackoffMax,
			CommitInterval: cfg.Consumer.CommitInterval,
			StartOffset:    cfg.Consumer.StartOffset,
			Dialer: &kafka.Dialer{
				Timeout:       defaultDialTimeout,
				DualStack:     true,
				TLS:           tlsConfig,
				SASLMechanism: saslMechanism,
			},
		})

		zapLogger.Info(
			"[synp-ioc-kafka] created kafka reader",
			zap.Strings("brokers", cfg.Brokers),
			zap.String("topic", topic),
			zap.String("group_id", groupId),
		)

		return reader
	}

	// 注册生命周期钩子。
	lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			if err := writer.Close(); err != nil {
				zapLogger.Error("[synp-ioc-kafka] failed to close kafka writer", zap.Error(err))
				return fmt.Errorf("failed to close kafka writer: %w", err)
			}
			zapLogger.Info("[synp-ioc-kafka] kafka writer closed")
			return nil
		},
	})

	return kafkaFxResult{
		Writer:        writer,
		ReaderFactory: readerFactory,
	}, nil
}

type kafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`

	Producer kafkaProducerConfig `mapstructure:"producer"`
	Consumer kafkaConsumerConfig `mapstructure:"consumer"`

	TLS  kafkaTLSConfig  `mapstructure:"tls"`
	SASL kafkaSaslConfig `mapstructure:"sasl"`
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

type kafkaTLSConfig struct {
	CAFile string `mapstructure:"ca_file"`
}

type kafkaSaslConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// loadKafkaConfig 加载 Kafka 配置。
func loadKafkaConfig() *kafkaConfig {
	cfg := &kafkaConfig{}
	if err := viper.UnmarshalKey("kafka", cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal kafka config: %w", err))
	}
	return cfg
}

// configureKafkaTLS 配置 TLS。
func configureKafkaTLS(tlsCfg kafkaTLSConfig, logger *zap.Logger) (*tls.Config, error) {
	if tlsCfg.CAFile == "" {
		return nil, errors.New("CA file is required")
	}

	tlsConf := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false, // 强制 TLS 认证
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(tlsCfg.CAFile)) {
		logger.Error("[synp-ioc-kafka] failed to append CA certificate to pool for kafka")
		return nil, fmt.Errorf("failed to append CA certificate to pool for kafka")
	}
	tlsConf.RootCAs = caCertPool

	logger.Info("[synp-ioc-kafka] successfully configured TLS for kafka")

	return tlsConf, nil
}

// configureKafkaSasl 配置 SASL/SCRAM-SHA-256 认证。
func configureKafkaSasl(saslCfg kafkaSaslConfig, logger *zap.Logger) (sasl.Mechanism, error) {
	if saslCfg.Username == "" || saslCfg.Password == "" {
		return nil, errors.New("username and password are required")
	}

	mechanism, err := scram.Mechanism(scram.SHA256, saslCfg.Username, saslCfg.Password)
	if err != nil {
		logger.Error(
			"[synp-ioc-kafka] failed to create SASL mechanism",
			zap.String("username", saslCfg.Username),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create SASL mechanism: %w", err)
	}

	logger.Info(
		"[synp-ioc-kafka] successfully configured SASL/SCRAM-SHA-256 for kafka",
		zap.String("username", saslCfg.Username),
	)

	return mechanism, nil
}

// getKafkaCompression 获取压缩算法。
func getKafkaCompression(compression string) kafka.Compression {
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

// getKafkaRequiredAcks 获取 RequiredAcks 配置。
func getKafkaRequiredAcks(acks int) kafka.RequiredAcks {
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

// createKafkaTransport 创建带有 TLS 和 SASL 的 Transport。
func createKafkaTransport(tlsConfig *tls.Config, saslMechanism sasl.Mechanism) *kafka.Transport {
	transport := &kafka.Transport{
		TLS:  tlsConfig,
		SASL: saslMechanism,
	}
	return transport
}
