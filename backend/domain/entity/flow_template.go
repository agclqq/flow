package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// 模板名称长度上限。
const maxNameLength = 128

// FlowTemplate 是流水线模板的领域实体，承载模板元数据与核心业务规则。
// 内容快照（节点与 DAG）由 FlowTemplateVersion 承载。
type FlowTemplate struct {
	ID   int64
	Name string
	// Owners 模板级全局负责人。
	Owners []string
	// Description 模板描述。
	Description string
	// Version 当前最新版本号（对应最新发布版本的编号）。
	Version int
	// Status 表示模板是否发布过：draft=从未发布，published=已发布（git 模型下发布后保持不变）。
	Status    valueobject.TemplateStatus
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewFlowTemplate 创建流水线模板实体，初始为草稿状态（临时版本）。
// ID 由数据库自增生成。
func NewFlowTemplate(name string) (*FlowTemplate, error) {
	if name == "" {
		return nil, errors.New("flow template name must not be empty")
	}
	if len(name) > maxNameLength {
		return nil, errors.New("flow template name exceeds maximum length")
	}
	now := time.Now()
	return &FlowTemplate{
		Name:      name,
		Owners:    []string{},
		Version:   1,
		Status:    valueobject.TemplateStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// PublishSnapshot 发布快照后记录最新发布版本号，状态置为已发布（发布后永久保持）。
func (t *FlowTemplate) PublishSnapshot(version int) error {
	if version <= 0 {
		return errors.New("published version must be positive")
	}
	t.Status = valueobject.TemplateStatusPublished
	t.Version = version
	t.UpdatedAt = time.Now()
	return nil
}

// IsDeleted 判断模板是否已被软删除。
func (t *FlowTemplate) IsDeleted() bool {
	return t.DeletedAt != nil
}

// SoftDelete 标记模板为软删除，删除后不再支持调度。
func (t *FlowTemplate) SoftDelete() {
	now := time.Now()
	t.DeletedAt = &now
	t.UpdatedAt = now
}
