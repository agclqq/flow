package job

import (
	"context"
	"log/slog"
	"time"

	"flow/application/service"
)

// CronReconcileJob 每 10 秒执行一次定时触发的对账：
// 重触发超时的 claimed 凭据（自愈），并清理已终结的超龄凭据。
type CronReconcileJob struct {
	svc    *service.FlowInstanceService
	logger *slog.Logger
}

// NewCronReconcileJob 构造对账任务。
func NewCronReconcileJob(svc *service.FlowInstanceService, logger *slog.Logger) *CronReconcileJob {
	return &CronReconcileJob{svc: svc, logger: logger}
}

// Name 返回任务名称。
func (j *CronReconcileJob) Name() string {
	return "cron_reconcile_job"
}

// Run 执行一轮对账。
func (j *CronReconcileJob) Run(ctx context.Context) error {
	j.svc.ReconcileCronTriggers(ctx, time.Now())
	return nil
}
