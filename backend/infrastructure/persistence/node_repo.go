package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// NodeRepository 是 NodeRepository 接口的 gorm 实现。
type NodeRepository struct {
	db *gorm.DB
}

// NewNodeRepository 构造仓储实现。
func NewNodeRepository(db *gorm.DB) repository.NodeRepository {
	return &NodeRepository{db: db}
}

// Create 持久化节点。
func (r *NodeRepository) Create(ctx context.Context, node *entity.Node) error {
	model := toNodeModel(node)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	node.ID = model.ID
	return nil
}

// GetByID 按主键查询节点。
func (r *NodeRepository) GetByID(ctx context.Context, id int64) (*entity.Node, error) {
	var model NodeModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toNodeEntity(&model), nil
}

// Update 更新节点。
func (r *NodeRepository) Update(ctx context.Context, node *entity.Node) error {
	return r.db.WithContext(ctx).Save(toNodeModel(node)).Error
}

// Delete 删除节点。
func (r *NodeRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&NodeModel{}, "id = ?", id).Error
}

// ListByVersion 查询某个版本下的所有节点。
func (r *NodeRepository) ListByVersion(ctx context.Context, templateID, versionID int64) ([]*entity.Node, error) {
	var models []NodeModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND version_id = ?", templateID, versionID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	nodes := make([]*entity.Node, 0, len(models))
	for i := range models {
		nodes = append(nodes, toNodeEntity(&models[i]))
	}
	return nodes, nil
}

// DeleteByVersion 删除某个版本下的所有节点。
func (r *NodeRepository) DeleteByVersion(ctx context.Context, templateID, versionID int64) error {
	return r.db.WithContext(ctx).
		Where("template_id = ? AND version_id = ?", templateID, versionID).
		Delete(&NodeModel{}).Error
}

func toNodeModel(node *entity.Node) *NodeModel {
	ownersJSON, _ := json.Marshal(node.Owners)
	approversJSON, _ := json.Marshal(node.Approvers)
	requiredPredJSON, _ := json.Marshal(node.RequiredPredecessorIDs)
	requiredEdgesJSON, _ := json.Marshal(node.RequiredEdgeIDs)
	k8sVolumesJSON, _ := json.Marshal(node.K8sVolumes)
	k8sMountsJSON, _ := json.Marshal(node.K8sVolumeMounts)
	k8sResourcesJSON, _ := json.Marshal(node.K8sResources)
	if node.K8sVolumes == nil {
		k8sVolumesJSON = []byte("[]")
	}
	if node.K8sVolumeMounts == nil {
		k8sMountsJSON = []byte("[]")
	}
	if node.K8sResources.IsEmpty() {
		k8sResourcesJSON = []byte("{}")
	}
	return &NodeModel{
		ID:                     node.ID,
		TemplateID:             node.TemplateID,
		VersionID:              node.VersionID,
		Type:                   node.Type.Int(),
		Name:                   node.Name,
		Description:            node.Description,
		PreScript:              node.PreScript,
		Script:                 node.Script,
		PostScript:             node.PostScript,
		Owners:                 string(ownersJSON),
		Approvers:              string(approversJSON),
		FromNodeID:             node.FromNodeID,
		ToNodeID:               node.ToNodeID,
		Condition:              node.Condition,
		IsDefault:              node.IsDefault,
		JoinType:               node.StartMode.Int(),
		StartScript:            node.StartScript,
		RequiredPredecessorIDs: string(requiredPredJSON),
		RequiredEdgeIDs:        string(requiredEdgesJSON),
		PositionX:              node.PositionX,
		PositionY:              node.PositionY,
		RuntimeKind:            node.RuntimeKind,
		ResourceID:             node.ResourceID,
		K8sAuthID:              node.K8sAuthID,
		K8sNamespace:           node.K8sNamespace,
		K8sImage:               node.K8sImage,
		K8sVolumes:             string(k8sVolumesJSON),
		K8sVolumeMounts:        string(k8sMountsJSON),
		K8sResources:           string(k8sResourcesJSON),
	}
}

func toNodeEntity(m *NodeModel) *entity.Node {
	var owners []string
	if m.Owners != "" {
		_ = json.Unmarshal([]byte(m.Owners), &owners)
	}
	if owners == nil {
		owners = []string{}
	}
	var approvers []string
	if m.Approvers != "" {
		_ = json.Unmarshal([]byte(m.Approvers), &approvers)
	}
	if approvers == nil {
		approvers = []string{}
	}
	var requiredPredIDs []int64
	if m.RequiredPredecessorIDs != "" {
		_ = json.Unmarshal([]byte(m.RequiredPredecessorIDs), &requiredPredIDs)
	}
	if requiredPredIDs == nil {
		requiredPredIDs = []int64{}
	}
	var requiredEdgeIDs []int64
	if m.RequiredEdgeIDs != "" {
		_ = json.Unmarshal([]byte(m.RequiredEdgeIDs), &requiredEdgeIDs)
	}
	if requiredEdgeIDs == nil {
		requiredEdgeIDs = []int64{}
	}
	var k8sVolumes []entity.K8sVolume
	if m.K8sVolumes != "" {
		_ = json.Unmarshal([]byte(m.K8sVolumes), &k8sVolumes)
	}
	if k8sVolumes == nil {
		k8sVolumes = []entity.K8sVolume{}
	}
	var k8sMounts []entity.K8sVolumeMount
	if m.K8sVolumeMounts != "" {
		_ = json.Unmarshal([]byte(m.K8sVolumeMounts), &k8sMounts)
	}
	if k8sMounts == nil {
		k8sMounts = []entity.K8sVolumeMount{}
	}
	var k8sResources entity.K8sResourceRequirements
	if m.K8sResources != "" && m.K8sResources != "{}" {
		_ = json.Unmarshal([]byte(m.K8sResources), &k8sResources)
	}
	if k8sResources.Requests == nil {
		k8sResources.Requests = map[string]string{}
	}
	if k8sResources.Limits == nil {
		k8sResources.Limits = map[string]string{}
	}
	return &entity.Node{
		ID:                     m.ID,
		TemplateID:             m.TemplateID,
		VersionID:              m.VersionID,
		Type:                   valueobject.NodeTypeFromInt(m.Type),
		Name:                   m.Name,
		Description:            m.Description,
		PreScript:              m.PreScript,
		Script:                 m.Script,
		PostScript:             m.PostScript,
		Owners:                 owners,
		Approvers:              approvers,
		FromNodeID:             m.FromNodeID,
		ToNodeID:               m.ToNodeID,
		Condition:              m.Condition,
		IsDefault:              m.IsDefault,
		StartMode:              valueobject.StartModeFromInt(m.JoinType),
		StartScript:            m.StartScript,
		RequiredPredecessorIDs: requiredPredIDs,
		RequiredEdgeIDs:        requiredEdgeIDs,
		PositionX:              m.PositionX,
		PositionY:              m.PositionY,
		RuntimeKind:            m.RuntimeKind,
		ResourceID:             m.ResourceID,
		K8sAuthID:              m.K8sAuthID,
		K8sNamespace:           m.K8sNamespace,
		K8sImage:               m.K8sImage,
		K8sVolumes:             k8sVolumes,
		K8sVolumeMounts:        k8sMounts,
		K8sResources:           k8sResources,
	}
}
