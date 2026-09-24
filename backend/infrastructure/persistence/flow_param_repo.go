package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowParamRepository 是 FlowParamRepository 接口的 gorm 实现。
type FlowParamRepository struct {
	db *gorm.DB
}

// NewFlowParamRepository 构造仓储实现。
func NewFlowParamRepository(db *gorm.DB) repository.FlowParamRepository {
	return &FlowParamRepository{db: db}
}

func (r *FlowParamRepository) Create(ctx context.Context, param *entity.FlowParam) error {
	model := toParamModel(param)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	param.ID = model.ID
	return nil
}

func (r *FlowParamRepository) GetByID(ctx context.Context, id int64) (*entity.FlowParam, error) {
	var model FlowParamModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toParamEntity(&model), nil
}

func (r *FlowParamRepository) Update(ctx context.Context, param *entity.FlowParam) error {
	return r.db.WithContext(ctx).Save(toParamModel(param)).Error
}

func (r *FlowParamRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FlowParamModel{}, "id = ?", id).Error
}

func (r *FlowParamRepository) ListByVersion(ctx context.Context, templateID, versionID int64) ([]*entity.FlowParam, error) {
	var models []FlowParamModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND version_id = ?", templateID, versionID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	params := make([]*entity.FlowParam, 0, len(models))
	for i := range models {
		params = append(params, toParamEntity(&models[i]))
	}
	return params, nil
}

// --- 转换函数 ---

func toParamModel(p *entity.FlowParam) *FlowParamModel {
	return &FlowParamModel{
		ID:           p.ID,
		TemplateID:   p.TemplateID,
		VersionID:    p.VersionID,
		Name:         p.Name,
		Type:         p.Type.Int(),
		DefaultValue: p.DefaultValue,
		Required:     p.Required,
		Description:  p.Description,
	}
}

func toParamEntity(m *FlowParamModel) *entity.FlowParam {
	return &entity.FlowParam{
		ID:           m.ID,
		TemplateID:   m.TemplateID,
		VersionID:    m.VersionID,
		Name:         m.Name,
		Type:         valueobject.ParamTypeFromInt(m.Type),
		DefaultValue: m.DefaultValue,
		Required:     m.Required,
		Description:  m.Description,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
