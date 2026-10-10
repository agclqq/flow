package distributedlock

import (
	"context"
	"time"
)

// Handle 表示一次成功加锁后的持有者信息。
type Handle struct {
	Type      string
	HolderKey string
}

// Lock 分布式锁抽象，支持数据库或 Redis 实现。
type Lock interface {
	TryAcquire(ctx context.Context, lockType, holderKey string, ttl time.Duration) (bool, error)
	Renew(ctx context.Context, handle Handle, ttl time.Duration) (bool, error)
	Release(ctx context.Context, handle Handle) error
}
