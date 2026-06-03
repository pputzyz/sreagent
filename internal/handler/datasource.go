package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
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

func NewDataSourceHandler(svc *service.DataSourceService, logger *zap.Logger) *DataSourceHandler {
	return &DataSourceHandler{svc: svc, log: logger}
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

	ds.AuthConfig = "" // mask credentials before returning
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
	search := c.Query("search")

	list, total, err := h.svc.List(c.Request.Context(), dsType, search, pq.Page, pq.PageSize)
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

	ds.AuthConfig = "" // mask credentials before returning
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

// TestConnectionRaw tests connectivity to a datasource endpoint without requiring a saved datasource.
// P1-15: POST /api/v1/datasources/test-connection
func (h *DataSourceHandler) TestConnectionRaw(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required"`
		Endpoint   string `json:"endpoint" binding:"required"`
		AuthType   string `json:"auth_type"`
		AuthConfig string `json:"auth_config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := validateEndpointScheme(req.Endpoint); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	result, err := h.svc.TestConnectionRaw(c.Request.Context(), req.Type, req.Endpoint, req.AuthType, req.AuthConfig)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// toTime converts a unix timestamp in seconds (with fractional precision) to time.Time.
// Handles millisecond/sub-second precision correctly (P1-3).
func toTime(unixSec float64) time.Time {
	if unixSec <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(unixSec * 1000))
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

	queryTime := toTime(req.Time)

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

	start := toTime(req.Start)
	end := toTime(req.End)

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
		Index      string  `json:"index"`                    // Elasticsearch index
		DateField  string  `json:"date_field"`               // Elasticsearch date field
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	start := toTime(req.Start)
	end := toTime(req.End)

	result, err := h.svc.QueryLogs(c.Request.Context(), id, service.LogQueryParams{
		Expression: req.Expression,
		Start:      start,
		End:        end,
		Limit:      req.Limit,
		Index:      req.Index,
		DateField:  req.DateField,
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
		Step       string  `json:"step"`        // e.g. "1m", "5m", "1h"
		Index      string  `json:"index"`       // Elasticsearch index
		DateField  string  `json:"date_field"`  // Elasticsearch date field
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

	start := toTime(req.Start)
	end := toTime(req.End)

	result, err := h.svc.QueryLogHistogram(c.Request.Context(), id, service.LogHistogramParams{
		Expression: req.Expression,
		Start:      start,
		End:        end,
		Step:       req.Step,
		Index:      req.Index,
		DateField:  req.DateField,
	})
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, result)
}

// Proxy proxies HTTP GET requests to the target datasource endpoint.
// GET /api/v1/datasources/:id/proxy/*path
// This is the Nightingale pattern: transparent proxy for datasource API calls.
// P1-7: Restricted to GET only. P1-9: Path whitelist for security.
func (h *DataSourceHandler) Proxy(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	// Extract the path after /proxy and normalize to prevent traversal.
	proxyPath := path.Clean(c.Param("path"))
	if proxyPath == "" || proxyPath == "." {
		proxyPath = "/"
	}

	// P1-9: Path whitelist validation
	allowedPrefixes := []string{
		"/api/v1/labels", "/api/v1/label/", "/api/v1/series",
		"/api/v1/query", "/api/v1/query_range",
		"/api/v1/status/", "/api/v1/metadata",
	}
	blockedPrefixes := []string{
		"/api/v1/admin", "/_security", "/_cat/", "/_cluster",
	}

	for _, blocked := range blockedPrefixes {
		if strings.HasPrefix(proxyPath, blocked) {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "access to this path is blocked"))
			return
		}
	}

	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(proxyPath, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "path not in proxy whitelist"))
		return
	}

	// Forward query parameters
	params := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	body, err := h.svc.ProxyToDatasource(c.Request.Context(), id, proxyPath, params)
	if err != nil {
		Error(c, err)
		return
	}

	// Return raw response (JSON passthrough)
	c.Data(200, "application/json", body)
}

// DsQuery is the unified query endpoint (Nightingale pattern).
// POST /api/v1/ds-query
// Supports multiple queries against different datasources concurrently.
func (h *DataSourceHandler) DsQuery(c *gin.Context) {
	var req struct {
		Queries []struct {
			DatasourceID uint    `json:"datasource_id" binding:"required"`
			Expression   string  `json:"expression" binding:"required"`
			Start        float64 `json:"start"` // unix seconds (float for ms precision), 0 = instant
			End          float64 `json:"end"`   // unix seconds
			Step         string  `json:"step"`  // e.g. "15s"
		} `json:"queries" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// P1-8: Limit concurrent queries per request
	const maxQueriesPerRequest = 50
	if len(req.Queries) > maxQueriesPerRequest {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("too many queries: max %d", maxQueriesPerRequest)))
		return
	}

	type queryResult struct {
		Index  int         `json:"index"`
		Data  interface{} `json:"data"`
		Error string      `json:"error,omitempty"`
	}

	results := make([]queryResult, len(req.Queries))
	var wg sync.WaitGroup

	for i, q := range req.Queries {
		wg.Add(1)
		go func(idx int, query struct {
			DatasourceID uint    `json:"datasource_id" binding:"required"`
			Expression   string  `json:"expression" binding:"required"`
			Start        float64 `json:"start"`
			End          float64 `json:"end"`
			Step         string  `json:"step"`
		}) {
			defer wg.Done()
			ctx := c.Request.Context()

			if query.Start > 0 && query.End > 0 {
				// Range query
				start := toTime(query.Start)
				end := toTime(query.End)
				step := query.Step
				if step == "" {
					step = "15s"
				}
				data, err := h.svc.QueryRange(ctx, query.DatasourceID, query.Expression, start, end, step)
				if err != nil {
					results[idx] = queryResult{Index: idx, Error: err.Error()}
				} else {
					results[idx] = queryResult{Index: idx, Data: data}
				}
			} else {
				// Instant query
				data, err := h.svc.QueryDatasource(ctx, query.DatasourceID, query.Expression, time.Time{})
				if err != nil {
					results[idx] = queryResult{Index: idx, Error: err.Error()}
				} else {
					results[idx] = queryResult{Index: idx, Data: data}
				}
			}
		}(i, q)
	}

	wg.Wait()
	Success(c, results)
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

