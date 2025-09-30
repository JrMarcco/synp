package xws

import (
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

type Writer struct {
	writer *wsutil.Writer

	messageState *wsflate.MessageState
	flateWriter  *wsflate.Writer
}
