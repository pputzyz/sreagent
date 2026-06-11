package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RegisterLarkTools registers SRE tools available to the Lark bot Agent.
// These tools allow the Lark bot to query alert data, statistics, and on-call info.
func RegisterLarkTools(
	registry *AIToolRegistry,
	eventSvc *AlertEventService,
	statsSvc *DashboardStatsService,
	scheduleSvc *ScheduleService,
) {
	// ── query_alert_events: 查询告警事件列表 ──
	registry.Register(&AITool{
		Name:        "query_alert_events",
		Description: "查询告警事件列表，支持按状态和严重等级筛选。返回最近的告警事件摘要。",
		IO:          "read",
		RiskLevel:   0,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "告警状态筛选: firing, acknowledged, resolved, closed",
				},
				"severity": map[string]interface{}{
					"type":        "string",
					"description": "严重等级筛选: critical, warning, info, debug",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量上限，默认 10",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			status, _ := params["status"].(string)
			severity, _ := params["severity"].(string)
			limit, _ := aiToolToInt(params["limit"])
			if limit <= 0 {
				limit = 10
			}

			events, total, err := eventSvc.List(ctx, status, severity, 1, limit)
			if err != nil {
				return fmt.Sprintf("查询告警事件失败: %v", err), nil
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
				"total":  total,
				"events": summaries,
			}), nil
		},
	})

	// ── alert_statistics: 告警统计（MTTA/MTTR/按严重等级计数） ──
	registry.Register(&AITool{
		Name:        "alert_statistics",
		Description: "获取告警统计信息，包括 MTTA（平均认领时间）、MTTR（平均恢复时间）和按严重等级的分布。默认查询最近 24 小时。",
		IO:          "read",
		RiskLevel:   0,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"hours": map[string]interface{}{
					"type":        "integer",
					"description": "统计时间窗口（小时），默认 24",
				},
			},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			hours, _ := aiToolToInt(params["hours"])
			if hours <= 0 {
				hours = 24
			}

			stats, err := statsSvc.GetMTTRStats(ctx, hours)
			if err != nil {
				return fmt.Sprintf("获取告警统计失败: %v", err), nil
			}

			return marshalJSONOrError(stats), nil
		},
	})

	// ── acknowledge_alert: 认领告警 ──
	registry.Register(&AITool{
		Name:        "acknowledge_alert",
		Description: "认领（acknowledge）一个告警事件。需要提供告警事件 ID。此操作会将告警状态从 firing 变为 acknowledged。",
		IO:          "write",
		RiskLevel:   1,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"event_id": map[string]interface{}{
					"type":        "integer",
					"description": "告警事件 ID",
				},
				"user_id": map[string]interface{}{
					"type":        "integer",
					"description": "操作人用户 ID",
				},
			},
			"required": []string{"event_id", "user_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			eventID, _ := aiToolToUint(params["event_id"])
			userID, _ := aiToolToUint(params["user_id"])

			if eventID == 0 {
				return "", fmt.Errorf("event_id 不能为空")
			}
			if userID == 0 {
				return "", fmt.Errorf("user_id 不能为空")
			}

			if err := eventSvc.Acknowledge(ctx, eventID, userID); err != nil {
				return fmt.Sprintf("认领告警失败: %v", err), nil
			}

			return marshalJSONOrError(map[string]interface{}{
				"success":  true,
				"event_id": eventID,
				"message":  fmt.Sprintf("告警 #%d 已认领", eventID),
			}), nil
		},
	})

	// ── get_oncall: 获取当前值班人 ──
	registry.Register(&AITool{
		Name:        "get_oncall",
		Description: "获取当前值班人员信息，包括姓名、邮箱、电话。用于快速联系值班人处理故障。",
		IO:          "read",
		RiskLevel:   0,
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			if scheduleSvc == nil {
				return "值班排班服务未配置", nil
			}

			user, err := scheduleSvc.GetCurrentOnCallForAlert(ctx, map[string]string{})
			if err != nil || user == nil {
				return "当前无值班人，请在 SREAgent 中配置排班", nil
			}

			result := map[string]interface{}{
				"name":         user.DisplayName,
				"email":        user.Email,
				"phone":        user.Phone,
				"lark_user_id": user.LarkUserID,
			}
			return marshalJSONOrError(result), nil
		},
	})

	registry.logger.Info("Lark bot tools registered",
		zap.Strings("tools", []string{
			"query_alert_events",
			"alert_statistics",
			"acknowledge_alert",
			"get_oncall",
		}),
	)
}
