package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowInstanceService 管理流水线实例的生命周期。
type FlowInstanceService struct {
	tplRepo     repository.FlowTemplateRepository
	versionRepo repository.FlowTemplateVersionRepository
	instRepo    repository.FlowInstanceRepository
	paramRepo   repository.FlowParamRepository
	logRepo     repository.NodeOperationLogRepository
	fireRepo    repository.CronFireRepository
	scheduler   *Scheduler
	logger      *slog.Logger

	// cronLastScan 进程内单分钟扫描去重（多副本间的去重由 cron_fire_logs 唯一约束保证）。
	cronMu       sync.Mutex
	cronLastScan *time.Time
}

// NewFlowInstanceService 构造实例管理服务。
func NewFlowInstanceService(
	tplRepo repository.FlowTemplateRepository,
	versionRepo repository.FlowTemplateVersionRepository,
	instRepo repository.FlowInstanceRepository,
	paramRepo repository.FlowParamRepository,
	logRepo repository.NodeOperationLogRepository,
	fireRepo repository.CronFireRepository,
	scheduler *Scheduler,
	logger *slog.Logger,
) *FlowInstanceService {
	return &FlowInstanceService{
		tplRepo:     tplRepo,
		versionRepo: versionRepo,
		instRepo:    instRepo,
		paramRepo:   paramRepo,
		logRepo:     logRepo,
		fireRepo:    fireRepo,
		scheduler:   scheduler,
		logger:      logger,
	}
}

// RunDueCronInstances 扫描所有开启定时触发的模板，
// 触发 [上一分钟, 本轮分钟] 窗口内到期的实例。由每分钟一次的扫描任务调用。
func (s *FlowInstanceService) RunDueCronInstances(ctx context.Context, now time.Time) {
	minuteStart := now.Truncate(time.Minute)

	// 进程内每个分钟只扫描一次（多副本间的触发去重由 cron_fire_logs 唯一约束保证）。
	s.cronMu.Lock()
	if s.cronLastScan != nil && s.cronLastScan.Equal(minuteStart) {
		s.cronMu.Unlock()
		return
	}
	s.cronLastScan = &minuteStart
	s.cronMu.Unlock()

	// 回看窗口：覆盖 tick 延迟/进程短暂暂停导致的漏扫。
	// 超时的 claimed 凭据由 ReconcileCronTriggers（10 秒对账任务）负责重试。
	lookback := []time.Time{minuteStart.Add(-time.Minute), minuteStart}

	offset := 0
	const pageSize = 100
	for {
		tpls, total, err := s.tplRepo.List(ctx, offset, pageSize)
		if err != nil {
			s.logger.Error("cron scan list templates failed", "error", err)
			return
		}
		for _, tpl := range tpls {
			s.fireCronIfDue(ctx, tpl, lookback)
		}
		offset += pageSize
		if len(tpls) == 0 || offset >= int(total) {
			return
		}
	}
}

// fireCronIfDue 判断单个模板的表达式是否在回看窗口内到期，到期则异步创建实例。
func (s *FlowInstanceService) fireCronIfDue(ctx context.Context, tpl *entity.FlowTemplate, lookback []time.Time) {
	ver, err := s.versionRepo.GetLatestPublishedByTemplate(ctx, tpl.ID)
	if err != nil || !ver.Trigger.Cron || ver.Trigger.CronExpr == "" {
		return
	}
	sched, err := cron.ParseStandard(ver.Trigger.CronExpr)
	if err != nil {
		s.logger.Error("invalid cron expr", "template_id", tpl.ID, "expr", ver.Trigger.CronExpr, "error", err)
		return
	}

	for _, minute := range lookback {
		// 表达式在 minute 触发当且仅当 Next(minute-1s) == minute。
		if !sched.Next(minute.Add(-time.Second)).Equal(minute) {
			continue
		}
		// 数据库唯一约束抢占触发权（claimed 凭据）：多副本/重复扫描下只有一个成功。
		claim, fired, err := s.fireRepo.TryMarkFired(ctx, tpl.ID, ver.ID, minute)
		if err != nil {
			s.logger.Error("mark cron fired failed", "template_id", tpl.ID, "error", err)
			continue
		}
		if !fired {
			continue
		}
		s.logger.Info("cron trigger firing", "template_id", tpl.ID, "expr", ver.Trigger.CronExpr, "fire_minute", minute.Format("15:04"))
		go func(claim *entity.CronFireLog, tpl *entity.FlowTemplate, ver *entity.FlowTemplateVersion) {
			runCtx := context.WithoutCancel(ctx)
			resp, err := s.createAndRun(runCtx, tpl, ver, valueobject.TriggerTypeCron, nil)
			if err != nil {
				// 创建失败：凭据保持 claimed，由对账回收后自动重试。
				s.logger.Error("cron triggered instance failed", "template_id", tpl.ID, "error", err)
				return
			}
			if err := s.fireRepo.MarkFired(runCtx, claim.ID, resp.ID); err != nil {
				s.logger.Error("mark cron fire completed failed", "claim_id", claim.ID, "error", err)
			}
		}(claim, tpl, ver)
	}
}

