// Package i18n provides lightweight, dependency-free localization for backend
// user-facing strings (currently the structured error catalog).
//
// Scope: this localizes the canonical messages of the predefined AppError values
// (internal/pkg/errors) plus the two hardcoded fallbacks in the response helper.
// Custom messages built via errors.WithMessage(base, "<free text>") are NOT in the
// catalog and pass through unchanged — translating every ad-hoc fmt.Errorf string
// across the services is a separate, larger effort.
package i18n

import "strings"

const (
	// ZhCN is Simplified Chinese.
	ZhCN = "zh-CN"
	// En is English — the source language of the AppError catalog.
	En = "en"
)

// Negotiate resolves a locale from an explicit override (e.g. an X-Lang header set
// by the SPA to mirror its in-app language toggle) and/or the Accept-Language header.
// Falls back to English so existing API consumers see no behavior change.
func Negotiate(explicit, acceptLanguage string) string {
	if l := normalize(explicit); l != "" {
		return l
	}
	if l := normalize(acceptLanguage); l != "" {
		return l
	}
	return En
}

func normalize(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "zh"):
		return ZhCN
	case strings.Contains(s, "en"):
		return En
	default:
		return ""
	}
}

// translations maps locale -> canonical English message -> localized message.
// English is the source language, so it needs no table.
var translations = map[string]map[string]string{
	ZhCN: {
		// 10000-10099 Validation
		"bad request":                "请求错误",
		"invalid parameter":          "参数错误",
		"missing required parameter": "缺少必填参数",
		"business error":             "业务处理失败",
		// 10100-10199 Authentication
		"unauthorized":             "未授权",
		"invalid or expired token": "登录已失效或令牌过期",
		"invalid credentials":      "用户名或密码错误",
		// 10200-10299 Authorization
		"forbidden":     "没有权限执行该操作",
		"no permission": "权限不足",
		// 10300-10399 Not found
		"resource not found":              "资源不存在",
		"user not found":                  "用户不存在",
		"alert rule not found":            "告警规则不存在",
		"alert event not found":           "告警事件不存在",
		"datasource not found":            "数据源不存在",
		"notification channel not found":  "通知渠道不存在",
		"notification policy not found":   "通知策略不存在",
		"team not found":                  "团队不存在",
		"notify rule not found":           "通知规则不存在",
		"notify media not found":          "通知媒介不存在",
		"message template not found":      "消息模板不存在",
		"subscribe rule not found":        "订阅规则不存在",
		"business group not found":        "业务分组不存在",
		"cannot delete built-in resource": "内置资源不可删除",
		"template rendering failed":       "模板渲染失败",
		"collaboration channel not found": "协作空间不存在",
		"incident not found":              "故障不存在",
		// 10400-10499 Conflict
		"resource already exists":  "资源已存在",
		"name already taken":       "名称已被占用",
		"invalid state transition": "非法的状态流转",
		"version conflict, resource was modified by another request": "版本冲突，资源已被其他请求修改",
		// 10500-10599 Rate limit
		"rate limit exceeded": "请求过于频繁，请稍后再试",
		// 50000+ Internal
		"internal server error": "服务器内部错误",
		"database error":        "数据库错误",
		"redis error":           "缓存服务错误",
		"external api error":    "外部接口调用失败",

		// AI / Agent
		"AI feature is not enabled":                              "AI 功能未启用，请在系统设置中配置并启用 AI",
		"AI is not enabled":                                      "AI 未启用",
		"failed to load AI config":                               "加载 AI 配置失败",
		"LLM planning call failed":                               "LLM 规划调用失败",
		"failed to parse planning result":                        "解析规划结果失败",
		"tool does not exist":                                    "工具不存在",
		"tool execution failed":                                  "工具执行失败",
		"tool is not in allowed list":                            "工具不在允许列表中",
		"invalid time range format":                              "无效的时间范围格式",
		"query time range exceeds maximum allowed by AI tools":   "查询时间范围超过 AI 工具允许的最大值",

		// Inspection / Report
		"failed to create inspection run":    "创建巡检运行记录失败",
		"inspection agent execution failed":  "巡检 Agent 执行失败",
		"failed to load inspection tasks":    "加载巡检任务失败",
		"failed to create report run":        "创建报告运行记录失败",
		"report agent execution failed":      "报告 Agent 执行失败",
		"failed to load report tasks":        "加载报告任务失败",
		"invalid cron expression":            "无效的 cron 表达式",

		// Diagnostic workflow
		"query failed":                       "查询失败",
		"change_correlation query failed":    "change_correlation 查询失败",

		// Rule generator
		"expression is empty":   "表达式为空",
		"PromQL syntax error":   "PromQL 语法错误",

		// Lark tools
		"event_id is required": "event_id 不能为空",
		"task_id is required":  "task_id 不能为空",
	},
}

// LocalizeMessage returns the localized form of a canonical English message for the
// given locale. Unknown messages (including custom WithMessage text) and the English
// locale return the input unchanged.
func LocalizeMessage(locale, msg string) string {
	if locale == "" || locale == En {
		return msg
	}
	if table, ok := translations[locale]; ok {
		if t, ok := table[msg]; ok {
			return t
		}
	}
	return msg
}
