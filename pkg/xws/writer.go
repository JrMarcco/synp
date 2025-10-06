package xws

import (
	"compress/flate"
	"io"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

// Writer 是对 gobwas/ws 的封装，用于写入 WebSocket 消息。
type Writer struct {
	writer *wsutil.Writer

	messageState *wsflate.MessageState
	flateWriter  *wsflate.Writer
}

func (w *Writer) Write(payload []byte) (int, error) {
	if w.messageState.IsCompressed() {
		return w.writeCompressed(payload)
	}

	return w.writeUncompressed(payload)
}

func (w *Writer) writeCompressed(payload []byte) (int, error) {
	w.flateWriter.Reset(w.writer)

	n, err := w.flateWriter.Write(payload)
	if err != nil {
		return 0, err
	}

	// 完成 flate wirter 压缩，即写入尾标记。
	err = w.flateWriter.Close()
	if err != nil {
		return 0, err
	}

	return n, w.writer.Flush()
}

func (w *Writer) writeUncompressed(payload []byte) (int, error) {
	n, err := w.writer.Write(payload)
	if err != nil {
		return 0, err
	}

	return n, w.writer.Flush()
}

func NewServerSideWriter(dst io.Writer, compressed bool) *Writer {
	messageState := wsflate.MessageState{}
	messageState.SetCompressed(compressed)

	state := ws.StateServerSide | ws.StateExtended
	opCode := ws.OpBinary

	rtn := &Writer{
		writer:       wsutil.NewWriter(dst, state, opCode),
		messageState: &messageState,
	}

	if compressed {
		rtn.flateWriter = wsflate.NewWriter(nil, func(w io.Writer) wsflate.Compressor {
			fw, _ := flate.NewWriter(w, flate.DefaultCompression)
			return fw
		})
	}

	// 注意传入的是 messageState 的指针，
	// 否则在后续的 SetExtensions 中，
	// messageState 的值不会被更新。
	rtn.writer.SetExtensions(&messageState)

	return rtn
}
