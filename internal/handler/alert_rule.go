package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type AlertRuleHandler struct {
	svc      *service.AlertRuleService
	auditSvc *service.AuditLogService
}

func NewAlertRuleHandler(svc *service.AlertRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{svc: svc}
}

func (h *AlertRuleHandler) SetAuditService(svc *service.AuditLogService) {
	h.auditSvc = svc
}

type CreateAlertRuleRequest struct {
	Name                 string               `json:"name" binding:"required"`
	DisplayName          string               `json:"display_name"`
	Description          string               `json:"description"`
	DataSourceID         *uint                `json:"datasource_id"`
	DatasourceType       model.DataSourceType `json:"datasource_type"`
	Expression           string               `json:"expression" binding:"required"`
	ForDuration          string               `json:"for_duration"`
	Severity             model.AlertSeverity  `json:"severity" binding:"required"`
	Labels               model.JSONLabels     `json:"labels"`
	Annotations          model.JSONLabels     `json:"annotations"`
	GroupName            string               `json:"group_name"`
	Category             string               `json:"category"`
	GroupWaitSeconds     int                  `json:"group_wait_seconds"`
	GroupIntervalSeconds int                  `json:"group_interval_seconds"`
	// Source indicates the origin of this rule (e.g. "ai", "import", "manual").
	Source string `json:"source"`
}

func (h *AlertRuleHandler) Create(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	rule := &model.AlertRule{
		Name:                 req.Name,
		DisplayName:          req.DisplayName,
		Description:          req.Description,
		DataSourceID:         req.DataSourceID,
		DatasourceType:       req.DatasourceType,
		Expression:           req.Expression,
		ForDuration:          req.ForDuration,
		Severity:             req.Severity,
		Labels:               req.Labels,
		Annotations:          req.Annotations,
		GroupName:            req.GroupName,
		Category:             req.Category,
		GroupWaitSeconds:     req.GroupWaitSeconds,
		GroupIntervalSeconds: req.GroupIntervalSeconds,
		Status:               model.RuleStatusEnabled,
		CreatedBy:            GetCurrentUserID(c),
	}

	if err := h.svc.Create(c.Request.Context(), rule, req.Source); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := rule.CreatedBy
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionCreate, ResourceType: model.AuditResourceAlertRule,
			ResourceID: &rule.ID, ResourceName: rule.Name, IP: c.ClientIP(),
		})
	}
	Success(c, rule)
}

func (h *AlertRuleHandler) Get(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, rule.MaskHeartbeatToken())
}

func (h *AlertRuleHandler) List(c *gin.Context) {
	pq := GetPageQuery(c)
	severity := c.Query("severity")
	status := c.Query("status")
	groupName := c.Query("group_name")
	category := c.Query("category")

	list, total, err := h.svc.List(c.Request.Context(), severity, status, groupName, category, pq.Page, pq.PageSize)
	if err != nil {
		Error(c, err)
		return
	}

	masked := make([]model.AlertRule, len(list))
	for i, r := range list {
		masked[i] = r.MaskHeartbeatToken()
	}

	SuccessPage(c, masked, total, pq.Page, pq.PageSize)
}

func (h *AlertRuleHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	rule := &model.AlertRule{
		Name:                 req.Name,
		DisplayName:          req.DisplayName,
		Description:          req.Description,
		DataSourceID:         req.DataSourceID,
		DatasourceType:       req.DatasourceType,
		Expression:           req.Expression,
		ForDuration:          req.ForDuration,
		Severity:             req.Severity,
		Labels:               req.Labels,
		Annotations:          req.Annotations,
		GroupName:            req.GroupName,
		Category:             req.Category,
		GroupWaitSeconds:     req.GroupWaitSeconds,
		GroupIntervalSeconds: req.GroupIntervalSeconds,
		UpdatedBy:            GetCurrentUserID(c),
	}
	rule.ID = id

	if err := h.svc.Update(c.Request.Context(), rule); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := rule.UpdatedBy
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionUpdate, ResourceType: model.AuditResourceAlertRule,
			ResourceID: &rule.ID, ResourceName: rule.Name, IP: c.ClientIP(),
		})
	}
	Success(c, rule)
}

