package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// ChangeEventHandler handles change event API endpoints.
type ChangeEventHandler struct {
	svc        *service.ChangeEventService
	incidentSvc *service.IncidentService
}

func NewChangeEventHandler(svc *service.ChangeEventService) *ChangeEventHandler {
	return &ChangeEventHandler{svc: svc}
}

// SetIncidentService injects the incident service for incident-scoped change queries.
func (h *ChangeEventHandler) SetIncidentService(svc *service.IncidentService) {
	h.incidentSvc = svc
}

// List godoc
// @Summary 列出变更事件
// @Tags ChangeEvent
// @Produce json
// @Param service query string false "服务名过滤"
// @Param environment query string false "环境过滤"
// @Param source query string false "来源过滤"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} handler.SuccessResponse
// @Router /changes [get]
func (h *ChangeEventHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	svc := c.Query("service")
	env := c.Query("environment")
	source := c.Query("source")

	// Support incident_id filtering: find changes in the incident's time window
	if incidentIDStr := c.Query("incident_id"); incidentIDStr != "" && h.incidentSvc != nil {
		incidentID, err := strconv.ParseUint(incidentIDStr, 10, 64)
		if err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid incident_id"))
			return
		}
		inc, err := h.incidentSvc.GetByID(c.Request.Context(), uint(incidentID))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				Error(c, apperr.ErrNotFound)
				return
			}
			Error(c, apperr.Wrap(apperr.ErrDatabase, err))
			return
		}
		end := inc.ClosedAt
		if end == nil {
			now := time.Now()
			end = &now
		}
		events, err := h.svc.FindByTimeWindow(c.Request.Context(), "", inc.TriggeredAt, *end)
		if err != nil {
			Error(c, apperr.Wrap(apperr.ErrDatabase, err))
			return
		}
		SuccessPage(c, events, int64(len(events)), 1, len(events))
		return
	}

	events, total, err := h.svc.List(c.Request.Context(), svc, env, source, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, events, total, pq.Page, pq.PageSize)
}

// Get godoc
// @Summary 获取变更事件详情
// @Tags ChangeEvent
// @Produce json
// @Param id path int true "事件 ID"
// @Success 200 {object} model.ChangeEvent
// @Router /changes/{id} [get]
func (h *ChangeEventHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	event, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.ErrNotFound)
		return
	}

	Success(c, event)
}

// Ingest godoc
// @Summary 接入变更事件（CI/CD webhook）
// @Tags ChangeEvent
// @Accept json
// @Produce json
// @Param body body model.ChangeEvent true "变更事件"
// @Success 200 {object} model.ChangeEvent
// @Router /changes [post]
func (h *ChangeEventHandler) Ingest(c *gin.Context) {
	var event model.ChangeEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.Ingest(c.Request.Context(), &event); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, event)
}

// Delete godoc
// @Summary 删除变更事件
// @Tags ChangeEvent
// @Produce json
// @Param id path int true "事件 ID"
// @Success 200
// @Router /changes/{id} [delete]
func (h *ChangeEventHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, nil)
}