// Start 手动启动流水线实例（运行最新已发布快照）。
func (s *FlowInstanceService) Start(ctx context.Context, templateID int64, req dto.CreateFlowInstanceRequest) (*dto.FlowInstanceResponse, error) {
	tpl, err := s.tplRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	ver, err := s.versionRepo.GetLatestPublishedByTemplate(ctx, templateID)
	if err != nil {
		return nil, errors.New("template has no published version to run")
	}
	// 以发布快照固化的触发配置为准。
	if !ver.Trigger.Manual {
		return nil, errors.New("manual trigger is disabled for the published version")
	}

	triggerType := valueobject.TriggerType(req.TriggerType)
	if !triggerType.Valid() {
		return nil, errors.New("invalid trigger type")
	}

	return s.createAndRun(ctx, tpl, ver, triggerType, req.Params)
}

// StartWebhook 通过 webhook 启动流水线实例：校验发布快照的 webhook 开关与密钥。
// params 为请求体中的参数（值统一转为字符串）。
func (s *FlowInstanceService) StartWebhook(ctx context.Context, templateID int64, secret string, params map[string]string) (*dto.FlowInstanceResponse, error) {
	tpl, err := s.tplRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	ver, err := s.versionRepo.GetLatestPublishedByTemplate(ctx, templateID)
	if err != nil {
		return nil, errors.New("template has no published version to run")
	}
	// 以发布快照固化的触发配置为准。
	if !ver.Trigger.Webhook {
		return nil, errors.New("webhook trigger is disabled for the published version")
	}
	if ver.Trigger.WebhookSecret != "" && secret != ver.Trigger.WebhookSecret {
		return nil, errors.New("invalid webhook secret")
	}

	return s.createAndRun(ctx, tpl, ver, valueobject.TriggerTypeWebhook, params)
}

// ReconcileCronTriggers 对账任务（每 10 秒执行一次）：
//  1. 重触发超时的 claimed 凭据（进程异常终止/实例创建失败的自愈）；
//  2. 清理已终结（fired）且超过保留期的凭据。
func (s *FlowInstanceService) ReconcileCronTriggers(ctx context.Context, now time.Time) {
	stale, err := s.fireRepo.ClaimStaleForRetry(ctx, now.Add(-10*time.Second))
	if err != nil {
		s.logger.Error("claim stale cron triggers failed", "error", err)
		return
	}
	for _, claim := range stale {
		s.reFireCronClaim(ctx, claim)
	}
	if err := s.fireRepo.DeleteFinishedBefore(ctx, now.Add(-10*time.Minute)); err != nil {
		s.logger.Error("delete finished cron claims failed", "error", err)
	}
}

// reFireCronClaim 重新执行一条超时凭据对应的触发。
// 成功后凭据标记为 fired；失败则保持 claimed（created_at 已顺延），10 秒后再次重试。
func (s *FlowInstanceService) reFireCronClaim(ctx context.Context, claim *entity.CronFireLog) {
	tpl, err := s.tplRepo.GetByID(ctx, claim.TemplateID)
	if err != nil {
		s.logger.Error("cron retry load template failed", "template_id", claim.TemplateID, "error", err)
		return
	}
	ver, err := s.versionRepo.GetByID(ctx, claim.VersionID)
	if err != nil {
		s.logger.Error("cron retry load version failed", "version_id", claim.VersionID, "error", err)
		return
	}
	s.logger.Info("cron retry firing", "template_id", tpl.ID, "fire_minute", claim.FireMinute.Format("15:04"))
	resp, err := s.createAndRun(ctx, tpl, ver, valueobject.TriggerTypeCron, nil)
	if err != nil {
		s.logger.Error("cron retry triggered instance failed", "template_id", tpl.ID, "error", err)
		return
	}
	if err := s.fireRepo.MarkFired(ctx, claim.ID, resp.ID); err != nil {
		s.logger.Error("mark cron retry fired failed", "claim_id", claim.ID, "error", err)
	}
}

