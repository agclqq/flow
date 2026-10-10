package cron

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"flow/domain/distributedlock"
	"flow/infrastructure/config"
)

const maxRenewFailures = 3

// LeaderService 抢分布式锁后作为唯一 Leader 运行动态 cron。
type LeaderService struct {
	lock          distributedlock.Lock
	lockType      string
	ttl           time.Duration
	renewInterval time.Duration
	scheduler     *DynamicScheduler
	logger        *slog.Logger
	acquireWait   time.Duration
}

// NewLeaderService 构造 Leader 调度服务。
func NewLeaderService(
	lock distributedlock.Lock,
	cfg config.LockConfig,
	scheduler *DynamicScheduler,
	logger *slog.Logger,
) (*LeaderService, error) {
	lockType := cfg.CronLeaderType
	if lockType == "" {
		lockType = "cron_scheduler_leader"
	}
	ttl, err := time.ParseDuration(cfg.TTL)
	if err != nil || ttl <= 0 {
		ttl = 60 * time.Second
	}
	renew, err := time.ParseDuration(cfg.RenewInterval)
	if err != nil || renew <= 0 {
		renew = ttl / 3
	}
	return &LeaderService{
		lock:          lock,
		lockType:      lockType,
		ttl:           ttl,
		renewInterval: renew,
		scheduler:     scheduler,
		logger:        logger,
		acquireWait:   3 * time.Second,
	}, nil
}

// Run 在 ctx 取消前循环抢锁；成为 Leader 时运行动态调度直至丢锁或 ctx 结束。
func (l *LeaderService) Run(ctx context.Context) {
	holderKey := newHolderKey()
	l.logger.Info("cron leader loop started", "holder_key", holderKey)

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("cron leader loop stopped")
			return
		default:
		}

		acquired, err := l.lock.TryAcquire(ctx, l.lockType, holderKey, l.ttl)
		if err != nil {
			l.logger.Error("cron leader acquire failed", "error", err)
			l.sleepOrDone(ctx, l.acquireWait)
			continue
		}
		if !acquired {
			l.sleepOrDone(ctx, l.acquireWait)
			continue
		}

		handle := distributedlock.Handle{Type: l.lockType, HolderKey: holderKey}
		l.logger.Info("became cron scheduler leader", "holder_key", holderKey)
		l.runLeaderSession(ctx, handle)
	}
}

func (l *LeaderService) runLeaderSession(parent context.Context, handle distributedlock.Handle) {
	sessionCtx, cancel := context.WithCancel(parent)
	defer cancel()

	l.scheduler.Start(sessionCtx)
	defer l.scheduler.Stop()

	renewFailures := 0
	ticker := time.NewTicker(l.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-parent.Done():
			_ = l.lock.Release(context.Background(), handle)
			return
		case <-ticker.C:
			ok, err := l.lock.Renew(sessionCtx, handle, l.ttl)
			if err != nil {
				l.logger.Error("cron leader renew failed", "error", err)
				ok = false
			}
			if !ok {
				renewFailures++
				if renewFailures >= maxRenewFailures {
					l.logger.Warn("lost cron scheduler leader lock")
					_ = l.lock.Release(context.Background(), handle)
					return
				}
				continue
			}
			renewFailures = 0
		}
	}
}

func (l *LeaderService) sleepOrDone(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func newHolderKey() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().UnixNano())
}
