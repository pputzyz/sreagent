package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// LeaderChecker is a minimal interface to avoid import cycle with engine package.
type LeaderChecker interface {
	IsLeader() bool
}

// InspectionScheduler 管理巡检任务的定时调度
type InspectionScheduler struct {
	taskRepo   *repository.InspectionRepository
	executor   *InspectionExecutor
	leader     LeaderChecker
	larkSvc    *LarkBotService
	httpClient *http.Client
	logger     *zap.Logger

	cron    *cron.Cron
	mu      sync.Mutex
	entries map[uint]cron.EntryID // taskID → cron entry
}

// NewInspectionScheduler 创建巡检调度器
func NewInspectionScheduler(
	taskRepo *repository.InspectionRepository,
	executor *InspectionExecutor,
	leader LeaderChecker,
	larkSvc *LarkBotService,
	logger *zap.Logger,
) *InspectionScheduler {
	return &InspectionScheduler{
		taskRepo:   taskRepo,
		executor:   executor,
		leader:     leader,
		larkSvc:    larkSvc,
		httpClient: safehttp.NewSafeClient(10 * time.Second),
		logger:     logger,
		// WithChain(Recover) ensures a panic inside a job is recovered and logged
		// instead of crashing the whole server process. cron/v3 does NOT recover by default.
		cron:    cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cron.DefaultLogger))),
		entries: make(map[uint]cron.EntryID),
	}
}

// Start 从 DB 加载所有启用任务并启动调度器
func (s *InspectionScheduler) Start(ctx context.Context) error {
	tasks, err := s.taskRepo.ListEnabledTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load inspection tasks: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			s.logger.Error("注册巡检任务失败",
				zap.Uint("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.Error(err),
			)
		}
	}

	s.cron.Start()
	s.logger.Info("巡检调度器已启动", zap.Int("tasks", len(tasks)))
	return nil
}

// Stop 停止调度器
func (s *InspectionScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("巡检调度器已停止")
}

// AddTask 动态添加一个巡检任务到调度器
func (s *InspectionScheduler) AddTask(task model.InspectionTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已存在，先移除
	if entryID, ok := s.entries[task.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, task.ID)
	}

	if !task.Enabled {
		return nil
	}

	return s.addTask(task)
}

// RemoveTask 从调度器移除一个巡检任务
func (s *InspectionScheduler) RemoveTask(taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
		s.logger.Info("已移除巡检任务", zap.Uint("task_id", taskID))
	}
}

// addTask 内部方法，调用方需持有锁
func (s *InspectionScheduler) addTask(task model.InspectionTask) error {
	taskCopy := task
	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.runWithLeaderCheck(&taskCopy)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", task.CronExpr, err)
	}

	s.entries[task.ID] = entryID
	s.logger.Info("已注册巡检任务",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.String("cron", task.CronExpr),
	)
	return nil
}

// runWithLeaderCheck 检查 leader 后执行巡检
// B8-6: Re-fetches task from DB to avoid using a stale snapshot from registration time.
func (s *InspectionScheduler) runWithLeaderCheck(task *model.InspectionTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if s.leader != nil && !s.leader.IsLeader() {
		s.logger.Debug("非 leader 节点，跳过巡检", zap.Uint("task_id", task.ID))
		return
	}

	// B8-6: Re-fetch task from DB to pick up any updates since registration.
	freshTask, err := s.taskRepo.GetTask(ctx, task.ID)
	if err != nil {
		s.logger.Error("巡检任务重新加载失败，使用缓存快照",
			zap.Uint("task_id", task.ID), zap.Error(err))
		// Fall through with stale task — better to run with old config than skip entirely.
	} else {
		task = freshTask
	}

	if !task.Enabled {
		s.logger.Debug("巡检任务已禁用，跳过执行", zap.Uint("task_id", task.ID))
		return
	}

	s.logger.Info("开始执行巡检任务",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
	)

	run, err := s.executor.Run(ctx, task)
	if err != nil {
		s.logger.Error("巡检任务执行失败",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
		return
	}

	// 发送通知（飞书卡片等）
	s.notifyResult(task, run)
}

// OutputChannel 输出渠道配置
type OutputChannel struct {
	Type  string   `json:"type"` // lark_bot, email, webhook
	BotID string   `json:"bot_id,omitempty"`
	To    []string `json:"to,omitempty"`
	URL   string   `json:"url,omitempty"`
}

// notifyResult 将巡检结果发送到配置的输出渠道
func (s *InspectionScheduler) notifyResult(task *model.InspectionTask, run *model.InspectionRun) {
	var channels []OutputChannel
	if err := json.Unmarshal([]byte(task.OutputChannels), &channels); err != nil {
		s.logger.Warn("解析输出渠道配置失败", zap.Uint("task_id", task.ID), zap.Error(err))
		return
	}

	summary := fmt.Sprintf("[巡检] %s\n状态: %s\n%s", task.Name, run.Status, run.ReportSummary)

	for _, ch := range channels {
		switch ch.Type {
		case "lark_bot":
			if s.larkSvc == nil {
				s.logger.Warn("飞书服务未配置，跳过通知", zap.Uint("task_id", task.ID))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := s.larkSvc.SendMessage(ctx, ch.BotID, summary)
			cancel()
			if err != nil {
				s.logger.Error("巡检结果飞书通知失败", zap.Uint("task_id", task.ID), zap.Error(err))
			} else {
				s.logger.Info("巡检结果飞书通知已发送", zap.Uint("task_id", task.ID))
			}
		case "webhook":
			if ch.URL == "" {
				s.logger.Warn("webhook URL 为空，跳过通知", zap.Uint("task_id", task.ID))
				continue
			}
			payload := map[string]interface{}{
				"task_id":     task.ID,
				"task_name":   task.Name,
				"status":      run.Status,
				"summary":     run.ReportSummary,
				"started_at":  run.StartedAt,
				"finished_at": run.FinishedAt,
			}
			body, _ := json.Marshal(payload)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ch.URL, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// HMAC-SHA256 signing for webhook payload integrity verification (B8-12).
			// If SREAGENT_WEBHOOK_SECRET is set, compute HMAC and attach X-Signature-256 header.
			if secret := os.Getenv("SREAGENT_WEBHOOK_SECRET"); secret != "" {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write(body)
				sig := hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("X-Signature-256", "sha256="+sig)
			}
			resp, err := s.httpClient.Do(req)
			cancel()
			if err != nil {
				s.logger.Error("巡检结果 webhook 通知失败", zap.Uint("task_id", task.ID), zap.Error(err))
			} else {
				_ = resp.Body.Close()
				s.logger.Info("巡检结果 webhook 通知已发送", zap.Uint("task_id", task.ID), zap.Int("status", resp.StatusCode))
			}
		default:
			s.logger.Warn("未知的输出渠道类型", zap.String("type", ch.Type))
		}
	}
}

// ListEntries 返回当前调度中的任务 ID 列表（调试用）
func (s *InspectionScheduler) ListEntries() []uint {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]uint, 0, len(s.entries))
	for id := range s.entries {
		ids = append(ids, id)
	}
	return ids
}
