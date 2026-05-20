package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// AgentTask 表示一个 Agent 任务
type AgentTask struct {
	ID             string      `json:"id"`
	ConversationID uint        `json:"conversation_id,omitempty"`
	Query          string      `json:"query"`
	Status         string      `json:"status"` // planning, executing, completed, failed
	Steps          []AgentStep `json:"steps"`
	Result         string      `json:"result"`
	Error          string      `json:"error,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
}

// AgentStep 表示 Agent 执行的一步
type AgentStep struct {
	Index       int                    `json:"index"`
	Description string                 `json:"description"`
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Result      string                 `json:"result"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	Duration    int64                  `json:"duration_ms"`
}

// AgentService 管理 Agent 任务的规划与执行
type AgentService struct {
	aiSvc    *AIService
	toolReg  *AIToolRegistry
	convRepo *repository.AIConversationRepository
	logger   *zap.Logger

	// 内存任务存储（用于快速轮询，DB 用于持久化）
	mu    sync.RWMutex
	tasks map[string]*AgentTask
}

// NewAgentService 创建 Agent 服务
func NewAgentService(aiSvc *AIService, convRepo *repository.AIConversationRepository, toolReg *AIToolRegistry, logger *zap.Logger) *AgentService {
	s := &AgentService{
		aiSvc:    aiSvc,
		toolReg:  toolReg,
		convRepo: convRepo,
		logger:   logger,
		tasks:    make(map[string]*AgentTask),
	}
	// 定期清理过期任务，防止 OOM
	go s.cleanupLoop()
	return s
}

// SetToolRegistry 延迟注入工具注册表（DI 两阶段初始化）
func (s *AgentService) SetToolRegistry(reg *AIToolRegistry) {
	s.toolReg = reg
}

// cleanupLoop 每 10 分钟清理超过 1 小时的已完成任务
func (s *AgentService) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		for id, t := range s.tasks {
			if t.CompletedAt != nil && t.CompletedAt.Before(cutoff) {
				delete(s.tasks, id)
			}
		}
		s.mu.Unlock()
	}
}

// guardrails 常量
const (
	agentMaxSteps     = 10
	agentStepTimeout  = 30 * time.Second
	agentTotalTimeout = 5 * time.Minute
)

// stepPlan 用于解析 LLM 返回的步骤规划
type stepPlan struct {
	Steps []planStep `json:"steps"`
}

