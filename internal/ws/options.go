package ws

import (
	"github.com/JrMarcco/jit/bean/option"
	"github.com/JrMarcco/synp/internal/pkg/limiter"
)

func SvrWithConnLimiter(connLimiter *limiter.TokenLimiter) option.Opt[Server] {
	return func(s *Server) {
		s.connLimiter = connLimiter
	}
}
