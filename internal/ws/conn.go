package ws

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/JrMarcco/jit/bean/option"
	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/pkg/compression"
	"github.com/JrMarcco/synp/pkg/session"
	"github.com/JrMarcco/synp/pkg/xws"
	"go.uber.org/zap"
)

var (
	ErrConnClosed = errors.New("connection closed")
)

var _ synp.Conn = (*Conn)(nil)

// Conn 代表一个 WebSocket 连接。
// Conn 封装了 net.Conn，只负责消息的读写。
type Conn struct {
	id   string
	sess session.Session

	netConn net.Conn

	reader      *xws.Reader
	readTimeout time.Duration

	writer       *xws.Writer
	writeTimeout time.Duration

	compressionState compression.State

	// 重试策略
	initRetryInterval time.Duration
	maxRetryInterval  time.Duration
	maxRetryCount     int32

	// 通信通道
	sendChan    chan []byte
	receiveChan chan []byte

	ctx        context.Context
	cancelFunc context.CancelFunc

	closeOnce sync.Once

	logger *zap.Logger
}

func (c *Conn) Id() string {
	return c.id
}

func (c *Conn) Session() session.Session {
	return c.sess
}

func (c *Conn) Send(payload []byte) error {
	//TODO: not implemented
	panic("not implemented")
}

func (c *Conn) Receive() <-chan []byte {
	//TODO: not implemented
	panic("not implemented")
}

func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		//TODO: not implemented
		panic("not implemented")
	})
	return err
}

func (c *Conn) sendLoop() {
	defer func() {
		_ = c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case payload, ok := <-c.sendChan:
			if !ok {
				return
			}

			if !c.trySend(payload) {
				// 发送失败，关闭连接。
				return
			}
		}
	}
}

// trySend 是实际发送消息给客户端的逻辑。
// 在发送失败时，会根据配置使用指数退避策略进行重试，最终重试失败才会返回 false。
// 注意：
//
//	只允许在发生超时时进行重试。
func (c *Conn) trySend(payload []byte) bool {
	//TODO: not implemented
	panic("not implemented")
}

func (c *Conn) receiveLoop() {
	defer func() {
		close(c.receiveChan)
		_ = c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 控制读数据的超时时间。
		// 注意：
		//  这里传入 0 值会禁用超时控制。
		_ = c.netConn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
}

func NewConn(
	parentCtx context.Context,
	id string,
	sess session.Session,
	netConn net.Conn,
	opts ...option.Opt[Conn],
) *Conn {
	ctx, cancel := context.WithCancel(parentCtx)

	c := &Conn{
		id:      id,
		sess:    sess,
		netConn: netConn,

		ctx:        ctx,
		cancelFunc: cancel,
	}

	option.Apply(c, opts...)

	// 启动收发数据的 goroutine。
	go c.sendLoop()
	go c.receiveLoop()

	return c
}
