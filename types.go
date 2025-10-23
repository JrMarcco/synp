package synp

import (
	"context"
	"errors"
	"net"

	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/compression"
	"github.com/JrMarcco/synp/internal/pkg/session"
	"go.uber.org/multierr"
)

var (
	ErrRateLimited = errors.New("request too frequently, please try again later")
)

//go:generate mockgen -source=./types.go -destination=./mock/synp.mock.go -package=synpmock -typed

type Server interface{}

// Upgrader 是连接升级器，用于将 HTTP 连接升级为 WebSocket 连接。
type Upgrader interface {
	Name() string
	Upgrade(conn net.Conn) (session.Session, *compression.State, error)
}

// Conn 是用户连接的抽象，封装了底层的网络连接 ( 如 WebSocket、TCP 连接 ) 。
type Conn interface {
	Id() string
	Session() session.Session

	Send(payload []byte) error
	Receive() <-chan []byte

	UpdateActivityTime()

	Closed() <-chan struct{}

	Close() error
}

type ConnManager interface {
	NewConn(ctx context.Context, netConn net.Conn, sess session.Session, compressionState *compression.State) (Conn, error)

	RemoveConn(connKey string, device session.Device) bool
	RemoveUserConns(connKey string) bool

	FindConn(connKey string, device session.Device) (Conn, bool)
	FindUserConns(connKey string) ([]Conn, bool)
}

// ConnEventHandler 是连接事件的回调接口。
type ConnEventHandler interface {
	OnConnect(conn Conn) error
	OnDisconnect(conn Conn) error

	// OnReceiveFromFrontend 收到后端（业务服务端）消息的回调。
	OnReceiveFromFrontend(conn Conn, payload []byte) error

	// OnReceiveFromBackend 收到前端（业务客户端）消息的回调，通常用于发送消息到后端。
	OnReceiveFromBackend(conn Conn, pushMsg *messagev1.PushMessage) error
}

type ConnEventHandlerWrapper struct {
	handlers []ConnEventHandler
}

var _ ConnEventHandler = (*ConnEventHandlerWrapper)(nil)

func (w *ConnEventHandlerWrapper) OnConnect(conn Conn) error {
	var err error
	for _, handler := range w.handlers {
		err = multierr.Append(err, handler.OnConnect(conn))
	}
	return err
}

func (w *ConnEventHandlerWrapper) OnDisconnect(conn Conn) error {
	var err error
	for _, handler := range w.handlers {
		err = multierr.Append(err, handler.OnDisconnect(conn))
	}
	return err
}

func (w *ConnEventHandlerWrapper) OnReceiveFromFrontend(conn Conn, payload []byte) error {
	var err error
	for _, handler := range w.handlers {
		handleErr := handler.OnReceiveFromFrontend(conn, payload)
		if errors.Is(handleErr, ErrRateLimited) {
			// 限流直接中断。
			return nil
		}
		err = multierr.Append(err, handleErr)
	}
	return err
}

func (w *ConnEventHandlerWrapper) OnReceiveFromBackend(conn Conn, pushMsg *messagev1.PushMessage) error {
	var err error
	for _, handler := range w.handlers {
		err = multierr.Append(err, handler.OnReceiveFromBackend(conn, pushMsg))
	}
	return err
}

func NewConnEventHandlerWrapper(handlers ...ConnEventHandler) *ConnEventHandlerWrapper {
	return &ConnEventHandlerWrapper{
		handlers: handlers,
	}
}
