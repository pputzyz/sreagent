package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// TaskTplHandler handles task template API endpoints.
type TaskTplHandler struct {
	svc    *service.TaskTplService
	logger *zap.Logger
}

// CreateTaskTplRequest is the request body for creating/updating a task template.
// The Password field is included here because the model.TaskTpl struct has json:"-"
// to prevent password leakage in responses, but we still need to accept it on write.
type CreateTaskTplRequest struct {
	Name      string `json:"name" binding:"required"`
	Script    string `json:"script"`
	Args      string `json:"args"`
	Batch     int    `json:"batch"`
	Tolerance int    `json:"tolerance"`
	Timeout   int    `json:"timeout"`
	Account   string `json:"account"`
	Password  string `json:"password"`
	Pause     string `json:"pause"`
	Hosts     string `json:"hosts"`
	Tags      string `json:"tags"`
	Note      string `json:"note"`
}

// NewTaskTplHandler creates a new TaskTplHandler.
func NewTaskTplHandler(svc *service.TaskTplService, logger *zap.Logger) *TaskTplHandler {
	return &TaskTplHandler{svc: svc, logger: logger}
}

// List godoc
// @Summary List task templates
// @Tags TaskTemplate
// @Produce json
// @Param keyword query string false "Search keyword"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} types.PageData
// @Router /task-tpls [get]
func (h *TaskTplHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	keyword := c.Query("keyword")

	list, total, err := h.svc.List(c.Request.Context(), keyword, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.Wrap(apperr.ErrDatabase, err))
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Get godoc
// @Summary Get task template detail
// @Tags TaskTemplate
// @Produce json
// @Param id path int true "Template ID"
// @Success 200 {object} model.TaskTpl
// @Router /task-tpls/{id} [get]
func (h *TaskTplHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	tpl, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, tpl)
}

// Create godoc
// @Summary Create task template
// @Tags TaskTemplate
// @Accept json
// @Produce json
// @Param body body model.TaskTpl true "Template data"
// @Success 200 {object} model.TaskTpl
// @Router /task-tpls [post]
func (h *TaskTplHandler) Create(c *gin.Context) {
	var req CreateTaskTplRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	username := GetCurrentUsername(c)
	tpl := &model.TaskTpl{
		Name:      req.Name,
		Script:    req.Script,
		Args:      req.Args,
		Batch:     req.Batch,
		Tolerance: req.Tolerance,
		Timeout:   req.Timeout,
		Account:   req.Account,
		Password:  req.Password,
		Pause:     req.Pause,
		Hosts:     req.Hosts,
		Tags:      req.Tags,
		Note:      req.Note,
		CreateBy:  username,
		UpdateBy:  username,
	}

	if err := h.svc.Create(c.Request.Context(), tpl); err != nil {
		Error(c, err)
		return
	}

	Success(c, tpl)
}

// Update godoc
// @Summary Update task template
// @Tags TaskTemplate
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param body body model.TaskTpl true "Template data"
// @Success 200 {object} model.TaskTpl
// @Router /task-tpls/{id} [put]
func (h *TaskTplHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateTaskTplRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	existing.Name = req.Name
	existing.Script = req.Script
	existing.Args = req.Args
	existing.Batch = req.Batch
	existing.Tolerance = req.Tolerance
	existing.Timeout = req.Timeout
	existing.Account = req.Account
	if req.Password != "" {
		existing.Password = req.Password
	}
	existing.Pause = req.Pause
	existing.Hosts = req.Hosts
	existing.Tags = req.Tags
	existing.Note = req.Note
	existing.ID = id
	existing.UpdateBy = GetCurrentUsername(c)

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, err)
		return
	}

	Success(c, existing)
}

// Delete godoc
// @Summary Delete task template
// @Tags TaskTemplate
// @Produce json
// @Param id path int true "Template ID"
// @Success 200
// @Router /task-tpls/{id} [delete]
func (h *TaskTplHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	Success(c, nil)
}
