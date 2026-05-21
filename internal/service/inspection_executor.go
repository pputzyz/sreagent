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

// InspectionExecutor 执行单次巡检任务
type InspectionExecutor struct {
	taskRepo *repository.InspectionRepository
	runRepo  *repository.InspectionRepository
	agentSvc *AgentService
	logger   *zap.Logger
}

// NewInspectionExecutor 创建巡检执行器
func NewInspectionExecutor(
	taskRepo *repository.InspectionRepository,
	agentSvc *AgentService,
	logger *zap.Logger,
) *InspectionExecutor {
	return &InspectionExecutor{
		taskRepo: taskRepo,
		runRepo:  taskRepo,
		agentSvc: agentSvc,
		logger:   logger,
	}
}

// InspectionFinding 结构化发现项
type InspectionFinding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Object   string `json:"object"`
	Detail   string `json:"detail"`
}

// InspectionReport 巡检报告结构
type InspectionReport struct {
	Summary  string              `json:"summary"`
	Findings []InspectionFinding `json:"findings"`
}

var jsonBlockRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?})\\s*```")

// Run 执行一次巡检任务，返回 run ID 和可能的错误
func (e *InspectionExecutor) Run(ctx context.Context, task *model.InspectionTask) (*model.InspectionRun, error) {
	run := &model.InspectionRun{
		TaskID:    task.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := e.runRepo.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("创建巡检运行记录失败: %w", err)
	}

	e.logger.Info("巡检任务开始执行",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Uint("run_id", run.ID),
	)

	// 解析 allowed tools
	var allowedTools []string
	if task.AllowedTools != "" {
		if err := json.Unmarshal([]byte(task.AllowedTools), &allowedTools); err != nil {
			e.logger.Warn("解析 allowed_tools 失败，使用全部只读工具", zap.Error(err))
			allowedTools = nil
		}
	}

	// 构建 prompt
	systemPrompt := buildInspectionSystemPrompt()
	userPrompt := buildInspectionUserPrompt(task.Name, task.Description)

	// 调用 Agent 执行（巡检任务使用 system user ID 0）
	result, err := e.agentSvc.RunUntilDone(ctx, 0, systemPrompt, userPrompt, allowedTools, 15)
	if err != nil {
		run.Status = "failed"
		run.ErrorMsg = err.Error()
		now := time.Now()
		run.FinishedAt = &now
		_ = e.runRepo.UpdateRun(ctx, run)
		return run, fmt.Errorf("巡检 Agent 执行失败: %w", err)
	}

	run.AIConversationID = &result.ConversationID

	// 解析报告
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
		e.logger.Error("更新巡检运行记录失败", zap.Error(err))
	}

	e.logger.Info("巡检任务执行完成",
		zap.Uint("task_id", task.ID),
		zap.Uint("run_id", run.ID),
		zap.String("status", run.Status),
		zap.Int("findings", len(report.Findings)),
	)

	return run, nil
}

// parseReport 从 LLM 输出中提取结构化巡检报告
func (e *InspectionExecutor) parseReport(output string) InspectionReport {
	report := InspectionReport{
		Summary:  "巡检完成",
		Findings: nil,
	}

	matches := jsonBlockRe.FindStringSubmatch(output)
	if len(matches) < 2 {
		// 没有找到 JSON 块，用纯文本作为 summary
		report.Summary = truncateString(strings.TrimSpace(output), 500)
		return report
	}

	var parsed InspectionReport
	if err := json.Unmarshal([]byte(matches[1]), &parsed); err != nil {
		e.logger.Warn("解析巡检报告 JSON 失败", zap.Error(err))
		report.Summary = truncateString(strings.TrimSpace(output), 500)
		return report
	}

	return parsed
}
