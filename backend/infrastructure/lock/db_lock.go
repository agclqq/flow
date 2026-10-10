package lock

import (
	"context"
	"time"

	"gorm.io/gorm"

	"flow/domain/distributedlock"
)

// DBLock 使用 distributed_locks 表实现分布式锁。
type DBLock struct {
	db *gorm.DB
}

// NewDBLock 构造数据库锁实现。
func NewDBLock(db *gorm.DB) *DBLock {
	return &DBLock{db: db}
}

func (l *DBLock) TryAcquire(ctx context.Context, lockType, holderKey string, ttl time.Duration) (bool, error) {
	secs := int64(ttl.Seconds())
	if secs < 1 {
		secs = 1
	}
	result := l.db.WithContext(ctx).Exec(`
		UPDATE distributed_locks
		SET holder_key = ?, expires_at = NOW() + (? * INTERVAL '1 second'), updated_at = NOW()
		WHERE lock_type = ? AND expires_at < NOW()
	`, holderKey, secs, lockType)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (l *DBLock) Renew(ctx context.Context, handle distributedlock.Handle, ttl time.Duration) (bool, error) {
	secs := int64(ttl.Seconds())
	if secs < 1 {
		secs = 1
	}
	result := l.db.WithContext(ctx).Exec(`
		UPDATE distributed_locks
		SET expires_at = NOW() + (? * INTERVAL '1 second'), updated_at = NOW()
		WHERE lock_type = ? AND holder_key = ? AND expires_at > NOW()
	`, secs, handle.Type, handle.HolderKey)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (l *DBLock) Release(ctx context.Context, handle distributedlock.Handle) error {
	result := l.db.WithContext(ctx).Exec(`
		UPDATE distributed_locks
		SET expires_at = NOW(), updated_at = NOW()
		WHERE lock_type = ? AND holder_key = ?
	`, handle.Type, handle.HolderKey)
	return result.Error
}
