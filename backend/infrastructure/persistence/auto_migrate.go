package persistence

import (
	"gorm.io/gorm"
)

// AutoMigrate 执行所有表结构的自动迁移。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&FlowTemplateModel{},
		&FlowTemplateVersionModel{},
		&NodeModel{},
		&ExecutorResourceModel{},
		&NodeOperationLogModel{},
		&CronFireLogModel{},
		&FlowParamModel{},
		&FlowInstanceModel{},
		&NodeInstanceModel{},
		&TriggerConfigModel{},
		&NotifyTemplateModel{},
		&FlowNotifyBindingModel{},
		&K8sConfigModel{},
		&BareMetalConfigModel{},
		&RepoConfigModel{},
	)
}
