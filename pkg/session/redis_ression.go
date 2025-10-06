package session

import (
	"github.com/go-redis/redis"
)

var _ Session = (*RedisSession)(nil)

// RedisSession 为 Session 的 Redis 实现。
type RedisSession struct {
	rdb redis.Cmdable

	key      string
	userInfo User
}

func (s *RedisSession) UserInfo() User {
	//TODO: not implemented
	panic("not implemented")
}
