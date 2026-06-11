package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// ReportTaskHandler handles report task and run API endpoints.
type ReportTaskHandler struct {
	taskSvc  *service.ReportTaskService
	schedSvc *service.ReportScheduler
	execSvc  *service.ReportExecutor
}

// NewReportTaskHandler creates a new ReportTaskHandler.
func NewReportTaskHandler(
	taskSvc *service.ReportTaskService,
	schedSvc *service.ReportScheduler,
	execSvc *service.ReportExecutor,
) *ReportTaskHandler {
	return &ReportTaskHandler{taskSvc: taskSvc, schedSvc: schedSvc, execSvc: execSvc}
}

// --- Task CRUD ---

func (h *ReportTaskHandler) ListTasks(c *gin.Context) {
	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		v := e == "true" || e == "1"
		enabled = &v
	}

	tasks, err := h.taskSvc.ListTasks(c.Request.Context(), enabled)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, tasks, int64(len(tasks)), 1, len(tasks))
}

func (h *ReportTaskHandler) GetTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	task, err := h.taskSvc.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, task)
}

func (h *ReportTaskHandler) CreateTask(c *gin.Context) {
	var task model.ReportTask
	if err := c.ShouldBindJSON(&task); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	uid, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauthorized)
		return
	}
	task.CreatedBy = uid

	if err := h.taskSvc.CreateTask(c.Request.Context(), &task); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// Register to scheduler
	if task.Enabled {
		if err := h.schedSvc.AddTask(task); err != nil {
			zap.L().Error("failed to register report task to scheduler", zap.Uint("task_id", task.ID), zap.Error(err))
		}
	}

	Success(c, task)
}

func (h *ReportTaskHandler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	existing, err := h.taskSvc.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	if err := c.ShouldBindJSON(existing); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.taskSvc.UpdateTask(c.Request.Context(), existing); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// Update scheduler
	if existing.Enabled {
		if err := h.schedSvc.AddTask(*existing); err != nil {
			zap.L().Error("failed to update report task in scheduler", zap.Uint("task_id", existing.ID), zap.Error(err))
		}
	} else {
		h.schedSvc.RemoveTask(existing.ID)
	}

	Success(c, existing)
}

func (h *ReportTaskHandler) DeleteTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.taskSvc.DeleteTask(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	h.schedSvc.RemoveTask(uint(id))
	Success(c, nil)
}

// --- Run operations ---

func (h *ReportTaskHandler) ListRuns(c *gin.Context) {
	pq := GetPageQuery(c)
	var taskID *uint
	if tid := c.Query("task_id"); tid != "" {
		v, err := strconv.ParseUint(tid, 10, 64)
		if err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid task_id"))
			return
		}
		u := uint(v)
		taskID = &u
	}

	runs, total, err := h.taskSvc.ListRuns(c.Request.Context(), taskID, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, runs, total, pq.Page, pq.PageSize)
}

func (h *ReportTaskHandler) GetRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	run, err := h.taskSvc.GetRun(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, run)
}

// RunNow triggers a report task immediately (async, does not wait for completion).
func (h *ReportTaskHandler) RunNow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	task, err := h.taskSvc.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// Execute asynchronously; use Background context to avoid cancellation after request ends.
	go func() {
		_, _ = h.execSvc.Run(context.Background(), task)
	}()

	Success(c, gin.H{"message": "报告任务已提交后台执行", "task_id": task.ID})
}
