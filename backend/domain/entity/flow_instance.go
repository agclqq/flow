package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// FlowInstance 是流水线实例，表示一次流水线的具体运行。
// 它引用某个已发布的版本快照，携带触发方式与入参。
type FlowInstance struct {
	ID                int64
	TemplateID        int64
	TemplateVersionID int64 // 引用的版本快照 ID
	TemplateVersion   int   // 版本号（冗余，便于展示/排查）
	Status            valueobject.FlowInstanceStatus
	TriggerType       valueobject.TriggerType
	Params            map[string]string // 触发时入参，以环境变量注入节点
	Env               map[string]string // 全局环境变量（节点输出会更新到这里）
	NodeInstances     []*NodeInstance
	FinishedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewFlowInstance 创建流水线实例，初始状态为未开始。
// ID 由数据库自增生成。
func NewFlowInstance(templateID, templateVersionID int64, templateVersion int, trigger valueobject.TriggerType) (*FlowInstance, error) {
	if templateID == 0 || templateVersionID == 0 {
		return nil, errors.New("template id and template version id must not be zero")
	}
	if !trigger.Valid() {
		return nil, errors.New("invalid trigger type")
	}
	now := time.Now()
	return &FlowInstance{
		TemplateID:        templateID,
		TemplateVersionID: templateVersionID,
		TemplateVersion:   templateVersion,
		Status:            valueobject.FlowInstanceStatusPending,
		TriggerType:       trigger,
		Params:            map[string]string{},
		Env:               map[string]string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// Transition 执行流水线实例状态流转，非法流转返回错误。
func (i *FlowInstance) Transition(target valueobject.FlowInstanceStatus) error {
	if !i.Status.CanTransitionTo(target) {
		return errors.New("invalid flow instance status transition")
	}
	i.Status = target
	i.UpdatedAt = time.Now()
	if target == valueobject.FlowInstanceStatusSucceeded || target == valueobject.FlowInstanceStatusFailed {
		now := time.Now()
		i.FinishedAt = &now
	}
	return nil
}

// SetParams 设置触发入参。
func (i *FlowInstance) SetParams(params map[string]string) {
	i.Params = params
}

// UpdateEnv 更新全局环境变量（合并节点输出）。
func (i *FlowInstance) UpdateEnv(env map[string]string) {
	for k, v := range env {
		i.Env[k] = v
	}
}

// AddNodeInstance 关联一个节点实例。
func (i *FlowInstance) AddNodeInstance(ni *NodeInstance) {
	i.NodeInstances = append(i.NodeInstances, ni)
}
