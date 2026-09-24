package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"flow/application/job"
	"flow/infrastructure/config"
	"flow/infrastructure/logger"
	"flow/interfaces/job/runner"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Printf("failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}

	log := logger.New()

	sampleJob := job.NewSampleJob(log)
	r := runner.New(log, sampleJob)

	if cfg.Cron.Enabled {
		if _, err := r.Schedule(cfg.Cron.Schedule, sampleJob); err != nil {
			log.Error("failed to schedule job", "error", err)
			panic(err)
		}
	}

	r.Start()

	// 等待退出信号，实现优雅停机。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("shutting down worker")
	r.Stop()
}
