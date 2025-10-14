package conn

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/JrMarcco/jit/bean/option"
	"github.com/JrMarcco/jit/xsync"
	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/JrMarcco/synp/internal/pkg/compression"
	"github.com/JrMarcco/synp/internal/pkg/session"
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
	RateLimit    int
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
	user := sess.UserInfo()

	cid := user.UniqueId()
	opts := m.convertToConnOpts(sess.UserInfo(), compressionState)

	c := NewConn(ctx, cid, sess, netConn, opts...)

	m.conns.Store(cid, c)
	m.logger.Info(
		"[synp-conn-manager] successfully create connection",
		zap.String("connection_id", cid),
		zap.Any("user", user),
	)

	m.len.Add(1)
	return c, nil
}

func (m *ConnManager) convertToConnOpts(user session.User, compressionState *compression.State) []option.Opt[Conn] {
	var opts []option.Opt[Conn]

	if compressionState != nil {
		opts = append(opts, ConnWithCompression(compressionState))
	}

	if m.cfg.ReadTimeout > 0 {
		opts = append(opts, ConnWithReadTimeout(m.cfg.ReadTimeout))
	}
	if m.cfg.WriteTimeout > 0 {
		opts = append(opts, ConnWithWriteTimeout(m.cfg.WriteTimeout))
	}

	if m.cfg.SendBufferSize > 0 {
		opts = append(opts, ConnWithWriteBuffer(m.cfg.SendBufferSize))
	}
	if m.cfg.ReceiveBufferSize > 0 {
		opts = append(opts, ConnWithReadBuffer(m.cfg.ReceiveBufferSize))
	}

	if m.cfg.InitRetryInterval > 0 && m.cfg.MaxRetryInterval > 0 && m.cfg.MaxRetryCount > 0 {
		opts = append(opts, ConnWithRetry(m.cfg.InitRetryInterval, m.cfg.MaxRetryInterval, m.cfg.MaxRetryCount))
	}

	opts = append(
		opts,
		ConnWithAutoClose(user.AutoClose),
		ConnWithRateLimit(m.cfg.RateLimit),
	)

	return opts
}

func (m *ConnManager) RemoveConn(id string) bool {
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

func (m *ConnManager) FindByUser(user session.User) (synp.Conn, bool) {
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
