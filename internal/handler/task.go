package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// TaskHandler handles task execution and record API endpoints.
type TaskHandler struct {
	executor *service.TaskExecutor
	recSvc   *service.TaskRecordService
	logger   *zap.Logger
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(executor *service.TaskExecutor, recSvc *service.TaskRecordService, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{executor: executor, recSvc: recSvc, logger: logger}
}

// Execute godoc
// @Summary Execute a task from template
// @Description Creates a task record and starts execution on specified hosts
// @Tags Task
// @Accept json
// @Produce json
// @Param body body service.ExecuteTaskRequest true "Execute request"
// @Success 200 {object} model.TaskRecord
// @Router /tasks [post]
func (h *TaskHandler) Execute(c *gin.Context) {
	var req service.ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	if userID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "user not found"))
		return
	}

	record, err := h.executor.ExecuteTask(c.Request.Context(), &req, userID)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrBusiness, err.Error()))
		return
	}

	Success(c, record)
}

// ExecuteDirect godoc
// @Summary Execute a task directly (without template)
// @Tags Task
// @Accept json
// @Produce json
// @Param body body object true "Direct execute request"
// @Success 200 {object} model.TaskRecord
// @Router /tasks/direct [post]
func (h *TaskHandler) ExecuteDirect(c *gin.Context) {
	var req struct {
		Script    string   `json:"script" binding:"required"`
		Args      string   `json:"args"`
		Account   string   `json:"account"`
		Timeout   int      `json:"timeout"`
		Batch     int      `json:"batch"`
		Tolerance int      `json:"tolerance"`
		Hosts     []string `json:"hosts" binding:"required"`
		Title     string   `json:"title"`
		EventID   uint     `json:"event_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	if userID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "user not found"))
		return
	}

	if req.Title == "" {
		req.Title = "Direct Execution"
	}

	record, err := h.executor.ExecuteDirect(c.Request.Context(),
		req.Script, req.Args, req.Account,
		req.Timeout, req.Batch, req.Tolerance,
		req.Hosts, req.Title, userID, req.EventID,
	)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrBusiness, err.Error()))
		return
	}

	Success(c, record)
}

// ListRecords godoc
// @Summary List task execution records
// @Tags Task
// @Produce json
// @Param tpl_id query int false "Filter by template ID"
// @Param event_id query int false "Filter by event ID"
// @Param status query int false "Filter by status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} types.PageData
// @Router /tasks [get]
func (h *TaskHandler) ListRecords(c *gin.Context) {
	pq := GetPageQuery(c)

	var tplID *uint
	if v := c.Query("tpl_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			u := uint(id)
			tplID = &u
		}
	}

	var eventID *uint
	if v := c.Query("event_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			u := uint(id)
			eventID = &u
		}
	}

	var status *int
	if v := c.Query("status"); v != "" {
		s, err := strconv.Atoi(v)
		if err == nil {
			status = &s
		}
	}

	list, total, err := h.recSvc.ListRecords(c.Request.Context(), tplID, eventID, status, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// GetRecord godoc
// @Summary Get task execution record detail
// @Tags Task
// @Produce json
// @Param id path int true "Record ID"
// @Success 200 {object} model.TaskRecord
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetRecord(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	record, err := h.recSvc.GetRecordByID(c.Request.Context(), id)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, record)
}

// ListHostRecords godoc
// @Summary Get host execution details for a task
// @Tags Task
// @Produce json
// @Param id path int true "Task Record ID"
// @Success 200 {object} []model.TaskHostRecord
// @Router /tasks/{id}/hosts [get]
func (h *TaskHandler) ListHostRecords(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	records, err := h.recSvc.ListHostRecords(c.Request.Context(), id)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, records)
}

// GetHostRecord godoc
// @Summary Get single host execution detail
// @Tags Task
// @Produce json
// @Param id path int true "Host Record ID"
// @Success 200 {object} model.TaskHostRecord
// @Router /tasks/hosts/{id} [get]
func (h *TaskHandler) GetHostRecord(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	record, err := h.recSvc.GetHostRecordByID(c.Request.Context(), id)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, record)
}
