package service

import (
	"context"
	"errors"
	"fmt"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowTemplateService 是流水线模板的应用服务，负责编排领域对象与仓储。
type FlowTemplateService struct {
	repo          repository.FlowTemplateRepository
	versionRepo   repository.FlowTemplateVersionRepository
	nodeRepo      repository.NodeRepository
	paramRepo     repository.FlowParamRepository
	triggerRepo   repository.TriggerConfigRepository
	bindingRepo   repository.FlowNotifyBindingRepository
}

// NewFlowTemplateService 构造应用服务。
func NewFlowTemplateService(
	repo repository.FlowTemplateRepository,
	versionRepo repository.FlowTemplateVersionRepository,
	nodeRepo repository.NodeRepository,
	paramRepo repository.FlowParamRepository,
	triggerRepo repository.TriggerConfigRepository,
	bindingRepo repository.FlowNotifyBindingRepository,
) *FlowTemplateService {
	return &FlowTemplateService{
		repo:        repo,
		versionRepo: versionRepo,
		nodeRepo:    nodeRepo,
		paramRepo:   paramRepo,
		triggerRepo: triggerRepo,
		bindingRepo: bindingRepo,
	}
}

// Create 创建流水线模板，同时创建初始草稿版本。
func (s *FlowTemplateService) Create(ctx context.Context, req dto.CreateFlowTemplateRequest) (*dto.FlowTemplateResponse, error) {
	tpl, err := entity.NewFlowTemplate(req.Name)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, tpl); err != nil {
		return nil, err
	}
	ver, err := entity.NewFlowTemplateVersion(tpl.ID, tpl.Version)
	if err != nil {
		return nil, err
	}
	if err := s.versionRepo.Create(ctx, ver); err != nil {
		return nil, err
	}
	resp := dto.ToTemplateResponse(tpl)
	return &resp, nil
}

// publishedTriggerOf 返回最新发布快照及其触发配置；未发布过返回 false。
func (s *FlowTemplateService) publishedTriggerOf(ctx context.Context, tpl *entity.FlowTemplate) (entity.TriggerConfig, int64, bool) {
	ver, err := s.versionRepo.GetLatestPublishedByTemplate(ctx, tpl.ID)
	if err != nil || ver == nil {
		return entity.TriggerConfig{}, 0, false
	}
	return ver.Trigger, ver.ID, true
}

// fillTriggerFields 把发布快照的触发信息填充到模板响应。
func fillTriggerFields(resp *dto.FlowTemplateResponse, trigger entity.TriggerConfig, versionID int64, published bool) {
	if !published {
		return
	}
	resp.ManualTrigger = trigger.Manual
	resp.CronEnabled = trigger.Cron
	resp.CronExpr = trigger.CronExpr
	resp.PublishedVersionID = versionID
}

// Get 按 ID 查询模板。
func (s *FlowTemplateService) Get(ctx context.Context, id int64) (*dto.FlowTemplateResponse, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := dto.ToTemplateResponse(tpl)
	trigger, verID, published := s.publishedTriggerOf(ctx, tpl)
	fillTriggerFields(&resp, trigger, verID, published)
	return &resp, nil
}

// List 分页查询模板。
func (s *FlowTemplateService) List(ctx context.Context, offset, limit int) (*dto.ListFlowTemplateResponse, error) {
	tpls, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FlowTemplateResponse, 0, len(tpls))
	for _, tpl := range tpls {
		resp := dto.ToTemplateResponse(tpl)
		trigger, verID, published := s.publishedTriggerOf(ctx, tpl)
		fillTriggerFields(&resp, trigger, verID, published)
		items = append(items, resp)
	}
	return &dto.ListFlowTemplateResponse{Items: items, Total: total}, nil
}

// Update 更新模板元数据。
func (s *FlowTemplateService) Update(ctx context.Context, id int64, req dto.UpdateFlowTemplateRequest) (*dto.FlowTemplateResponse, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		newTpl, err := entity.NewFlowTemplate(req.Name)
		if err != nil {
			return nil, err
		}
		tpl.Name = newTpl.Name
	}
	if req.Description != "" {
		tpl.Description = req.Description
	}
	if req.Owners != nil {
		tpl.Owners = req.Owners
	}
	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, err
	}
	resp := dto.ToTemplateResponse(tpl)
	trigger, verID, published := s.publishedTriggerOf(ctx, tpl)
	fillTriggerFields(&resp, trigger, verID, published)
	return &resp, nil
}

