package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowNotifyBindingRepository 是 FlowNotifyBindingRepository 接口的 gorm 实现。
type FlowNotifyBindingRepository struct {
	db *gorm.DB
}

// NewFlowNotifyBindingRepository 构造仓储实现。
func NewFlowNotifyBindingRepository(db *gorm.DB) repository.FlowNotifyBindingRepository {
	return &FlowNotifyBindingRepository{db: db}
}

func (r *FlowNotifyBindingRepository) Create(ctx context.Context, binding *entity.FlowNotifyBinding) error {
	model := toNotifyBindingModel(binding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	binding.ID = model.ID
	return nil
}

func (r *FlowNotifyBindingRepository) GetByID(ctx context.Context, id int64) (*entity.FlowNotifyBinding, error) {
	var model FlowNotifyBindingModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toNotifyBindingEntity(&model), nil
}

func (r *FlowNotifyBindingRepository) Update(ctx context.Context, binding *entity.FlowNotifyBinding) error {
	return r.db.WithContext(ctx).Save(toNotifyBindingModel(binding)).Error
}

func (r *FlowNotifyBindingRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FlowNotifyBindingModel{}, "id = ?", id).Error
}

func (r *FlowNotifyBindingRepository) ListByTemplate(ctx context.Context, templateID int64) ([]*entity.FlowNotifyBinding, error) {
	var models []FlowNotifyBindingModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("id ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	bindings := make([]*entity.FlowNotifyBinding, 0, len(models))
	for i := range models {
		bindings = append(bindings, toNotifyBindingEntity(&models[i]))
	}
	return bindings, nil
}

func (r *FlowNotifyBindingRepository) ReplaceByTemplate(ctx context.Context, templateID int64, bindings []*entity.FlowNotifyBinding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", templateID).Delete(&FlowNotifyBindingModel{}).Error; err != nil {
			return err
		}
		for _, b := range bindings {
			b.TemplateID = templateID
			model := toNotifyBindingModel(b)
			model.ID = 0
			if err := tx.Create(model).Error; err != nil {
				return err
			}
			b.ID = model.ID
		}
		return nil
	})
}

func toNotifyBindingModel(b *entity.FlowNotifyBinding) *FlowNotifyBindingModel {
	timingsJSON, _ := json.Marshal(b.Timings)
	return &FlowNotifyBindingModel{
		ID:               b.ID,
		TemplateID:       b.TemplateID,
		NodeID:           b.NodeID,
		NotifyTemplateID: b.NotifyTemplateID,
		Timings:          string(timingsJSON),
		ExtraContent:     b.ExtraContent,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
	}
}

func toNotifyBindingEntity(m *FlowNotifyBindingModel) *entity.FlowNotifyBinding {
	var timings []valueobject.NotifyTiming
	if m.Timings != "" {
		var raw []string
		_ = json.Unmarshal([]byte(m.Timings), &raw)
		for _, t := range raw {
			timings = append(timings, valueobject.NotifyTiming(t))
		}
	}
	if timings == nil {
		timings = []valueobject.NotifyTiming{}
	}
	return &entity.FlowNotifyBinding{
		ID:               m.ID,
		TemplateID:       m.TemplateID,
		NodeID:           m.NodeID,
		NotifyTemplateID: m.NotifyTemplateID,
		Timings:          timings,
		ExtraContent:     m.ExtraContent,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
