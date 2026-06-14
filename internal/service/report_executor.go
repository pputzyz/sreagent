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

// ReportExecutor executes a single report generation task.
type ReportExecutor struct {
	taskRepo  *repository.ReportTaskRepository
	runRepo   *repository.ReportTaskRepository
	eventRepo *repository.AlertEventRepository // platform statistics source
	agentSvc  *AgentService
	logger    *zap.Logger
}

// NewReportExecutor creates a new ReportExecutor.
func NewReportExecutor(
	taskRepo *repository.ReportTaskRepository,
	eventRepo *repository.AlertEventRepository,
	agentSvc *AgentService,
	logger *zap.Logger,
) *ReportExecutor {
	return &ReportExecutor{
		taskRepo:  taskRepo,
		runRepo:   taskRepo,
		eventRepo: eventRepo,
		agentSvc:  agentSvc,
		logger:    logger,
	}
}

// ReportFinding is a structured finding item in the report.
type ReportFinding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Object   string `json:"object"`
	Detail   string `json:"detail"`
}

// ReportOutput is the structured report output parsed from LLM response.
type ReportOutput struct {
	Summary  string          `json:"summary"`
	Findings []ReportFinding `json:"findings"`
}

var reportJSONBlockRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?})\\s*```")

// validateReportFinding fills defaults for missing fields in a parsed finding.
func validateReportFinding(f ReportFinding) ReportFinding {
	if f.Severity == "" {
		f.Severity = "info"
	}
	if f.Category == "" {
		f.Category = "general"
	}
	if f.Object == "" {
		f.Object = "未指定对象"
	}
	if f.Detail == "" {
		f.Detail = "无详细描述"
	}
	return f
}

// defaultReportTools is the tool whitelist for report agents when the task
// doesn't configure one. Read-only — a scheduled report must never mutate
// state, and an empty list would expand to the FULL registry downstream.
var defaultReportTools = []string{
	"query_alert_events",
	"alert_statistics",
	"query_instant",
	"list_alert_rules",
	"list_metrics",
	"search_similar_alerts",
	"search_knowledge",
	"get_oncall",
}

// Run executes a single report task. Returns the run record, the
// platform-computed statistics used (for card rendering), and possible error.
func (e *ReportExecutor) Run(ctx context.Context, task *model.ReportTask) (*model.ReportRun, *ReportAlertStats, error) {
	run := &model.ReportRun{
		TaskID:    task.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := e.runRepo.CreateRun(ctx, run); err != nil {
		return nil, nil, fmt.Errorf("failed to create report run: %w", err)
	}

	e.logger.Info("报告任务开始执行",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Uint("run_id", run.ID),
	)

	// Scope-driven platform statistics: the report's numbers come from DB
	// queries; the LLM only interprets them (anti-hallucination boundary).
	scope := parseReportScope(task.Scope)
	var stats *ReportAlertStats
	if e.eventRepo != nil {
		var statsErr error
		stats, statsErr = GatherReportAlertStats(ctx, e.eventRepo, scope, task.ReportType)
		if statsErr != nil {
			e.logger.Warn("报告统计数据收集失败，继续执行（Agent 仍可用工具查询）", zap.Error(statsErr))
		}
	}

	// Parse allowed tools (empty → read-only default set).
	allowedTools := append([]string(nil), defaultReportTools...)
	if task.AllowedTools != "" {
		var configured []string
		if err := json.Unmarshal([]byte(task.AllowedTools), &configured); err != nil {
			e.logger.Warn("解析 allowed_tools 失败，使用默认只读工具集", zap.Error(err))
		} else if len(configured) > 0 {
			allowedTools = configured
		}
	}

	// Build prompts
	systemPrompt := buildReportSystemPrompt()
	userPrompt := buildReportUserPrompt(task, scope, stats)

	// Execute via Agent (report tasks use system user ID 0)
	result, err := e.agentSvc.RunUntilDone(ctx, 0, systemPrompt, userPrompt, allowedTools, 15)
	if err != nil {
		run.Status = "failed"
		run.ErrorMsg = err.Error()
		now := time.Now()
		run.FinishedAt = &now
		_ = e.runRepo.UpdateRun(ctx, run)
		return run, stats, fmt.Errorf("report agent execution failed: %w", err)
	}

	run.AIConversationID = &result.ConversationID

	// Parse report
	report := e.parseReport(result.FinalAnswer)
	run.ReportMarkdown = result.FinalAnswer
	run.ReportSummary = report.Summary

	if len(report.Findings) > 0 {
		findingsJSON, _ := json.Marshal(report.Findings)
		run.FindingsJSON = string(findingsJSON)
	}

	run.Status = "success"
	now := time.Now()
	run.FinishedAt = &now

	if err := e.runRepo.UpdateRun(ctx, run); err != nil {
		e.logger.Error("更新报告运行记录失败", zap.Error(err))
	}

	e.logger.Info("报告任务执行完成",
		zap.Uint("task_id", task.ID),
		zap.Uint("run_id", run.ID),
		zap.String("status", run.Status),
		zap.Int("findings", len(report.Findings)),
	)

	return run, stats, nil
}

// parseReport extracts a structured report from LLM output.
func (e *ReportExecutor) parseReport(output string) ReportOutput {
	report := ReportOutput{
		Summary:  "报告生成完成",
		Findings: nil,
	}

	matches := reportJSONBlockRe.FindStringSubmatch(output)
	if len(matches) < 2 {
		report.Summary = truncateString(strings.TrimSpace(output), 500)
		return report
	}

	var parsed ReportOutput
	if err := json.Unmarshal([]byte(matches[1]), &parsed); err != nil {
		e.logger.Warn("解析报告 JSON 失败", zap.Error(err))
		report.Summary = truncateString(strings.TrimSpace(output), 500)
		return report
	}

	for i := range parsed.Findings {
		parsed.Findings[i] = validateReportFinding(parsed.Findings[i])
	}

	return parsed
}