type planStep struct {
	Description string                 `json:"description"`
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// StartAgent 异步启动 Agent 任务，立即返回任务 ID，后台执行完成后前端轮询获取结果
func (s *AgentService) StartAgent(userID uint, query string) (*AgentTask, error) {
	task := &AgentTask{
		ID:        uuid.New().String(),
		Query:     query,
		Status:    "planning",
		Steps:     nil,
		CreatedAt: time.Now(),
	}

	// 持久化会话到 DB
	if s.convRepo != nil {
		conv := &model.AIConversation{
			UserID: userID,
			Title:  truncateString(query, 100),
			Status: "active",
		}
		if err := s.convRepo.Create(context.Background(), conv); err != nil {
			s.logger.Warn("创建 AI 会话失败", zap.Error(err))
		} else {
			task.ConversationID = conv.ID
		}
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	go func() {
		ctx := context.Background()
		_, _ = s.runTask(ctx, task)
	}()

	s.logger.Info("Agent 任务已启动", zap.String("id", task.ID), zap.String("query", query))
	return task, nil
}

// RunAgent 执行一个 Agent 任务（同步模式，创建新任务并执行）
func (s *AgentService) RunAgent(ctx context.Context, userID uint, query string) (*AgentTask, error) {
	task := &AgentTask{
		ID:        uuid.New().String(),
		Query:     query,
		Status:    "planning",
		Steps:     nil,
		CreatedAt: time.Now(),
	}

	// 持久化会话到 DB
	if s.convRepo != nil {
		conv := &model.AIConversation{
			UserID: userID,
			Title:  truncateString(query, 100),
			Status: "active",
		}
		if err := s.convRepo.Create(ctx, conv); err != nil {
			s.logger.Warn("创建 AI 会话失败", zap.Error(err))
		} else {
			task.ConversationID = conv.ID
		}
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return s.runTask(ctx, task)
}

// runTask 执行已创建的任务（核心逻辑）
func (s *AgentService) runTask(ctx context.Context, task *AgentTask) (*AgentTask, error) {
	// 总超时
	ctx, cancel := context.WithTimeout(ctx, agentTotalTimeout)
	defer cancel()

	s.logger.Info("Agent 任务开始", zap.String("id", task.ID), zap.String("query", task.Query))

	// 第 1 步：规划
	steps, err := s.planSteps(ctx, task.Query)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("规划失败: %v", err)
		now := time.Now()
		task.CompletedAt = &now
		return task, err
	}

	// guardrails: 限制步骤数
	if len(steps) > agentMaxSteps {
		steps = steps[:agentMaxSteps]
		s.logger.Warn("Agent 步骤数超过上限，已截断",
			zap.Int("original", len(steps)),
			zap.Int("max", agentMaxSteps),
		)
	}

	task.Steps = steps
	task.Status = "executing"
	s.logger.Info("Agent 规划完成", zap.String("id", task.ID), zap.Int("steps", len(steps)))

	// 第 2 步：逐步执行
	for i := range task.Steps {
		step := &task.Steps[i]
		step.Status = "running"
		startTime := time.Now()

		err := s.executeStep(ctx, task, step)
		step.Duration = time.Since(startTime).Milliseconds()

		if err != nil {
			step.Status = "failed"
			step.Result = fmt.Sprintf("执行失败: %v", err)
			s.logger.Warn("Agent 步骤失败",
				zap.String("task_id", task.ID),
				zap.Int("step", step.Index),
				zap.Error(err),
			)

			// 让 LLM 决定是否跳过
			if !s.shouldContinue(ctx, task, i) {
				task.Status = "failed"
				task.Error = fmt.Sprintf("步骤 %d 失败且无法恢复: %v", step.Index, err)
				now := time.Now()
				task.CompletedAt = &now
				return task, nil
			}
			continue
		}

		step.Status = "completed"
		s.logger.Info("Agent 步骤完成",
			zap.String("task_id", task.ID),
			zap.Int("step", step.Index),
			zap.Int64("duration_ms", step.Duration),
		)
	}

	// 第 3 步：汇总
	summary, err := s.summarize(ctx, task)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("汇总失败: %v", err)
		now := time.Now()
		task.CompletedAt = &now
		return task, nil
	}

	task.Result = summary
	task.Status = "completed"
	now := time.Now()
	task.CompletedAt = &now

	s.logger.Info("Agent 任务完成", zap.String("id", task.ID))
	return task, nil
}

// ListConversations returns paginated conversations for a user.
func (s *AgentService) ListConversations(ctx context.Context, userID uint, page, pageSize int) ([]model.AIConversation, int64, error) {
	if s.convRepo == nil {
		return nil, 0, nil
	}
	return s.convRepo.ListByUser(ctx, userID, page, pageSize)
}

// GetConversation returns a conversation by ID.
func (s *AgentService) GetConversation(ctx context.Context, id uint) (*model.AIConversation, error) {
	if s.convRepo == nil {
		return nil, fmt.Errorf("conversation repository not available")
	}
	return s.convRepo.GetByID(ctx, id)
}

// DeleteConversation soft-deletes a conversation.
func (s *AgentService) DeleteConversation(ctx context.Context, id uint) error {
	if s.convRepo == nil {
		return fmt.Errorf("conversation repository not available")
	}
	return s.convRepo.Delete(ctx, id)
}

// ListToolCalls returns all tool calls for a conversation.
func (s *AgentService) ListToolCalls(ctx context.Context, conversationID uint) ([]model.AIToolCall, error) {
	if s.convRepo == nil {
		return nil, nil
	}
	return s.convRepo.ListToolCalls(ctx, conversationID)
}

// GetTask 获取任务详情
func (s *AgentService) GetTask(id string) (*AgentTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// planSteps 让 LLM 规划执行步骤
func (s *AgentService) planSteps(ctx context.Context, query string) ([]AgentStep, error) {
	tools := s.toolReg.List()

	// 构造工具列表描述
	toolDescs := make([]string, 0, len(tools))
	for _, t := range tools {
		toolDescs = append(toolDescs, fmt.Sprintf("工具名: %s\n  描述: %s", t.Name, t.Description))
	}

	systemPrompt := "你是一个 SRE 运维助手 Agent。你的任务是根据用户的查询，规划一系列操作步骤来解决问题。\n\n" +
		"可用工具列表：\n" + strings.Join(toolDescs, "\n\n") + "\n\n" +
		"请以 JSON 格式返回执行计划：\n" +
		`{"steps":[{"description":"步骤描述","tool":"工具名","parameters":{"key":"value"}}]}` +
		"\n\n规则：\n" +
		"1. 每个步骤必须使用上述工具之一\n" +
		"2. 步骤数不超过 10 步\n" +
		"3. 步骤之间可以有依赖关系\n" +
		"4. 如果不需要工具，返回空 steps 数组\n" +
		"5. 只输出 JSON，不要有其他内容"

	// 加载 AI 配置
	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载 AI 配置失败: %w", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("AI 未启用")
	}

	result, err := s.aiSvc.callLLMWithSystem(ctx, cfg, systemPrompt, fmt.Sprintf("用户查询: %s", query))
	if err != nil {
		return nil, fmt.Errorf("LLM 规划调用失败: %w", err)
	}

	// 解析 JSON
	cleaned := stripMarkdownCodeBlock(result)
	var plan stepPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("解析规划结果失败: %w (raw: %s)", err, truncateString(result, 200))
	}

	// 转换为 AgentStep
	steps := make([]AgentStep, 0, len(plan.Steps))
	for i, ps := range plan.Steps {
		steps = append(steps, AgentStep{
			Index:       i + 1,
			Description: ps.Description,
			Tool:        ps.Tool,
			Parameters:  ps.Parameters,
			Status:      "pending",
		})
	}

	return steps, nil
}

