package upstream

import (
	"github.com/JrMarcco/synp"
	commonv1 "github.com/JrMarcco/synp-api/api/go/common/v1"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
)

//go:generate mockgen -source=./types.go -destination=./mock/upstream.mock.go -package=upstreammock -typed Handler

// UMsgHandler 是 upstream 消息处理器的接口。
type UMsgHandler interface {
	// Handle 处理消息。
	// 注：
	//	Handle 方法没有 context.Context 参数是因为消息处理通常在 synp.Conn 的上下文中进行。
	Handle(conn synp.Conn, msg *messagev1.Message) error

	// CmdType 返回消息类型。
	CmdType() commonv1.CommandType
}
