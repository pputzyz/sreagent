package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type PresetRuleHandler struct {
	svc *service.PresetRuleService
}

func NewPresetRuleHandler(svc *service.PresetRuleService) *PresetRuleHandler {
	return &PresetRuleHandler{svc: svc}
}

func (h *PresetRuleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
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

func (h *PresetRuleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	rule, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

func (h *PresetRuleHandler) Categories(c *gin.Context) {
	cats, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, cats)
}

func (h *PresetRuleHandler) Apply(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	var override service.PresetRuleOverride
	if err := c.ShouldBindJSON(&override); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	rule, err := h.svc.Apply(c.Request.Context(), uint(id), &override)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, rule)
}

func (h *PresetRuleHandler) Import(c *gin.Context) {
	yamlContent, err := readYAMLInput(c)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	count, err := h.svc.ImportFromYAML(c.Request.Context(), yamlContent)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"imported": count})
}

func (h *PresetRuleHandler) BatchApply(c *gin.Context) {
	var req struct {
		PresetIDs            []uint `json:"preset_ids" binding:"required,min=1"`
		AutoMatchDatasource  bool   `json:"auto_match_datasource"`
		FallbackDatasourceID uint   `json:"fallback_datasource_id"`
		ChannelID            uint   `json:"channel_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	applied, failed := h.svc.BatchApply(c.Request.Context(), req.PresetIDs, req.AutoMatchDatasource, req.FallbackDatasourceID, req.ChannelID)
	Success(c, gin.H{
		"applied": applied,
		"failed":  failed,
	})
}

func (h *PresetRuleHandler) Delete(c *gin.Context) {
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
