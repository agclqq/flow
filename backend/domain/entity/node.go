package entity

import (
	"errors"

	"flow/domain/valueobject"
)

// Node 是流水线中的一个节点。
// 统一建模：start/end/task/edge/manual 共用一张表，
// 拓扑关系（连线）由 edge 节点的 from/to 字段承载，构成 DAG 邻接表。
type Node struct {
	ID         int64
	TemplateID int64
	VersionID  int64
	Type       valueobject.NodeType
	Name       string

	// 执行逻辑（三段 shell 脚本）。
	PreScript  string
	Script     string
	PostScript string

	// 负责人（task 节点支持多个）。
	Owners []string
	// Approvers 审批人（manual 节点支持多个）。
	Approvers []string
	// Description 节点描述。
	Description string

	// 连线关系：仅 edge 使用。
	FromNodeID int64
	ToNodeID   int64

	// Condition/IsDefault 已废弃，仅兼容旧数据存储。
	Condition string
	IsDefault bool

	// StartMode 启动门禁：any / all / custom，默认 any。
	StartMode valueobject.StartMode
	// StartScript 启动前脚本（custom 或条件分支），exit 0 才启动。
	StartScript string
	// RequiredPredecessorIDs custom 模式下必须成功的前驱节点 ID。
	RequiredPredecessorIDs []int64
	// RequiredEdgeIDs 已废弃，兼容旧 join 数据。
	RequiredEdgeIDs []int64

	// RuntimeKind 执行资源类型：local / ssh / k8s，空值视为 local。
	RuntimeKind string
	// ResourceID 引用的执行资源 ID（ssh/k8s 必填，local 忽略）。
	ResourceID int64

	// PositionX/PositionY DAG 画布坐标，仅可视化节点使用。
	PositionX *float64
	PositionY *float64
}

// NewNode 创建节点，校验类型合法性。
func NewNode(templateID, versionID int64, nodeType valueobject.NodeType, name string) (*Node, error) {
	if templateID == 0 || versionID == 0 {
		return nil, errors.New("template id and version id must not be zero")
	}
	if !nodeType.Valid() {
		return nil, errors.New("invalid node type")
	}
	return &Node{
		TemplateID:             templateID,
		VersionID:              versionID,
		Type:                   nodeType,
		Name:                   name,
		Owners:                 []string{},
		Approvers:              []string{},
		StartMode:              valueobject.StartModeAny,
		RequiredPredecessorIDs: []int64{},
		RequiredEdgeIDs:        []int64{},
	}, nil
}

// IsEdge 判断是否为流转节点（边）。
func (n *Node) IsEdge() bool {
	return n.Type == valueobject.NodeTypeEdge
}

// IsTask 判断是否为普通执行节点。
func (n *Node) IsTask() bool {
	return n.Type == valueobject.NodeTypeTask
}

// IsJoin 判断是否为废弃的聚合节点（兼容旧数据）。
func (n *Node) IsJoin() bool {
	return n.Type == valueobject.NodeTypeJoin
}

// IsExecutable 判断是否参与业务执行（含废弃 join 与 manual）。
func (n *Node) IsExecutable() bool {
	return n.IsTask() || n.IsManual() || n.IsJoin()
}

// IsManual 判断是否为人工审批节点。
func (n *Node) IsManual() bool {
	return n.Type == valueobject.NodeTypeManual
}

// SetEdge 为边节点设置连线关系（纯拓扑）。
func (n *Node) SetEdge(fromNodeID, toNodeID int64) error {
	if !n.IsEdge() {
		return errors.New("only edge node can set from/to")
	}
	if fromNodeID == 0 || toNodeID == 0 {
		return errors.New("edge from/to must not be zero")
	}
	n.FromNodeID = fromNodeID
	n.ToNodeID = toNodeID
	n.Condition = ""
	n.IsDefault = false
	return nil
}

// SetRequiredPredecessors 设置 custom 模式下必须完成的前驱节点。
func (n *Node) SetRequiredPredecessors(nodeIDs []int64) error {
	if !n.IsExecutable() {
		return errors.New("only executable node can set required predecessors")
	}
	n.RequiredPredecessorIDs = nodeIDs
	return nil
}

// AddOwner 为 task/manual 节点添加负责人。
func (n *Node) AddOwner(owner string) error {
	if !n.IsTask() && !n.IsManual() {
		return errors.New("only task or manual node can have owners")
	}
	if owner == "" {
		return errors.New("owner must not be empty")
	}
	n.Owners = append(n.Owners, owner)
	return nil
}
