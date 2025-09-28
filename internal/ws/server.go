package ws

import (
	"context"
	"errors"
	"net"

	"go.uber.org/zap"
)

type WebSocketServer struct {
	name   string
	config *WebSocketConfig

	upgrader Upgrader

	ctx    context.Context
	cancel context.CancelFunc

	listener net.Listener

	logger *zap.Logger
}

// Start 启动 WebSocket 服务器。
func (s *WebSocketServer) Start() error {
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
func (s *WebSocketServer) acceptConn() {
	for {
		//TODO: 判断是否还接收新连接。

		//TODO: 接收连接前先获取令牌。

		conn, err := s.listener.Accept()
		if err != nil {
			//TODO: 接收连接失败，归还令牌。

			s.logger.Error(
				"[synp] failed to accept connection",
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

			// 这里要确保循环继续而不是意外退出。
			continue
		}

		// 在追求极致性能的情况下，这里可以考虑使用连接池。
		// 在绝大多数情况下，goroutine : connection = 1 : 1 即可。
		go s.handleConn(conn)
	}
}

// handleConn 处理 WebSocket 连接。
func (s *WebSocketServer) handleConn(conn net.Conn) {
	//TODO: 归还令牌。

	// 关闭连接。
	defer func() {
		err := conn.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			s.logger.Warn(
				"[synp] failed to close connection",
				zap.String("step", "handle_conn"),
				zap.Error(err),
			)
		}
	}()

	// 处理 upgrade 请求。
	_, _, err := s.upgrader.Upgrade(conn)
	if err != nil {
		s.logger.Error(
			"[synp] failed to upgrade connection",
			zap.String("step", "handle_conn"),
			zap.Error(err),
		)
		return
	}

}
