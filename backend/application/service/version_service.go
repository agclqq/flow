package service

import (
	"context"
	"errors"
	"fmt"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/executor"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// VersionService 管理版本快照、节点、参数和触发配置。
type VersionService struct {
	tplRepo     repository.FlowTemplateRepository
	versionRepo repository.FlowTemplateVersionRepository
	nodeRepo    repository.NodeRepository
	paramRepo   repository.FlowParamRepository
	triggerRepo repository.TriggerConfigRepository
}

// NewVersionService 构造版本管理服务。
func NewVersionService(
	tplRepo repository.FlowTemplateRepository,
	versionRepo repository.FlowTemplateVersionRepository,
	nodeRepo repository.NodeRepository,
	paramRepo repository.FlowParamRepository,
	triggerRepo repository.TriggerConfigRepository,
) *VersionService {
	return &VersionService{
		tplRepo:     tplRepo,
		versionRepo: versionRepo,
		nodeRepo:    nodeRepo,
		paramRepo:   paramRepo,
		triggerRepo: triggerRepo,
	}
}

// --- 版本管理 ---

// ListVersions 列出某个模板下的所有版本。
func (s *VersionService) ListVersions(ctx context.Context, templateID int64) ([]dto.VersionListItem, error) {
	vers, err := s.versionRepo.ListByTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.VersionListItem, 0, len(vers))
	for _, v := range vers {
		items = append(items, dto.ToVersionListItem(v))
	}
	return items, nil
}

// GetVersion 获取版本详情（含节点列表）。
func (s *VersionService) GetVersion(ctx context.Context, versionID int64) (*dto.VersionResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToVersionResponse(ver)
	return &resp, nil
}

// PublishVersion 发布版本。
func (s *VersionService) PublishVersion(ctx context.Context, versionID int64) (*dto.VersionResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if err := ver.Publish(); err != nil {
		return nil, err
	}
	if err := s.versionRepo.Update(ctx, ver); err != nil {
		return nil, err
	}
	resp := dto.ToVersionResponse(ver)
	return &resp, nil
}

// GetEditingVersion 返回编辑页应加载的版本：优先草稿，否则最新已发布版本（只读预览，保存时才创建草稿）。
func (s *VersionService) GetEditingVersion(ctx context.Context, templateID int64) (*dto.EditingVersionResponse, error) {
	if _, err := s.tplRepo.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	vers, err := s.versionRepo.ListByTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	contentVer, err := pickContentVersion(vers)
	if err != nil {
		return nil, errors.New("no editable version found")
	}
	return &dto.EditingVersionResponse{
		VersionID: contentVer.ID,
		Version:   contentVer.Version,
		IsDraft:   contentVer.Status == valueobject.TemplateStatusDraft,
		Status:    contentVer.Status.String(),
	}, nil
}

