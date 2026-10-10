package persistence

import "time"

// DistributedLockModel 分布式锁表，每种 lock_type 唯一一行。
type DistributedLockModel struct {
	ID        int64     `gorm:"primaryKey"`
	LockType  string    `gorm:"type:varchar(64);uniqueIndex;not null;comment:'锁类型，全局唯一'"`
	HolderKey string    `gorm:"type:varchar(128);not null;default:'';comment:'持有者标识'"`
	ExpiresAt time.Time `gorm:"not null;comment:'锁过期时间（以数据库时间为准）'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (DistributedLockModel) TableName() string {
	return "distributed_locks"
}
