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

// K8sConfigHandler 管理 K8s 认证配置的 HTTP 处理器。
type K8sConfigHandler struct {
	svc *service.K8sConfigService
}

func NewK8sConfigHandler(svc *service.K8sConfigService) *K8sConfigHandler {
	return &K8sConfigHandler{svc: svc}
}

func (h *K8sConfigHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	ok(c, gin.H{"items": items})
}

func (h *K8sConfigHandler) Create(c *gin.Context) {
	var req dto.UpsertK8sConfigRequest
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

func (h *K8sConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, err)
		return
	}
	var req dto.UpsertK8sConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	resp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		handleK8sConfigError(c, err)
		return
	}
	ok(c, resp)
}

func (h *K8sConfigHandler) Delete(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		badRequest(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		handleK8sConfigError(c, err)
		return
	}
	ok(c, nil)
}

func handleK8sConfigError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, err)
		return
	}
	badRequest(c, err)
}
