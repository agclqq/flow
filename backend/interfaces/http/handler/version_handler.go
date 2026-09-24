package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	"flow/application/dto"
	"flow/application/service"
	"flow/domain/repository"
)

// VersionHandler 管理版本、节点、参数和触发配置的 HTTP 处理器。
type VersionHandler struct {
	svc *service.VersionService
}

// NewVersionHandler 构造处理器。
func NewVersionHandler(svc *service.VersionService) *VersionHandler {
	return &VersionHandler{svc: svc}
}

// --- 版本 ---

// CronNext 处理 GET /api/v1/cron/next?expr=...：解析 5 段 crontab 表达式并返回下次执行时间。
func (h *VersionHandler) CronNext(c *gin.Context) {
	expr := c.Query("expr")
	sched, err := cron.ParseStandard(expr)
	if err != nil {
		ok(c, gin.H{"valid": false, "error": err.Error()})
		return
	}
	next := sched.Next(time.Now())
	ok(c, gin.H{"valid": true, "next": next.Format("2006-01-02 15:04:05")})
}

// ListVersions 处理 GET /api/v1/flow-templates/:template_id/versions。
func (h *VersionHandler) ListVersions(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.ListVersions(c.Request.Context(), templateID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// GetVersion 处理 GET /api/v1/flow-templates/:template_id/versions/:version_id。
func (h *VersionHandler) GetVersion(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.GetVersion(c.Request.Context(), versionID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// PublishVersion 处理 POST /api/v1/flow-templates/:template_id/versions/:version_id/publish。
func (h *VersionHandler) PublishVersion(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.PublishVersion(c.Request.Context(), versionID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// GetEditingVersion 处理 GET /api/v1/flow-templates/:template_id/editing-version。
func (h *VersionHandler) GetEditingVersion(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.GetEditingVersion(c.Request.Context(), templateID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// EnsureDraftVersion 处理 POST /api/v1/flow-templates/:template_id/editing-version/ensure-draft。
func (h *VersionHandler) EnsureDraftVersion(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.EnsureDraftVersion(c.Request.Context(), templateID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// --- 节点 ---

// ListNodes 处理 GET /api/v1/flow-templates/:template_id/versions/:version_id/nodes。
func (h *VersionHandler) ListNodes(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.ListNodes(c.Request.Context(), versionID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// AddNode 处理 POST /api/v1/flow-templates/:template_id/versions/:version_id/nodes。
func (h *VersionHandler) AddNode(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.AddNode(c.Request.Context(), versionID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// UpdateNode 处理 PUT /api/v1/flow-templates/:template_id/versions/:version_id/nodes/:node_id。
func (h *VersionHandler) UpdateNode(c *gin.Context) {
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.UpdateNode(c.Request.Context(), nodeID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// DeleteNode 处理 DELETE /api/v1/flow-templates/:template_id/versions/:version_id/nodes/:node_id。
func (h *VersionHandler) DeleteNode(c *gin.Context) {
	nodeID, err := parseIDParam(c, "node_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.DeleteNode(c.Request.Context(), nodeID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// SaveGraph 处理 PUT /api/v1/flow-templates/:template_id/versions/:version_id/graph。
func (h *VersionHandler) SaveGraph(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.BatchGraphSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.SaveGraph(c.Request.Context(), versionID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// --- 参数 ---

// ListParams 处理 GET /api/v1/flow-templates/:template_id/versions/:version_id/params。
func (h *VersionHandler) ListParams(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.ListParams(c.Request.Context(), versionID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// AddParam 处理 POST /api/v1/flow-templates/:template_id/versions/:version_id/params。
func (h *VersionHandler) AddParam(c *gin.Context) {
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.CreateFlowParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.AddParam(c.Request.Context(), versionID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// UpdateParam 处理 PUT /api/v1/flow-templates/:template_id/versions/:version_id/params/:param_id。
func (h *VersionHandler) UpdateParam(c *gin.Context) {
	paramID, err := parseIDParam(c, "param_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpdateFlowParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.UpdateParam(c.Request.Context(), paramID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// DeleteParam 处理 DELETE /api/v1/flow-templates/:template_id/versions/:version_id/params/:param_id。
func (h *VersionHandler) DeleteParam(c *gin.Context) {
	paramID, err := parseIDParam(c, "param_id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.DeleteParam(c.Request.Context(), paramID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- 触发配置 ---

// GetTrigger 处理 GET /api/v1/flow-templates/:template_id/trigger。
func (h *VersionHandler) GetTrigger(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.GetTrigger(c.Request.Context(), templateID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// UpsertTrigger 处理 PUT /api/v1/flow-templates/:template_id/trigger。
func (h *VersionHandler) UpsertTrigger(c *gin.Context) {
	templateID, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpsertTriggerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.UpsertTrigger(c.Request.Context(), templateID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// --- 错误处理 ---

func (h *VersionHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	if err.Error() == "cannot modify a published version" {
		badRequest(c, err)
		return
	}
	internalError(c, err)
}
