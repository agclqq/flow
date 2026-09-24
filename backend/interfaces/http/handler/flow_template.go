package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"flow/application/dto"
	"flow/application/service"
	"flow/domain/repository"
)

// FlowTemplateHandler 是流水线模板的 HTTP 处理器，只负责参数绑定与响应。
type FlowTemplateHandler struct {
	svc *service.FlowTemplateService
}

// NewFlowTemplateHandler 构造处理器。
func NewFlowTemplateHandler(svc *service.FlowTemplateService) *FlowTemplateHandler {
	return &FlowTemplateHandler{svc: svc}
}

// Create 处理 POST /api/v1/flow-templates。
func (h *FlowTemplateHandler) Create(c *gin.Context) {
	var req dto.CreateFlowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, resp)
}

// Get 处理 GET /api/v1/flow-templates/:id。
func (h *FlowTemplateHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, resp)
}

// List 处理 GET /api/v1/flow-templates。
func (h *FlowTemplateHandler) List(c *gin.Context) {
	offset, limit := parsePagination(c)
	resp, err := h.svc.List(c.Request.Context(), offset, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, resp)
}

// Update 处理 PUT /api/v1/flow-templates/:id。
func (h *FlowTemplateHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpdateFlowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, resp)
}

// Publish 处理 POST /api/v1/flow-templates/:id/publish。
func (h *FlowTemplateHandler) Publish(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Publish(c.Request.Context(), id)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, resp)
}

// Delete 处理 DELETE /api/v1/flow-templates/:id。
func (h *FlowTemplateHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleQueryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Clone 处理 POST /api/v1/flow-templates/:id/clone。
func (h *FlowTemplateHandler) Clone(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Clone(c.Request.Context(), id)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, resp)
}

// GetNotifyBindings 处理 GET /api/v1/flow-templates/:id/notify-bindings。
func (h *FlowTemplateHandler) GetNotifyBindings(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.GetNotifyBindings(c.Request.Context(), id)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// ReplaceNotifyBindings 处理 PUT /api/v1/flow-templates/:id/notify-bindings。
func (h *FlowTemplateHandler) ReplaceNotifyBindings(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.ReplaceFlowNotifyBindingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	items, err := h.svc.ReplaceNotifyBindings(c.Request.Context(), id, req)
	if err != nil {
		h.handleQueryError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// handleQueryError 区分未找到与内部错误。
func (h *FlowTemplateHandler) handleQueryError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	internalError(c, err)
}

// parseIDParam 解析 URL 中的 int64 类型 ID 参数。
func parseIDParam(c *gin.Context, name string) (int64, error) {
	v, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// parsePagination 解析分页参数，提供默认值。
func parsePagination(c *gin.Context) (int, int) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}