// EnsureDraftVersion 确保存在可写入的草稿版本；若仅有已发布版本则复制为新草稿（模板 ID 不变）。
func (s *VersionService) EnsureDraftVersion(ctx context.Context, templateID int64) (*dto.EditingVersionResponse, error) {
	editing, err := s.GetEditingVersion(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if editing.IsDraft {
		return editing, nil
	}

	tpl, err := s.tplRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	srcVer, err := s.versionRepo.GetByID(ctx, editing.VersionID)
	if err != nil {
		return nil, err
	}

	newVer, err := entity.NewFlowTemplateVersion(templateID, tpl.Version)
	if err != nil {
		return nil, err
	}
	if err := s.versionRepo.Create(ctx, newVer); err != nil {
		return nil, err
	}
	if err := s.copyVersionContent(ctx, srcVer, newVer); err != nil {
		return nil, err
	}

	return &dto.EditingVersionResponse{
		VersionID: newVer.ID,
		Version:   newVer.Version,
		IsDraft:   true,
		Status:    newVer.Status.String(),
	}, nil
}

// copyVersionContent 把源版本的节点与参数内容复制到目标版本。
func copyVersionContent(
	ctx context.Context,
	nodeRepo repository.NodeRepository,
	paramRepo repository.FlowParamRepository,
	srcVer, dstVer *entity.FlowTemplateVersion,
) error {
	idMap := make(map[int64]int64)
	for _, n := range srcVer.Nodes {
		if n.IsEdge() {
			continue
		}
		nodeType := n.Type
		if nodeType == valueobject.NodeTypeJoin {
			nodeType = valueobject.NodeTypeTask
		}
		copyNode, err := entity.NewNode(dstVer.TemplateID, dstVer.ID, nodeType, n.Name)
		if err != nil {
			return err
		}
		copyNode.Description = n.Description
		copyNode.PreScript = n.PreScript
		copyNode.Script = n.Script
		copyNode.PostScript = n.PostScript
		copyNode.Owners = append([]string(nil), n.Owners...)
		copyNode.Approvers = append([]string(nil), n.Approvers...)
		copyNode.StartMode = n.StartMode
		copyNode.StartScript = n.StartScript
		copyNode.RequiredPredecessorIDs = append([]int64(nil), n.RequiredPredecessorIDs...)
		copyNode.RequiredEdgeIDs = append([]int64(nil), n.RequiredEdgeIDs...)
		copyNode.PositionX = cloneFloat64Ptr(n.PositionX)
		copyNode.PositionY = cloneFloat64Ptr(n.PositionY)
		copyNode.RuntimeKind = n.RuntimeKind
		copyNode.ResourceID = n.ResourceID
		if err := nodeRepo.Create(ctx, copyNode); err != nil {
			return err
		}
		idMap[n.ID] = copyNode.ID
	}
	for _, n := range srcVer.Nodes {
		if !n.IsEdge() {
			continue
		}
		copyNode, err := entity.NewNode(dstVer.TemplateID, dstVer.ID, n.Type, n.Name)
		if err != nil {
			return err
		}
		fromID := idMap[n.FromNodeID]
		toID := idMap[n.ToNodeID]
		if err := copyNode.SetEdge(fromID, toID); err != nil {
			return err
		}
		if err := nodeRepo.Create(ctx, copyNode); err != nil {
			return err
		}
		idMap[n.ID] = copyNode.ID
	}
	for _, n := range srcVer.Nodes {
		if len(n.RequiredPredecessorIDs) == 0 && len(n.RequiredEdgeIDs) == 0 {
			continue
		}
		copyNode, err := nodeRepo.GetByID(ctx, idMap[n.ID])
		if err != nil {
			return err
		}
		if len(n.RequiredPredecessorIDs) > 0 {
			mapped := make([]int64, 0, len(n.RequiredPredecessorIDs))
			for _, predID := range n.RequiredPredecessorIDs {
				mapped = append(mapped, idMap[predID])
			}
			if err := copyNode.SetRequiredPredecessors(mapped); err != nil {
				return err
			}
		} else {
			mapped := make([]int64, 0, len(n.RequiredEdgeIDs))
			for _, edgeID := range n.RequiredEdgeIDs {
				mapped = append(mapped, idMap[edgeID])
			}
			copyNode.RequiredEdgeIDs = mapped
		}
		if err := nodeRepo.Update(ctx, copyNode); err != nil {
			return err
		}
	}

	params, err := paramRepo.ListByVersion(ctx, srcVer.TemplateID, srcVer.ID)
	if err != nil {
		return err
	}
	for _, p := range params {
		copyParam, err := entity.NewFlowParam(dstVer.TemplateID, dstVer.ID, p.Name, p.Type)
		if err != nil {
			return err
		}
		copyParam.DefaultValue = p.DefaultValue
		copyParam.Required = p.Required
		copyParam.Description = p.Description
		if err := paramRepo.Create(ctx, copyParam); err != nil {
			return err
		}
	}
	return nil
}

func (s *VersionService) copyVersionContent(ctx context.Context, srcVer, dstVer *entity.FlowTemplateVersion) error {
	return copyVersionContent(ctx, s.nodeRepo, s.paramRepo, srcVer, dstVer)
}

// --- 节点管理 ---

// AddNode 向版本中添加节点。
func (s *VersionService) AddNode(ctx context.Context, versionID int64, req dto.CreateNodeRequest) (*dto.NodeResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if ver.Status == valueobject.TemplateStatusPublished {
		return nil, errors.New("cannot modify a published version")
	}

	nodeType := valueobject.NodeType(req.Type)
	if nodeType == valueobject.NodeTypeJoin {
		nodeType = valueobject.NodeTypeTask
	}
	node, err := entity.NewNode(ver.TemplateID, versionID, nodeType, req.Name)
	if err != nil {
		return nil, err
	}

	// 设置脚本（task 节点）。
	node.PreScript = req.PreScript
	node.Script = req.Script
	node.PostScript = req.PostScript
	node.Description = req.Description

	// 设置负责人与审批人。
	for _, o := range req.Owners {
		_ = node.AddOwner(o)
	}
	node.Approvers = req.Approvers
	if node.Approvers == nil {
		node.Approvers = []string{}
	}

	// 设置边节点属性。
	if nodeType == valueobject.NodeTypeEdge {
		if err := node.SetEdge(req.FromNodeID, req.ToNodeID); err != nil {
			return nil, err
		}
	} else if err := applyStartConfig(node, req.StartMode, req.StartScript, req.RequiredPredecessorIDs, req.JoinType, req.RequiredEdgeIDs); err != nil {
		return nil, err
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, err
	}

	resp := dto.ToNodeResponse(node)
	return &resp, nil
}

// UpdateNode 更新节点。
func (s *VersionService) UpdateNode(ctx context.Context, nodeID int64, req dto.UpdateNodeRequest) (*dto.NodeResponse, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	// 校验版本是否可编辑。
	ver, err := s.versionRepo.GetByID(ctx, node.VersionID)
	if err != nil {
		return nil, err
	}
	if ver.Status == valueobject.TemplateStatusPublished {
		return nil, errors.New("cannot modify a published version")
	}

	if req.Name != "" {
		node.Name = req.Name
	}
	node.Description = req.Description
	if req.PreScript != "" {
		node.PreScript = req.PreScript
	}
	if req.Script != "" {
		node.Script = req.Script
	}
	if req.PostScript != "" {
		node.PostScript = req.PostScript
	}
	if req.Owners != nil {
		node.Owners = req.Owners
	}
	if req.Approvers != nil {
		node.Approvers = req.Approvers
	}
	if node.IsEdge() {
		if req.FromNodeID != 0 && req.ToNodeID != 0 {
			if err := node.SetEdge(req.FromNodeID, req.ToNodeID); err != nil {
				return nil, err
			}
		}
	} else if err := applyStartConfig(node, req.StartMode, req.StartScript, req.RequiredPredecessorIDs, req.JoinType, req.RequiredEdgeIDs); err != nil {
		return nil, err
	}

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return nil, err
	}

	resp := dto.ToNodeResponse(node)
	return &resp, nil
}

