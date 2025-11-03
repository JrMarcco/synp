package ws

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/internal/pkg/limiter"
	"github.com/JrMarcco/synp/internal/pkg/xmq/consumer"
	wsc "github.com/JrMarcco/synp/internal/ws/conn"
	"github.com/cenkalti/backoff/v5"
	"go.uber.org/zap"
)

type Server struct {
	config *Config

	listener net.Listener

	upgrader       synp.Upgrader
	connManager    synp.ConnManager
	connEvtHandler synp.Handler

	consumers map[string]consumer.Consumer

	connLimiter *limiter.TokenLimiter
	backoff     *backoff.ExponentialBackOff

	ctx        context.Context
	cancelFunc context.CancelFunc

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
				zap.Error(err),
			)
		}
	}()

	// 处理 upgrade 请求。
	sess, compressionState, err := s.upgrader.Upgrade(conn)
	if err != nil {
		s.logger.Error(
			"[synp-server] failed to upgrade connection from HTTP to WebSocket",
			zap.Error(err),
		)
		return
	}

	// 创建、管理连接。
	// 注意，这里的连接指的是 synp.Conn 接口，并不是 net.Conn 接口。
	synpConn, err := s.connManager.NewConn(s.ctx, conn, sess, compressionState)
	if err != nil {
		s.logger.Error(
			"[synp-server] failed to create synp connection",
			zap.Error(err),
		)
		return
	}

	user := synpConn.Session().User()
	defer func() {
		s.connManager.RemoveConn(user.ConnKey(), user.Device)
		if err := synpConn.Close(); err != nil {
			s.logger.Error(
				"[synp-server] failed to close synp connection",
				zap.String("conn_id", synpConn.Id()),
				zap.Error(err),
			)
		}
	}()

	// 处理 on connect & on disconnect 事件。
	if err := s.connEvtHandler.OnConnect(synpConn); err != nil {
		s.logger.Error(
			"[synp-server] failed to handle on connect lifecycle event",
			zap.String("conn_id", synpConn.Id()),
			zap.Error(err),
		)
		// on connect 事件失败，直接返回。
		// 注：
		//  这里最好直接返回以防止后续的事件处理发生不可预料的错误。
		return
	}
	defer func() {
		if err := s.connEvtHandler.OnDisconnect(synpConn); err != nil {
			s.logger.Error(
				"[synp-server] failed to handle on disconnect lifecycle event",
				zap.String("conn_id", synpConn.Id()),
				zap.Error(err),
			)
		}
	}()

	// 处理收发消息。
	for {
		select {
		case message, ok := <-synpConn.Receive():
			if !ok {
				return
			}
			if err := s.connEvtHandler.OnReceiveFromFrontend(synpConn, message); err != nil {
				// 处理前端（业务客户端）发送的消息失败。
				s.logger.Error(
					"[synp-server] failed to handle on receive from frontend event",
					zap.Error(err),
				)

				// 如果连接已关闭，则直接返回 ( wsc => internal/ws/conn )。
				if errors.Is(err, wsc.ErrConnClosed) {
					return
				}
			}
		case <-synpConn.Closed():
			s.logger.Info("[synp-server] synp connection has been closed")
			return
		case <-s.ctx.Done():
			s.logger.Info("[synp-server] server has been closed")
			return
		}
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
	//TODO: 优雅关闭。
	<-s.ctx.Done()
	return s.Shutdown()
}
