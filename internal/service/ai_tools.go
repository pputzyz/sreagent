package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// marshalJSONOrError marshals v to JSON string, returning an error message string on failure.
func marshalJSONOrError(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"json marshal failed: %s"}`, err.Error())
	}
	return string(data)
}

// AI tool guardrail limits to prevent LLM-generated queries from overwhelming datasources.
const (
	maxQueryRange   = 24 * time.Hour // max time range for AI-generated PromQL queries
	minQueryStep    = 15             // minimum step in seconds
	maxResultSeries = 100            // max series to return to the LLM
)

// validLabelKeyRe validates Prometheus label keys to prevent path injection.
var validLabelKeyRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateLabelKey(key string) error {
	if !validLabelKeyRe.MatchString(key) {
		return fmt.Errorf("invalid label key %q: must match [a-zA-Z_][a-zA-Z0-9_]*", key)
	}
	return nil
}

// AITool 定义一个可被 AI 调用的工具
type AITool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
	IO          string                 `json:"io"`         // "read" | "write" | "none" — I/O 行为标注
	RiskLevel   int8                   `json:"risk_level"` // 0=read-only safe, 1=write, 2=destructive
	Execute     func(ctx context.Context, params map[string]interface{}) (string, error)
}

// AIToolRegistry 管理所有注册的工具
type AIToolRegistry struct {
	tools  map[string]*AITool
	logger *zap.Logger
}

// NewAIToolRegistry 创建空的工具注册表
func NewAIToolRegistry(logger *zap.Logger) *AIToolRegistry {
	return &AIToolRegistry{
		tools:  make(map[string]*AITool),
		logger: logger,
	}
}

// Register 注册一个工具
func (r *AIToolRegistry) Register(tool *AITool) {
	r.tools[tool.Name] = tool
}

// Get 按名称获取工具
func (r *AIToolRegistry) Get(name string) (*AITool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List 返回所有已注册工具的列表
func (r *AIToolRegistry) List() []*AITool {
	result := make([]*AITool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// ListFiltered 返回指定名称白名单中的工具。allowList 为空时返回全部。
func (r *AIToolRegistry) ListFiltered(allowList []string) []*AITool {
	if len(allowList) == 0 {
		return r.List()
	}
	set := make(map[string]struct{}, len(allowList))
	for _, n := range allowList {
		set[n] = struct{}{}
	}
	result := make([]*AITool, 0, len(allowList))
	for _, t := range r.tools {
		if _, ok := set[t.Name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// ToOpenAIToolsFiltered 将白名单中的工具转为 OpenAI function calling 格式。allowList 为空时返回全部。
func (r *AIToolRegistry) ToOpenAIToolsFiltered(allowList []string) []map[string]interface{} {
	tools := r.ListFiltered(allowList)
	result := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		tool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
		result = append(result, tool)
	}
	return result
}

// ToOpenAITools 将注册表中的工具转为 OpenAI function calling 格式
func (r *AIToolRegistry) ToOpenAITools() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		tool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
		result = append(result, tool)
	}
	return result
}

// RegisterBuiltinTools 注册所有内置工具（需要外部服务依赖）
// getEngineStatus 返回引擎状态的 JSON 序列化结果，为 nil 表示引擎未启用
func (r *AIToolRegistry) RegisterBuiltinTools(
	dsSvc DataSourceQuerier,
	ruleSvc AlertRuleOperator,
	incidentSvc *IncidentService,
	auditLogSvc *AuditLogService,
	eventSvc *AlertEventService,
	kbSvc *KnowledgeBaseService,
	getEngineStatus func() (interface{}, bool),
) {
	// ── query_datasource: 执行 PromQL 查询 ──
	r.Register(&AITool{
		Name:        "query_datasource",
		Description: "对指定数据源执行 PromQL 范围查询，返回时序数据。用于分析指标趋势、排查性能问题。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "PromQL 查询表达式",
				},
				"time_range": map[string]interface{}{
					"type":        "string",
					"description": "查询时间范围，如 '1h'、'30m'、'6h'，默认 1h",
				},
			},
			"required": []string{"datasource_id", "query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])
			query, _ := params["query"].(string)
			timeRange, _ := params["time_range"].(string)
			if timeRange == "" {
				timeRange = "1h"
			}

			duration, err := time.ParseDuration(timeRange)
			if err != nil {
				return "", fmt.Errorf("无效的时间范围格式 %q: %w", timeRange, err)
			}

			if duration > maxQueryRange {
				return "", fmt.Errorf("查询时间范围 %v 超过 AI 工具允许的最大值 %v", timeRange, maxQueryRange)
			}

			end := time.Now()
			start := end.Add(-duration)

			var step string
			if duration <= 30*time.Minute {
				step = "15s"
			} else if duration <= 2*time.Hour {
				step = "60s"
			} else {
				step = "300s"
			}

			resp, err := dsSvc.QueryRange(ctx, dsID, query, start, end, step)
			if err != nil {
				return fmt.Sprintf("查询失败: %v", err), nil
			}

			// Guardrail: truncate excessive series to protect LLM context
			totalSeries := len(resp.Series)
			if totalSeries > maxResultSeries {
				resp.Series = resp.Series[:maxResultSeries]
			}

			summary := fmt.Sprintf("查询结果: %d 条时间序列", totalSeries)
			if totalSeries > maxResultSeries {
				summary += fmt.Sprintf("（已截断至前 %d 条）", maxResultSeries)
			}
			totalPoints := 0
			for _, s := range resp.Series {
				totalPoints += len(s.Values)
			}
			summary += fmt.Sprintf("，共 %d 个数据点", totalPoints)
			if len(resp.Series) > 0 && len(resp.Series[0].Values) > 0 {
				last := resp.Series[0].Values[len(resp.Series[0].Values)-1]
				summary += fmt.Sprintf("\n最新值: %.4f (时间戳: %d)", last.Value, last.Timestamp)
			}
			return summary, nil
		},
	})

	// ── list_alert_rules: 查询告警规则 ──
	r.Register(&AITool{
		Name:        "list_alert_rules",
		Description: "查询告警规则列表，支持按严重等级、状态筛选。用于了解当前告警配置。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"severity": map[string]interface{}{
					"type":        "string",
					"description": "严重等级筛选: critical, warning, info, debug",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "状态筛选: active, disabled, draft",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码，默认 1",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			severity, _ := params["severity"].(string)
			status, _ := params["status"].(string)
			page, _ := aiToolToInt(params["page"])
			if page <= 0 {
				page = 1
			}

			rules, total, err := ruleSvc.List(ctx, severity, status, "", "", "", nil, page, 20)
			if err != nil {
				return fmt.Sprintf("查询失败: %v", err), nil
			}

			type ruleSummary struct {
				ID          uint   `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"display_name"`
				Severity    string `json:"severity"`
				Status      string `json:"status"`
				Expression  string `json:"expression"`
			}
			summaries := make([]ruleSummary, 0, len(rules))
			for _, rule := range rules {
				summaries = append(summaries, ruleSummary{
					ID:          rule.ID,
					Name:        rule.Name,
					DisplayName: rule.DisplayName,
					Severity:    string(rule.Severity),
					Status:      string(rule.Status),
					Expression:  rule.Expression,
				})
			}

			return marshalJSONOrError(map[string]interface{}{
				"total": total,
				"page":  page,
				"rules": summaries,
			}), nil
		},
	})

	// ── get_incident_detail: 获取故障详情 ──
	r.Register(&AITool{
		Name:        "get_incident_detail",
		Description: "获取指定故障（incident）的详细信息，包括状态、严重等级、负责人、关联告警等。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"incident_id": map[string]interface{}{
					"type":        "integer",
					"description": "故障 ID",
				},
			},
			"required": []string{"incident_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			incID, _ := aiToolToUint(params["incident_id"])
			inc, err := incidentSvc.GetByID(ctx, incID)
			if err != nil {
				return fmt.Sprintf("获取故障详情失败: %v", err), nil
			}

			return marshalJSONOrError(inc), nil
		},
	})

	// ── get_engine_status: 获取引擎状态 ──
	r.Register(&AITool{
		Name:        "get_engine_status",
		Description: "获取告警引擎的运行状态，包括是否运行中、规则总数、活跃告警数、运行时长等。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			status, ok := getEngineStatus()
			if !ok {
				return "告警引擎未启用", nil
			}
			return marshalJSONOrError(status), nil
		},
	})

	// ── search_audit_logs: 搜索审计日志 ──
	r.Register(&AITool{
		Name:        "search_audit_logs",
		Description: "搜索审计日志，支持按操作类型、资源类型、时间范围筛选。用于追踪操作历史和排查问题。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "操作类型筛选，如 create, update, delete, login",
				},
				"resource_type": map[string]interface{}{
					"type":        "string",
					"description": "资源类型筛选，如 alert_rule, user, datasource",
				},
				"days": map[string]interface{}{
					"type":        "integer",
					"description": "查询最近 N 天的日志，默认 7 天",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码，默认 1",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			action, _ := params["action"].(string)
			resourceType, _ := params["resource_type"].(string)
			days, _ := aiToolToInt(params["days"])
			if days <= 0 {
				days = 7
			}
			page, _ := aiToolToInt(params["page"])
			if page <= 0 {
				page = 1
			}

			now := time.Now()
			startTime := now.AddDate(0, 0, -days)

			filter := repository.AuditLogFilter{
				Action:       action,
				ResourceType: resourceType,
				StartTime:    &startTime,
				EndTime:      &now,
			}

			logs, total, err := auditLogSvc.List(ctx, filter, page, 20)
			if err != nil {
				return fmt.Sprintf("查询审计日志失败: %v", err), nil
			}

			type logSummary struct {
				ID           uint   `json:"id"`
				Action       string `json:"action"`
				ResourceType string `json:"resource_type"`
				ResourceName string `json:"resource_name"`
				Detail       string `json:"detail"`
				Status       string `json:"status"`
				CreatedAt    string `json:"created_at"`
			}
			summaries := make([]logSummary, 0, len(logs))
			for _, l := range logs {
				s := logSummary{
					ID:           l.ID,
					Action:       l.Action,
					ResourceType: l.ResourceType,
					ResourceName: l.ResourceName,
					Detail:       l.Detail,
					Status:       l.Status,
				}
				if !l.CreatedAt.IsZero() {
					s.CreatedAt = l.CreatedAt.Format(time.RFC3339)
				}
				summaries = append(summaries, s)
			}

			return marshalJSONOrError(map[string]interface{}{
				"total": total,
				"page":  page,
				"days":  days,
				"logs":  summaries,
			}), nil
		},
	})

	// ── list_metrics: 列出数据源的指标名 ──
	r.Register(&AITool{
		Name:        "list_metrics",
		Description: "列出某数据源的所有指标名（metric names）。可按前缀过滤。用于探索数据源有哪些可用指标。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
				"prefix": map[string]interface{}{
					"type":        "string",
					"description": "指标名前缀过滤，如 'mysql_'、'node_'、'redis_'",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量上限，默认 100",
				},
			},
			"required": []string{"datasource_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])
			prefix, _ := params["prefix"].(string)
			limit, _ := aiToolToInt(params["limit"])
			if limit <= 0 {
				limit = 100
			}

			// 通过 Prometheus /api/v1/label/__name__/values 获取指标名
			raw, err := dsSvc.ProxyToDatasource(ctx, dsID, "/api/v1/label/__name__/values", nil)
			if err != nil {
				return fmt.Sprintf("获取指标名失败: %v", err), nil
			}

			var resp struct {
				Status string   `json:"status"`
				Data   []string `json:"data"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Sprintf("解析响应失败: %v", err), nil
			}

			metrics := resp.Data
			if prefix != "" {
				filtered := make([]string, 0)
				for _, m := range metrics {
					if strings.HasPrefix(m, prefix) {
						filtered = append(filtered, m)
					}
				}
				metrics = filtered
			}

			if len(metrics) > limit {
				metrics = metrics[:limit]
			}

			return marshalJSONOrError(map[string]interface{}{
				"total":    len(resp.Data),
				"filtered": len(metrics),
				"prefix":   prefix,
				"metrics":  metrics,
			}), nil
		},
	})

	// ── list_label_keys: 列出数据源的 label keys ──
	r.Register(&AITool{
		Name:        "list_label_keys",
		Description: "列出某数据源的所有 label key（标签名）。用于了解数据源的标签维度。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
			},
			"required": []string{"datasource_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])

			raw, err := dsSvc.ProxyToDatasource(ctx, dsID, "/api/v1/labels", nil)
			if err != nil {
				return fmt.Sprintf("获取 label keys 失败: %v", err), nil
			}

			var resp struct {
				Status string   `json:"status"`
				Data   []string `json:"data"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Sprintf("解析响应失败: %v", err), nil
			}

			return marshalJSONOrError(map[string]interface{}{
				"total": len(resp.Data),
				"keys":  resp.Data,
			}), nil
		},
	})

	// ── list_label_values: 列出某 label key 的所有值 ──
	r.Register(&AITool{
		Name:        "list_label_values",
		Description: "列出某数据源中指定 label key 的所有值。用于探索标签值分布，如列出所有 job 名、instance 等。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
				"label_key": map[string]interface{}{
					"type":        "string",
					"description": "要查询的 label key，如 'job'、'instance'、'env'",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量上限，默认 100",
				},
			},
			"required": []string{"datasource_id", "label_key"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])
			labelKey, _ := params["label_key"].(string)
			if err := validateLabelKey(labelKey); err != nil {
				return "", err
			}
			limit, _ := aiToolToInt(params["limit"])
			if limit <= 0 {
				limit = 100
			}

			path := fmt.Sprintf("/api/v1/label/%s/values", labelKey)
			raw, err := dsSvc.ProxyToDatasource(ctx, dsID, path, nil)
			if err != nil {
				return fmt.Sprintf("获取 label values 失败: %v", err), nil
			}

			var resp struct {
				Status string   `json:"status"`
				Data   []string `json:"data"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Sprintf("解析响应失败: %v", err), nil
			}

			values := resp.Data
			if len(values) > limit {
				values = values[:limit]
			}

			return marshalJSONOrError(map[string]interface{}{
				"total":     len(resp.Data),
				"returned":  len(values),
				"label_key": labelKey,
				"values":    values,
			}), nil
		},
	})

	// ── query_instant: 即时查询 PromQL ──
	r.Register(&AITool{
		Name:        "query_instant",
		Description: "对指定数据源执行 PromQL 即时查询（当前时刻的快照值）。适用于查看当前状态、单值指标。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "PromQL 查询表达式",
				},
			},
			"required": []string{"datasource_id", "query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])
			query, _ := params["query"].(string)

			resp, err := dsSvc.QueryDatasource(ctx, dsID, query, time.Now())
			if err != nil {
				return fmt.Sprintf("即时查询失败: %v", err), nil
			}

			// Guardrail: truncate excessive series to protect LLM context
			totalSeries := len(resp.Series)
			if totalSeries > maxResultSeries {
				resp.Series = resp.Series[:maxResultSeries]
			}

			summary := fmt.Sprintf("查询结果: %d 条时间序列", totalSeries)
			if totalSeries > maxResultSeries {
				summary += fmt.Sprintf("（已截断至前 %d 条）", maxResultSeries)
			}
			for i, s := range resp.Series {
				if i >= 5 {
					summary += fmt.Sprintf("\n... 还有 %d 条", len(resp.Series)-5)
					break
				}
				labels := ""
				for k, v := range s.Labels {
					if labels != "" {
						labels += ", "
					}
					labels += fmt.Sprintf("%s=%s", k, v)
				}
				if len(s.Values) > 0 {
					summary += fmt.Sprintf("\n  {%s} = %.4f", labels, s.Values[len(s.Values)-1].Value)
				}
			}
			return summary, nil
		},
	})

	// ── get_metric_metadata: 获取指标元数据 ──
	r.Register(&AITool{
		Name:        "get_metric_metadata",
		Description: "获取某指标的元数据（help 文本、类型如 counter/gauge/histogram）。用于理解指标含义。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"datasource_id": map[string]interface{}{
					"type":        "integer",
					"description": "数据源 ID",
				},
				"metric_name": map[string]interface{}{
					"type":        "string",
					"description": "指标名，如 'http_requests_total'",
				},
			},
			"required": []string{"datasource_id", "metric_name"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			dsID, _ := aiToolToUint(params["datasource_id"])
			metricName, _ := params["metric_name"].(string)

			queryParams := map[string]string{"metric": metricName}
			raw, err := dsSvc.ProxyToDatasource(ctx, dsID, "/api/v1/metadata", queryParams)
			if err != nil {
				return fmt.Sprintf("获取元数据失败: %v", err), nil
			}

			var resp struct {
				Status string                              `json:"status"`
				Data   map[string][]map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Sprintf("解析响应失败: %v", err), nil
			}

			entries, ok := resp.Data[metricName]
			if !ok || len(entries) == 0 {
				return fmt.Sprintf("指标 %q 无元数据（可能是 untyped 或数据源不支持 metadata API）", metricName), nil
			}

			entry := entries[0]
			return marshalJSONOrError(map[string]interface{}{
				"metric": metricName,
				"type":   entry["type"],
				"help":   entry["help"],
				"unit":   entry["unit"],
			}), nil
		},
	})

	// ── search_similar_alerts: 搜索相似告警历史 ──
	r.Register(&AITool{
		Name:        "search_similar_alerts",
		Description: "搜索最近的告警事件。可按名称、严重等级、状态过滤。用于查找历史告警记录，辅助根因分析。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"alert_name": map[string]interface{}{
					"type":        "string",
					"description": "告警名称（模糊匹配）",
				},
				"severity": map[string]interface{}{
					"type":        "string",
					"description": "严重等级: critical / warning / info",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "状态: firing / acknowledged / resolved / closed",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量上限，默认 20",
				},
			},
			"required": []string{},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			alertName, _ := params["alert_name"].(string)
			severity, _ := params["severity"].(string)
			status, _ := params["status"].(string)
			limit, _ := aiToolToInt(params["limit"])
			if limit <= 0 {
				limit = 20
			}

			events, _, err := eventSvc.List(ctx, status, severity, 1, limit)
			if err != nil {
				return fmt.Sprintf("搜索告警历史失败: %v", err), nil
			}

			// 按 alert_name 过滤（List 不支持名称模糊匹配）
			if alertName != "" {
				filtered := make([]model.AlertEvent, 0)
				for _, e := range events {
					if strings.Contains(strings.ToLower(e.AlertName), strings.ToLower(alertName)) {
						filtered = append(filtered, e)
					}
				}
				events = filtered
			}

			type eventSummary struct {
				ID        uint   `json:"id"`
				AlertName string `json:"alert_name"`
				Severity  string `json:"severity"`
				Status    string `json:"status"`
				FiredAt   string `json:"fired_at"`
			}
			summaries := make([]eventSummary, 0, len(events))
			for _, e := range events {
				summaries = append(summaries, eventSummary{
					ID:        e.ID,
					AlertName: e.AlertName,
					Severity:  string(e.Severity),
					Status:    string(e.Status),
					FiredAt:   e.FiredAt.Format(time.RFC3339),
				})
			}

			return marshalJSONOrError(map[string]interface{}{
				"total":  len(summaries),
				"events": summaries,
			}), nil
		},
	})

	// ── search_knowledge: 搜索知识库 ──
	r.Register(&AITool{
		Name:        "search_knowledge",
		Description: "搜索知识库文档（SOP、事故案例、Runbook 等）。支持全文检索，可按来源过滤。用于查找排障手册、历史事故处理方案。",
		IO:          "read",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "来源过滤: sop / incident_case / runbook / template_example / wiki",
				},
				"top_k": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量上限，默认 10",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			query, _ := params["query"].(string)
			source, _ := params["source"].(string)
			topK, _ := aiToolToInt(params["top_k"])
			if topK <= 0 {
				topK = 10
			}

			docs, _, err := kbSvc.Search(ctx, query, source, 1, topK)
			if err != nil {
				return fmt.Sprintf("搜索知识库失败: %v", err), nil
			}

			type docSummary struct {
				ID           uint   `json:"id"`
				Source       string `json:"source"`
				Title        string `json:"title"`
				Summary      string `json:"summary"`
				HelpfulCount int    `json:"helpful_count"`
			}
			summaries := make([]docSummary, 0, len(docs))
			for _, d := range docs {
				summaries = append(summaries, docSummary{
					ID:           d.ID,
					Source:       string(d.Source),
					Title:        d.Title,
					Summary:      d.Summary,
					HelpfulCount: d.HelpfulCount,
				})
			}

			return marshalJSONOrError(map[string]interface{}{
				"total": len(summaries),
				"docs":  summaries,
			}), nil
		},
	})

	r.logger.Info("AI 工具注册表初始化完成",
		zap.Int("tool_count", len(r.tools)),
		zap.String("tools", strings.Join(aiToolNames(r), ", ")),
	)
}

