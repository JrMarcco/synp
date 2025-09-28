package auth

import (
	"context"

	"github.com/JrMarcco/jit/xjwt"
	"github.com/JrMarcco/synp/internal/pkg/session"
)

var _ Validator = (*JwtValidator)(nil)

type JwtValidator struct {
	jwtManager xjwt.Manager[session.UserInfo]
}

func (v *JwtValidator) Validate(_ context.Context, token string) (session.UserInfo, error) {
	claims, err := v.jwtManager.Decrypt(token)
	if err != nil {
		return session.UserInfo{}, err
	}
	return claims.Data, nil
}

func NewJwtValidator(jwtManager xjwt.Manager[session.UserInfo]) *JwtValidator {
	return &JwtValidator{jwtManager: jwtManager}
}
