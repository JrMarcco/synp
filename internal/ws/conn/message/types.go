package message

import (
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
)

//go:generate mockgen -source=./types.go -destination=./mock/message.mock.go -package=messagemock -typed Handler

// MsgHandler 是消息处理器的接口。
type MsgHandler interface {
	// Handle 处理消息。
	// 注：
	//	Handle 方法没有 context.Context 参数是因为消息处理通常在 synp.Conn 的上下文中进行。
	Handle(conn synp.Conn, msg *messagev1.Message) error
}
