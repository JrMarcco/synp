package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/JrMarcco/synp/internal/pkg/session"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/session_create.lua
var sessionCreateLua string

var (
	ErrSessionExists = errors.New("session exists")
	ErrSessionCreate = errors.New("failed to create session")
)

var _ session.Session = (*RedisSession)(nil)

// RedisSession 为 Session 的 Redis 实现。
type RedisSession struct {
	rdb redis.Cmdable

	key      string
	userInfo session.User
}

func (s *RedisSession) UserInfo() session.User {
	//TODO: not implemented
	panic("not implemented")
}

func (s *RedisSession) Set(ctx context.Context, key string, val string) error {
	//TODO: not implemented
	panic("not implemented")
}

func (s *RedisSession) Get(ctx context.Context, key string) (string, error) {
	//TODO: not implemented
	panic("not implemented")
}

// saveToRedis 将 Session 保存到 redis 中。
// 如果 Session 已存在，则返回 ErrSessionExists。
func (s *RedisSession) saveToRedis(ctx context.Context) error {
	args := []any{
		"sign_in_time",
		time.Now().Format(time.RFC3339Nano),
	}

	// 使用 lua 脚本保证原子性。
	res, err := s.rdb.Eval(
		ctx,
		sessionCreateLua,
		[]string{s.key},
		args...,
	).Result()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSessionCreate, err)
	}

	if res == "ok" {
		return nil
	}

	return ErrSessionExists
}

func newRedisSession(rdb redis.Cmdable, user session.User) *RedisSession {
	return &RedisSession{
		rdb:      rdb,
		key:      user.SessionKey(),
		userInfo: user,
	}
}

var _ session.Builder = (*RedisSessionBuilder)(nil)

type RedisSessionBuilder struct {
	rdb redis.Cmdable
}

func (b *RedisSessionBuilder) Build(ctx context.Context, user session.User) (session.Session, bool, error) {
	sess := newRedisSession(b.rdb, user)

	var err error
	if err = sess.saveToRedis(ctx); err == nil {
		return sess, true, nil
	}

	if errors.Is(err, ErrSessionExists) {
		return sess, false, nil
	}

	return nil, false, err
}

func NewRedisSessionBuilder(rdb redis.Cmdable) *RedisSessionBuilder {
	return &RedisSessionBuilder{rdb: rdb}
}
