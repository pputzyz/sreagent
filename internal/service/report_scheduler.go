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
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// ReportScheduler manages scheduled report task execution.
type ReportScheduler struct {
	taskRepo    *repository.ReportTaskRepository
	executor    *ReportExecutor
	leader      LeaderChecker
	larkSvc     *LarkBotService
	httpClient  *http.Client
	externalURL string // platform base URL for report detail links
	logger      *zap.Logger

	cron    *cron.Cron
	mu      sync.Mutex
	entries map[uint]cron.EntryID // taskID -> cron entry
}

// NewReportScheduler creates a new ReportScheduler.
func NewReportScheduler(
	taskRepo *repository.ReportTaskRepository,
	executor *ReportExecutor,
	leader LeaderChecker,
	larkSvc *LarkBotService,
	logger *zap.Logger,
) *ReportScheduler {
	return &ReportScheduler{
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

// Start loads all enabled tasks from DB and starts the scheduler.
func (s *ReportScheduler) Start(ctx context.Context) error {
	tasks, err := s.taskRepo.ListEnabledTasks(ctx)
	if err != nil {
		return fmt.Errorf("加载报告任务失败: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			s.logger.Error("注册报告任务失败",
				zap.Uint("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.Error(err),
			)
		}
	}

	s.cron.Start()
	s.logger.Info("报告调度器已启动", zap.Int("tasks", len(tasks)))
	return nil
}

// Stop stops the scheduler.
func (s *ReportScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("报告调度器已停止")
}

// AddTask dynamically adds a report task to the scheduler.
func (s *ReportScheduler) AddTask(task model.ReportTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if present
	if entryID, ok := s.entries[task.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, task.ID)
	}

	if !task.Enabled {
		return nil
	}

	return s.addTask(task)
}

// RemoveTask removes a report task from the scheduler.
func (s *ReportScheduler) RemoveTask(taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
		s.logger.Info("已移除报告任务", zap.Uint("task_id", taskID))
	}
}

// addTask is the internal method; caller must hold the lock.
func (s *ReportScheduler) addTask(task model.ReportTask) error {
	taskCopy := task
	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.runWithLeaderCheck(&taskCopy)
	})
	if err != nil {
		return fmt.Errorf("无效的 cron 表达式 %q: %w", task.CronExpr, err)
	}

	s.entries[task.ID] = entryID
	s.logger.Info("已注册报告任务",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.String("cron", task.CronExpr),
	)
	return nil
}

// runWithLeaderCheck checks leader status then executes the report task.
// Re-fetches task from DB to avoid using a stale snapshot from registration time.
func (s *ReportScheduler) runWithLeaderCheck(task *model.ReportTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if s.leader != nil && !s.leader.IsLeader() {
		s.logger.Debug("非 leader 节点，跳过报告", zap.Uint("task_id", task.ID))
		return
	}

	// Re-fetch task from DB to pick up any updates since registration.
	freshTask, err := s.taskRepo.GetTask(ctx, task.ID)
	if err != nil {
		s.logger.Error("报告任务重新加载失败，使用缓存快照",
			zap.Uint("task_id", task.ID), zap.Error(err))
	} else {
		task = freshTask
	}

	if !task.Enabled {
		s.logger.Debug("报告任务已禁用，跳过执行", zap.Uint("task_id", task.ID))
		return
	}

	s.logger.Info("开始执行报告任务",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
	)

	run, stats, err := s.executor.Run(ctx, task)
	if err != nil {
		s.logger.Error("报告任务执行失败",
			zap.Uint("task_id", task.ID),
			zap.Error(err),
		)
		if run == nil {
			return
		}
		// fall through: a failed run still notifies (failure card) below.
	}

	// Send notifications
	s.notifyResult(task, run, stats)
}

// ReportOutputChannel is the output channel configuration for report tasks.
type ReportOutputChannel struct {
	Type  string   `json:"type"` // lark_bot, email, webhook
	BotID string   `json:"bot_id,omitempty"`
	To    []string `json:"to,omitempty"`
	URL   string   `json:"url,omitempty"`
}

// SetExternalURL sets the platform base URL for "查看完整报告" buttons.
func (s *ReportScheduler) SetExternalURL(url string) {
	s.externalURL = strings.TrimRight(url, "/")
}

// notifyResult sends the report result to configured output channels.
// lark_bot channels receive a Card 2.0 report card (stat columns + trend
// line + severity pie + top-alerts bar + AI summary); plain text is only
// the fallback when card build/send fails.
func (s *ReportScheduler) notifyResult(task *model.ReportTask, run *model.ReportRun, stats *ReportAlertStats) {
	var channels []ReportOutputChannel
	if err := json.Unmarshal([]byte(task.OutputChannels), &channels); err != nil {
		s.logger.Warn("解析输出渠道配置失败", zap.Uint("task_id", task.ID), zap.Error(err))
		return
	}

	summary := fmt.Sprintf("[报告] %s\n状态: %s\n%s", task.Name, run.Status, run.ReportSummary)

	for _, ch := range channels {
		switch ch.Type {
		case "lark_bot":
			if s.larkSvc == nil {
				s.logger.Warn("飞书服务未配置，跳过通知", zap.Uint("task_id", task.ID))
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err := s.sendLarkReportCard(ctx, ch.BotID, task, run, stats)
			if err != nil {
				s.logger.Warn("报告卡片发送失败，降级为文本", zap.Uint("task_id", task.ID), zap.Error(err))
				err = s.larkSvc.SendMessage(ctx, ch.BotID, summary)
			}
			cancel()
			if err != nil {
				s.logger.Error("报告结果飞书通知失败", zap.Uint("task_id", task.ID), zap.Error(err))
			} else {
				s.logger.Info("报告结果飞书通知已发送", zap.Uint("task_id", task.ID))
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

			// HMAC-SHA256 signing for webhook payload integrity verification.
			if secret := os.Getenv("SREAGENT_WEBHOOK_SECRET"); secret != "" {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write(body)
				sig := hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("X-Signature-256", "sha256="+sig)
			}
			resp, err := s.httpClient.Do(req)
			cancel()
			if err != nil {
				s.logger.Error("报告结果 webhook 通知失败", zap.Uint("task_id", task.ID), zap.Error(err))
			} else {
				_ = resp.Body.Close()
				s.logger.Info("报告结果 webhook 通知已发送", zap.Uint("task_id", task.ID), zap.Int("status", resp.StatusCode))
			}
		default:
			s.logger.Warn("未知的输出渠道类型", zap.String("type", ch.Type))
		}
	}
}

// sendLarkReportCard renders and sends the Card 2.0 report card.
func (s *ReportScheduler) sendLarkReportCard(ctx context.Context, chatID string, task *model.ReportTask, run *model.ReportRun, stats *ReportAlertStats) error {
	detailURL := ""
	if s.externalURL != "" {
		detailURL = fmt.Sprintf("%s/platform/report-tasks?run_id=%d", s.externalURL, run.ID)
	}
	cardJSON, err := BuildReportCardJSON(task.Name, run, stats, detailURL)
	if err != nil {
		return fmt.Errorf("build report card: %w", err)
	}
	return s.larkSvc.SendCardJSON(ctx, chatID, cardJSON)
}

// ListEntries returns the list of currently scheduled task IDs (for debugging).
func (s *ReportScheduler) ListEntries() []uint {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]uint, 0, len(s.entries))
	for id := range s.entries {
		ids = append(ids, id)
	}
	return ids
}