func (h *AlertRuleHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionDelete, ResourceType: model.AuditResourceAlertRule,
			ResourceID: &id, IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// Import handles batch import of alert rules from a YAML or JSON file.
func (h *AlertRuleHandler) Import(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "file upload required: "+err.Error()))
		return
	}
	defer file.Close()

	datasourceIDStr := c.PostForm("datasource_id")
	var datasourceID *uint
	if datasourceIDStr != "" {
		var id uint64
		if _, err := fmt.Sscanf(datasourceIDStr, "%d", &id); err == nil {
			uid := uint(id)
			datasourceID = &uid
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read file: "+err.Error()))
		return
	}

	var ruleFile model.PrometheusRuleFile

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &ruleFile); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to parse YAML: "+err.Error()))
			return
		}
	case ".json":
		if err := json.Unmarshal(data, &ruleFile); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to parse JSON: "+err.Error()))
			return
		}
	default:
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "unsupported file format: "+ext+". Use .yaml, .yml, or .json"))
		return
	}

	// Convert Prometheus rules to AlertRule models
	var rules []model.AlertRule
	userID := GetCurrentUserID(c)
	for _, group := range ruleFile.Groups {
		for _, r := range group.Rules {
			severity := model.SeverityWarning
			if sev, ok := r.Labels["severity"]; ok {
				switch sev {
				case "critical":
					severity = model.SeverityCritical
				case "warning":
					severity = model.SeverityWarning
				case "info":
					severity = model.SeverityInfo
				}
			}

			rule := model.AlertRule{
				Name:         r.Alert,
				DisplayName:  r.Alert,
				Expression:   r.Expr,
				ForDuration:  r.For,
				Severity:     severity,
				Labels:       model.JSONLabels(r.Labels),
				Annotations:  model.JSONLabels(r.Annotations),
				GroupName:    group.Name,
				Status:       model.RuleStatusEnabled,
				DataSourceID: datasourceID,
				CreatedBy:    userID,
			}
			rules = append(rules, rule)
		}
	}

	success, failed, errors := h.svc.ImportRules(c.Request.Context(), rules)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"total":   len(rules),
			"success": success,
			"failed":  failed,
			"errors":  errors,
		},
	})
}

// LabelValidationPreview returns a dry-run preview of label validation across all rules.
func (h *AlertRuleHandler) LabelValidationPreview(c *gin.Context) {
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	result, err := h.svc.PreviewLabelValidation(c.Request.Context(), limit)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// ListCategories returns all distinct category values.
func (h *AlertRuleHandler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, categories)
}

// GetHeartbeatToken returns the full (unmasked) heartbeat token for a rule.
// This endpoint is adminOnly to prevent token leakage.
func (h *AlertRuleHandler) GetHeartbeatToken(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	token, err := h.svc.GetHeartbeatToken(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"heartbeat_token": token})
}

// Export exports alert rules as a Prometheus-compatible YAML or JSON file.
func (h *AlertRuleHandler) Export(c *gin.Context) {
	groupName := c.Query("group_name")
	category := c.Query("category")
	format := c.Query("format")

	// Fetch all rules (using a large page size to get all)
	list, _, err := h.svc.List(c.Request.Context(), "", "", groupName, category, 1, 10000)
	if err != nil {
		Error(c, err)
		return
	}

	// Group rules by GroupName
	groupMap := make(map[string][]model.PrometheusRule)
	for _, rule := range list {
		gn := rule.GroupName
		if gn == "" {
			gn = "default"
		}
		pr := model.PrometheusRule{
			Alert:       rule.Name,
			Expr:        rule.Expression,
			For:         rule.ForDuration,
			Labels:      map[string]string(rule.Labels),
			Annotations: map[string]string(rule.Annotations),
		}
		groupMap[gn] = append(groupMap[gn], pr)
	}

	// Build Prometheus rule file
	var ruleFile model.PrometheusRuleFile
	for name, rules := range groupMap {
		ruleFile.Groups = append(ruleFile.Groups, model.PrometheusRuleGroup{
			Name:  name,
			Rules: rules,
		})
	}

	if format == "json" {
		data, err := json.Marshal(&ruleFile)
		if err != nil {
			Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to marshal rules: "+err.Error()))
			return
		}
		c.Header("Content-Disposition", "attachment; filename=alert_rules.json")
		c.Data(http.StatusOK, "application/json", data)
		return
	}

	data, err := yaml.Marshal(&ruleFile)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, "failed to marshal rules: "+err.Error()))
		return
	}

	c.Header("Content-Disposition", "attachment; filename=alert_rules.yaml")
	c.Data(http.StatusOK, "application/x-yaml", data)
}

// ToggleStatus enables or disables an alert rule.
func (h *AlertRuleHandler) ToggleStatus(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	var req struct {
		Status model.AlertRuleStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionToggle, ResourceType: model.AuditResourceAlertRule,
			ResourceID: &id, Detail: string(req.Status), IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// batchIDsReq is shared by all batch endpoints.
type batchIDsReq struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchEnable enables multiple alert rules.
func (h *AlertRuleHandler) BatchEnable(c *gin.Context) {
	var req batchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchEnable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionToggle, ResourceType: model.AuditResourceAlertRule,
			Detail: "batch enable", IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// BatchDisable disables multiple alert rules.
func (h *AlertRuleHandler) BatchDisable(c *gin.Context) {
	var req batchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDisable(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionToggle, ResourceType: model.AuditResourceAlertRule,
			Detail: "batch disable", IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}

// BatchDelete deletes multiple alert rules.
func (h *AlertRuleHandler) BatchDelete(c *gin.Context) {
	var req batchIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		Error(c, err)
		return
	}
	if h.auditSvc != nil {
		uid := GetCurrentUserID(c)
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionDelete, ResourceType: model.AuditResourceAlertRule,
			Detail: "batch delete", IP: c.ClientIP(),
		})
	}
	Success(c, nil)
}
