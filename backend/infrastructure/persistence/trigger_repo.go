package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
)

// TriggerConfigRepository 是 TriggerConfigRepository 接口的 gorm 实现。
type TriggerConfigRepository struct {
	db *gorm.DB
}

// NewTriggerConfigRepository 构造仓储实现。
func NewTriggerConfigRepository(db *gorm.DB) repository.TriggerConfigRepository {
	return &TriggerConfigRepository{db: db}
}

func (r *TriggerConfigRepository) Create(ctx context.Context, cfg *entity.TriggerConfig) error {
	model := toTriggerModel(cfg)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	cfg.ID = model.ID
	return nil
}

func (r *TriggerConfigRepository) GetByTemplateID(ctx context.Context, templateID int64) (*entity.TriggerConfig, error) {
	var model TriggerConfigModel
	if err := r.db.WithContext(ctx).Where("template_id = ?", templateID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toTriggerEntity(&model), nil
}

func (r *TriggerConfigRepository) Update(ctx context.Context, cfg *entity.TriggerConfig) error {
	return r.db.WithContext(ctx).Save(toTriggerModel(cfg)).Error
}

func (r *TriggerConfigRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&TriggerConfigModel{}, "id = ?", id).Error
}

// --- 转换函数 ---

func toTriggerModel(c *entity.TriggerConfig) *TriggerConfigModel {
	return &TriggerConfigModel{
		ID:            c.ID,
		TemplateID:    c.TemplateID,
		Manual:        c.Manual,
		Webhook:       c.Webhook,
		Cron:          c.Cron,
		CronExpr:      c.CronExpr,
		RequireParams: c.RequireParams,
		WebhookSecret: c.WebhookSecret,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func toTriggerEntity(m *TriggerConfigModel) *entity.TriggerConfig {
	return &entity.TriggerConfig{
		ID:            m.ID,
		TemplateID:    m.TemplateID,
		Manual:        m.Manual,
		Webhook:       m.Webhook,
		Cron:          m.Cron,
		CronExpr:      m.CronExpr,
		RequireParams: m.RequireParams,
		WebhookSecret: m.WebhookSecret,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
