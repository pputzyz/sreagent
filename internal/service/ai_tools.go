package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// AITool 定义一个可被 AI 调用的工具
type AITool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
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
	dsSvc *DataSourceService,
	ruleSvc *AlertRuleService,
	incidentSvc *IncidentService,
	auditLogSvc *AuditLogService,
	eventSvc *AlertEventService,
	getEngineStatus func() (interface{}, bool),
) {
	// ── query_datasource: 执行 PromQL 查询 ──
	r.Register(&AITool{
		Name:        "query_datasource",
		Description: "对指定数据源执行 PromQL 范围查询，返回时序数据。用于分析指标趋势、排查性能问题。",
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

			end := time.Now()
			start := end.Add(-duration)

			step := "60s"
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

			summary := fmt.Sprintf("查询结果: %d 条时间序列", len(resp.Series))
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

			rules, total, err := ruleSvc.List(ctx, severity, status, "", "", page, 20)
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

			data, _ := json.Marshal(map[string]interface{}{
				"total": total,
				"page":  page,
				"rules": summaries,
			})
			return string(data), nil
		},
	})

	// ── get_incident_detail: 获取故障详情 ──
	r.Register(&AITool{
		Name:        "get_incident_detail",
		Description: "获取指定故障（incident）的详细信息，包括状态、严重等级、负责人、关联告警等。",
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

			data, _ := json.Marshal(inc)
			return string(data), nil
		},
	})

	// ── get_engine_status: 获取引擎状态 ──
	r.Register(&AITool{
		Name:        "get_engine_status",
		Description: "获取告警引擎的运行状态，包括是否运行中、规则总数、活跃告警数、运行时长等。",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			status, ok := getEngineStatus()
			if !ok {
				return "告警引擎未启用", nil
			}
			data, _ := json.Marshal(status)
			return string(data), nil
		},
	})

	// ── search_audit_logs: 搜索审计日志 ──
	r.Register(&AITool{
		Name:        "search_audit_logs",
		Description: "搜索审计日志，支持按操作类型、资源类型、时间范围筛选。用于追踪操作历史和排查问题。",
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

			data, _ := json.Marshal(map[string]interface{}{
				"total": total,
				"page":  page,
				"days":  days,
				"logs":  summaries,
			})
			return string(data), nil
		},
	})

	// ── list_metrics: 列出数据源的指标名 ──
	r.Register(&AITool{
		Name:        "list_metrics",
		Description: "列出某数据源的所有指标名（metric names）。可按前缀过滤。用于探索数据源有哪些可用指标。",
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

			data, _ := json.Marshal(map[string]interface{}{
				"total":   len(resp.Data),
				"filtered": len(metrics),
				"prefix":  prefix,
				"metrics": metrics,
			})
			return string(data), nil
		},
	})

	// ── list_label_keys: 列出数据源的 label keys ──
	r.Register(&AITool{
		Name:        "list_label_keys",
		Description: "列出某数据源的所有 label key（标签名）。用于了解数据源的标签维度。",
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

			data, _ := json.Marshal(map[string]interface{}{
				"total": len(resp.Data),
				"keys":  resp.Data,
			})
			return string(data), nil
		},
	})

	// ── list_label_values: 列出某 label key 的所有值 ──
	r.Register(&AITool{
		Name:        "list_label_values",
		Description: "列出某数据源中指定 label key 的所有值。用于探索标签值分布，如列出所有 job 名、instance 等。",
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

			data, _ := json.Marshal(map[string]interface{}{
				"total":      len(resp.Data),
				"returned":   len(values),
				"label_key":  labelKey,
				"values":     values,
			})
			return string(data), nil
		},
	})

	// ── query_instant: 即时查询 PromQL ──
	r.Register(&AITool{
		Name:        "query_instant",
		Description: "对指定数据源执行 PromQL 即时查询（当前时刻的快照值）。适用于查看当前状态、单值指标。",
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

			summary := fmt.Sprintf("查询结果: %d 条时间序列", len(resp.Series))
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
			data, _ := json.Marshal(map[string]interface{}{
				"metric": metricName,
				"type":   entry["type"],
				"help":   entry["help"],
				"unit":   entry["unit"],
			})
			return string(data), nil
		},
	})

	// ── search_similar_alerts: 搜索相似告警历史 ──
	r.Register(&AITool{
		Name:        "search_similar_alerts",
		Description: "搜索最近的告警事件。可按名称、严重等级、状态过滤。用于查找历史告警记录，辅助根因分析。",
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

			data, _ := json.Marshal(map[string]interface{}{
				"total":  len(summaries),
				"events": summaries,
			})
			return string(data), nil
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
