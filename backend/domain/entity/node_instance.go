package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// RunTimeSpan 是节点的一次运行时间段（失败重试会产生多段）。
type RunTimeSpan struct {
	StartAt time.Time  `json:"start_at"`
	EndAt   *time.Time `json:"end_at,omitempty"` // 进行中（仍在运行）时为空
}

// NodeInstance 是节点实例，表示流水线实例中某个节点的具体运行。
type NodeInstance struct {
	ID             int64
	FlowInstanceID int64
	NodeID         int64
	Type           valueobject.NodeType
	Status         valueobject.NodeInstanceStatus
	Env            map[string]string // 该节点执行时注入的环境变量
	OutputEnv      map[string]string // 节点输出的环境变量（写文件后被程序捕获）
	RunTimes       []RunTimeSpan     // 运行时间：每进入一次 Running 开启一段，离开时关闭
	RetryCount     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewNodeInstance 创建节点实例，初始状态为未开始。
// ID 由数据库自增生成。
func NewNodeInstance(flowInstanceID, nodeID int64, nodeType valueobject.NodeType) (*NodeInstance, error) {
	if flowInstanceID == 0 || nodeID == 0 {
		return nil, errors.New("flow instance id and node id must not be zero")
	}
	if !nodeType.Valid() {
		return nil, errors.New("invalid node type")
	}
	now := time.Now()
	return &NodeInstance{
		FlowInstanceID: flowInstanceID,
		NodeID:         nodeID,
		Type:           nodeType,
		Status:         valueobject.NodeInstanceStatusPending,
		Env:            map[string]string{},
		OutputEnv:      map[string]string{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Transition 执行节点实例状态流转，非法流转返回错误。
// 同时维护运行时间：进入 Running 开启新的时间段，进入终态时关闭当前段。
// 等待审批（waiting_approval）不关段：人工节点的结束时间取审批通过/拒绝的时刻。
func (ni *NodeInstance) Transition(target valueobject.NodeInstanceStatus) error {
	if !ni.Status.CanTransitionTo(target) {
		return errors.New("invalid node instance status transition")
	}
	now := time.Now()
	if target == valueobject.NodeInstanceStatusRunning {
		ni.RunTimes = append(ni.RunTimes, RunTimeSpan{StartAt: now})
	} else if target == valueobject.NodeInstanceStatusSucceeded ||
		target == valueobject.NodeInstanceStatusFailed ||
		target == valueobject.NodeInstanceStatusFailedSkipped {
		if len(ni.RunTimes) > 0 && ni.RunTimes[len(ni.RunTimes)-1].EndAt == nil {
			ni.RunTimes[len(ni.RunTimes)-1].EndAt = &now
		}
	}
	ni.Status = target
	ni.UpdatedAt = now
	return nil
}

// MarkRetrying 进入重试状态并累加重试次数。
func (ni *NodeInstance) MarkRetrying() error {
	if err := ni.Transition(valueobject.NodeInstanceStatusRetrying); err != nil {
		return err
	}
	ni.RetryCount++
	return nil
}

// SetOutputEnv 记录节点输出的环境变量。
func (ni *NodeInstance) SetOutputEnv(env map[string]string) {
	ni.OutputEnv = env
}
