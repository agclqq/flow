package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// NotifyTemplate 是通知内容模板，独立配置、可被多个流水线复用。
// 内容支持变量如 {{flow_instance_id}}、{{flow_name}}、{{node_name}}、{{node_owner}}。
type NotifyTemplate struct {
	ID   int64
	Name string
	// Content 通知内容模板。
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewNotifyTemplate 创建通知内容模板。
// ID 由数据库自增生成。
func NewNotifyTemplate(name, content string) (*NotifyTemplate, error) {
	if name == "" {
		return nil, errors.New("notify template name must not be empty")
	}
	if content == "" {
		return nil, errors.New("notify template content must not be empty")
	}
	now := time.Now()
	return &NotifyTemplate{
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FlowNotifyBinding 是流水线与通知内容模板的绑定。
// 每个流水线选择要复用的通知模板、触发时机，并可单独补充内容。
type FlowNotifyBinding struct {
	ID         int64
	TemplateID int64
	// NodeID 绑定的节点 ID，0 表示模板级绑定。
	NodeID int64
	// NotifyTemplateID 引用的通知内容模板（可复用）。
	NotifyTemplateID int64
	// Timings 通知触发时机（流水线成功/失败），可多个。
	Timings []valueobject.NotifyTiming
	// ExtraContent 额外补充内容（可选，追加到通知模板之后）。
	ExtraContent string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewFlowNotifyBinding 创建绑定。
// ID 由数据库自增生成。
func NewFlowNotifyBinding(templateID, notifyTemplateID int64) (*FlowNotifyBinding, error) {
	if templateID == 0 || notifyTemplateID == 0 {
		return nil, errors.New("template id and notify template id must not be zero")
	}
	now := time.Now()
	return &FlowNotifyBinding{
		TemplateID:       templateID,
		NotifyTemplateID: notifyTemplateID,
		Timings:          []valueobject.NotifyTiming{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// AddTiming 添加一个通知触发时机。
func (b *FlowNotifyBinding) AddTiming(timing valueobject.NotifyTiming) error {
	if !timing.Valid() {
		return errors.New("invalid notify timing")
	}
	b.Timings = append(b.Timings, timing)
	return nil
}
