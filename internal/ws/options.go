package ws

import (
	"github.com/jrmarcco/jit/bean/option"
	"github.com/jrmarcco/synp/internal/pkg/limiter"
)

func SvrWithConnLimiter(connLimiter *limiter.TokenLimiter) option.Opt[Server] {
	return func(s *Server) {
		s.connLimiter = connLimiter
	}
}
