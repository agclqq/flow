package valueobject

// TemplateStatus 表示流水线模板的发布状态。
// Draft 为临时版本（未发布），Published 为已发布的正式版本。
type TemplateStatus string

const (
	TemplateStatusDraft     TemplateStatus = "draft"
	TemplateStatusPublished TemplateStatus = "published"
)

// TemplateStatus 的 tinyint 存储值。
// 0: draft（临时版本）  1: published（已发布）
const (
	TemplateStatusDraftInt     int8 = 0
	TemplateStatusPublishedInt int8 = 1
)

// Int 返回该状态的 tinyint 存储值。
func (s TemplateStatus) Int() int8 {
	switch s {
	case TemplateStatusPublished:
		return TemplateStatusPublishedInt
	default:
		return TemplateStatusDraftInt
	}
}

// TemplateStatusFromInt 将 tinyint 值转换为 TemplateStatus。
func TemplateStatusFromInt(v int8) TemplateStatus {
	switch v {
	case TemplateStatusPublishedInt:
		return TemplateStatusPublished
	default:
		return TemplateStatusDraft
	}
}

// Valid 校验状态值是否合法。
func (s TemplateStatus) Valid() bool {
	return s == TemplateStatusDraft || s == TemplateStatusPublished
}

// String 返回状态的字符串表示。
func (s TemplateStatus) String() string {
	return string(s)
}

// FlowInstanceStatus 表示流水线实例的状态。
// 状态流转：未开始→执行中；执行中→执行成功；执行中→执行失败；执行失败→重试；重试→执行中。
type FlowInstanceStatus string

const (
	FlowInstanceStatusPending   FlowInstanceStatus = "pending"   // 未开始
	FlowInstanceStatusRunning   FlowInstanceStatus = "running"   // 执行中
	FlowInstanceStatusSucceeded FlowInstanceStatus = "succeeded" // 执行成功
	FlowInstanceStatusFailed    FlowInstanceStatus = "failed"    // 执行失败
	FlowInstanceStatusRetry     FlowInstanceStatus = "retry"     // 重试
)

// FlowInstanceStatus 的 tinyint 存储值。
// 0: pending（未开始） 1: running（执行中） 2: succeeded（执行成功） 3: failed（执行失败） 4: retry（重试）
const (
	FlowInstanceStatusPendingInt   int8 = 0
	FlowInstanceStatusRunningInt   int8 = 1
	FlowInstanceStatusSucceededInt int8 = 2
	FlowInstanceStatusFailedInt    int8 = 3
	FlowInstanceStatusRetryInt     int8 = 4
)

// Int 返回该状态的 tinyint 存储值。
func (s FlowInstanceStatus) Int() int8 {
	switch s {
	case FlowInstanceStatusRunning:
		return FlowInstanceStatusRunningInt
	case FlowInstanceStatusSucceeded:
		return FlowInstanceStatusSucceededInt
	case FlowInstanceStatusFailed:
		return FlowInstanceStatusFailedInt
	case FlowInstanceStatusRetry:
		return FlowInstanceStatusRetryInt
	default:
		return FlowInstanceStatusPendingInt
	}
}

// FlowInstanceStatusFromInt 将 tinyint 值转换为 FlowInstanceStatus。
func FlowInstanceStatusFromInt(v int8) FlowInstanceStatus {
	switch v {
	case FlowInstanceStatusRunningInt:
		return FlowInstanceStatusRunning
	case FlowInstanceStatusSucceededInt:
		return FlowInstanceStatusSucceeded
	case FlowInstanceStatusFailedInt:
		return FlowInstanceStatusFailed
	case FlowInstanceStatusRetryInt:
		return FlowInstanceStatusRetry
	default:
		return FlowInstanceStatusPending
	}
}

// FlowInstanceStateMachine 定义流水线实例的合法状态流转。
var FlowInstanceStateMachine = map[FlowInstanceStatus][]FlowInstanceStatus{
	FlowInstanceStatusPending:   {FlowInstanceStatusRunning},
	FlowInstanceStatusRunning:   {FlowInstanceStatusSucceeded, FlowInstanceStatusFailed},
	FlowInstanceStatusFailed:    {FlowInstanceStatusRetry},
	FlowInstanceStatusRetry:     {FlowInstanceStatusRunning},
	FlowInstanceStatusSucceeded: {},
}