// createAndRun 创建实例、补全缺失参数的默认值，并异步启动调度。
func (s *FlowInstanceService) createAndRun(
	ctx context.Context,
	tpl *entity.FlowTemplate,
	ver *entity.FlowTemplateVersion,
	triggerType valueobject.TriggerType,
	params map[string]string,
) (*dto.FlowInstanceResponse, error) {
	inst, err := entity.NewFlowInstance(tpl.ID, ver.ID, ver.Version, triggerType)
	if err != nil {
		return nil, err
	}
	merged := map[string]string{}
	for k, v := range params {
		merged[k] = v
	}
	// 缺失的参数回退默认值：优先用户设置的默认值，否则取类型零值。
	declared, err := s.paramRepo.ListByVersion(ctx, tpl.ID, ver.ID)
	if err != nil {
		return nil, err
	}
	// 必填参数校验：请求中未提供的必填参数直接报错（手动/webhook/定时触发统一拦截）。
	missing := make([]string, 0, len(declared))
	for _, p := range declared {
		if !p.Required {
			continue
		}
		if v, ok := params[p.Name]; !ok || v == "" {
			missing = append(missing, p.Name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("缺少必填参数: %s", strings.Join(missing, ", "))
	}
	for _, p := range declared {
		if _, ok := merged[p.Name]; ok {
			continue
		}
		if p.DefaultValue != "" {
			merged[p.Name] = p.DefaultValue
			continue
		}
		switch p.Type {
		case "number":
			merged[p.Name] = "0"
		case "boolean":
			merged[p.Name] = "false"
		default:
			merged[p.Name] = ""
		}
	}
	inst.SetParams(merged)

	// 注入实例级系统参数：脚本与实例详情均可引用（node_owner 为节点级，调度时单独注入）。
	inst.UpdateEnv(map[string]string{
		"template_id":    fmt.Sprintf("%d", tpl.ID),
		"template_name":  tpl.Name,
		"template_owner": strings.Join(tpl.Owners, ","),
	})

	if err := s.instRepo.Create(ctx, inst); err != nil {
		return nil, err
	}

	if err := inst.Transition(valueobject.FlowInstanceStatusRunning); err != nil {
		return nil, err
	}
	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}

	s.logger.Info("starting flow instance", "instance_id", inst.ID, "template_id", tpl.ID)
	// 异步执行调度：启动请求立即返回，执行进度由实例详情页轮询展示。
	// WithoutCancel 保留请求内的取消信号之外的内容，避免响应结束后脚本被终止。
	go s.runScheduler(context.WithoutCancel(ctx), inst)

	return s.Get(ctx, inst.ID)
}

func (s *FlowInstanceService) runScheduler(ctx context.Context, inst *entity.FlowInstance) {
	var err error
	if inst.Status == valueobject.FlowInstanceStatusRunning {
		// 首次运行或恢复：若已有节点实例则 Continue，否则 Run。
		nis, listErr := s.instRepo.ListNodeInstances(ctx, inst.ID)
		if listErr != nil {
			s.logger.Error("list node instances failed", "instance_id", inst.ID, "error", listErr)
			return
		}
		if len(nis) == 0 {
			err = s.scheduler.Run(ctx, inst)
		} else {
			err = s.scheduler.Continue(ctx, inst)
		}
	} else {
		err = s.scheduler.Continue(ctx, inst)
	}

	if err != nil {
		if errors.Is(err, ErrWorkflowPaused) {
			s.logger.Info("workflow paused waiting for approval", "instance_id", inst.ID)
		} else {
			s.logger.Error("scheduler run failed", "instance_id", inst.ID, "error", err)
		}
	}
	if updateErr := s.instRepo.Update(ctx, inst); updateErr != nil {
		s.logger.Error("update instance status failed", "instance_id", inst.ID, "error", updateErr)
	}
}

// Stop 停止流水线实例（标记为失败）。
func (s *FlowInstanceService) Stop(ctx context.Context, instanceID int64) error {
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.Status == valueobject.FlowInstanceStatusSucceeded {
		return errors.New("cannot stop a succeeded instance")
	}
	if inst.Status == valueobject.FlowInstanceStatusFailed {
		return errors.New("instance already failed")
	}
	if err := inst.Transition(valueobject.FlowInstanceStatusFailed); err != nil {
		return err
	}
	return s.instRepo.Update(ctx, inst)
}

// Get 获取实例详情（含节点实例）。
func (s *FlowInstanceService) Get(ctx context.Context, instanceID int64) (*dto.FlowInstanceResponse, error) {
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	nis, err := s.instRepo.ListNodeInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	inst.NodeInstances = nis
	resp := dto.ToInstanceResponse(inst)
	return &resp, nil
}

// List 列出某个模板下的实例列表。
func (s *FlowInstanceService) List(ctx context.Context, templateID int64, offset, limit int) (*dto.ListFlowInstanceResponse, error) {
	instances, total, err := s.instRepo.ListByTemplate(ctx, templateID, offset, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FlowInstanceResponse, 0, len(instances))
	for _, inst := range instances {
		items = append(items, dto.ToInstanceResponse(inst))
	}
	return &dto.ListFlowInstanceResponse{Items: items, Total: total}, nil
}

// Retry 重试失败的实例。
func (s *FlowInstanceService) Retry(ctx context.Context, instanceID int64) (*dto.FlowInstanceResponse, error) {
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Status != valueobject.FlowInstanceStatusFailed {
		return nil, errors.New("only failed instances can be retried")
	}

	if err := inst.Transition(valueobject.FlowInstanceStatusRetry); err != nil {
		return nil, err
	}
	if err := inst.Transition(valueobject.FlowInstanceStatusRunning); err != nil {
		return nil, err
	}
	inst.FinishedAt = nil
	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}

	// 重置失败的节点实例为重试中：调度器将其视为待执行并重新运行。
	// failed_skipped（失败跳过）视为成功，保持原状态参与下游判定。
	nis, err := s.instRepo.ListNodeInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	for _, ni := range nis {
		if ni.Status != valueobject.NodeInstanceStatusFailed {
			continue
		}
		if err := ni.MarkRetrying(); err != nil {
			continue
		}
		_ = s.instRepo.UpdateNodeInstance(ctx, ni)
	}

	s.logger.Info("retrying flow instance", "instance_id", inst.ID)
	// 异步执行调度：请求立即返回，进度由实例详情页轮询展示。
	go s.runScheduler(context.WithoutCancel(ctx), inst)

	return s.Get(ctx, inst.ID)
}

