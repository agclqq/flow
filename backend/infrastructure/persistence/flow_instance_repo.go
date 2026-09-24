package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowInstanceRepository 是 FlowInstanceRepository 接口的 gorm 实现。
type FlowInstanceRepository struct {
	db *gorm.DB
}

// NewFlowInstanceRepository 构造仓储实现。
func NewFlowInstanceRepository(db *gorm.DB) repository.FlowInstanceRepository {
	return &FlowInstanceRepository{db: db}
}

func (r *FlowInstanceRepository) Create(ctx context.Context, inst *entity.FlowInstance) error {
	model := toInstanceModel(inst)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	inst.ID = model.ID
	return nil
}

func (r *FlowInstanceRepository) GetByID(ctx context.Context, id int64) (*entity.FlowInstance, error) {
	var model FlowInstanceModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toInstanceEntity(&model), nil
}

func (r *FlowInstanceRepository) Update(ctx context.Context, inst *entity.FlowInstance) error {
	return r.db.WithContext(ctx).Save(toInstanceModel(inst)).Error
}

func (r *FlowInstanceRepository) ListByTemplate(ctx context.Context, templateID int64, offset, limit int) ([]*entity.FlowInstance, int64, error) {
	var models []FlowInstanceModel
	var total int64
	q := r.db.WithContext(ctx).Model(&FlowInstanceModel{}).Where("template_id = ?", templateID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	instances := make([]*entity.FlowInstance, 0, len(models))
	for i := range models {
		instances = append(instances, toInstanceEntity(&models[i]))
	}
	return instances, total, nil
}

func (r *FlowInstanceRepository) ListNodeInstances(ctx context.Context, flowInstanceID int64) ([]*entity.NodeInstance, error) {
	var models []NodeInstanceModel
	if err := r.db.WithContext(ctx).
		Where("flow_instance_id = ?", flowInstanceID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	nis := make([]*entity.NodeInstance, 0, len(models))
	for i := range models {
		nis = append(nis, toNodeInstanceEntity(&models[i]))
	}
	return nis, nil
}

func (r *FlowInstanceRepository) CreateNodeInstance(ctx context.Context, ni *entity.NodeInstance) error {
	model := toNodeInstanceModel(ni)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	ni.ID = model.ID
	return nil
}

func (r *FlowInstanceRepository) UpdateNodeInstance(ctx context.Context, ni *entity.NodeInstance) error {
	model := toNodeInstanceModel(ni)
	return r.db.WithContext(ctx).Model(&NodeInstanceModel{}).Where("id = ?", model.ID).Updates(map[string]interface{}{
		"status":      model.Status,
		"env":         model.Env,
		"output_env":  model.OutputEnv,
		"run_times":   model.RunTimes,
		"retry_count": model.RetryCount,
		"updated_at":  time.Now(),
	}).Error
}

// --- 转换函数 ---

func toInstanceModel(inst *entity.FlowInstance) *FlowInstanceModel {
	paramsJSON, _ := json.Marshal(inst.Params)
	envJSON, _ := json.Marshal(inst.Env)
	return &FlowInstanceModel{
		ID:                inst.ID,
		TemplateID:        inst.TemplateID,
		TemplateVersionID: inst.TemplateVersionID,
		TemplateVersion:   inst.TemplateVersion,
		Status:            inst.Status.Int(),
		TriggerType:       inst.TriggerType.Int(),
		Params:            string(paramsJSON),
		Env:               string(envJSON),
		FinishedAt:        inst.FinishedAt,
		CreatedAt:         inst.CreatedAt,
		UpdatedAt:         inst.UpdatedAt,
	}
}

func toInstanceEntity(m *FlowInstanceModel) *entity.FlowInstance {
	var params, env map[string]string
	_ = json.Unmarshal([]byte(m.Params), &params)
	_ = json.Unmarshal([]byte(m.Env), &env)
	if params == nil {
		params = map[string]string{}
	}
	if env == nil {
		env = map[string]string{}
	}
	return &entity.FlowInstance{
		ID:                m.ID,
		TemplateID:        m.TemplateID,
		TemplateVersionID: m.TemplateVersionID,
		TemplateVersion:   m.TemplateVersion,
		Status:            valueobject.FlowInstanceStatusFromInt(m.Status),
		TriggerType:       valueobject.TriggerTypeFromInt(m.TriggerType),
		Params:            params,
		Env:               env,
		FinishedAt:        m.FinishedAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func toNodeInstanceModel(ni *entity.NodeInstance) *NodeInstanceModel {
	envJSON, _ := json.Marshal(ni.Env)
	outputJSON, _ := json.Marshal(ni.OutputEnv)
	runTimesJSON, _ := json.Marshal(ni.RunTimes)
	return &NodeInstanceModel{
		ID:             ni.ID,
		FlowInstanceID: ni.FlowInstanceID,
		NodeID:         ni.NodeID,
		Type:           ni.Type.Int(),
		Status:         ni.Status.Int(),
		Env:            string(envJSON),
		OutputEnv:      string(outputJSON),
		RunTimes:       string(runTimesJSON),
		RetryCount:     ni.RetryCount,
	}
}

func toNodeInstanceEntity(m *NodeInstanceModel) *entity.NodeInstance {
	var env, output map[string]string
	_ = json.Unmarshal([]byte(m.Env), &env)
	_ = json.Unmarshal([]byte(m.OutputEnv), &output)
	if env == nil {
		env = map[string]string{}
	}
	if output == nil {
		output = map[string]string{}
	}
	var runTimes []entity.RunTimeSpan
	if m.RunTimes != "" {
		_ = json.Unmarshal([]byte(m.RunTimes), &runTimes)
	}
	return &entity.NodeInstance{
		ID:             m.ID,
		FlowInstanceID: m.FlowInstanceID,
		NodeID:         m.NodeID,
		Type:           valueobject.NodeTypeFromInt(m.Type),
		Status:         valueobject.NodeInstanceStatusFromInt(m.Status),
		Env:            env,
		OutputEnv:      output,
		RunTimes:       runTimes,
		RetryCount:     m.RetryCount,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
