package repository

import (
	"context"

	"flow/domain/entity"
)

// ExecutorResourceRepository 定义执行资源的持久化契约。
type ExecutorResourceRepository interface {
	Create(ctx context.Context, res *entity.ExecutorResource) error
	Update(ctx context.Context, res *entity.ExecutorResource) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*entity.ExecutorResource, error)
	List(ctx context.Context) ([]*entity.ExecutorResource, error)
}
