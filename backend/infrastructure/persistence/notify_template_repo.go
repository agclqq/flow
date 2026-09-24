package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
)

// NotifyTemplateRepository 是 NotifyTemplateRepository 接口的 gorm 实现。
type NotifyTemplateRepository struct {
	db *gorm.DB
}

// NewNotifyTemplateRepository 构造仓储实现。
func NewNotifyTemplateRepository(db *gorm.DB) repository.NotifyTemplateRepository {
	return &NotifyTemplateRepository{db: db}
}

func (r *NotifyTemplateRepository) Create(ctx context.Context, tpl *entity.NotifyTemplate) error {
	model := toNotifyTemplateModel(tpl)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	tpl.ID = model.ID
	return nil
}

func (r *NotifyTemplateRepository) GetByID(ctx context.Context, id int64) (*entity.NotifyTemplate, error) {
	var model NotifyTemplateModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toNotifyTemplateEntity(&model), nil
}

func (r *NotifyTemplateRepository) Update(ctx context.Context, tpl *entity.NotifyTemplate) error {
	return r.db.WithContext(ctx).Save(toNotifyTemplateModel(tpl)).Error
}

func (r *NotifyTemplateRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&NotifyTemplateModel{}, "id = ?", id).Error
}

func (r *NotifyTemplateRepository) List(ctx context.Context, offset, limit int) ([]*entity.NotifyTemplate, int64, error) {
	var models []NotifyTemplateModel
	var total int64
	if err := r.db.WithContext(ctx).Model(&NotifyTemplateModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*entity.NotifyTemplate, 0, len(models))
	for i := range models {
		items = append(items, toNotifyTemplateEntity(&models[i]))
	}
	return items, total, nil
}

func toNotifyTemplateModel(t *entity.NotifyTemplate) *NotifyTemplateModel {
	return &NotifyTemplateModel{
		ID:        t.ID,
		Name:      t.Name,
		Content:   t.Content,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toNotifyTemplateEntity(m *NotifyTemplateModel) *entity.NotifyTemplate {
	return &entity.NotifyTemplate{
		ID:        m.ID,
		Name:      m.Name,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
