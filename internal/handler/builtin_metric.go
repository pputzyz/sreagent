package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type BuiltinMetricHandler struct {
	svc        *service.BuiltinMetricService
	filterSvc  *service.MetricFilterService
	log        *zap.Logger
}

func NewBuiltinMetricHandler(svc *service.BuiltinMetricService, filterSvc *service.MetricFilterService, log *zap.Logger) *BuiltinMetricHandler {
	return &BuiltinMetricHandler{svc: svc, filterSvc: filterSvc, log: log}
}

// List returns builtin metrics with filtering and pagination.
func (h *BuiltinMetricHandler) List(c *gin.Context) {
	collector := c.Query("collector")
	typ := c.Query("typ")
	query := c.Query("query")
	unit := c.Query("unit")
	lang := c.GetHeader("Lang")
	if lang == "" {
		lang = "zh"
	}
	pq := GetPageQuery(c)

	metrics, total, err := h.svc.List(c.Request.Context(), collector, typ, query, unit, lang, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	SuccessPage(c, metrics, total, pq.Page, pq.PageSize)
}

// Get returns a single builtin metric by ID.
func (h *BuiltinMetricHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "invalid id"))
		return
	}
	m, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, m)
}

// Create creates a new builtin metric.
func (h *BuiltinMetricHandler) Create(c *gin.Context) {
	var req CreateBuiltinMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	m := &model.BuiltinMetric{
		Collector:      req.Collector,
		Typ:            req.Typ,
		Name:           req.Name,
		Unit:           req.Unit,
		Note:           req.Note,
		Lang:           req.Lang,
		Expression:     req.Expression,
		ExpressionType: req.ExpressionType,
		MetricType:     req.MetricType,
		ExtraFieldsJSON: req.ExtraFields,
		TranslationJSON: req.Translation,
		CreatedBy:      uidStr,
		UpdatedBy:      uidStr,
	}
	if m.Lang == "" {
		m.Lang = "zh"
	}
	if m.ExpressionType == "" {
		m.ExpressionType = "metric_name"
	}

	if err := h.svc.Create(c.Request.Context(), m); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, m)
}

// Update updates an existing builtin metric.
func (h *BuiltinMetricHandler) Update(c *gin.Context) {
	var req UpdateBuiltinMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	existing, err := h.svc.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		Error(c, err)
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	existing.Collector = req.Collector
	existing.Typ = req.Typ
	existing.Name = req.Name
	existing.Unit = req.Unit
	existing.Note = req.Note
	existing.Expression = req.Expression
	existing.ExpressionType = req.ExpressionType
	existing.MetricType = req.MetricType
	existing.ExtraFieldsJSON = req.ExtraFields
	existing.TranslationJSON = req.Translation
	existing.UpdatedBy = uidStr

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, nil)
}

// Delete deletes builtin metrics by IDs.
func (h *BuiltinMetricHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), req.IDs); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, nil)
}

// Types returns distinct metric types.
func (h *BuiltinMetricHandler) Types(c *gin.Context) {
	collector := c.Query("collector")
	query := c.Query("query")
	lang := c.GetHeader("Lang")

	types, err := h.svc.GetTypes(c.Request.Context(), collector, query, lang)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, types)
}

// Collectors returns distinct collectors.
func (h *BuiltinMetricHandler) Collectors(c *gin.Context) {
	typ := c.Query("typ")
	query := c.Query("query")
	lang := c.GetHeader("Lang")

	collectors, err := h.svc.GetCollectors(c.Request.Context(), typ, query, lang)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, collectors)
}

// BatchCreate creates multiple metrics (for import).
func (h *BuiltinMetricHandler) BatchCreate(c *gin.Context) {
	var req struct {
		Metrics []model.BuiltinMetric `json:"metrics" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	results := make(map[string]string, len(req.Metrics))
	for _, m := range req.Metrics {
		m.ID = 0
		m.CreatedBy = uidStr
		m.UpdatedBy = uidStr
		if err := h.svc.Create(c.Request.Context(), &m); err != nil {
			results[m.Name] = err.Error()
		} else {
			results[m.Name] = ""
		}
	}
	Success(c, results)
}

// --- Metric Filter handlers ---

// ListFilters returns filters for the current user.
func (h *BuiltinMetricHandler) ListFilters(c *gin.Context) {
	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	filters, err := h.filterSvc.List(c.Request.Context(), uidStr)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, filters)
}

// CreateFilter creates a new metric filter.
func (h *BuiltinMetricHandler) CreateFilter(c *gin.Context) {
	var req CreateMetricFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	f := &model.MetricFilter{
		Name:           req.Name,
		ConfigsJSON:    req.Configs,
		GroupsPermJSON: req.GroupsPerm,
		CreatedBy:      uidStr,
		UpdatedBy:      uidStr,
	}

	if err := h.filterSvc.Create(c.Request.Context(), f); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, f)
}

// UpdateFilter updates a metric filter.
func (h *BuiltinMetricHandler) UpdateFilter(c *gin.Context) {
	var req UpdateMetricFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)
	uidStr := strconv.FormatUint(uint64(userID), 10)

	f := &model.MetricFilter{
		ID:             req.ID,
		Name:           req.Name,
		ConfigsJSON:    req.Configs,
		GroupsPermJSON: req.GroupsPerm,
		UpdatedBy:      uidStr,
	}

	if err := h.filterSvc.Update(c.Request.Context(), f); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, nil)
}

// DeleteFilter deletes metric filters by IDs.
func (h *BuiltinMetricHandler) DeleteFilter(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.filterSvc.Delete(c.Request.Context(), req.IDs); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, err.Error()))
		return
	}
	Success(c, nil)
}

// --- Request types ---

type CreateBuiltinMetricRequest struct {
	Collector      string                 `json:"collector"`
	Typ            string                 `json:"typ"`
	Name           string                 `json:"name" binding:"required"`
	Unit           string                 `json:"unit"`
	Note           string                 `json:"note"`
	Lang           string                 `json:"lang"`
	Expression     string                 `json:"expression" binding:"required"`
	ExpressionType string                 `json:"expression_type"`
	MetricType     string                 `json:"metric_type"`
	ExtraFields    map[string]string      `json:"extra_fields"`
	Translation    []model.TranslationEntry `json:"translation"`
}

type UpdateBuiltinMetricRequest struct {
	ID             uint                   `json:"id" binding:"required"`
	Collector      string                 `json:"collector"`
	Typ            string                 `json:"typ"`
	Name           string                 `json:"name" binding:"required"`
	Unit           string                 `json:"unit"`
	Note           string                 `json:"note"`
	Expression     string                 `json:"expression" binding:"required"`
	ExpressionType string                 `json:"expression_type"`
	MetricType     string                 `json:"metric_type"`
	ExtraFields    map[string]string      `json:"extra_fields"`
	Translation    []model.TranslationEntry `json:"translation"`
}

type CreateMetricFilterRequest struct {
	Name        string              `json:"name" binding:"required"`
	Configs     []model.FilterConfig `json:"configs"`
	GroupsPerm  []model.GroupPerm    `json:"groups_perm"`
}

type UpdateMetricFilterRequest struct {
	ID          uint                `json:"id" binding:"required"`
	Name        string              `json:"name" binding:"required"`
	Configs     []model.FilterConfig `json:"configs"`
	GroupsPerm  []model.GroupPerm    `json:"groups_perm"`
}