// aiToolNames 返回所有工具名称的列表（用于日志）
func aiToolNames(r *AIToolRegistry) []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// RegisterMCPTools discovers and registers tools from all enabled MCP servers.
// Each MCP tool is registered with the prefix "mcp_{serverName}_" to avoid name collisions.
// Connection failures to individual servers are logged but do not block other servers.
func (r *AIToolRegistry) RegisterMCPTools(mcpSvc *MCPServerService) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	servers, err := mcpSvc.ListEnabled(ctx)
	if err != nil {
		r.logger.Error("failed to list enabled MCP servers for tool registration", zap.Error(err))
		return
	}

	registered := 0
	for _, srv := range servers {
		tools, err := mcpSvc.ListTools(ctx, &srv)
		if err != nil {
			r.logger.Warn("failed to discover MCP tools, skipping server",
				zap.String("server", srv.Name),
				zap.Uint("server_id", srv.ID),
				zap.Error(err),
			)
			continue
		}

		for _, tool := range tools {
			toolName := fmt.Sprintf("mcp_%s_%s", srv.Name, tool.Name)
			// Capture for closure
			srvURL := srv.URL
			srvHeaders := srv.GetHeadersMap()
			mcpToolName := tool.Name
			toolDesc := tool.Description
			toolSchema := tool.InputSchema

			// B8-4: Heuristic: infer IO/RiskLevel from description keywords when MCP
			// protocol does not provide annotation fields.
			// NOTE: This keyword-based classification is fragile — a tool named "get_user_settings"
			// that modifies data would be misclassified as read-only. MCP protocol extensions
			// (e.g. tool annotations with readOnlyHint/writeOnlyHint) should be preferred when
			// available. Until then, the conservative "write" default and explicit read-hint
			// matching provide a reasonable safety net but may produce false negatives for
			// write operations described with read-like verbs.
			ioType := "write"    // conservative default
			riskLevel := int8(1) // moderate risk default
			descLower := strings.ToLower(toolDesc)
			readHints := []string{"read", "list", "get", "query", "search", "fetch", "find", "show", "describe"}
			for _, hint := range readHints {
				if strings.Contains(descLower, hint) {
					ioType = "read"
					riskLevel = int8(0)
					break
				}
			}

			r.Register(&AITool{
				Name:        toolName,
				Description: fmt.Sprintf("[MCP:%s] %s", srv.Name, toolDesc),
				Parameters:  toolSchema,
				IO:          ioType,
				RiskLevel:   riskLevel,
				Execute: func(execCtx context.Context, params map[string]interface{}) (string, error) {
					client := NewMCPClient()
					result, err := client.CallTool(execCtx, srvURL, srvHeaders, mcpToolName, params)
					if err != nil {
						return "", fmt.Errorf("MCP tool %q call failed: %w", mcpToolName, err)
					}
					// Extract text from result content
					var texts []string
					for _, c := range result.Content {
						if c.Text != "" {
							texts = append(texts, c.Text)
						}
					}
					if len(texts) == 0 {
						return marshalJSONOrError(result), nil
					}
					return strings.Join(texts, "\n"), nil
				},
			})
			registered++
		}
	}

	if registered > 0 {
		r.logger.Info("MCP tools registered",
			zap.Int("count", registered),
			zap.Int("servers", len(servers)),
		)
	}
}

// aiToolToUint 将 interface{} 转为 uint，支持 float64（JSON 解析默认类型）和 int
func aiToolToUint(v interface{}) (uint, bool) {
	switch val := v.(type) {
	case float64:
		return uint(val), true
	case int:
		return uint(val), true
	case int64:
		return uint(val), true
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return uint(n), true
	default:
		return 0, false
	}
}

// aiToolToInt 将 interface{} 转为 int，支持 float64（JSON 解析默认类型）
func aiToolToInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}
