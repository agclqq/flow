package entity

import "time"

// CronFireLog 记录定时触发的抢占标记。
// (template_id, fire_minute) 唯一：多副本/重复扫描下只有一个能插入成功，
// 以数据库唯一约束实现定时触发的分布式去重。
// 抢占状态：claimed = 已抢占待创建实例；fired = 实例已创建；exhausted = 重试次数用尽。
const (
	CronFireStatusClaimed   = "claimed"
	CronFireStatusFired     = "fired"
	CronFireStatusExhausted = "exhausted"
)

// MaxCronClaimRetries 重试创建实例的最大次数（不含首次抢占）。
const MaxCronClaimRetries = 3

type CronFireLog struct {
	ID         int64
	TemplateID int64
	// FireMinute 触发所属分钟（截断到分钟）。
	FireMinute time.Time
	// VersionID 触发时使用的发布快照版本 ID（重试时恢复上下文）。
	VersionID int64
	// Status 抢占状态：claimed / fired / exhausted。
	Status     string
	// RetryCount 已重试次数。
	RetryCount int
	// InstanceID 创建的实例 ID（fired 后回填；claimed 阶段为 0）。
	InstanceID int64
	CreatedAt  time.Time
}
