package auth

import (
	"context"

	"github.com/JrMarcco/jit/xjwt"
	"github.com/JrMarcco/synp/internal/pkg/session"
)

var _ Validator = (*JwtValidator)(nil)

type JwtValidator struct {
	jwtManager xjwt.Manager[session.User]
}

func (v *JwtValidator) Validate(_ context.Context, token string) (session.User, error) {
	claims, err := v.jwtManager.Decrypt(token)
	if err != nil {
		return session.User{}, err
	}
	return claims.Data, nil
}

func NewJwtValidator(jwtManager xjwt.Manager[session.User]) *JwtValidator {
	return &JwtValidator{jwtManager: jwtManager}
}
