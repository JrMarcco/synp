package ws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/pkg/auth"
	"github.com/JrMarcco/synp/pkg/compression"
	"github.com/JrMarcco/synp/pkg/session"
	"github.com/go-redis/redis"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"go.uber.org/zap"
)

var (
	ErrTokenRequired = errors.New("token is required")
	ErrInvalidUri    = errors.New("invalid uri")
	ErrInvalidToken  = errors.New("invalid token")
)

var _ synp.Upgrader = (*Upgrader)(nil)

type Upgrader struct {
	rdb redis.Cmdable

	validator         auth.Validator
	compressionConfig compression.Config

	logger *zap.Logger
}

func (u *Upgrader) Name() string {
	return "synp.upgrader"
}

func (u *Upgrader) Upgrade(conn net.Conn) (session.Session, compression.State, error) {
	var ext *wsflate.Extension
	if u.compressionConfig.Enabled {
		// 启用压缩时，创建压缩扩展。
		params := u.compressionConfig.ToParamters()
		ext = &wsflate.Extension{Parameters: params}

		u.logger.Info("[synp-upgrader] compression enabled", zap.Any("params", params))
	}

	var user session.User
	var sess session.Session
	var autoClose bool
	upgrader := ws.Upgrader{
		// 协商过程，这里主要是压缩相关的协商（是否启用以及压缩算法）。
		Negotiate: func(opt httphead.Option) (httphead.Option, error) {
			if ext != nil {
				return ext.Negotiate(opt)
			}
			return httphead.Option{}, nil
		},
		OnRequest: func(uri []byte) error {
			// 验证 token 并提取用户信息。
			var err error
			if user, err = u.extractUserInfo(uri); err != nil {
				return err
			}
			return nil
		},
		OnHeader: func(key, value []byte) error {
			// 解析 auto close 参数。
			if strings.EqualFold(string(key), "x-auto-close") {
				autoClose = string(value) == "true"

				u.logger.Warn(
					"[synp-upgrader] auto close parameter parsed",
					zap.String("header_key", string(key)),
					zap.String("header_value", string(value)),
					zap.Bool("auto_close", autoClose),
				)
			}
			return nil
		},
		OnBeforeUpgrade: func() (header ws.HandshakeHeader, err error) {
			// 设置 auto close 参数。
			user.AutoClose = autoClose

			//TODO: 初始化 session。

			return ws.HandshakeHeaderString(""), nil
		},
	}

	state := compression.State{
		Enabled: false,
	}

	if _, err := upgrader.Upgrade(conn); err != nil {
		return nil, compression.State{}, err
	}

	// 检查协商压缩的结果。
	if ext != nil {
		if params, accepted := ext.Accepted(); accepted {
			state.Enabled = true
			state.Ext = ext
			state.Params = params

			u.logger.Info(
				"[synp-upgrader] successfully negotiated compression",
				zap.Any("negotiated_params", params),
			)
			return sess, state, nil
		}

		u.logger.Warn("[synp-upgrader] failed to negotiate compression, downgrade to no compression")
	}

	return sess, state, nil
}

// extractToken 从 URI 中提取 token。
func (u *Upgrader) extractToken(uri []byte) (string, error) {
	parsedURL, err := url.Parse(string(uri))
	if err != nil {
		return "", ErrInvalidUri
	}

	token := parsedURL.Query().Get("token")
	if token == "" {
		return "", ErrTokenRequired
	}
	return token, nil
}

// getUserInfo 从 URI 中获取用户信息。
func (u *Upgrader) extractUserInfo(uri []byte) (session.User, error) {
	token, err := u.extractToken(uri)
	if err != nil {
		u.logger.Error("[synp-upgrader] failed to extract token from uri", zap.Error(err))
		return session.User{}, err
	}

	var user session.User
	user, err = u.validator.Validate(context.Background(), token)
	if err != nil {
		u.logger.Error("[synp-upgrader] failed to validate token", zap.Error(err))
		return session.User{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	return user, nil
}
