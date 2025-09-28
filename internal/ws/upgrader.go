package ws

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/JrMarcco/synp/internal/pkg/auth"
	"github.com/JrMarcco/synp/internal/pkg/compression"
	"github.com/JrMarcco/synp/internal/pkg/session"
	"github.com/go-redis/redis"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"go.uber.org/zap"
)

var (
	ErrTokenRequired = errors.New("token is required")
)

// Upgrader 是 WebSocket 升级器，用于将 HTTP 请求升级为 WebSocket 连接。
type Upgrader struct {
	rdb redis.Cmdable

	validator         auth.Validator
	compressionConfig compression.Config

	logger *zap.Logger
}

func (u *Upgrader) Upgrade(conn net.Conn) (session.Session, *compression.State, error) {
	var err error

	var ext *wsflate.Extension
	if u.compressionConfig.Enabled {
		// 启用压缩时，创建压缩扩展。
		params := u.compressionConfig.ToParamters()
		ext = &wsflate.Extension{Parameters: params}

		u.logger.Info("[synp] compression enabled", zap.Any("params", params))
	}

	// var ui session.UserInfo
	var sess session.Session
	upgrader := ws.Upgrader{
		Negotiate: func(opt httphead.Option) (httphead.Option, error) {
			return httphead.Option{}, nil
		},
		OnRequest: func(uri []byte) error {
			if _, err = u.extractUserInfo(uri); err != nil {
				return err
			}
			return nil
		},
		OnHeader: func(key, value []byte) error {
			return nil
		},
		OnBeforeUpgrade: func() (header ws.HandshakeHeader, err error) {
			return ws.HandshakeHeaderString(""), nil
		},
	}

	if _, err = upgrader.Upgrade(conn); err != nil {
		return nil, nil, err
	}

	// 检查协商压缩的结果。
	var state *compression.State
	if ext != nil {
		if params, accepted := ext.Accepted(); accepted {
			state = &compression.State{
				Enabled: true,
				Ext:     ext,
				Params:  params,
			}

			u.logger.Info("[synp] successfully negotiated compression", zap.Any("negotiated_params", params))
			return sess, state, nil
		}

		state = &compression.State{Enabled: false}
		u.logger.Warn("[synp] failed to negotiate compression, downgrade to no compression")
	}

	return sess, state, nil
}

// extractToken 从 URI 中提取 token。
func (u *Upgrader) extractToken(uri []byte) (string, error) {
	parsedURL, err := url.Parse(string(uri))
	if err != nil {
		return "", err
	}

	params := parsedURL.Query()
	token := params.Get("token")
	if token == "" {
		return "", ErrTokenRequired
	}
	return token, nil
}

// getUserInfo 从 URI 中获取用户信息。
func (u *Upgrader) extractUserInfo(uri []byte) (session.UserInfo, error) {
	token, err := u.extractToken(uri)
	if err != nil {
		u.logger.Error("[synp] failed to extract token from uri", zap.Error(err))
		return session.UserInfo{}, err
	}

	var ui session.UserInfo
	ui, err = u.validator.Validate(context.Background(), token)
	if err != nil {
		u.logger.Error("[synp] failed to validate token", zap.Error(err))
		return session.UserInfo{}, err
	}
	return ui, nil
}
