package lock

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"flow/domain/distributedlock"
	"flow/infrastructure/config"
)

// NewFromConfig 按配置创建分布式锁实现。
func NewFromConfig(cfg config.LockConfig, db *gorm.DB) (distributedlock.Lock, *redis.Client, error) {
	switch cfg.Provider {
	case "redis":
		if cfg.Redis.Addr == "" {
			return nil, nil, fmt.Errorf("lock.provider=redis but lock.redis.addr is empty")
		}
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		return NewRedisLock(client, cfg.Redis.KeyPrefix), client, nil
	default:
		return NewDBLock(db), nil, nil
	}
}
