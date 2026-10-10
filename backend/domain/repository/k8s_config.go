package repository

import (
	"context"

	"flow/domain/entity"
)

// K8sConfigRepository 定义 K8s 认证配置的持久化契约。
type K8sConfigRepository interface {
	Create(ctx context.Context, cfg *entity.K8sConfig) error
	GetByID(ctx context.Context, id int64) (*entity.K8sConfig, error)
	List(ctx context.Context) ([]*entity.K8sConfig, error)
	Update(ctx context.Context, cfg *entity.K8sConfig) error
	Delete(ctx context.Context, id int64) error
}
