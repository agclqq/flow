package repository

import (
	"context"

	"flow/domain/entity"
)

// FlowInstanceRepository 定义流水线实例的持久化契约。
type FlowInstanceRepository interface {
	Create(ctx context.Context, inst *entity.FlowInstance) error
	GetByID(ctx context.Context, id int64) (*entity.FlowInstance, error)
	Update(ctx context.Context, inst *entity.FlowInstance) error
	// ListByTemplate 查询某个模板下的实例列表。
	ListByTemplate(ctx context.Context, templateID int64, offset, limit int) ([]*entity.FlowInstance, int64, error)
	// ListNodeInstances 查询某个实例下的所有节点实例。
	ListNodeInstances(ctx context.Context, flowInstanceID int64) ([]*entity.NodeInstance, error)
	// CreateNodeInstance 创建节点实例。
	CreateNodeInstance(ctx context.Context, ni *entity.NodeInstance) error
	// UpdateNodeInstance 更新节点实例。
	UpdateNodeInstance(ctx context.Context, ni *entity.NodeInstance) error
}
