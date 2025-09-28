package limiter

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type TokenLimiterConfig struct {
	InitCapacity     int64         `yaml:"init_capacity"`
	MaxCapacity      int64         `yaml:"max_capacity"`
	IncreaseStep     int64         `yaml:"increase_step"`
	IncreaseInterval time.Duration `yaml:"increase_interval"`
}

// TokenLimiter 令牌桶算法限流器。
type TokenLimiter struct {
	cfg TokenLimiterConfig

	tokens       chan struct{}
	currCapacity atomic.Int64

	ctx        context.Context
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

// Start 开启一个 goroutine，逐步增加桶的容量。
func (l *TokenLimiter) Start(ctx context.Context) {
	ticker := time.NewTicker(l.cfg.IncreaseInterval)
	defer ticker.Stop()

	l.logger.Info("[synp-limiter] start capacity increasing goroutine")

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("[synp-limiter] outter context canceled, stop capacity increasing goroutine")
			return

		case <-l.ctx.Done():
			l.logger.Info("[synp-limiter] inner context canceled, stop capacity increasing goroutine")
			return

		case <-ticker.C:
			curr := l.currCapacity.Load()
			if curr >= l.cfg.MaxCapacity {
				l.logger.Info("[synp-limiter] capacity reached max, stop capacity increasing goroutine")
				return
			}

			newCap := min(curr+l.cfg.IncreaseStep, l.cfg.MaxCapacity)

			for range newCap - curr {
				l.tokens <- struct{}{}
			}
			l.currCapacity.Store(newCap)

			l.logger.Info(
				"[synp-limiter] capacity increased",
				zap.Int64("from", curr),
				zap.Int64("to", newCap),
				zap.Int64("increase_step", l.cfg.IncreaseStep),
			)
		}
	}
}

// Acquire 尝试获取令牌，成功获取返回 true。
// 注：
//
//	这是一个非阻塞操作，如果当前没有可用令牌会立即返回 false。
func (l *TokenLimiter) Acquire() bool {
	select {
	case <-l.tokens:
		return true
	default:
		return false
	}
}

// Release 释放（归还）令牌，同样是一个非阻塞操作。
func (l *TokenLimiter) Release() bool {
	select {
	case l.tokens <- struct{}{}:
		return true
	default:
		// 理论上不会执行到这里。
		// 执行到这里通常意味着 Release 的调用次数超过 Acquire 的调用次数。
		l.logger.Warn("[synp-limiter] failed to release token, bucket is full")
		return false
	}
}

func (l *TokenLimiter) Close() error {
	l.cancelFunc()
	l.logger.Info("[synp-limiter] token limiter closed")
	return nil
}

func (l *TokenLimiter) Cap() int64 {
	return l.currCapacity.Load()
}
