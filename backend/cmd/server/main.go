package main

import (
	"fmt"
	"log"

	"flow/application/job"
	"flow/application/service"
	"flow/domain/executor"
	"flow/interfaces/job/runner"
	"flow/infrastructure/config"
	"flow/infrastructure/logger"
	"flow/infrastructure/persistence"
	"flow/interfaces/http/router"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Printf("failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}

	slog := logger.New()

	db, err := persistence.NewDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		panic(err)
	}

	// Repositories.
	tplRepo := persistence.NewFlowTemplateRepository(db)
	versionRepo := persistence.NewFlowTemplateVersionRepository(db)
	nodeRepo := persistence.NewNodeRepository(db)
	paramRepo := persistence.NewFlowParamRepository(db)
	triggerRepo := persistence.NewTriggerConfigRepository(db)
	instRepo := persistence.NewFlowInstanceRepository(db)
	notifyRepo := persistence.NewNotifyTemplateRepository(db)
	bindingRepo := persistence.NewFlowNotifyBindingRepository(db)
	resourceRepo := persistence.NewExecutorResourceRepository(db)
	logRepo := persistence.NewNodeOperationLogRepository(db)
	fireRepo := persistence.NewCronFireRepository(db)

	// Executor.
	localExec := executor.NewLocalExecutor()

	// Services.
	resourceSvc := service.NewExecutorResourceService(resourceRepo)

	// Scheduler.
	scheduler := service.NewScheduler(localExec, resourceSvc, instRepo, versionRepo, slog)

	tplSvc := service.NewFlowTemplateService(tplRepo, versionRepo, nodeRepo, paramRepo, triggerRepo, bindingRepo)
	verSvc := service.NewVersionService(tplRepo, versionRepo, nodeRepo, paramRepo, triggerRepo)
	instSvc := service.NewFlowInstanceService(tplRepo, versionRepo, instRepo, paramRepo, logRepo, fireRepo, scheduler, slog)
	notifySvc := service.NewNotifyService(notifyRepo)

	// 定时调度：每分钟扫描一次开启 cron 的模板并触发到期实例。
	cronScanJob := job.NewCronScanJob(instSvc, slog)
	cronReconcileJob := job.NewCronReconcileJob(instSvc, slog)
	cronRunner := runner.New(slog, cronScanJob, cronReconcileJob)
	// 对齐分钟边界（每分钟第 0 秒触发），保证定时实例的启动延迟最小。
	if _, err := cronRunner.Schedule("* * * * *", cronScanJob); err != nil {
		slog.Error("schedule cron scan job failed", "error", err)
	}
	if _, err := cronRunner.Schedule("@every 10s", cronReconcileJob); err != nil {
		slog.Error("schedule cron reconcile job failed", "error", err)
	}

	svcs := &router.Services{
		Template: tplSvc,
		Version:  verSvc,
		Instance: instSvc,
		Notify:   notifySvc,
		Resource: resourceSvc,
	}

	cronRunner.Start()

	r := router.New(svcs, slog)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("http server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("http server stopped", "error", err)
		panic(err)
	}
}
