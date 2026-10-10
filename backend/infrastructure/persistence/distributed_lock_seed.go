package persistence

import (
	"time"

	"gorm.io/gorm"
)

const LockTypeCronSchedulerLeader = "cron_scheduler_leader"

// SeedDistributedLocks 初始化锁表固定行（每种 lock_type 一行）。
func SeedDistributedLocks(db *gorm.DB) error {
	types := []string{LockTypeCronSchedulerLeader}
	for _, lockType := range types {
		var count int64
		if err := db.Model(&DistributedLockModel{}).Where("lock_type = ?", lockType).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&DistributedLockModel{
			LockType:  lockType,
			HolderKey: "",
			ExpiresAt: time.Unix(0, 0).UTC(),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
