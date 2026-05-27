package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
)

// InspectionHandler handles inspection task and run API endpoints.
type InspectionHandler struct {
	taskRepo *repository.InspectionRepository
	schedSvc *service.InspectionScheduler
	execSvc  *service.InspectionExecutor
}

// NewInspectionHandler creates a new InspectionHandler.
func NewInspectionHandler(
	taskRepo *repository.InspectionRepository,
	schedSvc *service.InspectionScheduler,
	execSvc *service.InspectionExecutor,
) *InspectionHandler {
	return &InspectionHandler{taskRepo: taskRepo, schedSvc: schedSvc, execSvc: execSvc}
}

// --- Task CRUD ---

func (h *InspectionHandler) ListTasks(c *gin.Context) {
	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		v := e == "true" || e == "1"
		enabled = &v
	}

	tasks, err := h.taskRepo.ListTasks(c.Request.Context(), enabled)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, tasks, int64(len(tasks)), 1, len(tasks))
}

func (h *InspectionHandler) GetTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, task)
}

func (h *InspectionHandler) CreateTask(c *gin.Context) {
	var task model.InspectionTask
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

	if err := h.taskRepo.CreateTask(c.Request.Context(), &task); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// 注册到调度器
	if task.Enabled {
		if err := h.schedSvc.AddTask(task); err != nil {
			zap.L().Error("failed to register inspection task to scheduler", zap.Uint("task_id", task.ID), zap.Error(err))
		}
	}

	Success(c, task)
}

func (h *InspectionHandler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	existing, err := h.taskRepo.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	if err := c.ShouldBindJSON(existing); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.taskRepo.UpdateTask(c.Request.Context(), existing); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// 更新调度器
	if existing.Enabled {
		if err := h.schedSvc.AddTask(*existing); err != nil {
			zap.L().Error("failed to update inspection task in scheduler", zap.Uint("task_id", existing.ID), zap.Error(err))
		}
	} else {
		h.schedSvc.RemoveTask(existing.ID)
	}

	Success(c, existing)
}

func (h *InspectionHandler) DeleteTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.taskRepo.DeleteTask(c.Request.Context(), uint(id)); err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	h.schedSvc.RemoveTask(uint(id))
	Success(c, nil)
}

// --- Run operations ---

func (h *InspectionHandler) ListRuns(c *gin.Context) {
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

	runs, total, err := h.taskRepo.ListRuns(c.Request.Context(), taskID, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, runs, total, pq.Page, pq.PageSize)
}

func (h *InspectionHandler) GetRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	run, err := h.taskRepo.GetRun(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	Success(c, run)
}

// RunNow godoc
// @Summary 手动触发巡检任务
// @Description 立即执行指定的巡检任务（不等待完成，后台执行）
// @Tags Inspection
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} model.InspectionRun
// @Router /inspection/tasks/{id}/run [post]
func (h *InspectionHandler) RunNow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	// 异步执行，立即返回（使用 Background context 避免请求结束后 context 被取消）
	go func() {
		_, _ = h.execSvc.Run(context.Background(), task)
	}()

	Success(c, gin.H{"message": "巡检任务已提交后台执行", "task_id": task.ID})
}

// ValidateCron godoc
// @Summary 校验 cron 表达式
// @Description 验证 cron 表达式是否合法并返回最近 5 次触发时间
// @Tags Inspection
// @Accept json
// @Produce json
// @Param body body object true "cron expression"
// @Success 200 {object} object
// @Router /inspection/validate-cron [post]
func (h *InspectionHandler) ValidateCron(c *gin.Context) {
	var req struct {
		CronExpr string `json:"cron_expr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(req.CronExpr)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "无效的 cron 表达式: "+err.Error()))
		return
	}

	var nextRuns []string
	now := time.Now()
	t := now
	for i := 0; i < 5; i++ {
		t = sched.Next(t)
		nextRuns = append(nextRuns, t.Format(time.RFC3339))
	}

	Success(c, gin.H{"valid": true, "next_runs": nextRuns})
}
