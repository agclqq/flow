package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// TriggerConfig 是流水线的触发配置，描述支持的触发方式及是否要求入参。
type TriggerConfig struct {
	ID         int64
	TemplateID int64
	// Manual 是否支持页面按钮触发。
	Manual bool
	// Webhook 是否支持 webhook 触发。
	Webhook bool
	// Cron 是否支持 crontab 定时触发。
	Cron bool
	// CronExpr 定时触发表达式。
	CronExpr string
	// RequireParams 已废弃：缺失参数一律回退默认值。
	RequireParams bool
	// WebhookSecret webhook 触发密钥（发布时随快照固化）。
	WebhookSecret string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewTriggerConfig 创建触发配置。
// ID 由数据库自增生成。
func NewTriggerConfig(templateID int64) (*TriggerConfig, error) {
	if templateID == 0 {
		return nil, errors.New("template id must not be zero")
	}
	now := time.Now()
	return &TriggerConfig{
		TemplateID: templateID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// HasTrigger 判断是否支持指定触发方式。
func (c *TriggerConfig) HasTrigger(t valueobject.TriggerType) bool {
	switch t {
	case valueobject.TriggerTypeManual:
		return c.Manual
	case valueobject.TriggerTypeWebhook:
		return c.Webhook
	case valueobject.TriggerTypeCron:
		return c.Cron
	default:
		return false
	}
}