// Approve 审批通过 manual 节点并继续执行。
func (s *FlowInstanceService) Approve(ctx context.Context, templateID, instanceID, nodeID int64, operator string) (*dto.FlowInstanceResponse, error) {
	inst, n, ni, err := s.loadManualNode(ctx, templateID, instanceID, nodeID)
	if err != nil {
		return nil, err
	}
	if inst.Status != valueobject.FlowInstanceStatusRunning {
		return nil, errors.New("instance is not running")
	}
	if err := s.recordOperationLog(ctx, instanceID, ni.ID, nodeID, entity.NodeActionApprove, operator); err != nil {
		return nil, err
	}
	if err := s.scheduler.ApproveManualNode(ctx, inst, n, ni); err != nil {
		if updateErr := s.instRepo.Update(ctx, inst); updateErr != nil {
			s.logger.Error("update instance failed", "error", updateErr)
		}
		return nil, err
	}
	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}
	if inst.Status == valueobject.FlowInstanceStatusRunning {
		// 异步继续执行剩余节点：审批请求立即返回，进度由实例详情页轮询展示。
		go s.runScheduler(context.WithoutCancel(ctx), inst)
	}
	return s.Get(ctx, inst.ID)
}

// Reject 拒绝 manual 节点，整个流水线失败。
func (s *FlowInstanceService) Reject(ctx context.Context, templateID, instanceID, nodeID int64, operator string) (*dto.FlowInstanceResponse, error) {
	inst, _, ni, err := s.loadManualNode(ctx, templateID, instanceID, nodeID)
	if err != nil {
		return nil, err
	}
	if inst.Status != valueobject.FlowInstanceStatusRunning {
		return nil, errors.New("instance is not running")
	}
	if ni.Status != valueobject.NodeInstanceStatusWaitingApproval {
		return nil, errors.New("node is not waiting for approval")
	}
	if err := s.recordOperationLog(ctx, instanceID, ni.ID, nodeID, entity.NodeActionReject, operator); err != nil {
		return nil, err
	}
	if err := ni.Transition(valueobject.NodeInstanceStatusFailed); err != nil {
		return nil, err
	}
	if err := s.instRepo.UpdateNodeInstance(ctx, ni); err != nil {
		return nil, err
	}
	if err := inst.Transition(valueobject.FlowInstanceStatusFailed); err != nil {
		return nil, err
	}
	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}
	return s.Get(ctx, inst.ID)
}

