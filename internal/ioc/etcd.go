package ioc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type etcdParams struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func InitEtcd(params etcdParams) *clientv3.Client {
	type tlsConfig struct {
		Enabled  bool   `mapstructure:"enabled"`
		CertFile string `mapstructure:"cert_file"`
		KeyFile  string `mapstructure:"key_file"`
		CAFile   string `mapstructure:"ca_file"`

		ServerName         string `mapstructure:"server_name"`
		InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	}

	type config struct {
		Endpoints []string `mapstructure:"endpoints"`

		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`

		DialTimeout time.Duration `mapstructure:"dial_timeout"`

		TLS tlsConfig `mapstructure:"tls"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("etcd", &cfg); err != nil {
		panic(err)
	}

	clientCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		Username:    cfg.Username,
		Password:    cfg.Password,
		DialTimeout: time.Duration(cfg.DialTimeout) * time.Millisecond,
	}

	// 配置 tls。
	if cfg.TLS.Enabled {
		tlsCfg := &tls.Config{
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}

		if cfg.TLS.ServerName != "" {
			tlsCfg.ServerName = cfg.TLS.ServerName
		}

		// 加载 Cert 文件。
		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				params.Logger.Error(
					"failed to load x509 key pair for etcd",
					zap.String("cert_file", cfg.TLS.CertFile),
					zap.String("key_file", cfg.TLS.KeyFile),
					zap.Error(err),
				)
				panic(fmt.Errorf("failed to load x509 key pair for etcd: %w", err))
			}

			tlsCfg.Certificates = []tls.Certificate{cert}

			// 检查证书的公钥算法。
			if len(cert.Certificate) > 0 {
				parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
				if err == nil {
					params.Logger.Info(
						"successfully loaded x509 key pair for etcd",
						zap.String("public_key_algorithm", parsedCert.PublicKeyAlgorithm.String()),
						zap.String("signature_algorithm", parsedCert.SignatureAlgorithm.String()),
					)
				}
			}
		}

		// 加载 CA 文件。
		if cfg.TLS.CAFile != "" {
			caCert, err := os.ReadFile(cfg.TLS.CAFile)
			if err != nil {
				params.Logger.Error(
					"failed to load CA file for etcd",
					zap.String("ca_file", cfg.TLS.CAFile),
					zap.Error(err),
				)
				panic(fmt.Errorf("failed to load CA file for etcd: %w", err))
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				params.Logger.Error(
					"failed to append CA certificate to pool for etcd",
					zap.Error(err),
				)
				panic(fmt.Errorf("failed to append CA certificate to pool for etcd: %w", err))
			}

			tlsCfg.RootCAs = caCertPool
			params.Logger.Info("successfully loaded CA file for etcd")
		}

		clientCfg.TLS = tlsCfg
		params.Logger.Info("successfully configured TLS for etcd")
	}

	client, err := clientv3.New(clientCfg)
	if err != nil {
		params.Logger.Error("failed to create etcd client", zap.Error(err))
		panic(fmt.Errorf("failed to create etcd client: %w", err))
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DialTimeout)*time.Millisecond)
	defer cancel()

	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		params.Logger.Error("failed to connect to etcd", zap.Error(err))

		// 连接失败直接关闭 client。
		_ = client.Close()
		panic(fmt.Errorf("failed to connect to etcd: %w", err))
	}

	params.Logger.Info("successfully connected to etcd")

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			err := client.Close()
			if err != nil {
				params.Logger.Error("failed to close etcd client", zap.Error(err))
				return err
			}

			params.Logger.Info("etcd client closed")
			return nil
		},
	})

	return client
}
