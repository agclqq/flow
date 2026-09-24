package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"flow/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CronFireRepository 是定时触发抢占标记的持久化实现。
type CronFireRepository struct {
	db *gorm.DB
}

// NewCronFireRepository 构造仓储实现。
func NewCronFireRepository(db *gorm.DB) *CronFireRepository {
	return &CronFireRepository{db: db}
}

func toCronFireModel(l *entity.CronFireLog) *CronFireLogModel {
	return &CronFireLogModel{
		TemplateID: l.TemplateID,
		VersionID:  l.VersionID,
		FireMinute: l.FireMinute,
		Status:     l.Status,
		InstanceID: l.InstanceID,
		CreatedAt:  l.CreatedAt,
	}
}

func toCronFireEntity(m *CronFireLogModel) *entity.CronFireLog {
	return &entity.CronFireLog{
		ID:         m.ID,
		TemplateID: m.TemplateID,
		VersionID:  m.VersionID,
		FireMinute: m.FireMinute,
		Status:     m.Status,
		InstanceID: m.InstanceID,
		CreatedAt:  m.CreatedAt,
	}
}

// TryMarkFired 抢占 (templateID, fireMinute) 的触发权，创建 status=claimed 的凭据行。
// 返回凭据与是否抢占成功；false 表示已被本副本或其他副本抢占。
func (r *CronFireRepository) TryMarkFired(
	ctx context.Context,
	templateID, versionID int64,
	fireMinute time.Time,
) (*entity.CronFireLog, bool, error) {
	claim := &entity.CronFireLog{
		TemplateID: templateID,
		VersionID:  versionID,
		FireMinute: fireMinute,
		Status:     entity.CronFireStatusClaimed,
		CreatedAt:  time.Now(),
	}
	m := toCronFireModel(claim)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		// 唯一索引冲突 = 已被抢占。兼容 gorm 错误翻译与驱动原始错误两种情况。
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key value") {
			return nil, false, nil
		}
		return nil, false, err
	}
	claim.ID = m.ID
	return claim, true, nil
}

// MarkFired 实例创建成功后把凭据标记为 fired 并回填实例 ID。
func (r *CronFireRepository) MarkFired(ctx context.Context, claimID, instanceID int64) error {
	return r.db.WithContext(ctx).Model(&CronFireLogModel{}).
		Where("id = ?", claimID).
		Updates(map[string]any{"status": entity.CronFireStatusFired, "instance_id": instanceID}).Error
}

// ClaimStaleForRetry 认领超时的 claimed 凭据以便重试：
// FOR UPDATE SKIP LOCKED 保证多副本不重复认领；同时把 created_at 顺延到当前，
// 让其他副本在一个重试周期内不会再看到它。
// 超过最大重试次数（MaxCronClaimRetries）的凭据标记为 exhausted，不再重试。
func (r *CronFireRepository) ClaimStaleForRetry(ctx context.Context, cutoff time.Time) ([]*entity.CronFireLog, error) {
	var retry []*entity.CronFireLog
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []CronFireLogModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND created_at < ?", entity.CronFireStatusClaimed, cutoff).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		now := time.Now()
		exhaustedIDs := make([]int64, 0)
		retryIDs := make([]int64, 0)
		for i := range rows {
			if rows[i].RetryCount >= entity.MaxCronClaimRetries {
				exhaustedIDs = append(exhaustedIDs, rows[i].ID)
				continue
			}
			retryIDs = append(retryIDs, rows[i].ID)
			rows[i].RetryCount++
			rows[i].CreatedAt = now
			retry = append(retry, toCronFireEntity(&rows[i]))
		}
		if len(exhaustedIDs) > 0 {
			if err := tx.Model(&CronFireLogModel{}).Where("id IN ?", exhaustedIDs).
				Update("status", entity.CronFireStatusExhausted).Error; err != nil {
				return err
			}
		}
		if len(retryIDs) > 0 {
			if err := tx.Model(&CronFireLogModel{}).Where("id IN ?", retryIDs).
				Updates(map[string]any{"created_at": now, "retry_count": gorm.Expr("retry_count + 1")}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return retry, err
}

// DeleteFinishedBefore 清理已终结（fired / exhausted）且超过保留期的凭据。
// 只清理终态：claimed 的行由重试循环负责，永不清理（直到成功或重试耗尽）。
func (r *CronFireRepository) DeleteFinishedBefore(ctx context.Context, cutoff time.Time) error {
	return r.db.WithContext(ctx).
		Where("status IN ? AND created_at < ?", []string{
			string(entity.CronFireStatusFired), string(entity.CronFireStatusExhausted),
		}, cutoff).
		Delete(&CronFireLogModel{}).Error
}