// GetESIndices returns non-hidden Elasticsearch indices for the given datasource.
// GET /api/v1/datasources/:id/es-indices
func (h *DataSourceHandler) GetESIndices(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	indices, err := h.svc.GetESIndices(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, indices)
}

// GetESFields returns field names and types for a given Elasticsearch index.
// GET /api/v1/datasources/:id/es-fields?index=my-index
func (h *DataSourceHandler) GetESFields(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	index := c.Query("index")
	if index == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "index query parameter is required"))
		return
	}

	fields, err := h.svc.GetESFields(c.Request.Context(), id, index)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, fields)
}

// VariableStreamSSE streams real-time variable values via SSE.
// GET /api/v1/datasources/variables/stream?datasource_id=1&expression=up&interval=30&regex=.*
//
// Each SSE message: data: {"variable":"name","value":"val","timestamp":"RFC3339"}\n\n
// On error:         data: {"error":"..."}\n\n
func (h *DataSourceHandler) VariableStreamSSE(c *gin.Context) {
	dsIDStr := c.Query("datasource_id")
	if dsIDStr == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "datasource_id is required"))
		return
	}
	var dsID uint
	if _, err := fmt.Sscanf(dsIDStr, "%d", &dsID); err != nil || dsID == 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "datasource_id must be a positive integer"))
		return
	}

	expression := c.Query("expression")
	if expression == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "expression is required"))
		return
	}

	intervalSec := 30
	if v := c.Query("interval"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &intervalSec); err != nil || n != 1 || intervalSec < 5 {
			intervalSec = 30
		}
	}
	intervalSec = int(math.Min(float64(intervalSec), 3600)) // cap at 1 hour

	regexFilter := c.Query("regex")

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send initial values immediately
	h.sendVariableSnapshot(c, dsID, expression, regexFilter)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			h.sendVariableSnapshot(c, dsID, expression, regexFilter)
			if c.IsAborted() {
				return
			}
		}
	}
}

// sendVariableSnapshot executes the datasource query and writes SSE data.
func (h *DataSourceHandler) sendVariableSnapshot(c *gin.Context, dsID uint, expression, regexFilter string) {
	result, err := h.svc.QueryDatasource(c.Request.Context(), dsID, expression, time.Time{})
	if err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		c.Writer.Flush()
		return
	}

	// Build the value list from series labels
	var values []string
	for _, s := range result.Series {
		for k, v := range s.Labels {
			if k != "__name__" {
				values = append(values, v)
				break // take first non-__name__ label
			}
		}
	}

	// Apply optional regex filter
	if regexFilter != "" {
		re, err := regexp.Compile(regexFilter)
		if err == nil {
			filtered := values[:0]
			for _, v := range values {
				if re.MatchString(v) {
					filtered = append(filtered, v)
				}
			}
			values = filtered
		}
	}

	payload := map[string]interface{}{
		"variable":  expression,
		"value":     values,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}
