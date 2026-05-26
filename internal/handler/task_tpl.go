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
	var tpl model.TaskTpl
	if err := c.ShouldBindJSON(&tpl); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	username := GetCurrentUsername(c)
	tpl.CreateBy = username
	tpl.UpdateBy = username

	if err := h.svc.Create(c.Request.Context(), &tpl); err != nil {
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

	if err := c.ShouldBindJSON(existing); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

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
