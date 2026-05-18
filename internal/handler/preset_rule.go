package handler

import (
	"io"
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
	Success(c, gin.H{"list": list, "total": total})
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
	var req struct {
		YAML string `json:"yaml" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Try multipart form fallback
		file, _, fErr := c.Request.FormFile("file")
		if fErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "yaml content is required"))
			return
		}
		defer file.Close()
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read file"))
			return
		}
		req.YAML = string(data)
	}

	count, err := h.svc.ImportFromYAML(c.Request.Context(), []byte(req.YAML))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"imported": count})
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
