package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// FlowTemplateVersion 是流水线模板的一个版本快照，
// 承载某一次发布时的完整内容（节点与 DAG 定义）。
// 每次发布产生一个新版本，实例运行引用具体版本快照。
type FlowTemplateVersion struct {
	ID         int64
	TemplateID int64
	Version    int
	Status     valueobject.TemplateStatus
	Nodes      []*Node
	// Trigger 触发配置快照（发布时从草稿触发配置固化，实例启动以快照为准）。
	Trigger   TriggerConfig
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewFlowTemplateVersion 创建版本快照，初始为草稿（临时版本）。
// ID 由数据库自增生成。
func NewFlowTemplateVersion(templateID int64, version int) (*FlowTemplateVersion, error) {
	if templateID == 0 {
		return nil, errors.New("template id must not be zero")
	}
	if version < 1 {
		return nil, errors.New("version must be positive")
	}
	now := time.Now()
	return &FlowTemplateVersion{
		TemplateID: templateID,
		Version:    version,
		Status:     valueobject.TemplateStatusDraft,
		Nodes:      []*Node{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Publish 发布版本快照，草稿转为正式版本。
func (v *FlowTemplateVersion) Publish() error {
	if v.Status == valueobject.TemplateStatusPublished {
		return errors.New("version already published")
	}
	v.Status = valueobject.TemplateStatusPublished
	v.UpdatedAt = time.Now()
	return nil
}

// AddNode 向版本快照中追加节点。
func (v *FlowTemplateVersion) AddNode(node *Node) error {
	if node == nil {
		return errors.New("node must not be nil")
	}
	v.Nodes = append(v.Nodes, node)
	return nil
}

// GetNode 按 ID 查找节点。
func (v *FlowTemplateVersion) GetNode(id int64) *Node {
	for _, n := range v.Nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// StartNode 返回开始节点，若不存在或不止一个则返回错误。
func (v *FlowTemplateVersion) StartNode() (*Node, error) {
	var start *Node
	for _, n := range v.Nodes {
		if n.Type == valueobject.NodeTypeStart {
			if start != nil {
				return nil, errors.New("multiple start nodes found")
			}
			start = n
		}
	}
	if start == nil {
		return nil, errors.New("start node not found")
	}
	return start, nil
}

// EndNode 返回结束节点，若不存在或不止一个则返回错误。
func (v *FlowTemplateVersion) EndNode() (*Node, error) {
	var end *Node
	for _, n := range v.Nodes {
		if n.Type == valueobject.NodeTypeEnd {
			if end != nil {
				return nil, errors.New("multiple end nodes found")
			}
			end = n
		}
	}
	if end == nil {
		return nil, errors.New("end node not found")
	}
	return end, nil
}
