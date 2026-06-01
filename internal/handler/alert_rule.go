package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/upload"
	"github.com/sreagent/sreagent/internal/service"
)

type AlertRuleHandler struct {
	svc      *service.AlertRuleService
	auditSvc *service.AuditLogService
	log      *zap.Logger
}

func NewAlertRuleHandler(svc *service.AlertRuleService, logger ...*zap.Logger) *AlertRuleHandler {
	l := zap.NewNop()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &AlertRuleHandler{svc: svc, log: l}
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
	// Status allows the caller to set the initial status (draft/active/disabled).
	// Defaults to "active" if empty.
	Status model.AlertRuleStatus `json:"status"`
	// Source indicates the origin of this rule (e.g. "ai", "import", "manual").
	Source string `json:"source"`
	// Evaluation
	EvalInterval     int    `json:"eval_interval"`
	RuleType         string `json:"rule_type"`
	RecoveryHold     string `json:"recovery_hold"`
	NoDataEnabled    bool   `json:"nodata_enabled"`
	NoDataDuration   string `json:"nodata_duration"`
	SuppressEnabled  bool   `json:"suppress_enabled"`
	// Ownership
	BizGroupID *uint `json:"biz_group_id"`
	TeamID     *uint `json:"team_id"`
	// Heartbeat
	HeartbeatToken    string `json:"heartbeat_token"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	// SLA
	AckSlaMinutes int `json:"ack_sla_minutes"`
	// Multi-query
	Queries    []model.RuleQuery `json:"queries"`
	TriggerExp string            `json:"trigger_exp"`
	JoinType   string            `json:"join_type"`
	JoinKeys   []string          `json:"join_keys"`
	// Variable filling
	VarConfig *model.VarConfig `json:"var_config"`
	// Channel
	ChannelID *uint `json:"channel_id"`
}

// UpdateAlertRuleRequest uses pointer types so nil = "not sent in JSON".
// The Update handler only overwrites fields that are explicitly provided.
type UpdateAlertRuleRequest struct {
	Name                 *string               `json:"name"`
	DisplayName          *string               `json:"display_name"`
	Description          *string               `json:"description"`
	DataSourceID         *uint                 `json:"datasource_id"`
	DatasourceType       *model.DataSourceType `json:"datasource_type"`
	Expression           *string               `json:"expression"`
	ForDuration          *string               `json:"for_duration"`
	Severity             *model.AlertSeverity  `json:"severity"`
	Labels               *model.JSONLabels     `json:"labels"`
	Annotations          *model.JSONLabels     `json:"annotations"`
	GroupName            *string               `json:"group_name"`
	Category             *string               `json:"category"`
	GroupWaitSeconds     *int                  `json:"group_wait_seconds"`
	GroupIntervalSeconds *int                  `json:"group_interval_seconds"`
	Status               *model.AlertRuleStatus `json:"status"`
	// Evaluation
	EvalInterval     *int    `json:"eval_interval"`
	RuleType         *string `json:"rule_type"`
	RecoveryHold     *string `json:"recovery_hold"`
	NoDataEnabled    *bool   `json:"nodata_enabled"`
	NoDataDuration   *string `json:"nodata_duration"`
	SuppressEnabled  *bool   `json:"suppress_enabled"`
	// Ownership
	BizGroupID *uint `json:"biz_group_id"`
	TeamID     *uint `json:"team_id"`
	// Heartbeat
	HeartbeatToken    *string `json:"heartbeat_token"`
	HeartbeatInterval *int    `json:"heartbeat_interval"`
	// SLA
	AckSlaMinutes *int `json:"ack_sla_minutes"`
	// Multi-query
	Queries    *[]model.RuleQuery `json:"queries"`
	TriggerExp *string            `json:"trigger_exp"`
	JoinType   *string            `json:"join_type"`
	JoinKeys   *[]string          `json:"join_keys"`
	// Variable filling
	VarConfig *model.VarConfig `json:"var_config"`
	// Channel
	ChannelID *uint `json:"channel_id"`
}

func (h *AlertRuleHandler) Create(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// --- Parameter validation ---
	if !req.Severity.IsValid() {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "severity must be one of: critical, warning, info, p0, p1, p2, p3, p4"))
		return
	}
	if req.EvalInterval != 0 && (req.EvalInterval < 0 || req.EvalInterval > 86400) {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "eval_interval must be between 1 and 86400 seconds"))
		return
	}
	if req.ForDuration != "" {
		if _, err := time.ParseDuration(req.ForDuration); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "for_duration must be a valid Go duration (e.g. 5m, 1h, 30s): "+err.Error()))
			return
		}
	}
	if req.RecoveryHold != "" {
		if _, err := time.ParseDuration(req.RecoveryHold); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "recovery_hold must be a valid Go duration (e.g. 5m, 1h): "+err.Error()))
			return
		}
	}
	if req.NoDataDuration != "" {
		if _, err := time.ParseDuration(req.NoDataDuration); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "nodata_duration must be a valid Go duration (e.g. 5m, 1h): "+err.Error()))
			return
		}
	}

	// Default to active if caller did not specify a status.
	status := req.Status
	if status == "" {
		status = model.RuleStatusActive
	}

	userID := GetCurrentUserID(c)
	h.log.Info("alert rule create",
		zap.Uint("user_id", userID),
		zap.String("name", req.Name),
		zap.String("request_id", c.GetString("request_id")))

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
		Status:               status,
		CreatedBy:            userID,
		// Evaluation
		EvalInterval:     req.EvalInterval,
		RuleType:         model.AlertRuleType(req.RuleType),
		RecoveryHold:     req.RecoveryHold,
		NoDataEnabled:    req.NoDataEnabled,
		NoDataDuration:   req.NoDataDuration,
		SuppressEnabled:  req.SuppressEnabled,
		// Ownership
		BizGroupID: req.BizGroupID,
		TeamID:     req.TeamID,
		// Heartbeat
		HeartbeatToken:    req.HeartbeatToken,
		HeartbeatInterval: req.HeartbeatInterval,
		// SLA
		AckSlaMinutes: req.AckSlaMinutes,
		// Multi-query
		Queries:    req.Queries,
		TriggerExp: req.TriggerExp,
		JoinType:   req.JoinType,
		JoinKeys:   req.JoinKeys,
		// Variable filling
		VarConfig: req.VarConfig,
		// Channel
		ChannelID: req.ChannelID,
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
	keyword := c.Query("keyword")

	var datasourceID *uint
	if dsStr := c.Query("datasource_id"); dsStr != "" {
		if id, err := strconv.ParseUint(dsStr, 10, 64); err == nil {
			uid := uint(id)
			datasourceID = &uid
		}
	}

	// Team-scoped listing: admin sees all, non-admin sees only own team's rules.
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == "admin"
	teamIDs := middleware.GetUserTeamIDs(c)

	list, total, err := h.svc.ListScoped(c.Request.Context(), isAdmin, teamIDs, severity, status, groupName, category, keyword, datasourceID, pq.Page, pq.PageSize)
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

	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	// --- Parameter validation (PATCH: only validate fields that are explicitly provided) ---
	if req.Severity != nil && !req.Severity.IsValid() {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "severity must be one of: critical, warning, info, p0, p1, p2, p3, p4"))
		return
	}
	if req.EvalInterval != nil && *req.EvalInterval < 0 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "eval_interval must not be negative"))
		return
	}
	if req.EvalInterval != nil && *req.EvalInterval > 86400 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "eval_interval must not exceed 86400 seconds (1 day)"))
		return
	}
	if req.ForDuration != nil && *req.ForDuration != "" {
		if _, err := time.ParseDuration(*req.ForDuration); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "for_duration must be a valid Go duration (e.g. 5m, 1h, 30s): "+err.Error()))
			return
		}
	}
	if req.RecoveryHold != nil && *req.RecoveryHold != "" {
		if _, err := time.ParseDuration(*req.RecoveryHold); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "recovery_hold must be a valid Go duration (e.g. 5m, 1h): "+err.Error()))
			return
		}
	}
	if req.NoDataDuration != nil && *req.NoDataDuration != "" {
		if _, err := time.ParseDuration(*req.NoDataDuration); err != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "nodata_duration must be a valid Go duration (e.g. 5m, 1h): "+err.Error()))
			return
		}
	}

	// Fetch existing rule so we only overwrite fields explicitly provided in JSON.
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}

	// Merge non-nil request fields into existing rule (PATCH semantics).
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.DisplayName != nil {
		existing.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.DataSourceID != nil {
		existing.DataSourceID = req.DataSourceID
	}
	if req.DatasourceType != nil {
		existing.DatasourceType = *req.DatasourceType
	}
	if req.Expression != nil {
		existing.Expression = *req.Expression
	}
	if req.ForDuration != nil {
		existing.ForDuration = *req.ForDuration
	}
	if req.Severity != nil {
		existing.Severity = *req.Severity
	}
	if req.Labels != nil {
		existing.Labels = *req.Labels
	}
	if req.Annotations != nil {
		existing.Annotations = *req.Annotations
	}
	if req.GroupName != nil {
		existing.GroupName = *req.GroupName
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.GroupWaitSeconds != nil {
		existing.GroupWaitSeconds = *req.GroupWaitSeconds
	}
	if req.GroupIntervalSeconds != nil {
		existing.GroupIntervalSeconds = *req.GroupIntervalSeconds
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	// Evaluation
	if req.EvalInterval != nil {
		existing.EvalInterval = *req.EvalInterval
	}
	if req.RuleType != nil {
		existing.RuleType = model.AlertRuleType(*req.RuleType)
	}
	if req.RecoveryHold != nil {
		existing.RecoveryHold = *req.RecoveryHold
	}
	if req.NoDataEnabled != nil {
		existing.NoDataEnabled = *req.NoDataEnabled
	}
	if req.NoDataDuration != nil {
		existing.NoDataDuration = *req.NoDataDuration
	}
	if req.SuppressEnabled != nil {
		existing.SuppressEnabled = *req.SuppressEnabled
	}
	// Ownership
	if req.BizGroupID != nil {
		existing.BizGroupID = req.BizGroupID
	}
	if req.TeamID != nil {
		existing.TeamID = req.TeamID
	}
	// Heartbeat
	if req.HeartbeatToken != nil {
		existing.HeartbeatToken = *req.HeartbeatToken
	}
	if req.HeartbeatInterval != nil {
		existing.HeartbeatInterval = *req.HeartbeatInterval
	}
	// SLA
	if req.AckSlaMinutes != nil {
		existing.AckSlaMinutes = *req.AckSlaMinutes
	}
	// Multi-query
	if req.Queries != nil {
		existing.Queries = *req.Queries
	}
	if req.TriggerExp != nil {
		existing.TriggerExp = *req.TriggerExp
	}
	if req.JoinType != nil {
		existing.JoinType = *req.JoinType
	}
	if req.JoinKeys != nil {
		existing.JoinKeys = *req.JoinKeys
	}
	// Variable filling
	if req.VarConfig != nil {
		existing.VarConfig = req.VarConfig
	}
	// Channel
	if req.ChannelID != nil {
		existing.ChannelID = req.ChannelID
	}

	existing.UpdatedBy = userID

	ruleName := existing.Name
	h.log.Info("alert rule update",
		zap.Uint("user_id", userID),
		zap.Uint("rule_id", id),
		zap.String("name", ruleName),
		zap.String("request_id", c.GetString("request_id")))

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		Error(c, err)
		return
	}

	if h.auditSvc != nil {
		uid := existing.UpdatedBy
		h.auditSvc.Record(&model.AuditLog{
			UserID: &uid, Username: GetCurrentUsername(c),
			Action: model.AuditActionUpdate, ResourceType: model.AuditResourceAlertRule,
			ResourceID: &existing.ID, ResourceName: ruleName, IP: c.ClientIP(),
		})
	}
	Success(c, existing)
}

func (h *AlertRuleHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}

	h.log.Info("alert rule delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("rule_id", id),
		zap.String("request_id", c.GetString("request_id")))

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
	defer func() { _ = file.Close() }()

	const maxUploadSize = 10 << 20 // 10 MB
	if header.Size > maxUploadSize {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "file too large (max 10MB)"))
		return
	}

	datasourceIDStr := c.PostForm("datasource_id")
	var datasourceID *uint
	if datasourceIDStr != "" {
		var id uint64
		if _, err := fmt.Sscanf(datasourceIDStr, "%d", &id); err == nil {
			uid := uint(id)
			datasourceID = &uid
		}
	}

	// Validate MIME type and file extension.
	validated, err := upload.ValidateYAMLUpload(header.Filename, file)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "file validation failed: "+err.Error()))
		return
	}

	data, err := io.ReadAll(io.LimitReader(validated, maxUploadSize+1))
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
				Status:       model.RuleStatusActive,
				DataSourceID: datasourceID,
				CreatedBy:    userID,
			}
			rules = append(rules, rule)
		}
	}

	h.log.Info("alert rule import",
		zap.Uint("user_id", userID),
		zap.Int("total_rules", len(rules)),
		zap.String("request_id", c.GetString("request_id")))

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

	// Team-scoped listing: admin sees all, non-admin sees only own team's rules.
	currentRole, _ := c.Get("role")
	isAdmin := currentRole == "admin"
	teamIDs := middleware.GetUserTeamIDs(c)

	// Fetch all rules (using a large page size to get all)
	list, _, err := h.svc.ListScoped(c.Request.Context(), isAdmin, teamIDs, "", "", groupName, category, "", nil, 1, 10000)
	if err != nil {
		Error(c, err)
		return
	}

	// Mask heartbeat tokens before export
	for i := range list {
		list[i] = list[i].MaskHeartbeatToken()
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

	h.log.Info("alert rule toggle status",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Uint("rule_id", id),
		zap.String("status", string(req.Status)),
		zap.String("request_id", c.GetString("request_id")))

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

	h.log.Info("alert rule batch enable",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Int("count", len(req.IDs)),
		zap.String("request_id", c.GetString("request_id")))

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

	h.log.Info("alert rule batch disable",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Int("count", len(req.IDs)),
		zap.String("request_id", c.GetString("request_id")))

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

	h.log.Info("alert rule batch delete",
		zap.Uint("user_id", GetCurrentUserID(c)),
		zap.Int("count", len(req.IDs)),
		zap.String("request_id", c.GetString("request_id")))

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
