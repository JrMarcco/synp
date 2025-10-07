package synp

import (
	"context"
	"net"

	"github.com/JrMarcco/synp/pkg/compression"
	"github.com/JrMarcco/synp/pkg/session"
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
type ConnEventHandler interface{}
