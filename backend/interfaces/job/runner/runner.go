package runner

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"

	"flow/application/job"
)

// Runner 封装 robfig/cron 调度器，支持定时与手工触发。
type Runner struct {
	cron   *cron.Cron
	logger *slog.Logger
	jobs   []job.Job
}

// New 构造调度器并注册传入的任务。
func New(logger *slog.Logger, jobs ...job.Job) *Runner {
	return &Runner{
		cron:   cron.New(),
		logger: logger,
		jobs:   jobs,
	}
}

// Start 启动调度器。
func (r *Runner) Start() {
	r.cron.Start()
	r.logger.Info("cron runner started")
}

// Stop 停止调度器。
func (r *Runner) Stop() {
	ctx := r.cron.Stop()
	<-ctx.Done()
	r.logger.Info("cron runner stopped")
}

// Schedule 按 crontab 表达式注册一个任务，返回任务入口 ID。
func (r *Runner) Schedule(spec string, j job.Job) (cron.EntryID, error) {
	return r.cron.AddFunc(spec, func() {
		r.run(j)
	})
}

// RunAll 手工触发所有已注册任务。
func (r *Runner) RunAll(ctx context.Context) {
	for _, j := range r.jobs {
		r.runWithContext(ctx, j)
	}
}

// RunByName 手工触发指定名称的任务。
func (r *Runner) RunByName(ctx context.Context, name string) bool {
	for _, j := range r.jobs {
		if j.Name() == name {
			r.runWithContext(ctx, j)
			return true
		}
	}
	return false
}

// run 在无上下文的情况下执行任务。
func (r *Runner) run(j job.Job) {
	r.runWithContext(context.Background(), j)
}

// runWithContext 执行任务并记录错误日志。
func (r *Runner) runWithContext(ctx context.Context, j job.Job) {
	if err := j.Run(ctx); err != nil {
		r.logger.Error("job failed", "job", j.Name(), "error", err)
		return
	}
	r.logger.Info("job finished", "job", j.Name())
}
