package handler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// validateEndpointScheme ensures the endpoint URL uses only http or https.
func validateEndpointScheme(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("endpoint URL scheme must be http or https, got %q", scheme)
	}
	return nil
}

type DataSourceHandler struct {
	svc      *service.DataSourceService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

func NewDataSourceHandler(svc *service.DataSourceService, logger ...*zap.Logger) *DataSourceHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &DataSourceHandler{svc: svc, log: l}
}

// SetAuditService injects the audit log service (called after construction to avoid circular DI).
func (h *DataSourceHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

// CreateDataSourceRequest is the request body for creating/updating a datasource.
type CreateDataSourceRequest struct {
	Name                string               `json:"name" binding:"required"`
	Type                model.DataSourceType `json:"type" binding:"required"`
	Endpoint            string               `json:"endpoint" binding:"required,url"`
	Description         string               `json:"description"`
	Labels              model.JSONLabels     `json:"labels"`
	AuthType            string               `json:"auth_type"`
	AuthConfig          string               `json:"auth_config"`
	HealthCheckInterval int                  `json:"health_check_interval"`
	IsEnabled           *bool                `json:"is_enabled"`
}

// Create creates a new datasource.
func (h *DataSourceHandler) Create(c *gin.Context) {
	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := validateEndpointScheme(req.Endpoint); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("datasource create",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.String("name", req.Name),
		zap.String("type", string(req.Type)),
		zap.String("request_id", c.GetString("request_id")))

	ds := &model.DataSource{
		Name:                req.Name,
		Type:                req.Type,
		Endpoint:            req.Endpoint,
		Description:         req.Description,
		Labels:              req.Labels,
		AuthType:            req.AuthType,
		AuthConfig:          req.AuthConfig,
		HealthCheckInterval: req.HealthCheckInterval,
		IsEnabled:           req.IsEnabled == nil || *req.IsEnabled, // default true
	}

	if err := h.svc.Create(c.Request.Context(), ds); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionCreate,
			ResourceType: model.AuditResourceDatasource, ResourceID: &ds.ID, ResourceName: ds.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, ds)
}

// Get returns a single datasource by ID.
func (h *DataSourceHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	ds, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, ds)
}

// List returns a paginated list of datasources.
func (h *DataSourceHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	dsType := c.Query("type")

	list, total, err := h.svc.List(c.Request.Context(), dsType, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	SuccessPage(c, list, total, pq.Page, pq.PageSize)
}

// Update updates a datasource.
func (h *DataSourceHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := validateEndpointScheme(req.Endpoint); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	h.log.Info("datasource update",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("datasource_id", id),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

	ds := &model.DataSource{
		Name:                req.Name,
		Type:                req.Type,
		Endpoint:            req.Endpoint,
		Description:         req.Description,
		Labels:              req.Labels,
		AuthType:            req.AuthType,
		AuthConfig:          req.AuthConfig,
		HealthCheckInterval: req.HealthCheckInterval,
	}
	if req.IsEnabled != nil {
		ds.IsEnabled = *req.IsEnabled
	}
	ds.ID = id

	if err := h.svc.Update(c.Request.Context(), ds); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionUpdate,
			ResourceType: model.AuditResourceDatasource, ResourceID: &id, ResourceName: req.Name,
			IP: c.ClientIP(),
		})
	}

	Success(c, ds)
}

// Delete deletes a datasource.
func (h *DataSourceHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("datasource delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("datasource_id", id),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Action: model.AuditActionDelete,
			ResourceType: model.AuditResourceDatasource, ResourceID: &id,
			IP: c.ClientIP(),
		})
	}

	Success(c, nil)
}

