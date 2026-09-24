package repository

import (
	"context"
	"errors"

	"flow/domain/entity"
)

// ErrNotFound 表示记录不存在。
var ErrNotFound = errors.New("record not found")

// FlowTemplateRepository 定义流水线模板的持久化契约，
// 由基础设施层提供具体实现，领域层仅依赖该接口。
type FlowTemplateRepository interface {
	Create(ctx context.Context, tpl *entity.FlowTemplate) error
	GetByID(ctx context.Context, id int64) (*entity.FlowTemplate, error)
	Update(ctx context.Context, tpl *entity.FlowTemplate) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*entity.FlowTemplate, int64, error)
}
