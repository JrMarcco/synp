package ws

import (
	"sync/atomic"
	"time"

	"github.com/JrMarcco/jit/xsync"
	"github.com/JrMarcco/synp"
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

	DefaultCloseTimeout  = time.Second
	DefaultUserRateLimit = 10
)

type ConnManagerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	InitRetryInterval time.Duration
	MaxRetryInterval  time.Duration
	MaxRetryCount     int32

	SendBufferSize    int
	ReceiveBufferSize int

	CloseTimeout  time.Duration
	UserRateLimit int64
}

var _ synp.ConnManager = (*ConnManager)(nil)

type ConnManager struct {
	cfg *ConnManagerConfig

	conns *xsync.Map[string, synp.Conn]
	len   atomic.Int64

	logger *zap.Logger
}
