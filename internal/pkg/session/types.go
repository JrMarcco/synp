package session

import (
	"context"
	"fmt"
)

//go:generate mockgen -source=./types.go -destination=./mock/session.mock.go -package=sessionmock -typed Session

type Session interface {
	User() User

	Set(ctx context.Context, key, val string) error
	Get(ctx context.Context, key string) (string, error)

	Destroy(ctx context.Context) error
}

// Builder 为 Session 的构建器。
type Builder interface {
	// Build 新建一个 Session 或 返回一个已存在的 Session。
	// bool 参数表示返回的 Session 是否是新建的。
	Build(ctx context.Context, user User) (Session, bool, error)
}

// Device 设备类型
type Device string

const (
	DeviceMobile  Device = "mobile"
	DeviceTablet  Device = "tablet"
	DevicePC      Device = "pc"
	DeviceUnknown Device = "unknown"
)

// User 为 Session 的用户信息。
type User struct {
	BID       uint64 `json:"bid"`
	UID       uint64 `json:"uid"`
	Device    Device `json:"device"`    // 设备类型：mobile/tablet/pc
	AutoClose bool   `json:"autoClose"` // 空闲时是否自动关闭连接
}

func (u *User) ConnID() string {
	return fmt.Sprintf("%d:%d:%s", u.BID, u.UID, u.Device)
}

func (u *User) ConnKey() string {
	return fmt.Sprintf("%d:%d", u.BID, u.UID)
}

func (u *User) SessionKey() string {
	return fmt.Sprintf("synp:session:%d:%d", u.BID, u.UID)
}
