package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type AlertRuleTemplateHandler struct {
	svc *service.AlertRuleTemplateService
}

func NewAlertRuleTemplateHandler(svc *service.AlertRuleTemplateService) *AlertRuleTemplateHandler {
	return &AlertRuleTemplateHandler{svc: svc}
}

func (h *AlertRuleTemplateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	category := c.Query("category")
	search := c.Query("search")

	list, total, err := h.svc.List(c.Request.Context(), category, search, page, pageSize)
	if err != nil {
		Error(c, err)
		return
	}
	SuccessPage(c, list, total, page, pageSize)
}

func (h *AlertRuleTemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	tpl, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, tpl)
}

func (h *AlertRuleTemplateHandler) Create(c *gin.Context) {
	var tpl model.AlertRuleTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.Create(c.Request.Context(), &tpl); err != nil {
		Error(c, err)
		return
	}
	Success(c, tpl)
}

func (h *AlertRuleTemplateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	var tpl model.AlertRuleTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	tpl.ID = uint(id)
	if err := h.svc.Update(c.Request.Context(), &tpl); err != nil {
		Error(c, err)
		return
	}
	Success(c, tpl)
}

func (h *AlertRuleTemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

func (h *AlertRuleTemplateHandler) ListCategories(c *gin.Context) {
	cats, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, cats)
}

func (h *AlertRuleTemplateHandler) Apply(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	var overrides model.AlertRule
	if err := c.ShouldBindJSON(&overrides); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	rule, err := h.svc.ApplyTemplate(c.Request.Context(), uint(id), &overrides)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}
