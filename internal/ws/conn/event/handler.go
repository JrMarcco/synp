package event

import (
	"github.com/JrMarcco/synp"
	"go.uber.org/zap"
)

var _ synp.ConnEventHandler = (*Handler)(nil)

type Handler struct {
	logger *zap.Logger
}

func (h *Handler) OnConnect(conn synp.Conn) error {
	//TODO: not implemented
	panic("not implemented")
}

func (h *Handler) OnDisconnect(conn synp.Conn) error {
	//TODO: not implemented
	panic("not implemented")
}

func (h *Handler) OnReceiveFromFrontend(conn synp.Conn, payload []byte) error {
	//TODO: not implemented
	panic("not implemented")
}

func (h *Handler) OnPushToBackend(conn synp.Conn, payload []byte) error {
	//TODO: not implemented
	panic("not implemented")
}
