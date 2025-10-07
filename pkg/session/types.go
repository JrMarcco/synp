package session

import "fmt"

//go:generate mockgen -source=./types.go -destination=./mock/session.mock.go -package=sessionmock -typed Session

type Session interface {
	UserInfo() User
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
