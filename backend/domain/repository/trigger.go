package repository

import (
	"context"

	"flow/domain/entity"
)

// TriggerConfigRepository 定义触发配置的持久化契约。
type TriggerConfigRepository interface {
	Create(ctx context.Context, cfg *entity.TriggerConfig) error
	// GetByTemplateID 查询某个模板的触发配置（一个模板只有一条触发配置）。
	GetByTemplateID(ctx context.Context, templateID int64) (*entity.TriggerConfig, error)
	Update(ctx context.Context, cfg *entity.TriggerConfig) error
	Delete(ctx context.Context, id int64) error
}
