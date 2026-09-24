package repository

import (
	"context"

	"flow/domain/entity"
)

// FlowParamRepository 定义流水线参数定义的持久化契约。
type FlowParamRepository interface {
	Create(ctx context.Context, param *entity.FlowParam) error
	GetByID(ctx context.Context, id int64) (*entity.FlowParam, error)
	Update(ctx context.Context, param *entity.FlowParam) error
	Delete(ctx context.Context, id int64) error
	// ListByVersion 查询某个版本下的所有参数定义。
	ListByVersion(ctx context.Context, templateID, versionID int64) ([]*entity.FlowParam, error)
}