// DeleteNode 删除节点。
func (s *VersionService) DeleteNode(ctx context.Context, nodeID int64) error {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return err
	}
	ver, err := s.versionRepo.GetByID(ctx, node.VersionID)
	if err != nil {
		return err
	}
	if ver.Status == valueobject.TemplateStatusPublished {
		return errors.New("cannot modify a published version")
	}
	return s.nodeRepo.Delete(ctx, nodeID)
}

// ListNodes 列出版本下的所有节点。
func (s *VersionService) ListNodes(ctx context.Context, versionID int64) ([]dto.NodeResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.NodeResponse, 0, len(ver.Nodes))
	for _, n := range ver.Nodes {
		items = append(items, dto.ToNodeResponse(n))
	}
	return items, nil
}

// SaveGraph 批量替换草稿版本中的全部节点。
func (s *VersionService) SaveGraph(ctx context.Context, versionID int64, req dto.BatchGraphSaveRequest) ([]dto.NodeResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if ver.Status == valueobject.TemplateStatusPublished {
		return nil, errors.New("cannot modify a published version")
	}

	if err := s.nodeRepo.DeleteByVersion(ctx, ver.TemplateID, versionID); err != nil {
		return nil, err
	}

	idMap := make(map[int64]int64)
	created := make([]*entity.Node, 0, len(req.Nodes))

	for _, item := range req.Nodes {
		if valueobject.NodeType(item.Type) == valueobject.NodeTypeEdge {
			continue
		}
		node, err := s.buildNodeFromRequest(ver.TemplateID, versionID, item.CreateNodeRequest)
		if err != nil {
			return nil, err
		}
		if err := s.nodeRepo.Create(ctx, node); err != nil {
			return nil, err
		}
		refID := item.ID
		if refID == 0 {
			refID = node.ID
		}
		idMap[refID] = node.ID
		created = append(created, node)
	}

	for _, item := range req.Nodes {
		if valueobject.NodeType(item.Type) != valueobject.NodeTypeEdge {
			continue
		}
		node, err := entity.NewNode(ver.TemplateID, versionID, valueobject.NodeTypeEdge, item.Name)
		if err != nil {
			return nil, err
		}
		fromID := idMap[item.FromNodeID]
		toID := idMap[item.ToNodeID]
		if fromID == 0 || toID == 0 {
			return nil, errors.New("edge references unknown node id")
		}
		if err := node.SetEdge(fromID, toID); err != nil {
			return nil, err
		}
		if err := s.nodeRepo.Create(ctx, node); err != nil {
			return nil, err
		}
		if item.ID != 0 {
			idMap[item.ID] = node.ID
		}
		created = append(created, node)
	}

	for _, n := range created {
		if len(n.RequiredPredecessorIDs) == 0 && len(n.RequiredEdgeIDs) == 0 {
			continue
		}
		if len(n.RequiredPredecessorIDs) > 0 {
			mapped := make([]int64, 0, len(n.RequiredPredecessorIDs))
			for _, predID := range n.RequiredPredecessorIDs {
				newID, ok := idMap[predID]
				if !ok {
					return nil, errors.New("node references unknown predecessor id")
				}
				mapped = append(mapped, newID)
			}
			if err := n.SetRequiredPredecessors(mapped); err != nil {
				return nil, err
			}
		} else {
			mapped := make([]int64, 0, len(n.RequiredEdgeIDs))
			for _, edgeID := range n.RequiredEdgeIDs {
				newID, ok := idMap[edgeID]
				if !ok {
					return nil, errors.New("node references unknown edge id")
				}
				mapped = append(mapped, newID)
			}
			n.RequiredEdgeIDs = mapped
		}
		if err := s.nodeRepo.Update(ctx, n); err != nil {
			return nil, err
		}
	}

	items := make([]dto.NodeResponse, 0, len(created))
	for _, n := range created {
		items = append(items, dto.ToNodeResponse(n))
	}
	return items, nil
}

