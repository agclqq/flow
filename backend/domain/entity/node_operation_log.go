package entity

import (
	"errors"
	"time"
)

// 人工节点操作日志的动作常量。
const (
	NodeActionApprove = "approve"
	NodeActionReject  = "reject"
)

// NodeOperationLog 记录人工节点的操作流水（谁在什么时间做了什么）。
type NodeOperationLog struct {
	ID             int64
	InstanceID     int64
	NodeInstanceID int64
	NodeID         int64
	// Action 动作：approve / reject。
	Action   string
	// Operator 操作人。
	Operator string
	CreatedAt time.Time
}

// NewNodeOperationLog 创建操作日志。
func NewNodeOperationLog(instanceID, nodeInstanceID, nodeID int64, action, operator string) (*NodeOperationLog, error) {
	if action != NodeActionApprove && action != NodeActionReject {
		return nil, errors.New("invalid node action")
	}
	if operator == "" {
		operator = "-"
	}
	return &NodeOperationLog{
		InstanceID:     instanceID,
		NodeInstanceID: nodeInstanceID,
		NodeID:         nodeID,
		Action:         action,
		Operator:       operator,
		CreatedAt:      time.Now(),
	}, nil
}