// recordOperationLog 记录人工节点操作日志；日志失败不阻断主流程。
func (s *FlowInstanceService) recordOperationLog(ctx context.Context, instanceID, nodeInstanceID, nodeID int64, action, operator string) error {
	log, err := entity.NewNodeOperationLog(instanceID, nodeInstanceID, nodeID, action, operator)
	if err != nil {
		return err
	}
	if err := s.logRepo.Create(ctx, log); err != nil {
		s.logger.Error("record node operation log failed", "instance_id", instanceID, "node_id", nodeID, "error", err)
	}
	return nil
}

// ListNodeOperationLogs 查询某实例某节点的人工操作日志（倒序）。
func (s *FlowInstanceService) ListNodeOperationLogs(ctx context.Context, instanceID, nodeID int64) ([]dto.NodeOperationLogResponse, error) {
	logs, err := s.logRepo.ListByInstanceNode(ctx, instanceID, nodeID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.NodeOperationLogResponse, 0, len(logs))
	for _, l := range logs {
		items = append(items, dto.NodeOperationLogResponse{
			ID:        l.ID,
			Operator:  l.Operator,
			Action:    l.Action,
			CreatedAt: l.CreatedAt,
		})
	}
	return items, nil
}

// SkipNode 手动跳过运行中的节点：终止其脚本执行并标记为跳过，调度继续后续节点。
func (s *FlowInstanceService) SkipNode(ctx context.Context, templateID, instanceID, nodeID int64) (*dto.FlowInstanceResponse, error) {
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.TemplateID != templateID {
		return nil, errors.New("instance does not belong to template")
	}
	nis, err := s.instRepo.ListNodeInstances(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	var ni *entity.NodeInstance
	for _, item := range nis {
		if item.NodeID == nodeID {
			ni = item
			break
		}
	}
	if ni == nil {
		return nil, repository.ErrNotFound
	}
	if err := s.scheduler.SkipNode(ctx, ni); err != nil {
		return nil, err
	}
	s.logger.Info("node skipped manually", "instance_id", instanceID, "node_id", nodeID)
	return s.Get(ctx, instanceID)
}

func (s *FlowInstanceService) loadManualNode(ctx context.Context, templateID, instanceID, nodeID int64) (*entity.FlowInstance, *entity.Node, *entity.NodeInstance, error) {
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, nil, nil, err
	}
	if inst.TemplateID != templateID {
		return nil, nil, nil, errors.New("instance does not belong to template")
	}
	ver, err := s.versionRepo.GetByID(ctx, inst.TemplateVersionID)
	if err != nil {
		return nil, nil, nil, err
	}
	n := ver.GetNode(nodeID)
	if n == nil {
		return nil, nil, nil, repository.ErrNotFound
	}
	if n.Type != valueobject.NodeTypeManual {
		return nil, nil, nil, errors.New("node is not a manual node")
	}
	nis, err := s.instRepo.ListNodeInstances(ctx, instanceID)
	if err != nil {
		return nil, nil, nil, err
	}
	var ni *entity.NodeInstance
	for _, item := range nis {
		if item.NodeID == nodeID {
			ni = item
			break
		}
	}
	if ni == nil {
		return nil, nil, nil, repository.ErrNotFound
	}
	return inst, n, ni, nil
}
