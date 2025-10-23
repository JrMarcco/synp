package conn

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JrMarcco/jit/bean/option"
	"github.com/JrMarcco/jit/xsync"
	"github.com/JrMarcco/synp"
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

// DeviceConns 管理单个用户的多设备连接。
// 这里不直接使用 sync.Map 是因为一个用户最多只会有 3 个设备连接。
// 相比起直接使用 sync.Map 性能更好且内存占用更低。
type DeviceConns struct {
	mu    sync.RWMutex
	conns map[session.Device]synp.Conn
}

func newDeviceConns() *DeviceConns {
	return &DeviceConns{
		// 一个用户最多只会有 3 个设备连接。
		conns: make(map[session.Device]synp.Conn, 3),
	}
}

func (dc *DeviceConns) add(device session.Device, conn synp.Conn) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.conns[device] = conn
}

func (dc *DeviceConns) remove(device session.Device) (synp.Conn, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if conn, ok := dc.conns[device]; ok {
		delete(dc.conns, device)
		return conn, true
	}
	return nil, false
}

func (dc *DeviceConns) find(device session.Device) (synp.Conn, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	conn, ok := dc.conns[device]
	return conn, ok
}

func (dc *DeviceConns) findAll() ([]synp.Conn, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	conns := make([]synp.Conn, 0, len(dc.conns))
	for _, conn := range dc.conns {
		conns = append(conns, conn)
	}
	return conns, len(conns) > 0
}

func (dc *DeviceConns) clear() int {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	cnt := len(dc.conns)
	clear(dc.conns)
	return cnt
}

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

	conns   *xsync.Map[string, *DeviceConns]
	connCnt atomic.Int64
	userCnt atomic.Int64

	logger *zap.Logger
}

func (m *ConnManager) NewConn(ctx context.Context, netConn net.Conn, sess session.Session, compressionState *compression.State) (synp.Conn, error) {
	user := sess.User()
	connKey := user.ConnKey()
	device := user.Device

	// 检查同一设备是否已有连接，如果有则关闭旧连接。
	if _, exists := m.loadExistingConn(connKey, device); exists {
		m.logger.Info(
			"[synp-conn-manager] found existing connection for same device, closing old connection",
			zap.String("conn_key", connKey),
			zap.String("device", string(device)),
			zap.Any("user", user),
		)

		// 移除旧连接。
		_ = m.RemoveConn(connKey, device)
	}

	// 创建新连接
	connId := user.ConnId()
	opts := m.convertToConnOpts(user, compressionState)
	newConn := NewConn(ctx, connId, sess, netConn, opts...)

	// 存储连接
	m.storeConn(connKey, device, newConn)
	m.logger.Info(
		"[synp-conn-manager] successfully create connection",
		zap.String("conn_id", connId),
		zap.String("device", string(device)),
		zap.Any("user", user),
	)
	return newConn, nil
}

func (m *ConnManager) loadExistingConn(connKey string, device session.Device) (synp.Conn, bool) {
	dc, ok := m.conns.Load(connKey)
	if !ok {
		return nil, false
	}

	conn, ok := dc.find(device)
	if !ok {
		return nil, false
	}
	return conn, true
}

func (m *ConnManager) storeConn(cid string, device session.Device, conn synp.Conn) {
	dc, ok := m.conns.LoadOrStore(cid, newDeviceConns())
	if !ok {
		// LoadOrStore 返回 false 代表新创建了 DeviceConns。
		// 用户数 +1。
		m.userCnt.Add(1)
	}
	dc.add(device, conn)
	m.connCnt.Add(1)

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

func (m *ConnManager) RemoveConn(connKey string, device session.Device) bool {
	dc, ok := m.conns.Load(connKey)
	if !ok {
		return false
	}

	conn, ok := dc.remove(device)
	if ok {
		defer func() {
			// 关闭连接。
			err := conn.Close()
			if err != nil {
				m.logger.Warn(
					"[synp-conn-manager] failed to close connection",
					zap.String("conn_id", conn.Id()),
					zap.Error(err),
				)
			}
		}()

		m.connCnt.Add(-1)

		if len(dc.conns) == 0 {
			m.conns.Delete(connKey)
			m.userCnt.Add(-1)
		}
	}

	return ok
}

func (m *ConnManager) RemoveUserConns(connKey string) bool {
	dc, ok := m.conns.Load(connKey)
	if !ok {
		return false
	}

	m.userCnt.Add(-1)

	cnt := dc.clear()
	m.connCnt.Add(-int64(cnt))
	return true
}

func (m *ConnManager) FindConn(connKey string, device session.Device) (synp.Conn, bool) {
	dc, ok := m.conns.Load(connKey)
	if !ok {
		return nil, false
	}

	conn, ok := dc.find(device)
	if !ok {
		return nil, false
	}
	return conn, true
}

func (m *ConnManager) FindUserConns(connKey string) ([]synp.Conn, bool) {
	dc, ok := m.conns.Load(connKey)
	if !ok {
		return nil, false
	}
	return dc.findAll()
}

func NewConnManager(cfg *ConnManagerConfig, logger *zap.Logger) *ConnManager {
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
		cfg:    cfg,
		conns:  &xsync.Map[string, *DeviceConns]{},
		logger: logger,
	}
}