// Publish 发布模板：把当前草稿固化为一条新的已发布版本（类似 git 打 tag），
// 草稿保留继续编辑。草稿按状态查找（不依赖版本号推断），
// 发布后 current_version = 快照版本号 + 1，实例始终运行最新已发布快照。
func (s *FlowTemplateService) Publish(ctx context.Context, id int64) (*dto.FlowTemplateResponse, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 按状态找草稿，兼容历史数据中版本号错位的情况。
	vers, err := s.versionRepo.ListByTemplate(ctx, tpl.ID)
	if err != nil {
		return nil, err
	}
	var draftRef *entity.FlowTemplateVersion
	maxPublished := 0
	for _, v := range vers {
		if v.Status == valueobject.TemplateStatusDraft {
			if draftRef == nil || v.Version > draftRef.Version {
				draftRef = &entity.FlowTemplateVersion{ID: v.ID, Version: v.Version}
			}
		} else if v.Version > maxPublished {
			maxPublished = v.Version
		}
	}
	if draftRef == nil {
		return nil, errors.New("no draft to publish")
	}
	draftVer, err := s.versionRepo.GetByID(ctx, draftRef.ID)
	if err != nil {
		return nil, err
	}

	// 快照版本号：优先沿用草稿编号，若与历史已发布版本冲突则顺延。
	snapshotNumber := draftVer.Version
	if snapshotNumber <= maxPublished {
		snapshotNumber = maxPublished + 1
	}
	snapshot, err := entity.NewFlowTemplateVersion(tpl.ID, snapshotNumber)
	if err != nil {
		return nil, err
	}
	if err := snapshot.Publish(); err != nil {
		return nil, err
	}
	// 触发配置快照：以保存时的草稿触发配置为准。
	if cfg, cfgErr := s.triggerRepo.GetByTemplateID(ctx, tpl.ID); cfgErr == nil && cfg != nil {
		snapshot.Trigger = *cfg
	} else {
		snapshot.Trigger = entity.TriggerConfig{Manual: true}
	}
	if err := s.versionRepo.Create(ctx, snapshot); err != nil {
		return nil, err
	}
	if err := copyVersionContent(ctx, s.nodeRepo, s.paramRepo, draftVer, snapshot); err != nil {
		return nil, err
	}

	// 草稿保留继续编辑，版本号顺延避免与快照冲突。
	draftVer.Version = snapshotNumber + 1
	if err := s.versionRepo.Update(ctx, draftVer); err != nil {
		return nil, err
	}

	if err := tpl.PublishSnapshot(snapshotNumber); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, err
	}
	resp := dto.ToTemplateResponse(tpl)
	fillTriggerFields(&resp, snapshot.Trigger, snapshot.ID, true)
	return &resp, nil
}

