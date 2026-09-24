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

// FlowTemplateRepository 是 FlowTemplateRepository 接口的 gorm 实现。
type FlowTemplateRepository struct {
	db *gorm.DB
}

// NewFlowTemplateRepository 构造仓储实现。
func NewFlowTemplateRepository(db *gorm.DB) repository.FlowTemplateRepository {
	return &FlowTemplateRepository{db: db}
}

// Create 持久化一个流水线模板。
func (r *FlowTemplateRepository) Create(ctx context.Context, tpl *entity.FlowTemplate) error {
	model := toModel(tpl)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	tpl.ID = model.ID
	return nil
}

// GetByID 按主键查询模板，软删除记录不会被返回。
func (r *FlowTemplateRepository) GetByID(ctx context.Context, id int64) (*entity.FlowTemplate, error) {
	var model FlowTemplateModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toEntity(&model), nil
}

// Update 更新模板。
func (r *FlowTemplateRepository) Update(ctx context.Context, tpl *entity.FlowTemplate) error {
	return r.db.WithContext(ctx).Save(toModel(tpl)).Error
}

// Delete 执行软删除。
func (r *FlowTemplateRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FlowTemplateModel{}, "id = ?", id).Error
}

// List 分页查询模板，返回记录与总数。
func (r *FlowTemplateRepository) List(ctx context.Context, offset, limit int) ([]*entity.FlowTemplate, int64, error) {
	var models []FlowTemplateModel
	var total int64
	if err := r.db.WithContext(ctx).Model(&FlowTemplateModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	tpls := make([]*entity.FlowTemplate, 0, len(models))
	for i := range models {
		tpls = append(tpls, toEntity(&models[i]))
	}
	return tpls, total, nil
}

func toModel(tpl *entity.FlowTemplate) *FlowTemplateModel {
	ownersJSON, _ := json.Marshal(tpl.Owners)
	m := &FlowTemplateModel{
		ID:             tpl.ID,
		Name:           tpl.Name,
		Owners:         string(ownersJSON),
		Description:    tpl.Description,
		CurrentVersion: tpl.Version,
		Status:         tpl.Status.Int(),
		CreatedAt:      tpl.CreatedAt,
		UpdatedAt:      tpl.UpdatedAt,
	}
	if tpl.DeletedAt != nil {
		m.DeletedAt = gorm.DeletedAt{Time: *tpl.DeletedAt, Valid: true}
	}
	return m
}

func toEntity(m *FlowTemplateModel) *entity.FlowTemplate {
	var owners []string
	if m.Owners != "" {
		_ = json.Unmarshal([]byte(m.Owners), &owners)
	}
	if owners == nil {
		owners = []string{}
	}
	tpl := &entity.FlowTemplate{
		ID:          m.ID,
		Name:        m.Name,
		Owners:      owners,
		Description: m.Description,
		Version:     m.CurrentVersion,
		Status:    valueobject.TemplateStatusFromInt(m.Status),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		tpl.DeletedAt = &t
	}
	return tpl
}