func (s *VersionService) buildNodeFromRequest(templateID, versionID int64, req dto.CreateNodeRequest) (*entity.Node, error) {
	nodeType := valueobject.NodeType(req.Type)
	if nodeType == valueobject.NodeTypeJoin {
		nodeType = valueobject.NodeTypeTask
	}
	node, err := entity.NewNode(templateID, versionID, nodeType, req.Name)
	if err != nil {
		return nil, err
	}
	node.Description = req.Description
	node.PreScript = req.PreScript
	node.Script = req.Script
	node.PostScript = req.PostScript
	for _, o := range req.Owners {
		_ = node.AddOwner(o)
	}
	node.Approvers = req.Approvers
	if node.Approvers == nil {
		node.Approvers = []string{}
	}
	if err := applyStartConfig(node, req.StartMode, req.StartScript, req.RequiredPredecessorIDs, req.JoinType, req.RequiredEdgeIDs); err != nil {
		return nil, err
	}
	if nodeType != valueobject.NodeTypeEdge {
		node.PositionX = req.PositionX
		node.PositionY = req.PositionY
	}
	// 执行资源：默认本地（调试）；ssh/k8s 必须引用已登记的资源 ID。
	switch req.RuntimeKind {
	case "", executor.RuntimeKindLocal:
		node.RuntimeKind = executor.RuntimeKindLocal
	case executor.RuntimeKindSSH, executor.RuntimeKindK8s:
		if req.ResourceID == 0 {
			return nil, fmt.Errorf("%s runtime requires resource_id", req.RuntimeKind)
		}
		node.RuntimeKind = req.RuntimeKind
		node.ResourceID = req.ResourceID
	default:
		return nil, fmt.Errorf("unknown runtime kind: %s", req.RuntimeKind)
	}
	if nodeType == valueobject.NodeTypeStart || nodeType == valueobject.NodeTypeEnd {
		sanitizeTerminalNode(node)
	}
	return node, nil
}

func sanitizeTerminalNode(node *entity.Node) {
	switch node.Type {
	case valueobject.NodeTypeStart:
		node.Name = "开始"
	case valueobject.NodeTypeEnd:
		node.Name = "结束"
	}
	node.Description = ""
	node.PreScript = ""
	node.Script = ""
	node.PostScript = ""
	node.Owners = []string{}
	node.Approvers = []string{}
	node.StartMode = valueobject.StartModeAny
	node.StartScript = ""
	node.RequiredPredecessorIDs = []int64{}
	node.RequiredEdgeIDs = []int64{}
}

func applyStartConfig(
	node *entity.Node,
	startMode, startScript string,
	requiredPredecessorIDs []int64,
	legacyJoinType string,
	legacyRequiredEdgeIDs []int64,
) error {
	if !node.IsExecutable() {
		return nil
	}
	mode := startMode
	if mode == "" && legacyJoinType != "" {
		mode = legacyJoinType
	}
	if mode != "" {
		sm := valueobject.StartMode(mode)
		if !sm.Valid() {
			return errors.New("invalid start mode")
		}
		node.StartMode = sm
	}
	node.StartScript = startScript
	if len(requiredPredecessorIDs) > 0 {
		return node.SetRequiredPredecessors(requiredPredecessorIDs)
	}
	if len(legacyRequiredEdgeIDs) > 0 {
		node.RequiredEdgeIDs = legacyRequiredEdgeIDs
	}
	return nil
}

