package runner

import (
	"context"
	"testing"

	"flow/infrastructure/logger"
)

// countingJob 是一个可计数的任务实现，用于验证调度触发。
type countingJob struct {
	name  string
	count int
}

func (j *countingJob) Name() string { return j.name }

func (j *countingJob) Run(ctx context.Context) error {
	j.count++
	return nil
}

func TestRunnerRunByName(t *testing.T) {
	j := &countingJob{name: "count"}
	r := New(logger.New(), j)

	found := r.RunByName(context.Background(), "count")
	if !found {
		t.Fatal("expected job to be found")
	}
	if j.count != 1 {
		t.Fatalf("expected count 1, got %d", j.count)
	}

	found = r.RunByName(context.Background(), "missing")
	if found {
		t.Fatal("expected missing job not found")
	}
}

func TestRunnerRunAll(t *testing.T) {
	j1 := &countingJob{name: "a"}
	j2 := &countingJob{name: "b"}
	r := New(logger.New(), j1, j2)

	r.RunAll(context.Background())

	if j1.count != 1 || j2.count != 1 {
		t.Fatalf("expected both jobs run once, got %d and %d", j1.count, j2.count)
	}
}
