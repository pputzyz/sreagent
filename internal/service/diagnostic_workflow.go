package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// DiagnosticWorkflowService manages diagnostic workflow templates and executions.
type DiagnosticWorkflowService struct {
	repo           *repository.DiagnosticWorkflowRepository
	dsSvc          DataSourceQuerier
	aiSvc          *AIService
	changeEventSvc *ChangeEventService
	logger         *zap.Logger
}

func NewDiagnosticWorkflowService(
	repo *repository.DiagnosticWorkflowRepository,
	dsSvc DataSourceQuerier,
	aiSvc *AIService,
	changeEventSvc *ChangeEventService,
	logger *zap.Logger,
) *DiagnosticWorkflowService {
	return &DiagnosticWorkflowService{repo: repo, dsSvc: dsSvc, aiSvc: aiSvc, changeEventSvc: changeEventSvc, logger: logger}
}

// --- Workflow CRUD ---

func (s *DiagnosticWorkflowService) Create(ctx context.Context, wf *model.DiagnosticWorkflow) error {
	return s.repo.Create(ctx, wf)
}

func (s *DiagnosticWorkflowService) GetByID(ctx context.Context, id uint) (*model.DiagnosticWorkflow, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DiagnosticWorkflowService) List(ctx context.Context, category string, enabled *bool, page, pageSize int) ([]model.DiagnosticWorkflow, int64, error) {
	return s.repo.List(ctx, category, enabled, page, pageSize)
}

func (s *DiagnosticWorkflowService) Update(ctx context.Context, wf *model.DiagnosticWorkflow) error {
	return s.repo.Update(ctx, wf)
}

func (s *DiagnosticWorkflowService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// --- Steps CRUD ---

func (s *DiagnosticWorkflowService) ReplaceSteps(ctx context.Context, workflowID uint, steps []model.DiagnosticWorkflowStep) error {
	return s.repo.ReplaceSteps(ctx, workflowID, steps)
}

func (s *DiagnosticWorkflowService) ListSteps(ctx context.Context, workflowID uint) ([]model.DiagnosticWorkflowStep, error) {
	return s.repo.ListSteps(ctx, workflowID)
}

// --- Run ---

func (s *DiagnosticWorkflowService) StartRun(ctx context.Context, workflowID uint, incidentID *uint, userID *uint) (*model.DiagnosticRun, error) {
	wf, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	run := &model.DiagnosticRun{
		WorkflowID:  workflowID,
		IncidentID:  incidentID,
		UserID:      userID,
		CurrentStep: 0,
	}

	// P1-22: If require_approval is true, set status to pending_approval and wait.
	if wf.RequireApproval {
		run.Status = "pending_approval"
		if err := s.repo.CreateRun(ctx, run); err != nil {
			return nil, err
		}
		s.logger.Info("diagnostic run pending approval", zap.Uint("run_id", run.ID), zap.Uint("wf_id", workflowID))
		return run, nil
	}

	run.Status = "running"
	now := time.Now()
	run.StartedAt = &now

	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// Execute steps asynchronously with a 30-minute timeout.
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("diagnostic workflow panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
			}
		}()
		s.executeRun(runCtx, run, wf)
	}()

	return run, nil
}

// ApproveRun transitions a pending_approval run to running and starts execution.
// P1-22: Approval endpoint for diagnostic workflows with require_approval=true.
func (s *DiagnosticWorkflowService) ApproveRun(ctx context.Context, runID uint, userID *uint) (*model.DiagnosticRun, error) {
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("run not found: %w", err)
	}
	if run.Status != "pending_approval" {
		return nil, fmt.Errorf("run is not pending approval (current status: %s)", run.Status)
	}

	wf, err := s.repo.GetByID(ctx, run.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	run.Status = "running"
	now := time.Now()
	run.StartedAt = &now
	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return nil, err
	}

	// Execute steps asynchronously with a 30-minute timeout.
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("diagnostic workflow panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
			}
		}()
		s.executeRun(runCtx, run, wf)
	}()

	s.logger.Info("diagnostic run approved", zap.Uint("run_id", runID), zap.Uint("wf_id", wf.ID))
	return run, nil
}

func (s *DiagnosticWorkflowService) GetRun(ctx context.Context, id uint) (*model.DiagnosticRun, error) {
	return s.repo.GetRun(ctx, id)
}

