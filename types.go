package synp

import (
	"context"
	"net"

	"github.com/JrMarcco/synp/pkg/compression"
	"github.com/JrMarcco/synp/pkg/session"
	"go.uber.org/multierr"
)

//go:generate mockgen -source=./types.go -destination=./mock/synp.mock.go -package=synpmock -typed

type Server interface{}

// Upgrader 是连接升级器，用于将 HTTP 连接升级为 WebSocket 连接。
type Upgrader interface {
	Name() string
	Upgrade(conn net.Conn) (session.Session, compression.State, error)
}

// Conn 是用户连接的抽象，封装了底层的网络连接（如 WebSocket、TCP 连接）。
type Conn interface {
	Id() string
	Session() session.Session

	Send(payload []byte) error
	Receive() <-chan []byte

	Close() error
}

type ConnManager interface {
	NewConn(ctx context.Context, netConn net.Conn, sess session.Session, compressionState *compression.State) (Conn, error)
	RemoveConn(ctx context.Context, id string) bool

	FindByUser(ctx context.Context, user session.User) (Conn, bool)
}

// ConnEventHandler 是连接事件的回调接口。
type ConnEventHandler interface {
	OnConnect(conn Conn) error
	OnDisconnect(conn Conn) error

	OnReceiveFromFrontend(conn Conn, payload []byte) error
	OnPushToBackend(conn Conn, payload []byte) error
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
	//TODO: not implemented
	panic("not implemented")
}

func (w *ConnEventHandlerWrapper) OnPushToBackend(conn Conn, payload []byte) error {
	//TODO: not implemented
	panic("not implemented")
}

func NewConnEventHandlerWrapper(handlers ...ConnEventHandler) *ConnEventHandlerWrapper {
	return &ConnEventHandlerWrapper{
		handlers: handlers,
	}
}
