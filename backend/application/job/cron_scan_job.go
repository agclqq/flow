package job

import (
	"context"
	"log/slog"
	"time"

	"flow/application/service"
)

// CronScanJob 每分钟扫描一次开启定时触发的模板，
// 触发本轮分钟到期的实例（触发配置来自各模板最新发布快照）。
type CronScanJob struct {
	svc    *service.FlowInstanceService
	logger *slog.Logger
}

// NewCronScanJob 构造定时扫描任务。
func NewCronScanJob(svc *service.FlowInstanceService, logger *slog.Logger) *CronScanJob {
	return &CronScanJob{svc: svc, logger: logger}
}

// Name 返回任务名称。
func (j *CronScanJob) Name() string {
	return "cron_scan_job"
}

// Run 执行一轮扫描。建议以 @every 1m 注册，保证分钟级触发精度。
func (j *CronScanJob) Run(ctx context.Context) error {
	j.svc.RunDueCronInstances(ctx, time.Now())
	return nil
}
