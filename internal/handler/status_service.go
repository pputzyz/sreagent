package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/service"
)

type StatusServiceHandler struct {
	svc *service.StatusServiceService
}

func NewStatusServiceHandler(svc *service.StatusServiceService) *StatusServiceHandler {
	return &StatusServiceHandler{svc: svc}
}

func (h *StatusServiceHandler) List(c *gin.Context) {
	services, err := h.svc.List(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, services)
}

func (h *StatusServiceHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	svc, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, svc)
}

func (h *StatusServiceHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,max=128"`
		Status      string `json:"status" binding:"required,oneof=operational degraded outage maintenance"`
		Description string `json:"description" binding:"max=512"`
		URL         string `json:"url" binding:"max=512"`
		Icon        string `json:"icon" binding:"max=64"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	svc := &model.StatusService{
		Name:        req.Name,
		Status:      req.Status,
		Description: req.Description,
		URL:         req.URL,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
	}
	if err := h.svc.Create(c.Request.Context(), svc); err != nil {
		Error(c, err)
		return
	}
	Success(c, svc)
}

func (h *StatusServiceHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	var req struct {
		Name        *string `json:"name" binding:"omitempty,max=128"`
		Status      *string `json:"status" binding:"omitempty,oneof=operational degraded outage maintenance"`
		Description *string `json:"description" binding:"omitempty,max=512"`
		URL         *string `json:"url" binding:"omitempty,max=512"`
		Icon        *string `json:"icon" binding:"omitempty,max=64"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.URL != nil {
		existing.URL = *req.URL
	}
	if req.Icon != nil {
		existing.Icon = *req.Icon
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}
	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, err)
		return
	}
	Success(c, existing)
}

func (h *StatusServiceHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