func (s *DiagnosticWorkflowService) ListRuns(ctx context.Context, workflowID *uint, incidentID *uint, status string, page, pageSize int) ([]model.DiagnosticRun, int64, error) {
	return s.repo.ListRuns(ctx, workflowID, incidentID, status, page, pageSize)
}

func (s *DiagnosticWorkflowService) ListRunSteps(ctx context.Context, runID uint) ([]model.DiagnosticRunStep, error) {
	return s.repo.ListRunSteps(ctx, runID)
}

// FindMatching finds workflows that match the given labels and severity.
func (s *DiagnosticWorkflowService) FindMatching(ctx context.Context, labels map[string]string, severity string) ([]model.DiagnosticWorkflow, error) {
	return s.repo.FindMatchingWorkflows(ctx, labels, severity)
}

// executeRun executes all steps in a diagnostic run.
func (s *DiagnosticWorkflowService) executeRun(ctx context.Context, run *model.DiagnosticRun, wf *model.DiagnosticWorkflow) {
	steps, err := s.repo.ListSteps(ctx, wf.ID)
	if err != nil {
		s.logger.Error("failed to load workflow steps", zap.Uint("wf_id", wf.ID), zap.Error(err))
		run.Status = "failed"
		run.ResultSummary = fmt.Sprintf("加载步骤失败: %v", err)
		now := time.Now()
		run.CompletedAt = &now
		if updateErr := s.repo.UpdateRun(ctx, run); updateErr != nil {
			s.logger.Error("failed to update run status to failed", zap.Uint("run_id", run.ID), zap.Error(updateErr))
		}
		return
	}

	for i, step := range steps {
		run.CurrentStep = i + 1
		if err := s.repo.UpdateRun(ctx, run); err != nil {
			s.logger.Error("failed to update run current step", zap.Uint("run_id", run.ID), zap.Error(err))
		}

		runStep := &model.DiagnosticRunStep{
			RunID:      run.ID,
			StepOrder:  step.StepOrder,
			StepName:   step.Name,
			StepType:   step.StepType,
			Expression: step.Expression,
			Status:     "running",
		}
		stepStart := time.Now()
		runStep.StartedAt = &stepStart
		if err := s.repo.CreateRunStep(ctx, runStep); err != nil {
			s.logger.Error("failed to create run step", zap.Uint("run_id", run.ID), zap.String("step_name", step.Name), zap.Error(err))
		}

		// Per-step timeout: 5 minutes max to prevent a single step from blocking the entire run.
		stepCtx, stepCancel := context.WithTimeout(ctx, 5*time.Minute)
		result, execErr := s.executeStep(stepCtx, &step, wf.TriggerLabels)
		stepCancel()
		stepEnd := time.Now()
		runStep.CompletedAt = &stepEnd
		runStep.DurationMs = stepEnd.Sub(stepStart).Milliseconds()

		if execErr != nil {
			runStep.Status = "failed"
			runStep.Error = execErr.Error()
			if err := s.repo.UpdateRunStep(ctx, runStep); err != nil {
				s.logger.Error("failed to update run step to failed", zap.Uint("run_id", run.ID), zap.String("step_name", step.Name), zap.Error(err))
			}

			if step.OnFailure == "abort" {
				run.Status = "failed"
				run.ResultSummary = fmt.Sprintf("步骤 %q 失败，已中止: %v", step.Name, execErr)
				now := time.Now()
				run.CompletedAt = &now
				if err := s.repo.UpdateRun(ctx, run); err != nil {
					s.logger.Error("failed to update run status to failed on abort", zap.Uint("run_id", run.ID), zap.Error(err))
				}
				return
			}
			// continue on failure
			continue
		}

		runStep.Result = truncateString(result, 5000)
		runStep.Status = "completed"
		if err := s.repo.UpdateRunStep(ctx, runStep); err != nil {
			s.logger.Error("failed to update run step to completed", zap.Uint("run_id", run.ID), zap.String("step_name", step.Name), zap.Error(err))
		}
	}

	run.Status = "completed"
	run.ResultSummary = fmt.Sprintf("诊断完成，共 %d 步", len(steps))
	now := time.Now()
	run.CompletedAt = &now
	if err := s.repo.UpdateRun(ctx, run); err != nil {
		s.logger.Error("failed to update run status to completed", zap.Uint("run_id", run.ID), zap.Error(err))
	}

	s.logger.Info("诊断工作流执行完成", zap.Uint("run_id", run.ID), zap.Uint("wf_id", wf.ID))
}

