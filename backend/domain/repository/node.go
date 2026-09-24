package repository

import (
	"context"

	"flow/domain/entity"
)

// NodeRepository 定义节点的持久化契约。
type NodeRepository interface {
	Create(ctx context.Context, node *entity.Node) error
	GetByID(ctx context.Context, id int64) (*entity.Node, error)
	Update(ctx context.Context, node *entity.Node) error
	Delete(ctx context.Context, id int64) error
	// ListByVersion 查询某个版本下的所有节点。
	ListByVersion(ctx context.Context, templateID, versionID int64) ([]*entity.Node, error)
	// DeleteByVersion 删除某个版本下的所有节点（用于整体替换）。
	DeleteByVersion(ctx context.Context, templateID, versionID int64) error
}
