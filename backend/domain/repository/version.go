package repository

import (
	"context"

	"flow/domain/entity"
)

// FlowTemplateVersionRepository 定义版本快照的持久化契约。
type FlowTemplateVersionRepository interface {
	Create(ctx context.Context, ver *entity.FlowTemplateVersion) error
	GetByID(ctx context.Context, id int64) (*entity.FlowTemplateVersion, error)
	// GetByTemplateAndVersion 按模板 ID 和版本号查询。
	GetByTemplateAndVersion(ctx context.Context, templateID int64, version int) (*entity.FlowTemplateVersion, error)
	// GetLatestPublishedByTemplate 返回模板最新的已发布版本（含节点）。
	GetLatestPublishedByTemplate(ctx context.Context, templateID int64) (*entity.FlowTemplateVersion, error)
	// ListByTemplate 查询某个模板下的所有版本。
	ListByTemplate(ctx context.Context, templateID int64) ([]*entity.FlowTemplateVersion, error)
	// ListPublishedCronTriggers 返回各模板最新已发布版本中开启 cron 的规则。
	ListPublishedCronTriggers(ctx context.Context) ([]entity.PublishedCronRule, error)
	Update(ctx context.Context, ver *entity.FlowTemplateVersion) error
}
