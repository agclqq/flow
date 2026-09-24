package persistence

import (
	"context"

	"flow/domain/entity"

	"gorm.io/gorm"
)

// NodeOperationLogRepository 是人工节点操作日志的持久化实现。
type NodeOperationLogRepository struct {
	db *gorm.DB
}

// NewNodeOperationLogRepository 构造仓储实现。
func NewNodeOperationLogRepository(db *gorm.DB) *NodeOperationLogRepository {
	return &NodeOperationLogRepository{db: db}
}

func toOperationLogModel(l *entity.NodeOperationLog) *NodeOperationLogModel {
	return &NodeOperationLogModel{
		InstanceID:     l.InstanceID,
		NodeInstanceID: l.NodeInstanceID,
		NodeID:         l.NodeID,
		Action:         l.Action,
		Operator:       l.Operator,
		CreatedAt:      l.CreatedAt,
	}
}

func toOperationLogEntity(m *NodeOperationLogModel) *entity.NodeOperationLog {
	return &entity.NodeOperationLog{
		ID:             m.ID,
		InstanceID:     m.InstanceID,
		NodeInstanceID: m.NodeInstanceID,
		NodeID:         m.NodeID,
		Action:         m.Action,
		Operator:       m.Operator,
		CreatedAt:      m.CreatedAt,
	}
}

func (r *NodeOperationLogRepository) Create(ctx context.Context, log *entity.NodeOperationLog) error {
	m := toOperationLogModel(log)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	log.ID, log.CreatedAt = m.ID, m.CreatedAt
	return nil
}

func (r *NodeOperationLogRepository) ListByInstanceNode(ctx context.Context, instanceID, nodeID int64) ([]*entity.NodeOperationLog, error) {
	var models []NodeOperationLogModel
	if err := r.db.WithContext(ctx).
		Where("instance_id = ? AND node_id = ?", instanceID, nodeID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.NodeOperationLog, 0, len(models))
	for i := range models {
		items = append(items, toOperationLogEntity(&models[i]))
	}
	return items, nil
}
