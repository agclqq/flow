package repository

import (
	"context"

	"flow/domain/entity"
)

// NodeOperationLogRepository 定义人工节点操作日志的持久化契约。
type NodeOperationLogRepository interface {
	Create(ctx context.Context, log *entity.NodeOperationLog) error
	// ListByInstanceNode 按实例与节点倒序列出操作日志。
	ListByInstanceNode(ctx context.Context, instanceID, nodeID int64) ([]*entity.NodeOperationLog, error)
}
