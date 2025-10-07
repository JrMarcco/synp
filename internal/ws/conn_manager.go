package ws

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/JrMarcco/jit/xsync"
	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/pkg/codec"
	"github.com/JrMarcco/synp/pkg/compression"
	"github.com/JrMarcco/synp/pkg/session"
	"go.uber.org/zap"
)

const (
	// 默认读写超时时间
	DefaultReadTiemout  = 15 * time.Second
	DefaultWriteTiemout = 10 * time.Second

	// 默认缓冲大小
	DefaultSendBufferSize    = 256
	DefaultReceiveBufferSize = 256

	// 默认重试策略
	DefaultInitRetryInterval = 1 * time.Second
	DefaultMaxRetryInterval  = 5 * time.Second
	DefaultMaxRetryCount     = 3

	DefaultCloseTimeout = time.Second
	DefaultRateLimit    = 10
)

type ConnManagerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	InitRetryInterval time.Duration
	MaxRetryInterval  time.Duration
	MaxRetryCount     int32

	SendBufferSize    int
	ReceiveBufferSize int

	CloseTimeout time.Duration
	RateLimit    int64
}

var _ synp.ConnManager = (*ConnManager)(nil)

type ConnManager struct {
	cfg *ConnManagerConfig

	conns *xsync.Map[string, synp.Conn]
	len   atomic.Int64

	codec codec.Codec

	logger *zap.Logger
}

func (m *ConnManager) NewConn(ctx context.Context, netConn net.Conn, sess session.Session, compressionState *compression.State) (synp.Conn, error) {
	//TODO: not implemented
	panic("not implemented")
}

func (m *ConnManager) RemoveConn(_ context.Context, id string) bool {
	_, ok := m.conns.LoadAndDelete(id)
	if ok {
		m.logger.Info(
			"[synp-conn-manager] successfully remove connection",
			zap.String("connection_id", id),
		)
		m.len.Add(-1)
	}
	return ok
}

func (m *ConnManager) FindByUser(_ context.Context, user session.User) (synp.Conn, bool) {
	return m.conns.Load(user.UniqueId())
}

func NewConnManager(codec codec.Codec, cfg *ConnManagerConfig, logger *zap.Logger) *ConnManager {
	if cfg == nil {
		cfg = &ConnManagerConfig{
			ReadTimeout:       DefaultReadTiemout,
			WriteTimeout:      DefaultWriteTiemout,
			InitRetryInterval: DefaultInitRetryInterval,
			MaxRetryInterval:  DefaultMaxRetryInterval,
			MaxRetryCount:     DefaultMaxRetryCount,
			SendBufferSize:    DefaultSendBufferSize,
			ReceiveBufferSize: DefaultReceiveBufferSize,
			CloseTimeout:      DefaultCloseTimeout,
			RateLimit:         DefaultRateLimit,
		}
	}

	return &ConnManager{
		conns:  &xsync.Map[string, synp.Conn]{},
		codec:  codec,
		cfg:    cfg,
		logger: logger,
	}
}
