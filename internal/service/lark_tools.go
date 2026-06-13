package service

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/repository"
)

// RegisterLarkTools registers SRE tools available to the Lark bot Agent.
// These tools allow the Lark bot to query alert data, statistics, on-call
// info, trigger inspections, and acknowledge alerts.
//
// Security: write tools take the operator from the agent context
// (AgentOperatorFromContext), never from LLM-controlled parameters.
func RegisterLarkTools(
	registry *AIToolRegistry,
	eventSvc *AlertEventService,
	statsSvc *DashboardStatsService,
	scheduleSvc *ScheduleService,
	inspectionExec *InspectionExecutor,
	inspectionRepo *repository.InspectionRepository,
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
					"description": "告警状态筛选: firing, acknowledged, assigned, silenced, resolved, closed",
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
			if limit > 50 {
				limit = 50
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
			if hours > 24*31 {
				hours = 24 * 31
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
		Name: "acknowledge_alert",
		Description: "认领（acknowledge）一个告警事件，将状态从 firing 变为 acknowledged。" +
			"操作以当前对话用户的身份执行；未绑定平台账号的用户无法执行。",
		IO:        "write",
		RiskLevel: 1,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"event_id": map[string]interface{}{
					"type":        "integer",
					"description": "告警事件 ID",
				},
			},
			"required": []string{"event_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			eventID, _ := aiToolToUint(params["event_id"])
			if eventID == 0 {
				return "", fmt.Errorf("event_id is required")
			}

			// Operator comes from the agent context (the mapped platform user
			// of the Lark sender) — NOT from an LLM-fillable parameter.
			operatorID := AgentOperatorFromContext(ctx)
			if operatorID == 0 {
				return "认领失败：当前用户未绑定 SREAgent 平台账号，无法执行写操作。请先在平台个人设置中绑定 Lark 账号。", nil
			}

			if err := eventSvc.Acknowledge(ctx, eventID, operatorID); err != nil {
				return fmt.Sprintf("认领告警失败: %v", err), nil
			}

			return marshalJSONOrError(map[string]interface{}{
				"success":  true,
				"event_id": eventID,
				"operator": operatorID,
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

	// ── run_inspection: 触发巡检任务 ──
	registry.Register(&AITool{
		Name: "run_inspection",
		Description: "触发一个已配置的巡检任务（按任务 ID）。巡检在后台异步执行（通常需要数分钟），" +
			"执行结果记录在平台「巡检记录」页面。返回触发确认。",
		IO:        "write",
		RiskLevel: 1,
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "integer",
					"description": "巡检任务 ID（可先用其他工具或平台页面查看任务列表）",
				},
			},
			"required": []string{"task_id"},
		},
		Execute: func(ctx context.Context, params map[string]interface{}) (string, error) {
			if inspectionExec == nil || inspectionRepo == nil {
				return "巡检服务未配置", nil
			}
			taskID, _ := aiToolToUint(params["task_id"])
			if taskID == 0 {
				return "", fmt.Errorf("task_id is required")
			}
			if AgentOperatorFromContext(ctx) == 0 {
				return "触发失败：当前用户未绑定 SREAgent 平台账号，无法执行写操作。", nil
			}

			task, err := inspectionRepo.GetTask(ctx, taskID)
			if err != nil {
				return fmt.Sprintf("巡检任务 #%d 不存在", taskID), nil
			}
			if !task.Enabled {
				return fmt.Sprintf("巡检任务 #%d（%s）已禁用", taskID, task.Name), nil
			}

			// Run asynchronously: inspections take minutes; the chat reply
			// only confirms the trigger. Results flow through the task's own
			// output channels.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						registry.logger.Error("run_inspection panic recovered",
							zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
					}
				}()
				runCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
				defer cancel()
				if _, err := inspectionExec.Run(runCtx, task); err != nil {
					registry.logger.Error("inspection triggered from Lark failed",
						zap.Uint("task_id", taskID), zap.Error(err))
				}
			}()

			return marshalJSONOrError(map[string]interface{}{
				"triggered": true,
				"task_id":   taskID,
				"task_name": task.Name,
				"message":   fmt.Sprintf("巡检任务「%s」已在后台触发，结果可稍后在平台「巡检记录」页查看", task.Name),
			}), nil
		},
	})

	registry.logger.Info("Lark bot tools registered",
		zap.Strings("tools", []string{
			"query_alert_events",
			"alert_statistics",
			"acknowledge_alert",
			"get_oncall",
			"run_inspection",
		}),
	)
}
