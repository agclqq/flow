package repository

import (
	"context"
	"time"

	"flow/domain/entity"
)

// CronFireRepository 定义定时触发抢占标记的持久化契约。
type CronFireRepository interface {
	// TryMarkFired 抢占 (templateID, fireMinute) 触发权，创建 claimed 凭据。
	// 返回凭据与是否抢占成功（多副本去重）。
	TryMarkFired(ctx context.Context, templateID, versionID int64, fireMinute time.Time) (*entity.CronFireLog, bool, error)
	// MarkFired 实例创建成功后标记凭据为 fired 并回填实例 ID。
	MarkFired(ctx context.Context, claimID, instanceID int64) error
	// ClaimStaleForRetry 认领超时的 claimed 凭据以便重试（created_at 顺延）。
	ClaimStaleForRetry(ctx context.Context, cutoff time.Time) ([]*entity.CronFireLog, error)
	// DeleteFinishedBefore 清理已终结（fired）且超过保留期的凭据。
	DeleteFinishedBefore(ctx context.Context, cutoff time.Time) error
}
