package auth

import (
	"context"

	"github.com/JrMarcco/synp/internal/pkg/session"
)

//go:generate mockgen -source=types.go -destination=mock/validator.mock.go -package=authmock -typed Validator

type Validator interface {
	Validate(ctx context.Context, token string) (session.User, error)
}
