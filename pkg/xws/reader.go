package xws

import (
	"io"
	"net"

	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

// Reader 是对 gobwas/ws 的封装，用于读取 WebSocket 消息。
type Reader struct {
	conn net.Conn

	reader *wsutil.Reader

	messageState *wsflate.MessageState
	flateReader  *wsflate.Reader

	handlerFunc wsutil.FrameHandlerFunc
}

func (r *Reader) Read() ([]byte, error) {
	for {
		header, err := r.reader.NextFrame()
		if err != nil {
			return nil, err
		}

		if header.OpCode.IsControl() {
			if err := r.handlerFunc(header, r.reader); err != nil {
				return nil, err
			}
			continue
		}

		if r.messageState.IsCompressed() {
			r.flateReader.Reset(r.reader)
			return io.ReadAll(r.flateReader)
		}

		return io.ReadAll(r.reader)
	}
}

func NewServerSideReader(conn net.Conn, compressed bool) *Reader {
	// not implemented
	panic("not implemented")
}

func NewClientSideReader(conn net.Conn, compressed bool) *Reader {
	// not implemented
	panic("not implemented")
}