// Delete 软删除模板。
func (s *FlowTemplateService) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// Clone 克隆模板：仅生成新模板 ID，名称及内容与原模板一致。
func (s *FlowTemplateService) Clone(ctx context.Context, id int64) (*dto.FlowTemplateResponse, error) {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newTpl, err := entity.NewFlowTemplate(src.Name)
	if err != nil {
		return nil, err
	}
	newTpl.Owners = append([]string(nil), src.Owners...)
	newTpl.Description = src.Description
	if err := s.repo.Create(ctx, newTpl); err != nil {
		return nil, err
	}

	newVer, err := entity.NewFlowTemplateVersion(newTpl.ID, newTpl.Version)
	if err != nil {
		return nil, err
	}
	if err := s.versionRepo.Create(ctx, newVer); err != nil {
		return nil, err
	}

	// 复制当前内容版本（优先草稿，否则最新已发布）。
	vers, err := s.versionRepo.ListByTemplate(ctx, src.ID)
	if err != nil {
		return nil, err
	}
	contentVer, err := pickContentVersion(vers)
	if err != nil {
		return nil, err
	}
	srcVer, err := s.versionRepo.GetByID(ctx, contentVer.ID)
	if err != nil {
		return nil, err
	}

	idMap := make(map[int64]int64)
	for _, n := range srcVer.Nodes {
		if n.IsEdge() {
			continue
		}
		nodeType := n.Type
		if nodeType == valueobject.NodeTypeJoin {
			nodeType = valueobject.NodeTypeTask
		}
		copyNode, err := entity.NewNode(newTpl.ID, newVer.ID, nodeType, n.Name)
		if err != nil {
			return nil, err
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
		if err := s.nodeRepo.Create(ctx, copyNode); err != nil {
			return nil, err
		}
		idMap[n.ID] = copyNode.ID
	}
	for _, n := range srcVer.Nodes {
		if !n.IsEdge() {
			continue
		}
		copyNode, err := entity.NewNode(newTpl.ID, newVer.ID, n.Type, n.Name)
		if err != nil {
			return nil, err
		}
		fromID := idMap[n.FromNodeID]
		toID := idMap[n.ToNodeID]
		if err := copyNode.SetEdge(fromID, toID); err != nil {
			return nil, err
		}
		if err := s.nodeRepo.Create(ctx, copyNode); err != nil {
			return nil, err
		}
		idMap[n.ID] = copyNode.ID
	}

	for _, n := range srcVer.Nodes {
		if len(n.RequiredPredecessorIDs) == 0 && len(n.RequiredEdgeIDs) == 0 {
			continue
		}
		copyID := idMap[n.ID]
		copyNode, err := s.nodeRepo.GetByID(ctx, copyID)
		if err != nil {
			return nil, err
		}
		if len(n.RequiredPredecessorIDs) > 0 {
			mapped := make([]int64, 0, len(n.RequiredPredecessorIDs))
			for _, predID := range n.RequiredPredecessorIDs {
				mapped = append(mapped, idMap[predID])
			}
			if err := copyNode.SetRequiredPredecessors(mapped); err != nil {
				return nil, err
			}
		} else {
			mapped := make([]int64, 0, len(n.RequiredEdgeIDs))
			for _, edgeID := range n.RequiredEdgeIDs {
				mapped = append(mapped, idMap[edgeID])
			}
			copyNode.RequiredEdgeIDs = mapped
		}
		if err := s.nodeRepo.Update(ctx, copyNode); err != nil {
			return nil, err
		}
	}

	params, err := s.paramRepo.ListByVersion(ctx, src.ID, srcVer.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range params {
		copyParam, err := entity.NewFlowParam(newTpl.ID, newVer.ID, p.Name, p.Type)
		if err != nil {
			return nil, err
		}
		copyParam.DefaultValue = p.DefaultValue
		copyParam.Required = p.Required
		copyParam.Description = p.Description
		if err := s.paramRepo.Create(ctx, copyParam); err != nil {
			return nil, err
		}
	}

	if trigger, err := s.triggerRepo.GetByTemplateID(ctx, src.ID); err == nil && trigger != nil {
		cfg, err := entity.NewTriggerConfig(newTpl.ID)
		if err != nil {
			return nil, err
		}
		cfg.Manual = trigger.Manual
		cfg.Webhook = trigger.Webhook
		cfg.Cron = trigger.Cron
		cfg.CronExpr = trigger.CronExpr
		cfg.RequireParams = trigger.RequireParams
		if err := s.triggerRepo.Create(ctx, cfg); err != nil {
			return nil, err
		}
	}

	bindings, err := s.bindingRepo.ListByTemplate(ctx, src.ID)
	if err != nil {
		return nil, err
	}
	if len(bindings) > 0 {
		newBindings := make([]*entity.FlowNotifyBinding, 0, len(bindings))
		for _, b := range bindings {
			nb, err := entity.NewFlowNotifyBinding(newTpl.ID, b.NotifyTemplateID)
			if err != nil {
				return nil, err
			}
			if b.NodeID != 0 {
				mapped, ok := idMap[b.NodeID]
				if !ok {
					continue
				}
				nb.NodeID = mapped
			}
			nb.Timings = append([]valueobject.NotifyTiming(nil), b.Timings...)
			nb.ExtraContent = b.ExtraContent
			newBindings = append(newBindings, nb)
		}
		if err := s.bindingRepo.ReplaceByTemplate(ctx, newTpl.ID, newBindings); err != nil {
			return nil, err
		}
	}

	resp := dto.ToTemplateResponse(newTpl)
	return &resp, nil
}

// GetNotifyBindings 获取模板的通知绑定。
func (s *FlowTemplateService) GetNotifyBindings(ctx context.Context, templateID int64) ([]dto.FlowNotifyBindingResponse, error) {
	if _, err := s.repo.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	bindings, err := s.bindingRepo.ListByTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FlowNotifyBindingResponse, 0, len(bindings))
	for _, b := range bindings {
		items = append(items, dto.ToNotifyBindingResponse(b))
	}
	return items, nil
}

// ReplaceNotifyBindings 替换模板的通知绑定。
func (s *FlowTemplateService) ReplaceNotifyBindings(ctx context.Context, templateID int64, req dto.ReplaceFlowNotifyBindingsRequest) ([]dto.FlowNotifyBindingResponse, error) {
	if _, err := s.repo.GetByID(ctx, templateID); err != nil {
		return nil, err
	}
	bindings := make([]*entity.FlowNotifyBinding, 0, len(req.Bindings))
	for _, item := range req.Bindings {
		b, err := entity.NewFlowNotifyBinding(templateID, item.NotifyTemplateID)
		if err != nil {
			return nil, err
		}
		b.NodeID = item.NodeID
		b.ExtraContent = item.ExtraContent
		for _, t := range item.Timings {
			timing := valueobject.NotifyTiming(t)
			if !timing.Valid() {
				return nil, fmt.Errorf("invalid notify timing: %s", t)
			}
			if err := b.AddTiming(timing); err != nil {
				return nil, err
			}
		}
		bindings = append(bindings, b)
	}
	if err := s.bindingRepo.ReplaceByTemplate(ctx, templateID, bindings); err != nil {
		return nil, err
	}
	return s.GetNotifyBindings(ctx, templateID)
}
