package dto

import (
	"time"

	"flow/domain/entity"
	"flow/domain/valueobject"
)

// --- FlowTemplate DTOs ---

type CreateFlowTemplateRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateFlowTemplateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owners      []string `json:"owners"`
}

type FlowTemplateResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Owners      []string  `json:"owners"`
	Version     int       `json:"version"`
	Status      string    `json:"status"`
	// ManualTrigger 最新发布快照是否开启手动启动（未发布过为 false）。
	ManualTrigger bool     `json:"manual_trigger"`
	// CronEnabled 最新发布快照是否开启定时触发。
	CronEnabled bool     `json:"cron_enabled"`
	// CronExpr 最新发布快照的定时表达式。
	CronExpr string   `json:"cron_expr"`
	// PublishedVersionID 最新发布快照的版本 ID（未发布过为 0）。
	PublishedVersionID int64     `json:"published_version_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ListFlowTemplateResponse struct {
	Items []FlowTemplateResponse `json:"items"`
	Total int64                  `json:"total"`
}

func ToTemplateResponse(tpl *entity.FlowTemplate) FlowTemplateResponse {
	owners := tpl.Owners
	if owners == nil {
		owners = []string{}
	}
	return FlowTemplateResponse{
		ID:          tpl.ID,
		Name:        tpl.Name,
		Description: tpl.Description,
		Owners:      owners,
		Version:     tpl.Version,
		Status:      tpl.Status.String(),
		CreatedAt:   tpl.CreatedAt,
		UpdatedAt:   tpl.UpdatedAt,
	}
}

type PublishFlowTemplateRequest struct{}

type TemplateStatus = valueobject.TemplateStatus

// --- FlowTemplateVersion DTOs ---

type VersionResponse struct {
	ID         int64          `json:"id"`
	TemplateID int64          `json:"template_id"`
	Version    int            `json:"version"`
	Status     string         `json:"status"`
	Nodes      []NodeResponse `json:"nodes"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type VersionListItem struct {
	ID        int64     `json:"id"`
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// EditingVersionResponse 描述当前编辑页应加载的版本快照。
type EditingVersionResponse struct {
	VersionID int64  `json:"version_id"`
	Version   int    `json:"version"`
	IsDraft   bool   `json:"is_draft"`
	Status    string `json:"status"`
}

func ToVersionResponse(ver *entity.FlowTemplateVersion) VersionResponse {
	nodes := make([]NodeResponse, 0, len(ver.Nodes))
	for _, n := range ver.Nodes {
		nodes = append(nodes, ToNodeResponse(n))
	}
	return VersionResponse{
		ID:         ver.ID,
		TemplateID: ver.TemplateID,
		Version:    ver.Version,
		Status:     ver.Status.String(),
		Nodes:      nodes,
		CreatedAt:  ver.CreatedAt,
		UpdatedAt:  ver.UpdatedAt,
	}
}

func ToVersionListItem(ver *entity.FlowTemplateVersion) VersionListItem {
	return VersionListItem{
		ID:        ver.ID,
		Version:   ver.Version,
		Status:    ver.Status.String(),
		CreatedAt: ver.CreatedAt,
	}
}

// --- ExecutorResource DTOs ---

type UpsertExecutorResourceRequest struct {
	// Name 资源名称。
	Name string `json:"name" binding:"required"`
	// Type 运行时类型：local / ssh / k8s。
	Type string `json:"type" binding:"required"`
	// Config 为类型对应的连接配置（JSON 字符串）。
	Config string `json:"config"`
}

type ExecutorResourceResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Node DTOs ---

type CreateNodeRequest struct {
	Type        string   `json:"type" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	PreScript   string   `json:"pre_script"`
	Script      string   `json:"script"`
	PostScript  string   `json:"post_script"`
	Owners      []string `json:"owners"`
	Approvers   []string `json:"approvers"`
	// 边节点字段（纯拓扑）
	FromNodeID int64 `json:"from_node_id"`
	ToNodeID   int64 `json:"to_node_id"`
	// 启动门禁
	StartMode               string  `json:"start_mode"`
	StartScript             string  `json:"start_script"`
	RequiredPredecessorIDs  []int64 `json:"required_predecessor_ids"`
	// 已废弃，兼容旧客户端
	Condition  string `json:"condition,omitempty"`
	IsDefault  bool   `json:"is_default,omitempty"`
	JoinType   string `json:"join_type,omitempty"`
	RequiredEdgeIDs []int64 `json:"required_edge_ids,omitempty"`
	// DAG 画布坐标
	PositionX *float64 `json:"position_x"`
	PositionY *float64 `json:"position_y"`
	// 执行资源：runtime_kind = local / ssh / k8s，resource_id 引用执行资源。
	RuntimeKind string `json:"runtime_kind"`
	ResourceID  int64  `json:"resource_id"`
}

type UpdateNodeRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PreScript   string   `json:"pre_script"`
	Script      string   `json:"script"`
	PostScript  string   `json:"post_script"`
	Owners      []string `json:"owners"`
	Approvers   []string `json:"approvers"`
	FromNodeID int64    `json:"from_node_id"`
	ToNodeID   int64    `json:"to_node_id"`
	StartMode              string   `json:"start_mode"`
	StartScript            string   `json:"start_script"`
	RequiredPredecessorIDs []int64  `json:"required_predecessor_ids"`
	Condition              string   `json:"condition,omitempty"`
	IsDefault              bool     `json:"is_default,omitempty"`
	JoinType               string   `json:"join_type,omitempty"`
	RequiredEdgeIDs        []int64  `json:"required_edge_ids,omitempty"`
	PositionX       *float64 `json:"position_x"`
	PositionY       *float64 `json:"position_y"`
}

type NodeResponse struct {
	ID              int64    `json:"id"`
	TemplateID      int64    `json:"template_id"`
	VersionID       int64    `json:"version_id"`
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PreScript       string   `json:"pre_script"`
	Script          string   `json:"script"`
	PostScript      string   `json:"post_script"`
	Owners          []string `json:"owners"`
	Approvers       []string `json:"approvers"`
	FromNodeID             int64    `json:"from_node_id"`
	ToNodeID               int64    `json:"to_node_id"`
	StartMode              string   `json:"start_mode"`
	StartScript            string   `json:"start_script"`
	RequiredPredecessorIDs []int64  `json:"required_predecessor_ids"`
	PositionX              *float64 `json:"position_x"`
	PositionY              *float64 `json:"position_y"`
	RuntimeKind            string   `json:"runtime_kind"`
	ResourceID             int64    `json:"resource_id"`
}

func ToNodeResponse(n *entity.Node) NodeResponse {
	owners := n.Owners
	if owners == nil {
		owners = []string{}
	}
	approvers := n.Approvers
	if approvers == nil {
		approvers = []string{}
	}
	return NodeResponse{
		ID:              n.ID,
		TemplateID:      n.TemplateID,
		VersionID:       n.VersionID,
		Type:            n.Type.String(),
		Name:            n.Name,
		Description:     n.Description,
		PreScript:       n.PreScript,
		Script:          n.Script,
		PostScript:      n.PostScript,
		Owners:          owners,
		Approvers:       approvers,
		FromNodeID:             n.FromNodeID,
		ToNodeID:               n.ToNodeID,
		StartMode:              n.StartMode.String(),
		StartScript:            n.StartScript,
		RequiredPredecessorIDs: n.RequiredPredecessorIDs,
		PositionX:       n.PositionX,
		PositionY:       n.PositionY,
		RuntimeKind:     n.RuntimeKind,
		ResourceID:      n.ResourceID,
	}
}

// --- FlowParam DTOs ---

type CreateFlowParamRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	DefaultValue string `json:"default_value"`
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

type UpdateFlowParamRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"default_value"`
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

type FlowParamResponse struct {
	ID           int64  `json:"id"`
	TemplateID   int64  `json:"template_id"`
	VersionID    int64  `json:"version_id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"default_value"`
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

func ToParamResponse(p *entity.FlowParam) FlowParamResponse {
	return FlowParamResponse{
		ID:           p.ID,
		TemplateID:   p.TemplateID,
		VersionID:    p.VersionID,
		Name:         p.Name,
		Type:         p.Type.String(),
		DefaultValue: p.DefaultValue,
		Required:     p.Required,
		Description:  p.Description,
	}
}

// --- TriggerConfig DTOs ---

type UpsertTriggerConfigRequest struct {
	Manual        bool   `json:"manual"`
	Webhook       bool   `json:"webhook"`
	Cron          bool   `json:"cron"`
	CronExpr      string `json:"cron_expr"`
	RequireParams bool   `json:"require_params"`
	// WebhookSecret webhook 触发密钥。
	WebhookSecret string `json:"webhook_secret"`
}

type TriggerConfigResponse struct {
	ID            int64  `json:"id"`
	TemplateID    int64  `json:"template_id"`
	Manual        bool   `json:"manual"`
	Webhook       bool   `json:"webhook"`
	Cron          bool   `json:"cron"`
	CronExpr      string `json:"cron_expr"`
	RequireParams bool   `json:"require_params"`
	WebhookSecret string `json:"webhook_secret"`
}

func ToTriggerConfigResponse(c *entity.TriggerConfig) TriggerConfigResponse {
	return TriggerConfigResponse{
		ID:            c.ID,
		TemplateID:    c.TemplateID,
		Manual:        c.Manual,
		Webhook:       c.Webhook,
		Cron:          c.Cron,
		CronExpr:      c.CronExpr,
		RequireParams: c.RequireParams,
		WebhookSecret: c.WebhookSecret,
	}
}

// --- 节点操作日志 DTOs ---

type NodeOperationLogResponse struct {
	ID        int64     `json:"id"`
	Operator  string    `json:"operator"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type NodeActionRequest struct {
	// Operator 操作人（可选）。
	Operator string `json:"operator"`
}

// --- FlowInstance DTOs ---

type CreateFlowInstanceRequest struct {
	TriggerType string            `json:"trigger_type" binding:"required"`
	Params      map[string]string `json:"params"`
}

type FlowInstanceResponse struct {
	ID                int64                  `json:"id"`
	TemplateID        int64                  `json:"template_id"`
	TemplateVersionID int64                  `json:"template_version_id"`
	TemplateVersion   int                    `json:"template_version"`
	Status            string                 `json:"status"`
	TriggerType       string                 `json:"trigger_type"`
	Params            map[string]string      `json:"params"`
	Env               map[string]string      `json:"env"`
	NodeInstances     []NodeInstanceResponse `json:"node_instances,omitempty"`
	FinishedAt        *time.Time             `json:"finished_at,omitempty"`
	DurationSeconds   *float64               `json:"duration_seconds,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type NodeInstanceResponse struct {
	ID             int64                `json:"id"`
	FlowInstanceID int64                `json:"flow_instance_id"`
	NodeID         int64                `json:"node_id"`
	Type           string               `json:"type"`
	Status         string               `json:"status"`
	Env            map[string]string    `json:"env"`
	OutputEnv      map[string]string    `json:"output_env"`
	RunTimes       []entity.RunTimeSpan `json:"run_times,omitempty"`
	RetryCount     int                  `json:"retry_count"`
}

type ListFlowInstanceResponse struct {
	Items []FlowInstanceResponse `json:"items"`
	Total int64                  `json:"total"`
}

func ToInstanceResponse(inst *entity.FlowInstance) FlowInstanceResponse {
	params := inst.Params
	if params == nil {
		params = map[string]string{}
	}
	env := inst.Env
	if env == nil {
		env = map[string]string{}
	}
	resp := FlowInstanceResponse{
		ID:                inst.ID,
		TemplateID:        inst.TemplateID,
		TemplateVersionID: inst.TemplateVersionID,
		TemplateVersion:   inst.TemplateVersion,
		Status:            inst.Status.String(),
		TriggerType:       inst.TriggerType.String(),
		Params:            params,
		Env:               env,
		FinishedAt:        inst.FinishedAt,
		CreatedAt:         inst.CreatedAt,
		UpdatedAt:         inst.UpdatedAt,
	}
	if inst.FinishedAt != nil {
		secs := inst.FinishedAt.Sub(inst.CreatedAt).Seconds()
		resp.DurationSeconds = &secs
	}
	if len(inst.NodeInstances) > 0 {
		nis := make([]NodeInstanceResponse, 0, len(inst.NodeInstances))
		for _, ni := range inst.NodeInstances {
			nis = append(nis, ToNodeInstanceResponse(ni))
		}
		resp.NodeInstances = nis
	}
	return resp
}

func ToNodeInstanceResponse(ni *entity.NodeInstance) NodeInstanceResponse {
	env := ni.Env
	if env == nil {
		env = map[string]string{}
	}
	output := ni.OutputEnv
	if output == nil {
		output = map[string]string{}
	}
	return NodeInstanceResponse{
		ID:             ni.ID,
		FlowInstanceID: ni.FlowInstanceID,
		NodeID:         ni.NodeID,
		Type:           ni.Type.String(),
		Status:         ni.Status.String(),
		Env:            env,
		OutputEnv:      output,
		RunTimes:       ni.RunTimes,
		RetryCount:     ni.RetryCount,
	}
}

// --- NotifyTemplate DTOs ---

type CreateNotifyTemplateRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type UpdateNotifyTemplateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type NotifyTemplateResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListNotifyTemplateResponse struct {
	Items []NotifyTemplateResponse `json:"items"`
	Total int64                    `json:"total"`
}

func ToNotifyTemplateResponse(t *entity.NotifyTemplate) NotifyTemplateResponse {
	return NotifyTemplateResponse{
		ID:        t.ID,
		Name:      t.Name,
		Content:   t.Content,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// --- FlowNotifyBinding DTOs ---

type FlowNotifyBindingItem struct {
	NodeID           int64    `json:"node_id"`
	NotifyTemplateID int64    `json:"notify_template_id" binding:"required"`
	Timings          []string `json:"timings"`
	ExtraContent     string   `json:"extra_content"`
}

type FlowNotifyBindingResponse struct {
	ID               int64    `json:"id"`
	TemplateID       int64    `json:"template_id"`
	NodeID           int64    `json:"node_id"`
	NotifyTemplateID int64    `json:"notify_template_id"`
	Timings          []string `json:"timings"`
	ExtraContent     string   `json:"extra_content"`
}

type ReplaceFlowNotifyBindingsRequest struct {
	Bindings []FlowNotifyBindingItem `json:"bindings"`
}

func ToNotifyBindingResponse(b *entity.FlowNotifyBinding) FlowNotifyBindingResponse {
	timings := make([]string, 0, len(b.Timings))
	for _, t := range b.Timings {
		timings = append(timings, t.String())
	}
	return FlowNotifyBindingResponse{
		ID:               b.ID,
		TemplateID:       b.TemplateID,
		NodeID:           b.NodeID,
		NotifyTemplateID: b.NotifyTemplateID,
		Timings:          timings,
		ExtraContent:     b.ExtraContent,
	}
}

// --- Batch Graph DTOs ---

type BatchGraphNodeRequest struct {
	ID int64 `json:"id"`
	CreateNodeRequest
}

type BatchGraphSaveRequest struct {
	Nodes []BatchGraphNodeRequest `json:"nodes" binding:"required"`
}