// CanTransitionTo 判断流水线实例状态能否流转到目标状态。
func (s FlowInstanceStatus) CanTransitionTo(target FlowInstanceStatus) bool {
	for _, t := range FlowInstanceStateMachine[s] {
		if t == target {
			return true
		}
	}
	return false
}

// IsTerminal 判断是否为终态（不再发生流转）。
func (s FlowInstanceStatus) IsTerminal() bool {
	return s == FlowInstanceStatusSucceeded
}

// String 返回状态的字符串表示。
func (s FlowInstanceStatus) String() string {
	return string(s)
}

// NodeInstanceStatus 表示节点实例的状态。
// 状态流转：未开始→执行中；执行中→执行成功；执行中→执行失败但跳过；
// 执行中→执行失败；执行失败→重试；重试→执行中。
type NodeInstanceStatus string

const (
	NodeInstanceStatusPending        NodeInstanceStatus = "pending"          // 未开始
	NodeInstanceStatusRunning        NodeInstanceStatus = "running"          // 执行中
	NodeInstanceStatusSucceeded      NodeInstanceStatus = "succeeded"        // 执行成功
	NodeInstanceStatusFailedSkipped  NodeInstanceStatus = "failed_skipped"   // 执行失败但跳过
	NodeInstanceStatusFailed         NodeInstanceStatus = "failed"           // 执行失败
	NodeInstanceStatusRetrying         NodeInstanceStatus = "retrying"          // 重试中
	NodeInstanceStatusWaitingApproval  NodeInstanceStatus = "waiting_approval" // 等待审批
	NodeInstanceStatusSkipped          NodeInstanceStatus = "skipped"          // 手动跳过
)

// NodeInstanceStatus 的 tinyint 存储值。
// 0: pending（未开始） 1: running（执行中） 2: succeeded（执行成功）
// 3: failed_skipped（执行失败但跳过） 4: failed（执行失败） 5: retrying（重试中） 6: waiting_approval（等待审批）
const (
	NodeInstanceStatusPendingInt          int8 = 0
	NodeInstanceStatusRunningInt          int8 = 1
	NodeInstanceStatusSucceededInt        int8 = 2
	NodeInstanceStatusFailedSkippedInt    int8 = 3
	NodeInstanceStatusFailedInt           int8 = 4
	NodeInstanceStatusRetryingInt         int8 = 5
	NodeInstanceStatusWaitingApprovalInt  int8 = 6
	NodeInstanceStatusSkippedInt          int8 = 7
)

// Int 返回该状态的 tinyint 存储值。
func (s NodeInstanceStatus) Int() int8 {
	switch s {
	case NodeInstanceStatusRunning:
		return NodeInstanceStatusRunningInt
	case NodeInstanceStatusSucceeded:
		return NodeInstanceStatusSucceededInt
	case NodeInstanceStatusFailedSkipped:
		return NodeInstanceStatusFailedSkippedInt
	case NodeInstanceStatusFailed:
		return NodeInstanceStatusFailedInt
	case NodeInstanceStatusRetrying:
		return NodeInstanceStatusRetryingInt
	case NodeInstanceStatusWaitingApproval:
		return NodeInstanceStatusWaitingApprovalInt
	case NodeInstanceStatusSkipped:
		return NodeInstanceStatusSkippedInt
	default:
		return NodeInstanceStatusPendingInt
	}
}

// NodeInstanceStatusFromInt 将 tinyint 值转换为 NodeInstanceStatus。
func NodeInstanceStatusFromInt(v int8) NodeInstanceStatus {
	switch v {
	case NodeInstanceStatusRunningInt:
		return NodeInstanceStatusRunning
	case NodeInstanceStatusSucceededInt:
		return NodeInstanceStatusSucceeded
	case NodeInstanceStatusFailedSkippedInt:
		return NodeInstanceStatusFailedSkipped
	case NodeInstanceStatusFailedInt:
		return NodeInstanceStatusFailed
	case NodeInstanceStatusRetryingInt:
		return NodeInstanceStatusRetrying
	case NodeInstanceStatusWaitingApprovalInt:
		return NodeInstanceStatusWaitingApproval
	case NodeInstanceStatusSkippedInt:
		return NodeInstanceStatusSkipped
	default:
		return NodeInstanceStatusPending
	}
}