// --- 参数管理 ---

// AddParam 向版本中添加参数定义。
func (s *VersionService) AddParam(ctx context.Context, versionID int64, req dto.CreateFlowParamRequest) (*dto.FlowParamResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if ver.Status == valueobject.TemplateStatusPublished {
		return nil, errors.New("cannot modify a published version")
	}

	paramType := valueobject.ParamType(req.Type)
	param, err := entity.NewFlowParam(ver.TemplateID, versionID, req.Name, paramType)
	if err != nil {
		return nil, err
	}
	param.DefaultValue = req.DefaultValue
	param.Required = req.Required
	param.Description = req.Description

	if err := s.paramRepo.Create(ctx, param); err != nil {
		return nil, err
	}

	resp := dto.ToParamResponse(param)
	return &resp, nil
}

// UpdateParam 更新参数定义。
func (s *VersionService) UpdateParam(ctx context.Context, paramID int64, req dto.UpdateFlowParamRequest) (*dto.FlowParamResponse, error) {
	param, err := s.paramRepo.GetByID(ctx, paramID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		param.Name = req.Name
	}
	if req.Type != "" {
		pt := valueobject.ParamType(req.Type)
		if !pt.Valid() {
			return nil, errors.New("invalid param type")
		}
		param.Type = pt
	}
	param.DefaultValue = req.DefaultValue
	param.Required = req.Required
	param.Description = req.Description

	if err := s.paramRepo.Update(ctx, param); err != nil {
		return nil, err
	}

	resp := dto.ToParamResponse(param)
	return &resp, nil
}

// DeleteParam 删除参数定义。
func (s *VersionService) DeleteParam(ctx context.Context, paramID int64) error {
	return s.paramRepo.Delete(ctx, paramID)
}

// ListParams 列出版本下的所有参数。
func (s *VersionService) ListParams(ctx context.Context, versionID int64) ([]dto.FlowParamResponse, error) {
	ver, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	params, err := s.paramRepo.ListByVersion(ctx, ver.TemplateID, versionID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FlowParamResponse, 0, len(params))
	for _, p := range params {
		items = append(items, dto.ToParamResponse(p))
	}
	return items, nil
}

// --- 触发配置管理 ---

// UpsertTrigger 创建或更新触发配置。
func (s *VersionService) UpsertTrigger(ctx context.Context, templateID int64, req dto.UpsertTriggerConfigRequest) (*dto.TriggerConfigResponse, error) {
	existing, _ := s.triggerRepo.GetByTemplateID(ctx, templateID)
	if existing != nil {
		// 更新。
		existing.Manual = req.Manual
		existing.Webhook = req.Webhook
		existing.Cron = req.Cron
		existing.CronExpr = req.CronExpr
		existing.RequireParams = req.RequireParams
		existing.WebhookSecret = req.WebhookSecret
		if err := s.triggerRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
		resp := dto.ToTriggerConfigResponse(existing)
		return &resp, nil
	}

	// 创建。
	cfg, err := entity.NewTriggerConfig(templateID)
	if err != nil {
		return nil, err
	}
	cfg.Manual = req.Manual
	cfg.Webhook = req.Webhook
	cfg.Cron = req.Cron
	cfg.CronExpr = req.CronExpr
	cfg.RequireParams = req.RequireParams

	if err := s.triggerRepo.Create(ctx, cfg); err != nil {
		return nil, err
	}
	resp := dto.ToTriggerConfigResponse(cfg)
	return &resp, nil
}

// GetTrigger 获取触发配置；不存在时返回默认配置（手动启动开启）。
func (s *VersionService) GetTrigger(ctx context.Context, templateID int64) (*dto.TriggerConfigResponse, error) {
	cfg, err := s.triggerRepo.GetByTemplateID(ctx, templateID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			defaultCfg, createErr := entity.NewTriggerConfig(templateID)
			if createErr != nil {
				return nil, createErr
			}
			defaultCfg.Manual = true
			resp := dto.ToTriggerConfigResponse(defaultCfg)
			return &resp, nil
		}
		return nil, err
	}
	resp := dto.ToTriggerConfigResponse(cfg)
	return &resp, nil
}
