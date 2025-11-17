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

var EtcdFxOpt = fx.Module("etcd", fx.Provide(initEtcd))

type etcdFxParams struct {
	fx.In

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func initEtcd(params etcdFxParams) *clientv3.Client {
	cfg := loadEtcdConfig()
	clientCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		Username:    cfg.Username,
		Password:    cfg.Password,
		DialTimeout: cfg.DialTimeout,
	}

	// 配置 tls。
	if cfg.TLS.Enabled {
		tlsCfg, err := configureEtcdTLS(cfg.TLS, params.Logger)
		if err != nil {
			panic(err)
		}

		clientCfg.TLS = tlsCfg
		params.Logger.Info("[synp-ioc-etcd] successfully configured TLS for etcd")
	}

	client, err := clientv3.New(clientCfg)
	if err != nil {
		params.Logger.Error("[synp-ioc-etcd] failed to create etcd client", zap.Error(err))
		panic(fmt.Errorf("failed to create etcd client: %w", err))
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		params.Logger.Error("[synp-ioc-etcd] failed to connect to etcd", zap.Error(err))

		// 连接失败直接关闭 client。
		_ = client.Close()
		panic(fmt.Errorf("[synp-ioc-etcd] failed to connect to etcd: %w", err))
	}

	params.Logger.Info("[synp-ioc-etcd] successfully connected to etcd")

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			err := client.Close()
			if err != nil {
				params.Logger.Error("[synp-ioc-etcd] failed to close etcd client", zap.Error(err))
				return err
			}

			params.Logger.Info("[synp-ioc-etcd] etcd client closed")
			return nil
		},
	})

	return client
}

type etcdConfig struct {
	Endpoints []string `mapstructure:"endpoints"`

	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`

	DialTimeout time.Duration `mapstructure:"dial_timeout"`

	TLS etcdTLSConfig `mapstructure:"tls"`
}

type etcdTLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`

	ServerName string `mapstructure:"server_name"`
}

func loadEtcdConfig() *etcdConfig {
	cfg := &etcdConfig{}
	if err := viper.UnmarshalKey("etcd", cfg); err != nil {
		panic(err)
	}
	return cfg
}

func configureEtcdTLS(tlsCfg etcdTLSConfig, logger *zap.Logger) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false, // 强制 TLS 认证
	}

	if tlsCfg.ServerName != "" {
		cfg.ServerName = tlsCfg.ServerName
	}

	// 加载 Cert 文件。
	if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			logger.Error(
				"[synp-ioc-etcd] failed to load x509 key pair for etcd",
				zap.String("cert_file", tlsCfg.CertFile),
				zap.String("key_file", tlsCfg.KeyFile),
				zap.Error(err),
			)
			return nil, fmt.Errorf("[synp-ioc-etcd] failed to load x509 key pair for etcd: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert}

		// 检查证书的公钥算法。
		if len(cert.Certificate) > 0 {
			parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return nil, fmt.Errorf("[synp-ioc-etcd] failed to parse x509 certificate for etcd: %w", err)
			}
			logger.Info(
				"[synp-ioc-etcd] successfully loaded x509 key pair for etcd",
				zap.String("public_key_algorithm", parsedCert.PublicKeyAlgorithm.String()),
				zap.String("signature_algorithm", parsedCert.SignatureAlgorithm.String()),
			)
		}
	}

	// 加载 CA 证书（用于验证服务器的证书是否可信）。
	if tlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			logger.Error(
				"[synp-ioc-etcd] failed to load CA file for etcd",
				zap.String("ca_file", tlsCfg.CAFile),
				zap.Error(err),
			)
			return nil, fmt.Errorf("[synp-ioc-etcd] failed to load CA file for etcd: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			logger.Error(
				"[synp-ioc-etcd] failed to append CA certificate to pool for etcd",
				zap.Error(err),
			)
			return nil, fmt.Errorf("[synp-ioc-etcd] failed to append CA certificate to pool for etcd: %w", err)
		}

		cfg.RootCAs = caCertPool
		logger.Info("[synp-ioc-etcd] successfully loaded CA file for etcd")
	}
	return cfg, nil
}
