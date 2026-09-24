package job

import (
	"context"
	"log/slog"
	"time"
)

// Job 定义可被 cron 调度或手工触发的任务契约。
type Job interface {
	// Name 返回任务名称，用于日志与调度标识。
	Name() string
	// Run 执行任务逻辑。
	Run(ctx context.Context) error
}

// SampleJob 是一个示例任务，演示定时/手工调度的完整形态。
type SampleJob struct {
	logger *slog.Logger
}

// NewSampleJob 构造示例任务。
func NewSampleJob(logger *slog.Logger) *SampleJob {
	return &SampleJob{logger: logger}
}

// Name 返回任务名称。
func (j *SampleJob) Name() string {
	return "sample_job"
}

// Run 执行任务逻辑，此处仅记录一条日志作为示例。
func (j *SampleJob) Run(ctx context.Context) error {
	j.logger.Info("sample job running", "job", j.Name(), "time", time.Now().Format(time.RFC3339))
	return nil
}
