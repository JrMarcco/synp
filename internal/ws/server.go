package ws

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/limiter"
	"github.com/cenkalti/backoff/v5"
	"go.uber.org/zap"
)

type Server struct {
	name   string
	config *Config

	upgrader Upgrader

	connLimiter *limiter.TokenLimiter
	backoff     *backoff.ExponentialBackOff

	ctx        context.Context
	cancelFunc context.CancelFunc

	listener net.Listener

	logger *zap.Logger
}

// Start 启动 WebSocket 服务器。
func (s *Server) Start() error {
	// 端口启用并监听 upgrade 请求，
	// 完成 WebSocket 的初始化工作。
	ln, err := net.Listen(s.config.Network, s.config.Address())
	if err != nil {
		return err
	}

	s.listener = ln

	go s.acceptConn()

	// 阻塞并等待关闭。
	<-s.ctx.Done()

	return nil
}

// acceptConn 接收 WebSocket 连接。
func (s *Server) acceptConn() {
	for {
		//TODO: 判断是否还接收新连接。

		// 接收连接前先获取令牌。
		if !s.connLimiter.Acquire() {
			next := s.backoff.NextBackOff()

			s.logger.Warn(
				"[synp-server] connection limit reached, reject new connection",
				zap.String("step", "accept_conn"),
				zap.Duration("next_backoff", next),
			)
			time.Sleep(next)
			continue
		}

		// 成功获取令牌，重置退避策略。
		s.backoff.Reset()

		conn, err := s.listener.Accept()
		if err != nil {
			// 接收连接失败，归还令牌。
			s.connLimiter.Release()

			s.logger.Error(
				"[synp-server] failed to accept connection",
				zap.String("step", "accept_conn"),
				zap.Error(err),
			)
			if errors.Is(err, net.ErrClosed) {
				// net.ErrClosed 表示 listener 已关闭，
				// 此时可以退出。
				return
			}

			var netOpErr *net.OpError
			if errors.As(err, &netOpErr) && (netOpErr.Timeout() || netOpErr.Temporary()) {
				// 如果是 timeout 或 temporary 错误，
				// 那么可以继续尝试接收连接。
				continue
			}

			// 确保循环继续而不是意外退出。
			continue
		}

		// 在追求极致性能的情况下，这里可以考虑使用连接池。
		// 在绝大多数情况下，goroutine : connection = 1 : 1 即可。
		go s.handleConn(conn)
	}
}

// handleConn 处理 WebSocket 连接。
func (s *Server) handleConn(conn net.Conn) {
	// 归还令牌。
	defer s.connLimiter.Release()

	// 关闭连接。
	defer func() {
		err := conn.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			s.logger.Warn(
				"[synp-server] failed to close connection",
				zap.String("step", "handle_conn"),
				zap.Error(err),
			)
		}
	}()

	// 处理 upgrade 请求。
	_, _, err := s.upgrader.Upgrade(conn)
	if err != nil {
		s.logger.Error(
			"[synp-server] failed to upgrade connection from HTTP to WebSocket",
			zap.String("step", "handle_conn"),
			zap.Error(err),
		)
		return
	}

}

func (s *Server) Shutdown() error {
	// 关闭限流器。
	if err := s.connLimiter.Close(); err != nil {
		s.logger.Error("[synp-server] failed to close connection limiter", zap.Error(err))
		return err
	}

	s.cancelFunc()
	return nil
}

func (s *Server) GracefulShutdown() error {

	<-s.ctx.Done()
	return s.Shutdown()
}
