package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appcron "flow/application/cron"
	"flow/application/service"
	"flow/domain/executor"
	"flow/infrastructure/config"
	"flow/infrastructure/lock"
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

	distLock, redisClient, err := lock.NewFromConfig(cfg.Lock, db)
	if err != nil {
		slog.Error("failed to create distributed lock", "error", err)
		panic(err)
	}
	if redisClient != nil {
		defer func() { _ = redisClient.Close() }()
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
	k8sConfigRepo := persistence.NewK8sConfigRepository(db)

	// Executor.
	localExec := executor.NewLocalExecutor()

	// Services.
	resourceSvc := service.NewExecutorResourceService(resourceRepo, k8sConfigRepo)
	k8sConfigSvc := service.NewK8sConfigService(k8sConfigRepo)

	// Scheduler.
	scheduler := service.NewScheduler(localExec, resourceSvc, instRepo, versionRepo, slog, cfg.Runtime.EnvOutDir)

	tplSvc := service.NewFlowTemplateService(tplRepo, versionRepo, nodeRepo, paramRepo, triggerRepo, bindingRepo)
	verSvc := service.NewVersionService(tplRepo, versionRepo, nodeRepo, paramRepo, triggerRepo)
	instSvc := service.NewFlowInstanceService(tplRepo, versionRepo, instRepo, paramRepo, logRepo, fireRepo, scheduler, slog)
	notifySvc := service.NewNotifyService(notifyRepo)

	svcs := &router.Services{
		Template: tplSvc,
		Version:  verSvc,
		Instance: instSvc,
		Notify:   notifySvc,
		Resource:  resourceSvc,
		K8sConfig: k8sConfigSvc,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dynamicScheduler := appcron.NewDynamicScheduler(slog, instSvc, versionRepo)
	leaderSvc, err := appcron.NewLeaderService(distLock, cfg.Lock, dynamicScheduler, slog)
	if err != nil {
		slog.Error("failed to create cron leader service", "error", err)
		panic(err)
	}
	go leaderSvc.Run(ctx)

	engine := router.New(svcs, slog)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	go func() {
		slog.Info("http server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
			panic(err)
		}
	}()

	<-ctx.Done()

	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown failed", "error", err)
	}

	slog.Info("server stopped")
}
