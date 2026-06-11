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
	taskRepo *repository.ReportTaskRepository
	runRepo  *repository.ReportTaskRepository
	agentSvc *AgentService
	logger   *zap.Logger
}

// NewReportExecutor creates a new ReportExecutor.
func NewReportExecutor(
	taskRepo *repository.ReportTaskRepository,
	agentSvc *AgentService,
	logger *zap.Logger,
) *ReportExecutor {
	return &ReportExecutor{
		taskRepo: taskRepo,
		runRepo:  taskRepo,
		agentSvc: agentSvc,
		logger:   logger,
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

// Run executes a single report task, returns the run record and possible error.
func (e *ReportExecutor) Run(ctx context.Context, task *model.ReportTask) (*model.ReportRun, error) {
	run := &model.ReportRun{
		TaskID:    task.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := e.runRepo.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("创建报告运行记录失败: %w", err)
	}

	e.logger.Info("报告任务开始执行",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Uint("run_id", run.ID),
	)

	// Parse allowed tools
	var allowedTools []string
	if task.AllowedTools != "" {
		if err := json.Unmarshal([]byte(task.AllowedTools), &allowedTools); err != nil {
			e.logger.Warn("解析 allowed_tools 失败，使用全部只读工具", zap.Error(err))
			allowedTools = nil
		}
	}

	// Build prompts
	systemPrompt := buildReportSystemPrompt()
	userPrompt := buildReportUserPrompt(task.Name, task.Description, task.PromptTemplate)

	// Execute via Agent (report tasks use system user ID 0)
	result, err := e.agentSvc.RunUntilDone(ctx, 0, systemPrompt, userPrompt, allowedTools, 15)
	if err != nil {
		run.Status = "failed"
		run.ErrorMsg = err.Error()
		now := time.Now()
		run.FinishedAt = &now
		_ = e.runRepo.UpdateRun(ctx, run)
		return run, fmt.Errorf("报告 Agent 执行失败: %w", err)
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

	return run, nil
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