// NodeInstanceStateMachine 定义节点实例的合法状态流转。
var NodeInstanceStateMachine = map[NodeInstanceStatus][]NodeInstanceStatus{
	NodeInstanceStatusPending:       {NodeInstanceStatusRunning},
	NodeInstanceStatusRunning:       {NodeInstanceStatusSucceeded, NodeInstanceStatusFailedSkipped, NodeInstanceStatusFailed, NodeInstanceStatusWaitingApproval, NodeInstanceStatusSkipped},
	NodeInstanceStatusWaitingApproval: {NodeInstanceStatusSucceeded, NodeInstanceStatusFailed},
	NodeInstanceStatusFailed:        {NodeInstanceStatusRetrying},
	NodeInstanceStatusRetrying:      {NodeInstanceStatusRunning},
	NodeInstanceStatusSucceeded:     {},
	NodeInstanceStatusFailedSkipped: {},
}

// CanTransitionTo 判断节点实例状态能否流转到目标状态。
func (s NodeInstanceStatus) CanTransitionTo(target NodeInstanceStatus) bool {
	for _, t := range NodeInstanceStateMachine[s] {
		if t == target {
			return true
		}
	}
	return false
}

// IsTerminal 判断是否为终态。
func (s NodeInstanceStatus) IsTerminal() bool {
	return s == NodeInstanceStatusSucceeded || s == NodeInstanceStatusFailedSkipped
}

// String 返回状态的字符串表示。
func (s NodeInstanceStatus) String() string {
	return string(s)
}

// TriggerType 表示流水线的触发方式。
type TriggerType string

const (
	TriggerTypeManual  TriggerType = "manual"  // 页面按钮触发
	TriggerTypeWebhook TriggerType = "webhook" // webhook 触发
	TriggerTypeCron    TriggerType = "cron"    // crontab 定时触发
)

// TriggerType 的 tinyint 存储值。
// 0: manual（页面按钮触发） 1: webhook（webhook 触发） 2: cron（crontab 定时触发）
const (
	TriggerTypeManualInt  int8 = 0
	TriggerTypeWebhookInt int8 = 1
	TriggerTypeCronInt    int8 = 2
)

// Int 返回该触发方式的 tinyint 存储值。
func (t TriggerType) Int() int8 {
	switch t {
	case TriggerTypeWebhook:
		return TriggerTypeWebhookInt
	case TriggerTypeCron:
		return TriggerTypeCronInt
	default:
		return TriggerTypeManualInt
	}
}

// TriggerTypeFromInt 将 tinyint 值转换为 TriggerType。
func TriggerTypeFromInt(v int8) TriggerType {
	switch v {
	case TriggerTypeWebhookInt:
		return TriggerTypeWebhook
	case TriggerTypeCronInt:
		return TriggerTypeCron
	default:
		return TriggerTypeManual
	}
}

// Valid 校验触发方式是否合法。
func (t TriggerType) Valid() bool {
	return t == TriggerTypeManual || t == TriggerTypeWebhook || t == TriggerTypeCron
}

// String 返回触发方式的字符串表示。
func (t TriggerType) String() string {
	return string(t)
}

// NotifyTiming 表示通知触发时机。
type NotifyTiming string

const (
	NotifyTimingSuccess NotifyTiming = "success" // 流水线成功时
	NotifyTimingFailure NotifyTiming = "failure" // 流水线失败时
)

// Valid 校验通知时机是否合法。
func (t NotifyTiming) Valid() bool {
	return t == NotifyTimingSuccess || t == NotifyTimingFailure
}

// String 返回通知时机的字符串表示。
func (t NotifyTiming) String() string {
	return string(t)
}
