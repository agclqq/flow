package persistence

import (
	"time"

	"gorm.io/gorm"
)

// FlowTemplateModel 是流水线模板的元数据映射。
// 模板保存名称、当前版本号及草稿状态等元数据，内容快照由 FlowTemplateVersionModel 承载。
type FlowTemplateModel struct {
	ID   int64  `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name string `gorm:"type:varchar(128);not null"`
	// Owners 模板级全局负责人，JSON 数组。
	Owners string `gorm:"type:jsonb"`
	// Description 模板描述。
	Description string `gorm:"type:varchar(512)"`
	// CurrentVersion 当前最新版本号。
	CurrentVersion int `gorm:"not null;default:0"`
	// Status 模板状态。tinyint 枚举：0=draft（临时版本） 1=published（已发布）。
	Status    int8 `gorm:"type:smallint;not null;default:0;comment:'模板状态 0=draft 1=published'"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (FlowTemplateModel) TableName() string {
	return "flow_templates"
}

// FlowTemplateVersionModel 是流水线模板版本快照的映射。
type FlowTemplateVersionModel struct {
	ID         int64 `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID int64 `gorm:"type:bigint;not null;index"`
	Version    int   `gorm:"not null"`
	// TriggerConfig 触发配置快照（JSON，发布时固化）。
	TriggerConfig string `gorm:"type:jsonb"`
	// Status 版本状态。tinyint 枚举：0=draft（临时版本） 1=published（已发布）。
	Status    int8           `gorm:"type:smallint;not null;default:0;comment:'版本状态 0=draft 1=published'"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (FlowTemplateVersionModel) TableName() string {
	return "flow_template_versions"
}

// NodeModel 是节点的映射。统一一张表，用 type 区分 start/end/task/edge。
// 拓扑关系由 edge 节点的 from/to 字段承载。
type NodeModel struct {
	ID         int64 `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID int64 `gorm:"type:bigint;not null;index"`
	VersionID  int64 `gorm:"type:bigint;not null;index"`
	// Type 节点类型。tinyint 枚举：0=start 1=end 2=task 3=edge 4=join 5=manual。
	Type       int8   `gorm:"type:smallint;not null;comment:'节点类型 0=start 1=end 2=task 3=edge 4=join 5=manual'"`
	Name       string `gorm:"type:varchar(128);not null"`
	Description string `gorm:"type:varchar(512)"`
	PreScript  string `gorm:"type:text"`
	Script     string `gorm:"type:text"`
	PostScript string `gorm:"type:text"`
	// Owners 负责人列表，多值以 JSON 数组存储。
	Owners string `gorm:"type:jsonb"`
	// Approvers 审批人列表，多值以 JSON 数组存储。
	Approvers string `gorm:"type:jsonb"`
	// 连线关系（edge 节点使用）。
	FromNodeID int64 `gorm:"type:bigint;index"`
	ToNodeID   int64 `gorm:"type:bigint;index"`
	// Condition/IsDefault 已废弃，兼容旧数据。
	Condition string `gorm:"type:text"`
	IsDefault bool   `gorm:"not null;default:false"`
	// JoinType 启动门禁策略（复用列）。0=all 1=any 2=custom。
	JoinType int8 `gorm:"type:smallint;not null;default:1;comment:'启动策略 0=all 1=any 2=custom'"`
	// StartScript 启动前脚本。
	StartScript string `gorm:"type:text"`
	// RequiredPredecessorIDs custom 模式必须成功的前驱节点 ID（JSON 数组）。
	RequiredPredecessorIDs string `gorm:"type:jsonb"`
	// RequiredEdgeIDs 已废弃，兼容旧 join 数据。
	RequiredEdgeIDs string `gorm:"type:jsonb"`
	// PositionX/PositionY DAG 画布坐标。
	PositionX *float64       `gorm:"type:double precision"`
	PositionY *float64       `gorm:"type:double precision"`
	// RuntimeKind 执行资源类型：local/ssh/k8s，空视为 local。
	RuntimeKind string `gorm:"type:varchar(32);not null;default:'local'"`
	// ResourceID 引用的执行资源 ID（ssh/k8s 使用）。
	ResourceID int64          `gorm:"type:bigint;not null;default:0"`
	CreatedAt  time.Time      `gorm:"not null"`
	UpdatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (NodeModel) TableName() string {
	return "nodes"
}

// ExecutorResourceModel 是节点执行资源（在哪里执行）的映射，全局共享。
type ExecutorResourceModel struct {
	ID   int64  `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name string `gorm:"type:varchar(128);not null"`
	// Type 运行时类型：local/ssh/k8s。
	Type       string    `gorm:"type:varchar(32);not null"`
	Config     string    `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (ExecutorResourceModel) TableName() string {
	return "executor_resources"
}

// FlowInstanceModel 是流水线实例的映射。
type FlowInstanceModel struct {
	ID                int64 `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID        int64 `gorm:"type:bigint;not null;index"`
	TemplateVersionID int64 `gorm:"type:bigint;not null;index"`
	TemplateVersion   int   `gorm:"not null"`
	// Status 实例状态。tinyint 枚举：0=pending（未开始） 1=running（执行中） 2=succeeded（成功） 3=failed（失败） 4=retry（重试）。
	Status int8 `gorm:"type:smallint;not null;index;comment:'实例状态 0=pending 1=running 2=succeeded 3=failed 4=retry'"`
	// TriggerType 触发方式。tinyint 枚举：0=manual（页面按钮） 1=webhook 2=cron（定时）。
	TriggerType int8           `gorm:"type:smallint;not null;comment:'触发方式 0=manual 1=webhook 2=cron'"`
	Params      string         `gorm:"type:jsonb"`
	Env         string         `gorm:"type:jsonb"`
	FinishedAt  *time.Time     `gorm:"index"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (FlowInstanceModel) TableName() string {
	return "flow_instances"
}

// NodeInstanceModel 是节点实例的映射。
type NodeInstanceModel struct {
	ID             int64 `gorm:"primaryKey;autoIncrement;type:bigint"`
	FlowInstanceID int64 `gorm:"type:bigint;not null;index"`
	NodeID         int64 `gorm:"type:bigint;not null;index"`
	// Type 节点类型。tinyint 枚举：0=start（开始） 1=end（结束） 2=task（普通） 3=edge（流转/边）。
	Type int8 `gorm:"type:smallint;not null;comment:'节点类型 0=start 1=end 2=task 3=edge'"`
	// Status 节点实例状态。tinyint 枚举：0=pending 1=running 2=succeeded 3=failed_skipped 4=failed 5=retrying 6=waiting_approval。
	Status     int8           `gorm:"type:smallint;not null;index;comment:'节点实例状态 0=pending 1=running 2=succeeded 3=failed_skipped 4=failed 5=retrying 6=waiting_approval'"`
	Env        string         `gorm:"type:jsonb"`
	OutputEnv  string         `gorm:"type:jsonb"`
	// RunTimes 运行时间段列表（JSON 数组，失败重试会有多段）。
	RunTimes   string         `gorm:"type:jsonb"`
	RetryCount int            `gorm:"not null;default:0"`
	CreatedAt  time.Time      `gorm:"not null"`
	UpdatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (NodeInstanceModel) TableName() string {
	return "node_instances"
}

// TriggerConfigModel 是触发配置的映射。
type TriggerConfigModel struct {
	ID            int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID    int64          `gorm:"type:bigint;not null;index"`
	Manual        bool           `gorm:"not null;default:false"`
	Webhook       bool           `gorm:"not null;default:false"`
	Cron          bool           `gorm:"not null;default:false"`
	CronExpr      string         `gorm:"type:varchar(128)"`
	RequireParams bool           `gorm:"not null;default:false"`
	// WebhookSecret webhook 触发密钥。
	WebhookSecret string         `gorm:"type:varchar(128)"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// NodeOperationLogModel 是人工节点操作日志的映射。
type NodeOperationLogModel struct {
	ID             int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	InstanceID     int64          `gorm:"type:bigint;not null;index"`
	NodeInstanceID int64          `gorm:"type:bigint;not null;index"`
	NodeID         int64          `gorm:"type:bigint;not null;index"`
	// Action 动作：approve / reject。
	Action        string         `gorm:"type:varchar(32);not null"`
	Operator      string         `gorm:"type:varchar(128);not null;default:'-'"`
	CreatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// CronFireLogModel 是定时触发抢占标记的映射。
// (template_id, fire_minute) 唯一，实现多副本下定时触发的去重。
type CronFireLogModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID int64     `gorm:"type:bigint;not null;uniqueIndex:idx_cron_template_minute"`
	FireMinute time.Time `gorm:"not null;uniqueIndex:idx_cron_template_minute"`
	// VersionID 触发时使用的发布快照版本 ID。
	VersionID int64     `gorm:"type:bigint;not null;default:0"`
	// Status 抢占状态：claimed（待创建实例）/ fired（实例已创建）/ exhausted（重试耗尽）。
	Status     string         `gorm:"type:varchar(16);not null;default:'claimed'"`
	// RetryCount 已重试次数。
	RetryCount int            `gorm:"not null;default:0"`
	InstanceID int64          `gorm:"type:bigint;not null;default:0"`
	CreatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (CronFireLogModel) TableName() string {
	return "cron_fire_logs"
}

// TableName 指定表名。
func (NodeOperationLogModel) TableName() string {
	return "node_operation_logs"
}

// TableName 指定表名。
func (TriggerConfigModel) TableName() string {
	return "trigger_configs"
}

// NotifyTemplateModel 是通知内容模板的映射（可被多个流水线复用）。
type NotifyTemplateModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name      string         `gorm:"type:varchar(128);not null"`
	Content   string         `gorm:"type:text"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (NotifyTemplateModel) TableName() string {
	return "notify_templates"
}

// FlowNotifyBindingModel 是流水线与通知模板绑定的映射。
type FlowNotifyBindingModel struct {
	ID               int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID       int64          `gorm:"type:bigint;not null;index"`
	NodeID           int64          `gorm:"type:bigint;not null;default:0;index"`
	NotifyTemplateID int64          `gorm:"type:bigint;not null;index"`
	Timings          string         `gorm:"type:jsonb"`
	ExtraContent     string         `gorm:"type:text"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (FlowNotifyBindingModel) TableName() string {
	return "flow_notify_bindings"
}

// FlowParamModel 是流水线参数定义的映射，挂到版本快照下。
type FlowParamModel struct {
	ID         int64  `gorm:"primaryKey;autoIncrement;type:bigint"`
	TemplateID int64  `gorm:"type:bigint;not null;index"`
	VersionID  int64  `gorm:"type:bigint;not null;index"`
	Name       string `gorm:"type:varchar(128);not null"`
	// Type 参数类型。tinyint 枚举：0=string（字符串） 1=number（数值） 2=boolean（布尔）。
	Type         int8           `gorm:"type:smallint;not null;comment:'参数类型 0=string 1=number 2=boolean'"`
	DefaultValue string         `gorm:"type:varchar(512)"`
	Required     bool           `gorm:"not null;default:false"`
	Description  string         `gorm:"type:varchar(512)"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (FlowParamModel) TableName() string {
	return "flow_params"
}

// K8sConfigModel 是 k8s 集群配置的映射。
type K8sConfigModel struct {
	ID         int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name       string         `gorm:"type:varchar(128);not null"`
	Server     string         `gorm:"type:varchar(256);not null"`
	Token      string         `gorm:"type:text"`
	Kubeconfig string         `gorm:"type:text"`
	Namespace  string         `gorm:"type:varchar(128)"`
	CreatedAt  time.Time      `gorm:"not null"`
	UpdatedAt  time.Time      `gorm:"not null"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (K8sConfigModel) TableName() string {
	return "k8s_configs"
}

// BareMetalConfigModel 是裸机执行环境配置的映射。
type BareMetalConfigModel struct {
	ID   int64  `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name string `gorm:"type:varchar(128);not null"`
	Host string `gorm:"type:varchar(256);not null"`
	Port int    `gorm:"not null"`
	// AuthType 认证方式。tinyint 枚举：0=password（用户+密码） 1=sshkey（SSH 私钥）。
	AuthType  int8           `gorm:"type:smallint;not null;default:0;comment:'认证方式 0=password 1=sshkey'"`
	User      string         `gorm:"type:varchar(128)"`
	Password  string         `gorm:"type:varchar(256)"`
	SSHKey    string         `gorm:"type:text"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (BareMetalConfigModel) TableName() string {
	return "bare_metal_configs"
}

// RepoConfigModel 是仓库配置的映射。
type RepoConfigModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;type:bigint"`
	Name      string         `gorm:"type:varchar(128);not null"`
	URL       string         `gorm:"type:varchar(512);not null"`
	User      string         `gorm:"type:varchar(128)"`
	Token     string         `gorm:"type:varchar(512)"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名。
func (RepoConfigModel) TableName() string {
	return "repo_configs"
}
