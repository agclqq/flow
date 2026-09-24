package repository

import (
	"context"

	"flow/domain/entity"
)

// NotifyTemplateRepository 定义通知内容模板的持久化契约。
type NotifyTemplateRepository interface {
	Create(ctx context.Context, tpl *entity.NotifyTemplate) error
	GetByID(ctx context.Context, id int64) (*entity.NotifyTemplate, error)
	Update(ctx context.Context, tpl *entity.NotifyTemplate) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*entity.NotifyTemplate, int64, error)
}

// FlowNotifyBindingRepository 定义流水线通知绑定的持久化契约。
type FlowNotifyBindingRepository interface {
	Create(ctx context.Context, binding *entity.FlowNotifyBinding) error
	GetByID(ctx context.Context, id int64) (*entity.FlowNotifyBinding, error)
	Update(ctx context.Context, binding *entity.FlowNotifyBinding) error
	Delete(ctx context.Context, id int64) error
	ListByTemplate(ctx context.Context, templateID int64) ([]*entity.FlowNotifyBinding, error)
	ReplaceByTemplate(ctx context.Context, templateID int64, bindings []*entity.FlowNotifyBinding) error
}