// executeStep 执行单个步骤
func (s *AgentService) executeStep(ctx context.Context, task *AgentTask, step *AgentStep) error {
	tool, ok := s.toolReg.Get(step.Tool)
	if !ok {
		return fmt.Errorf("工具 %q 不存在", step.Tool)
	}

	// 持久化工具调用记录
	var callID uint
	if s.convRepo != nil && task.ConversationID > 0 {
		paramsBytes, _ := json.Marshal(step.Parameters)
		call := &model.AIToolCall{
			ConversationID: task.ConversationID,
			StepIndex:      step.Index,
			ToolName:       step.Tool,
			Parameters:     string(paramsBytes),
			Status:         "running",
		}
		if err := s.convRepo.CreateToolCall(ctx, call); err != nil {
			s.logger.Warn("保存工具调用记录失败", zap.Error(err))
		} else {
			callID = call.ID
		}
	}

	// 工具执行超时 30s
	toolCtx, cancel := context.WithTimeout(ctx, agentStepTimeout)
	defer cancel()

	result, err := tool.Execute(toolCtx, step.Parameters)
	if err != nil {
		// 更新调用记录状态
		if callID > 0 {
			_ = s.convRepo.UpdateToolCall(ctx, &model.AIToolCall{
				ID:       callID,
				Status:   "failed",
				Error:    err.Error(),
				DurationMs: step.Duration,
			})
		}
		return fmt.Errorf("工具 %q 执行失败: %w", step.Tool, err)
	}
	step.Result = result

	// 更新调用记录
	if callID > 0 {
		_ = s.convRepo.UpdateToolCall(ctx, &model.AIToolCall{
			ID:         callID,
			Result:     truncateString(result, 5000),
			Status:     "completed",
			DurationMs: step.Duration,
		})
	}
	return nil
}

// shouldContinue 让 LLM 判断是否继续执行后续步骤
func (s *AgentService) shouldContinue(ctx context.Context, task *AgentTask, failedIdx int) bool {
	systemPrompt := "你是一个 SRE Agent。某个步骤执行失败了，请判断是否应该继续执行后续步骤。\n" +
		`返回 JSON: {"continue": true, "reason": "原因"}` + "\n\n" +
		"规则：\n" +
		"- 如果失败步骤是关键步骤（如查询告警），可能应该停止\n" +
		"- 如果失败步骤是可选的（如辅助诊断），可以继续\n" +
		"- 只输出 JSON"

	// 收集已完成步骤的结果
	var completed []string
	for i := 0; i <= failedIdx && i < len(task.Steps); i++ {
		st := task.Steps[i]
		completed = append(completed, fmt.Sprintf("步骤 %d (%s): 状态=%s, 结果=%s",
			st.Index, st.Description, st.Status, truncateString(st.Result, 100)))
	}

	userMsg := fmt.Sprintf("任务: %s\n失败步骤: %d (%s)\n已完成步骤:\n%s",
		task.Query, failedIdx+1, task.Steps[failedIdx].Description, strings.Join(completed, "\n"))

	// 加载 AI 配置
	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil {
		s.logger.Warn("shouldContinue 加载 AI 配置失败，默认停止", zap.Error(err))
		return false
	}

	result, err := s.aiSvc.callLLMWithSystem(ctx, cfg, systemPrompt, userMsg)
	if err != nil {
		s.logger.Warn("shouldContinue LLM 调用失败，默认停止", zap.Error(err))
		return false
	}

	cleaned := stripMarkdownCodeBlock(result)
	var decision struct {
		Continue bool   `json:"continue"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(cleaned), &decision); err != nil {
		return false
	}

	s.logger.Info("Agent 继续决策", zap.Bool("continue", decision.Continue), zap.String("reason", decision.Reason))
	return decision.Continue
}

// summarize 让 LLM 汇总所有步骤结果
func (s *AgentService) summarize(ctx context.Context, task *AgentTask) (string, error) {
	systemPrompt := "你是一个 SRE 运维助手。请根据以下 Agent 任务的执行结果，生成一份简洁的中文汇总报告。\n" +
		"格式：\n1. 简要总结\n2. 关键发现\n3. 建议的后续行动（如有）"

	var stepResults []string
	for _, step := range task.Steps {
		status := "[完成]"
		if step.Status == "failed" {
			status = "[失败]"
		} else if step.Status == "pending" {
			status = "[跳过]"
		}
		stepResults = append(stepResults, fmt.Sprintf("%s 步骤 %d: %s\n   工具: %s\n   结果: %s",
			status, step.Index, step.Description, step.Tool, truncateString(step.Result, 300)))
	}

	userMsg := fmt.Sprintf("用户查询: %s\n\n执行步骤:\n%s", task.Query, strings.Join(stepResults, "\n\n"))

	// 加载 AI 配置
	cfg, err := s.aiSvc.loadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("加载 AI 配置失败: %w", err)
	}

	return s.aiSvc.callLLMWithSystem(ctx, cfg, systemPrompt, userMsg)
}