// HealthCheck triggers a health check for a datasource.
func (h *DataSourceHandler) HealthCheck(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	result, err := h.svc.HealthCheck(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// Query tests an expression against a datasource.
// POST /api/v1/datasources/:id/query
func (h *DataSourceHandler) Query(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Expression string  `json:"expression" binding:"required"`
		Time       float64 `json:"time"` // unix timestamp in seconds, 0 or omitted = now
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	var queryTime time.Time
	if req.Time > 0 {
		queryTime = time.UnixMilli(int64(req.Time * 1000))
	}

	result, err := h.svc.QueryDatasource(c.Request.Context(), id, req.Expression, queryTime)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// RangeQuery executes a PromQL range query against a datasource.
// POST /api/v1/datasources/:id/query-range
func (h *DataSourceHandler) RangeQuery(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Expression string  `json:"expression" binding:"required"`
		Start      float64 `json:"start" binding:"required"` // unix timestamp in seconds
		End        float64 `json:"end" binding:"required"`   // unix timestamp in seconds
		Step       string  `json:"step" binding:"required"`  // e.g. "15s", "1m", "5m"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	start := time.Unix(int64(req.Start), 0)
	end := time.Unix(int64(req.End), 0)

	result, err := h.svc.QueryRange(c.Request.Context(), id, req.Expression, start, end, req.Step)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// LabelKeys returns label names from the target datasource (for PromQL autocompletion).
// GET /api/v1/datasources/:id/labels/keys
func (h *DataSourceHandler) LabelKeys(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	body, err := h.svc.ProxyToDatasource(c.Request.Context(), id, "/api/v1/labels", nil)
	if err != nil {
		Error(c, err)
		return
	}

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to parse label keys response"))
		return
	}

	Success(c, apiResp.Data)
}

// LabelValues returns values for a given label key from the target datasource.
// GET /api/v1/datasources/:id/labels/values?key=job
func (h *DataSourceHandler) LabelValues(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	key := c.Query("key")
	if key == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "key parameter is required"))
		return
	}

	body, err := h.svc.ProxyToDatasource(c.Request.Context(), id, "/api/v1/label/"+url.PathEscape(key)+"/values", nil)
	if err != nil {
		Error(c, err)
		return
	}

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to parse label values response"))
		return
	}

	Success(c, apiResp.Data)
}

// LogQuery executes a LogsQL query against a VictoriaLogs datasource and returns log entries.
// POST /api/v1/datasources/:id/log-query
func (h *DataSourceHandler) LogQuery(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Expression string  `json:"expression" binding:"required"`
		Start      float64 `json:"start" binding:"required"` // unix timestamp in seconds
		End        float64 `json:"end" binding:"required"`   // unix timestamp in seconds
		Limit      int     `json:"limit"`                    // max entries, default 100
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	start := time.Unix(int64(req.Start), 0)
	end := time.Unix(int64(req.End), 0)

	result, err := h.svc.QueryLogs(c.Request.Context(), id, service.LogQueryParams{
		Expression: req.Expression,
		Start:      start,
		End:        end,
		Limit:      req.Limit,
	})
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// LogHistogram returns log hit counts over time buckets for histogram visualization.
// POST /api/v1/datasources/:id/log-histogram
func (h *DataSourceHandler) LogHistogram(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Expression string  `json:"expression" binding:"required"`
		Start      float64 `json:"start" binding:"required"`
		End        float64 `json:"end" binding:"required"`
		Step       string  `json:"step"` // e.g. "1m", "5m", "1h"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// Auto-calculate step if not provided
	if req.Step == "" {
		diff := req.End - req.Start
		switch {
		case diff <= 3600:
			req.Step = "1m"
		case diff <= 86400:
			req.Step = "5m"
		case diff <= 604800:
			req.Step = "1h"
		default:
			req.Step = "1d"
		}
	}

	start := time.Unix(int64(req.Start), 0)
	end := time.Unix(int64(req.End), 0)

	result, err := h.svc.QueryLogHistogram(c.Request.Context(), id, service.LogHistogramParams{
		Expression: req.Expression,
		Start:      start,
		End:        end,
		Step:       req.Step,
	})
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// MetricNames returns metric names from the target datasource (for PromQL autocompletion).
// GET /api/v1/datasources/:id/metrics?search=http&limit=100
func (h *DataSourceHandler) MetricNames(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	params := map[string]string{
		"label_name": "__name__",
	}
	if search := c.Query("search"); search != "" {
		params["search"] = search
	}
	if limit := c.Query("limit"); limit != "" {
		params["limit"] = limit
	}

	body, err := h.svc.ProxyToDatasource(c.Request.Context(), id, "/api/v1/label/__name__/values", params)
	if err != nil {
		Error(c, err)
		return
	}

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrExternalAPI, "failed to parse metric names response"))
		return
	}

	Success(c, apiResp.Data)
}
