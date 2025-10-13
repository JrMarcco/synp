package ioc

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
	"github.com/xdg-go/scram"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var KafkaFxOpt = fx.Module("kafka", fx.Provide(InitKafka))

var SHA256 scram.HashGeneratorFcn = sha256.New

var _ sarama.SCRAMClient = (*scramClient)(nil)

type scramClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (sc *scramClient) Begin(username, password, authzID string) error {
	var err error
	sc.Client, err = sc.HashGeneratorFcn.NewClient(username, password, authzID)
	if err != nil {
		return err
	}

	sc.ClientConversation = sc.Client.NewConversation()
	return nil
}

func (sc *scramClient) Step(challenge string) (string, error) {
	resp, err := sc.ClientConversation.Step(challenge)
	if err != nil {
		return "", err
	}

	return resp, nil
}

func (sc *scramClient) Done() bool {
	return sc.ClientConversation.Done()
}

type kafkaFxParams struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func InitKafka(params kafkaFxParams) sarama.SyncProducer {
	type tlsConfig struct {
		CAFile             string `mapstructure:"ca_file"`
		InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	}

	type saslConfig struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}

	type config struct {
		Brokers []string `mapstructure:"brokers"`

		// Producer 配置
		RequiredAcks      int16         `mapstructure:"required_acks"`      // 0=NoResponse, 1=WaitForLocal, -1=WaitForAll
		Compression       string        `mapstructure:"compression"`        // none, gzip, snappy, lz4, zstd
		MaxMessageBytes   int           `mapstructure:"max_message_bytes"`  // 最大消息大小，默认 1MB
		RetryMax          int           `mapstructure:"retry_max"`          // 最大重试次数
		RetryBackoff      time.Duration `mapstructure:"retry_backoff"`      // 重试间隔（毫秒）
		Timeout           time.Duration `mapstructure:"timeout"`            // 超时时间（毫秒）
		IdempotentEnabled bool          `mapstructure:"idempotent_enabled"` // 是否启用幂等性
		MaxOpenRequests   int           `mapstructure:"max_open_requests"`  // 最大并发请求数

		TLS  tlsConfig  `mapstructure:"tls"`
		SASL saslConfig `mapstructure:"sasl"`
	}

	cfg := config{
		// 设置默认值
		RequiredAcks:      -1,                     // WaitForAll
		Compression:       "snappy",               // 使用 snappy 压缩
		MaxMessageBytes:   1024000,                // 1MB
		RetryMax:          3,                      // 最多重试 3 次
		RetryBackoff:      100 * time.Millisecond, // 重试间隔 100ms
		Timeout:           30 * time.Second,       // 超时 30s
		IdempotentEnabled: true,                   // 启用幂等性
		MaxOpenRequests:   8,                      // 最大并发请求数，当 idempotent_enabled=false 时生效
	}

	if err := viper.UnmarshalKey("kafka", &cfg); err != nil {
		params.Logger.Error("[synp-ioc] failed to unmarshal kafka config", zap.Error(err))
		panic(fmt.Errorf("failed to unmarshal kafka config: %w", err))
	}

	// 创建 Kafka 配置
	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version = sarama.V4_1_0_0 // 设置 Kafka 版本

	// Producer 配置
	kafkaCfg.Producer.RequiredAcks = sarama.RequiredAcks(cfg.RequiredAcks)
	kafkaCfg.Producer.Return.Successes = true
	kafkaCfg.Producer.Return.Errors = true
	kafkaCfg.Producer.Retry.Max = cfg.RetryMax
	kafkaCfg.Producer.Retry.Backoff = time.Duration(cfg.RetryBackoff) * time.Millisecond
	kafkaCfg.Producer.Timeout = time.Duration(cfg.Timeout) * time.Millisecond

	if cfg.MaxMessageBytes > 0 {
		kafkaCfg.Producer.MaxMessageBytes = cfg.MaxMessageBytes
	}

	if cfg.IdempotentEnabled {
		kafkaCfg.Producer.Idempotent = true
		kafkaCfg.Producer.RequiredAcks = sarama.WaitForAll
		kafkaCfg.Net.MaxOpenRequests = 1
	} else if cfg.MaxOpenRequests > 0 {
		kafkaCfg.Net.MaxOpenRequests = cfg.MaxOpenRequests
	}

	// 配置压缩
	switch cfg.Compression {
	case "gzip":
		kafkaCfg.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		kafkaCfg.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		kafkaCfg.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		kafkaCfg.Producer.Compression = sarama.CompressionZSTD
	default:
		kafkaCfg.Producer.Compression = sarama.CompressionNone
	}

	// 配置 tls。
	if cfg.TLS.CAFile != "" {
		tlsCfg := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}

		// 加载 CA 证书（用于验证服务器的证书是否可信）。
		caCert, err := os.ReadFile(cfg.TLS.CAFile)
		if err != nil {
			params.Logger.Error(
				"[synp-ioc] failed to load CA file for kafka",
				zap.String("ca_file", cfg.TLS.CAFile),
				zap.Error(err),
			)
			panic(fmt.Errorf("failed to load CA file for kafka: %w", err))
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			params.Logger.Error("[synp-ioc] failed to append CA certificate to pool for kafka")
			panic(fmt.Errorf("failed to append CA certificate to pool for kafka"))
		}

		tlsCfg.RootCAs = caCertPool

		kafkaCfg.Net.TLS.Enable = true
		kafkaCfg.Net.TLS.Config = tlsCfg

		params.Logger.Info(
			"[synp-ioc] successfully configured TLS for kafka",
			zap.String("ca_file", cfg.TLS.CAFile),
		)
	}

	// 配置 SASL/SCRAM-SHA-256 认证。
	if cfg.SASL.Username != "" && cfg.SASL.Password != "" {
		kafkaCfg.Net.SASL.Enable = true
		kafkaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		kafkaCfg.Net.SASL.User = cfg.SASL.Username
		kafkaCfg.Net.SASL.Password = cfg.SASL.Password
		kafkaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &scramClient{HashGeneratorFcn: SHA256}
		}

		params.Logger.Info(
			"[synp-ioc] successfully configured SASL/SCRAM-SHA-256 for kafka",
			zap.String("username", cfg.SASL.Username),
		)
	}

	// 创建 SyncProducer。
	producer, err := sarama.NewSyncProducer(cfg.Brokers, kafkaCfg)
	if err != nil {
		params.Logger.Error(
			"[synp-ioc] failed to create kafka producer",
			zap.Strings("brokers", cfg.Brokers),
			zap.Error(err),
		)
		panic(fmt.Errorf("failed to create kafka producer: %w", err))
	}

	params.Logger.Info(
		"[synp-ioc] successfully created kafka producer",
		zap.Strings("brokers", cfg.Brokers),
	)

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := producer.Close(); err != nil {
				params.Logger.Error("[synp-ioc] failed to close kafka producer", zap.Error(err))
				return fmt.Errorf("failed to close kafka producer: %w", err)
			}

			params.Logger.Info("[synp-ioc] kafka producer closed")
			return nil
		},
	})

	return producer
}
