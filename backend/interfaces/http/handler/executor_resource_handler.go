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

// ExecutorResourceHandler 管理执行资源的 HTTP 处理器。
type ExecutorResourceHandler struct {
	svc *service.ExecutorResourceService
}

// NewExecutorResourceHandler 构造处理器。
func NewExecutorResourceHandler(svc *service.ExecutorResourceService) *ExecutorResourceHandler {
	return &ExecutorResourceHandler{svc: svc}
}

// List 处理 GET /api/v1/executor-resources。
func (h *ExecutorResourceHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

// Get 处理 GET /api/v1/executor-resources/:id。
func (h *ExecutorResourceHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		handleResourceError(c, err)
		return
	}
	ok(c, resp)
}

// Create 处理 POST /api/v1/executor-resources。
func (h *ExecutorResourceHandler) Create(c *gin.Context) {
	var req dto.UpsertExecutorResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		badRequest(c, err)
		return
	}
	ok(c, resp)
}

// Update 处理 PUT /api/v1/executor-resources/:id。
func (h *ExecutorResourceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpsertExecutorResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		handleResourceError(c, err)
		return
	}
	ok(c, resp)
}

// Delete 处理 DELETE /api/v1/executor-resources/:id。
func (h *ExecutorResourceHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		handleResourceError(c, err)
		return
	}
	ok(c, nil)
}

func handleResourceError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	badRequest(c, err)
}
