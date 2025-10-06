package xws

import (
	"compress/flate"
	"io"
	"net"

	"github.com/gobwas/ws"
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

func NewServerSideReader(conn net.Conn) *Reader {
	messageState := &wsflate.MessageState{}
	handlerFunc := wsutil.ControlFrameHandler(conn, ws.StateServerSide)

	return &Reader{
		conn: conn,
		reader: &wsutil.Reader{
			Source:         conn,
			State:          ws.StateServerSide | ws.StateExtended,
			Extensions:     []wsutil.RecvExtension{messageState},
			OnIntermediate: handlerFunc,
		},
		messageState: messageState,
		flateReader: wsflate.NewReader(nil, func(r io.Reader) wsflate.Decompressor {
			return flate.NewReader(r)
		}),
		handlerFunc: handlerFunc,
	}
}

func NewClientSideReader(conn net.Conn) *Reader {
	messageState := &wsflate.MessageState{}
	handlerFunc := wsutil.ControlFrameHandler(conn, ws.StateClientSide)

	return &Reader{
		conn: conn,
		reader: &wsutil.Reader{
			Source:         conn,
			State:          ws.StateClientSide | ws.StateExtended,
			Extensions:     []wsutil.RecvExtension{messageState},
			OnIntermediate: handlerFunc,
		},
		messageState: messageState,
		flateReader: wsflate.NewReader(nil, func(r io.Reader) wsflate.Decompressor {
			return flate.NewReader(r)
		}),
		handlerFunc: handlerFunc,
	}
}
