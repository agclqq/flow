package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"flow/application/dto"
	"flow/application/service"
	"flow/domain/repository"
)

// NotifyHandler 管理通知模板的 HTTP 处理器。
type NotifyHandler struct {
	svc *service.NotifyService
}

// NewNotifyHandler 构造处理器。
func NewNotifyHandler(svc *service.NotifyService) *NotifyHandler {
	return &NotifyHandler{svc: svc}
}

// Create 处理 POST /api/v1/notify-templates。
func (h *NotifyHandler) Create(c *gin.Context) {
	var req dto.CreateNotifyTemplateRequest
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

// List 处理 GET /api/v1/notify-templates。
func (h *NotifyHandler) List(c *gin.Context) {
	offset, limit := parsePagination(c)
	resp, err := h.svc.List(c.Request.Context(), offset, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, resp)
}

// Get 处理 GET /api/v1/notify-templates/:id。
func (h *NotifyHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// Update 处理 PUT /api/v1/notify-templates/:id。
func (h *NotifyHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpdateNotifyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	ok(c, resp)
}

// Delete 处理 DELETE /api/v1/notify-templates/:id。
func (h *NotifyHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotifyHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	internalError(c, err)
}
