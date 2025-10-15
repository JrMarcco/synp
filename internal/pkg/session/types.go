package session

import (
	"context"
	"fmt"
)

//go:generate mockgen -source=./types.go -destination=./mock/session.mock.go -package=sessionmock -typed Session

type Session interface {
	UserInfo() User

	Set(ctx context.Context, key string, val string) error
	Get(ctx context.Context, key string) (string, error)
}

// Builder 为 Session 的构建器。
type Builder interface {
	// Build 新建一个 Session 或 返回一个已存在的 Session。
	// bool 参数表示返回的 Session 是否是新建的。
	Build(ctx context.Context, user User) (Session, bool, error)
}

// User 为 Session 的用户信息。
type User struct {
	Bid       uint64 `json:"bid"`
	Uid       uint64 `json:"uid"`
	AutoClose bool   `json:"auto_close"` // 空闲时是否自动关闭连接
}

func (u *User) UniqueId() string {
	return fmt.Sprintf("%d:%d", u.Bid, u.Uid)
}

func (u *User) SessionKey() string {
	return fmt.Sprintf("synp:session:%d:%d", u.Bid, u.Uid)
}
