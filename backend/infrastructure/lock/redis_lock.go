package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"flow/domain/distributedlock"
)

var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

// RedisLock 简化版 Redisson 风格分布式锁（SET NX + Lua 续期/释放）。
type RedisLock struct {
	client *redis.Client
	prefix string
}

// NewRedisLock 构造 Redis 锁实现。
func NewRedisLock(client *redis.Client, keyPrefix string) *RedisLock {
	if keyPrefix == "" {
		keyPrefix = "flow:lock:"
	}
	return &RedisLock{client: client, prefix: keyPrefix}
}

func (l *RedisLock) redisKey(lockType string) string {
	return l.prefix + lockType
}

func (l *RedisLock) TryAcquire(ctx context.Context, lockType, holderKey string, ttl time.Duration) (bool, error) {
	ok, err := l.client.SetNX(ctx, l.redisKey(lockType), holderKey, ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (l *RedisLock) Renew(ctx context.Context, handle distributedlock.Handle, ttl time.Duration) (bool, error) {
	ms := ttl.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	n, err := renewScript.Run(ctx, l.client, []string{l.redisKey(handle.Type)}, handle.HolderKey, ms).Int64()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (l *RedisLock) Release(ctx context.Context, handle distributedlock.Handle) error {
	_, err := releaseScript.Run(ctx, l.client, []string{l.redisKey(handle.Type)}, handle.HolderKey).Int64()
	return err
}
