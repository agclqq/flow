package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"flow/application/dto"
	"flow/application/service"
	"flow/domain/repository"
)

// InstanceHandler 管理流水线实例的 HTTP 处理器。
type InstanceHandler struct {
	svc *service.FlowInstanceService
}

// NewInstanceHandler 构造处理器。
func NewInstanceHandler(svc *service.FlowInstanceService) *InstanceHandler {
	return &InstanceHandler{svc: svc}
}

// Start 处理 POST /api/v1/flow-templates/:template_id/instances。
func (h *InstanceHandler) Start(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.CreateFlowInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Start(c.Request.Context(), templateID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// Get 处理 GET /api/v1/flow-templates/:template_id/instances/:instance_id。
func (h *InstanceHandler) Get(c *gin.Context) {
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), instanceID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// List 处理 GET /api/v1/flow-templates/:template_id/instances。
func (h *InstanceHandler) List(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.List(c.Request.Context(), templateID, offset, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, resp)
}

// Stop 处理 POST /api/v1/flow-templates/:template_id/instances/:instance_id/stop。
func (h *InstanceHandler) Stop(c *gin.Context) {
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.Stop(c.Request.Context(), instanceID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Retry 处理 POST /api/v1/flow-templates/:template_id/instances/:instance_id/retry。
func (h *InstanceHandler) Retry(c *gin.Context) {
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Retry(c.Request.Context(), instanceID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// Approve 处理 POST /api/v1/flow-templates/:id/instances/:instance_id/nodes/:node_id/approve。
func (h *InstanceHandler) Approve(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.NodeActionRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.svc.Approve(c.Request.Context(), templateID, instanceID, nodeID, req.Operator)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// Skip 处理 POST /api/v1/flow-templates/:id/instances/:instance_id/nodes/:node_id/skip。
func (h *InstanceHandler) Skip(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.SkipNode(c.Request.Context(), templateID, instanceID, nodeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// ListNodeOperationLogs 处理 GET /api/v1/flow-templates/:id/instances/:instance_id/nodes/:node_id/logs。
func (h *InstanceHandler) ListNodeOperationLogs(c *gin.Context) {
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.ListNodeOperationLogs(c.Request.Context(), instanceID, nodeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// Webhook 处理 POST /api/v1/flow-templates/:id/webhook。
// 密钥通过 query 参数 secret 或请求头 X-Webhook-Secret 传递；
// 请求体为 JSON 对象，作为流水线入参（值统一转为字符串）。
func (h *InstanceHandler) Webhook(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	secret := c.Query("secret")
	if secret == "" {
		secret = c.GetHeader("X-Webhook-Secret")
	}
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		body = nil
	}
	params := map[string]string{}
	for k, v := range body {
		switch val := v.(type) {
		case string:
			params[k] = val
		case float64:
			params[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case bool:
			params[k] = strconv.FormatBool(val)
		default:
			data, _ := json.Marshal(val)
			params[k] = string(data)
		}
	}
	resp, err := h.svc.StartWebhook(c.Request.Context(), templateID, secret, params)
	if err != nil {
		handleResourceError(c, err)
		return
	}
	ok(c, resp)
}

// Reject 处理 POST /api/v1/flow-templates/:id/instances/:instance_id/nodes/:node_id/reject。
func (h *InstanceHandler) Reject(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	instanceID, err := parseIDParam(c, "instance_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.NodeActionRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.svc.Reject(c.Request.Context(), templateID, instanceID, nodeID, req.Operator)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// --- 错误处理 ---

func (h *InstanceHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	msg := err.Error()
	if msg == "template must be published before starting an instance" ||
		msg == "published version not found" ||
		msg == "invalid trigger type" ||
		msg == "cannot stop a succeeded instance" ||
		msg == "instance already failed" ||
		msg == "only failed instances can be retried" ||
		msg == "instance is not running" ||
		msg == "node is not waiting for approval" ||
		msg == "node is not a manual node" ||
		msg == "instance does not belong to template" {
		badRequest(c, err)
		return
	}
	internalError(c, err)
}
