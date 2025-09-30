package xws

import (
	"net"

	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

type Reader struct {
	conn net.Conn

	reader *wsutil.Reader

	messageState *wsflate.MessageState
	flateReader  *wsflate.Reader

	handlerFunc wsutil.FrameHandlerFunc
}