// executeStep executes a single diagnostic step.
// triggerLabels are the workflow's trigger labels, used by label_check to compare against.
func (s *DiagnosticWorkflowService) executeStep(ctx context.Context, step *model.DiagnosticWorkflowStep, triggerLabels model.JSONLabels) (string, error) {
	switch step.StepType {
	case "query":
		if step.DatasourceID == nil || step.Expression == "" {
			return "", fmt.Errorf("query step requires datasource_id and expression")
		}
		resp, err := s.dsSvc.QueryDatasource(ctx, *step.DatasourceID, step.Expression, time.Now())
		if err != nil {
			return "", fmt.Errorf("query failed: %w", err)
		}
		summary := fmt.Sprintf("%d 条时间序列", len(resp.Series))
		for i, series := range resp.Series {
			if i >= 3 {
				break
			}
			if len(series.Values) > 0 {
				last := series.Values[len(series.Values)-1]
				summary += fmt.Sprintf("\n  %v = %.4f", series.Labels, last.Value)
			}
		}
		return summary, nil

	case "label_check":
		// Parse expected labels from ConditionExpr (JSON: {"key":"value", ...})
		var expected model.JSONLabels
		if step.ConditionExpr != "" {
			if err := json.Unmarshal([]byte(step.ConditionExpr), &expected); err != nil {
				return "label_check: invalid condition_expr", fmt.Errorf("parse label_check condition_expr: %w", err)
			}
		}
		if len(expected) == 0 {
			return "label_check: no labels configured", nil
		}

		// Use trigger labels from the workflow context as actual labels.
		actual := triggerLabels
		if actual == nil {
			actual = model.JSONLabels{}
		}

		// Check each expected label against actual labels.
		var mismatches []string
		for k, exp := range expected {
			got, ok := actual[k]
			if !ok {
				mismatches = append(mismatches, fmt.Sprintf("%s: missing (expected %s)", k, exp))
			} else if got != exp {
				mismatches = append(mismatches, fmt.Sprintf("%s: got %s, expected %s", k, got, exp))
			}
		}

		if len(mismatches) > 0 {
			return fmt.Sprintf("label_check failed: %s", strings.Join(mismatches, "; ")), nil
		}
		return "label_check: all labels match", nil

	case "change_correlation":
		if s.changeEventSvc == nil {
			return "change_correlation: 变更事件服务未配置", nil
		}
		// Query change events in a 30-minute window around the run start
		windowEnd := time.Now()
		windowStart := windowEnd.Add(-30 * time.Minute)
		svcName := ""
		if v, ok := triggerLabels["service"]; ok {
			svcName = v
		}
		changes, err := s.changeEventSvc.FindByTimeWindow(ctx, svcName, windowStart, windowEnd)
		if err != nil {
			return "", fmt.Errorf("change_correlation query failed: %w", err)
		}
		if len(changes) == 0 {
			return "change_correlation: 未发现相关变更", nil
		}
		summary := fmt.Sprintf("change_correlation: 发现 %d 条相关变更", len(changes))
		for _, ch := range changes {
			summary += fmt.Sprintf("\n- [%s] %s (%s at %s)", ch.ChangeType, ch.Description, ch.Source, ch.Timestamp.Format(time.RFC3339))
		}
		return summary, nil

	case "metric_correlation":
		// Compares a metric across two time windows (e.g. current hour vs same hour yesterday)
		// to detect anomalous deviations. Uses the step's datasource_id and expression.
		// ConditionExpr: JSON {"current_window":"1h","baseline_window":"24h","threshold":0.5}
		if step.DatasourceID == nil || step.Expression == "" {
			return "", fmt.Errorf("metric_correlation step requires datasource_id and expression")
		}
		var corrCfg struct {
			CurrentWindow  string  `json:"current_window"`
			BaselineWindow string  `json:"baseline_window"`
			Threshold      float64 `json:"threshold"` // fractional change threshold (0.5 = 50%)
		}
		corrCfg.CurrentWindow = "1h"
		corrCfg.BaselineWindow = "24h"
		corrCfg.Threshold = 0.5
		if step.ConditionExpr != "" {
			if err := json.Unmarshal([]byte(step.ConditionExpr), &corrCfg); err != nil {
				return "", fmt.Errorf("parse metric_correlation condition_expr: %w", err)
			}
		}
		currentDur, err := time.ParseDuration(corrCfg.CurrentWindow)
		if err != nil {
			return "", fmt.Errorf("invalid current_window %q: %w", corrCfg.CurrentWindow, err)
		}
		baselineDur, err := time.ParseDuration(corrCfg.BaselineWindow)
		if err != nil {
			return "", fmt.Errorf("invalid baseline_window %q: %w", corrCfg.BaselineWindow, err)
		}
		now := time.Now()
		// Query current window
		currentResp, err := s.dsSvc.QueryDatasource(ctx, *step.DatasourceID, step.Expression, now)
		if err != nil {
			return "", fmt.Errorf("metric_correlation current window query failed: %w", err)
		}
		// Query baseline window
		baselineResp, err := s.dsSvc.QueryDatasource(ctx, *step.DatasourceID, step.Expression, now.Add(-baselineDur))
		if err != nil {
			return "", fmt.Errorf("metric_correlation baseline window query failed: %w", err)
		}
		summary := fmt.Sprintf("metric_correlation: current window=%s, baseline=%s, threshold=%.0f%%",
			corrCfg.CurrentWindow, corrCfg.BaselineWindow, corrCfg.Threshold*100)
		// Compare last values
		if len(currentResp.Series) > 0 && len(baselineResp.Series) > 0 {
			currentVal := 0.0
			if len(currentResp.Series[0].Values) > 0 {
				currentVal = currentResp.Series[0].Values[len(currentResp.Series[0].Values)-1].Value
			}
			baselineVal := 0.0
			if len(baselineResp.Series[0].Values) > 0 {
				baselineVal = baselineResp.Series[0].Values[len(baselineResp.Series[0].Values)-1].Value
			}
			change := 0.0
			if baselineVal != 0 {
				change = (currentVal - baselineVal) / baselineVal
			}
			summary += fmt.Sprintf("\ncurrent=%.4f, baseline=%.4f, change=%.1f%%", currentVal, baselineVal, change*100)
			if change > corrCfg.Threshold || change < -corrCfg.Threshold {
				summary += "\nANOMALY DETECTED: change exceeds threshold"
			} else {
				summary += "\nNo significant deviation detected"
			}
		} else {
			summary += "\nInsufficient data for comparison"
		}
		_ = currentDur // used for logging; actual time refs use now
		return summary, nil

	case "http_probe":
		// Checks endpoint availability via HTTP GET.
		// Expression is the URL to probe. ConditionExpr: JSON {"expected_status":200,"timeout":"10s"}
		if step.Expression == "" {
			return "", fmt.Errorf("http_probe step requires expression (URL)")
		}
		var probeCfg struct {
			ExpectedStatus int    `json:"expected_status"`
			Timeout        string `json:"timeout"`
		}
		probeCfg.ExpectedStatus = 200
		probeCfg.Timeout = "10s"
		if step.ConditionExpr != "" {
			if err := json.Unmarshal([]byte(step.ConditionExpr), &probeCfg); err != nil {
				return "", fmt.Errorf("parse http_probe condition_expr: %w", err)
			}
		}
		timeoutDur, err := time.ParseDuration(probeCfg.Timeout)
		if err != nil {
			timeoutDur = 10 * time.Second
		}
		probeCtx, probeCancel := context.WithTimeout(ctx, timeoutDur)
		defer probeCancel()
		req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, step.Expression, nil)
		if err != nil {
			return "", fmt.Errorf("http_probe create request: %w", err)
		}
		client := safehttp.NewSafeClient(timeoutDur)
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("http_probe: %s — UNREACHABLE (%v)", step.Expression, err), nil
		}
		defer func() { _ = resp.Body.Close() }()
		statusOK := resp.StatusCode == probeCfg.ExpectedStatus
		summary := fmt.Sprintf("http_probe: %s — HTTP %d (expected %d) — %s",
			step.Expression, resp.StatusCode, probeCfg.ExpectedStatus,
			map[bool]string{true: "OK", false: "MISMATCH"}[statusOK])
		return summary, nil

	default:
		return fmt.Sprintf("step type %q does not support auto-execution (supported: query, label_check, change_correlation, metric_correlation, http_probe)", step.StepType),
			fmt.Errorf("unsupported step type: %s", step.StepType)
	}
}
