package cron

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"flow/application/service"
	"flow/domain/repository"
)

type scheduledRule struct {
	versionID int64
	expr      string
	entryID   cron.EntryID
}

// DynamicScheduler 在 Leader 进程内按模板维护 robfig/cron 条目。
type DynamicScheduler struct {
	logger      *slog.Logger
	instSvc     *service.FlowInstanceService
	versionRepo repository.FlowTemplateVersionRepository
	cron        *cron.Cron
	mu          sync.Mutex
	rules       map[int64]scheduledRule
	started     bool
}

// NewDynamicScheduler 构造动态调度器。
func NewDynamicScheduler(
	logger *slog.Logger,
	instSvc *service.FlowInstanceService,
	versionRepo repository.FlowTemplateVersionRepository,
) *DynamicScheduler {
	return &DynamicScheduler{
		logger:      logger,
		instSvc:     instSvc,
		versionRepo: versionRepo,
		cron:        cron.New(),
		rules:       make(map[int64]scheduledRule),
	}
}

// Start 启动 cron 引擎、全量同步规则，并注册对账与规则刷新任务。
func (s *DynamicScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	s.cron.Start()
	s.syncRules(ctx)
	s.instSvc.RunDueCronInstances(ctx, time.Now())

	if _, err := s.cron.AddFunc("@every 10s", func() {
		s.instSvc.ReconcileCronTriggers(context.Background(), time.Now())
	}); err != nil {
		s.logger.Error("schedule cron reconcile failed", "error", err)
	}
	if _, err := s.cron.AddFunc("@every 30s", func() {
		s.syncRules(context.Background())
	}); err != nil {
		s.logger.Error("schedule cron rule sync failed", "error", err)
	}

	s.logger.Info("dynamic cron scheduler started")
}

// Stop 停止 cron 并清空内存规则。
func (s *DynamicScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	s.rules = make(map[int64]scheduledRule)
	s.cron = cron.New()
	s.started = false
	s.logger.Info("dynamic cron scheduler stopped")
}

func (s *DynamicScheduler) syncRules(ctx context.Context) {
	desired, err := s.versionRepo.ListPublishedCronTriggers(ctx)
	if err != nil {
		s.logger.Error("sync cron rules failed", "error", err)
		return
	}

	desiredMap := make(map[int64]scheduledRule, len(desired))
	for _, rule := range desired {
		desiredMap[rule.TemplateID] = scheduledRule{
			versionID: rule.VersionID,
			expr:      rule.CronExpr,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for templateID, current := range s.rules {
		want, ok := desiredMap[templateID]
		if !ok || want.versionID != current.versionID || want.expr != current.expr {
			s.cron.Remove(current.entryID)
			delete(s.rules, templateID)
		}
	}

	for templateID, want := range desiredMap {
		current, ok := s.rules[templateID]
		if ok && current.versionID == want.versionID && current.expr == want.expr {
			continue
		}
		if ok {
			s.cron.Remove(current.entryID)
			delete(s.rules, templateID)
		}

		tid := templateID
		entryID, err := s.cron.AddFunc(want.expr, func() {
			s.instSvc.FireCronForTemplate(context.Background(), tid, time.Now())
		})
		if err != nil {
			s.logger.Error("add cron rule failed", "template_id", templateID, "expr", want.expr, "error", err)
			continue
		}
		s.rules[templateID] = scheduledRule{
			versionID: want.versionID,
			expr:      want.expr,
			entryID:   entryID,
		}
		s.logger.Info("cron rule registered", "template_id", templateID, "expr", want.expr, "version_id", want.versionID)
	}
}
