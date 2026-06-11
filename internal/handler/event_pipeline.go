package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// EventPipelineHandler handles event pipeline API requests.
type EventPipelineHandler struct {
	pipelineSvc *service.EventPipelineService
	execSvc     *service.EventPipelineExecutionService
	engine      *pipeline.Engine
	eventSvc    *service.AlertEventService
	auditSvc    *service.AuditLogService
	log         *zap.Logger
}

// NewEventPipelineHandler creates a new EventPipelineHandler.
func NewEventPipelineHandler(
	pipelineSvc *service.EventPipelineService,
	execSvc *service.EventPipelineExecutionService,
	engine *pipeline.Engine,
	eventSvc *service.AlertEventService,
	logger ...*zap.Logger,
) *EventPipelineHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &EventPipelineHandler{
		pipelineSvc: pipelineSvc,
		execSvc:     execSvc,
		engine:      engine,
		eventSvc:    eventSvc,
		log:         l,
	}
}

// SetAuditService injects the audit log service.
func (h *EventPipelineHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// EventPipelineRequest is the request body for create/update.
type EventPipelineRequest struct {
	Name             string                  `json:"name" binding:"required"`
	Description      string                  `json:"description"`
	Disabled         bool                    `json:"disabled"`
	FilterEnable     bool                    `json:"filter_enable"`
	LabelFilters     []model.TagFilter       `json:"label_filters"`
	ProcessorConfigs []model.ProcessorConfig `json:"processors"`
}

// List returns a paginated list of event pipelines.
func (h *EventPipelineHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	query := c.Query("query")

	var disabled *bool
	if d := c.Query("disabled"); d != "" {
		v := d == "true" || d == "1"
		disabled = &v
	}

	list, total, err := h.pipelineSvc.List(c.Request.Context(), pq.Page, pq.PageSize, disabled, query)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get returns a single event pipeline by ID.
func (h *EventPipelineHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	p, err := h.pipelineSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, p)
}

// Create creates a new event pipeline.
func (h *EventPipelineHandler) Create(c *gin.Context) {
	var req EventPipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	p := &model.EventPipeline{
		Name:             req.Name,
		Description:      req.Description,
		Disabled:         req.Disabled,
		FilterEnable:     req.FilterEnable,
		LabelFilters:     req.LabelFilters,
		ProcessorConfigs: req.ProcessorConfigs,
		CreatedBy:        userID,
		UpdatedBy:        userID,
	}
	if p.LabelFilters == nil {
		p.LabelFilters = []model.TagFilter{}
	}
	if p.ProcessorConfigs == nil {
		p.ProcessorConfigs = []model.ProcessorConfig{}
	}
	p.FE2DB()

	if err := h.pipelineSvc.Create(c.Request.Context(), p); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}

	p.DB2FE()

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		pid := p.ID
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: "event_pipeline", ResourceID: &pid, ResourceName: p.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, p)
}

// Update updates an existing event pipeline.
func (h *EventPipelineHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	existing, err := h.pipelineSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req EventPipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Disabled = req.Disabled
	existing.FilterEnable = req.FilterEnable
	existing.LabelFilters = req.LabelFilters
	existing.ProcessorConfigs = req.ProcessorConfigs
	existing.UpdatedBy = userID
	if existing.LabelFilters == nil {
		existing.LabelFilters = []model.TagFilter{}
	}
	if existing.ProcessorConfigs == nil {
		existing.ProcessorConfigs = []model.ProcessorConfig{}
	}
	existing.FE2DB()

	if err := h.pipelineSvc.Update(c.Request.Context(), existing); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}

	existing.DB2FE()

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: "event_pipeline", ResourceID: &id, ResourceName: existing.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, existing)
}

// Delete soft-deletes an event pipeline.
func (h *EventPipelineHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	existing, err := h.pipelineSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.pipelineSvc.Delete(c.Request.Context(), id); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: "event_pipeline", ResourceID: &id, ResourceName: existing.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// ListExecutions returns paginated executions for a pipeline.
func (h *EventPipelineHandler) ListExecutions(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	pq := GetPageQuery(c)

	list, total, err := h.execSvc.ListByPipelineID(c.Request.Context(), id, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// GetExecution returns a single execution record by ID.
func (h *EventPipelineHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "id is required"))
		return
	}

	exec, err := h.execSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, exec)
}

// CleanExecutions deletes execution records older than the specified days.
func (h *EventPipelineHandler) CleanExecutions(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 30
	}

	affected, err := h.execSvc.CleanOlderThan(c.Request.Context(), days)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, gin.H{"deleted": affected})
}

// TryRun tests a pipeline against the most recent alert event.
func (h *EventPipelineHandler) TryRun(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	p, err := h.pipelineSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	// Get the most recent firing event for testing
	events, _, err := h.eventSvc.List(c.Request.Context(), "firing", "", 1, 1)
	if err != nil || len(events) == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "no firing alert events available for testing"))
		return
	}

	event := &events[0]
	userID := GetCurrentUserID(c)
	triggerBy := "tryrun:" + strconv.FormatUint(uint64(userID), 10)

	resultEvent, exec, execErr := h.engine.Execute(c.Request.Context(), p, event, triggerBy)

	Success(c, gin.H{
		"execution": exec,
		"event":     resultEvent,
		"error": func() string {
			if execErr != nil {
				return execErr.Error()
			}
			return ""
		}(),
	})
}

// ListProcessorTypes returns the available processor types.
func (h *EventPipelineHandler) ListProcessorTypes(c *gin.Context) {
	Success(c, pipeline.AvailableTypes())
}
