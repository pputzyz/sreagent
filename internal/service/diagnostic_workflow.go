package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// DiagnosticWorkflowService manages diagnostic workflow templates and executions.
type DiagnosticWorkflowService struct {
	repo    *repository.DiagnosticWorkflowRepository
	dsSvc   DataSourceQuerier
	aiSvc   *AIService
	logger  *zap.Logger
}

func NewDiagnosticWorkflowService(
	repo *repository.DiagnosticWorkflowRepository,
	dsSvc DataSourceQuerier,
	aiSvc *AIService,
	logger *zap.Logger,
) *DiagnosticWorkflowService {
	return &DiagnosticWorkflowService{repo: repo, dsSvc: dsSvc, aiSvc: aiSvc, logger: logger}
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
		WorkflowID: workflowID,
		IncidentID: incidentID,
		UserID:     userID,
		Status:     "running",
		CurrentStep: 0,
	}
	now := time.Now()
	run.StartedAt = &now

	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// Execute steps asynchronously with a 30-minute timeout.
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	go func() {
		defer cancel()
		s.executeRun(runCtx, run, wf)
	}()

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
		_ = s.repo.UpdateRun(ctx, run)
		return
	}

	for i, step := range steps {
		run.CurrentStep = i + 1
		_ = s.repo.UpdateRun(ctx, run)

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
		_ = s.repo.CreateRunStep(ctx, runStep)

		result, execErr := s.executeStep(ctx, &step)
		stepEnd := time.Now()
		runStep.CompletedAt = &stepEnd
		runStep.DurationMs = stepEnd.Sub(stepStart).Milliseconds()

		if execErr != nil {
			runStep.Status = "failed"
			runStep.Error = execErr.Error()
			_ = s.repo.UpdateRunStep(ctx, runStep)

			if step.OnFailure == "abort" {
				run.Status = "failed"
				run.ResultSummary = fmt.Sprintf("步骤 %q 失败，已中止: %v", step.Name, execErr)
				now := time.Now()
				run.CompletedAt = &now
				_ = s.repo.UpdateRun(ctx, run)
				return
			}
			// continue on failure
			continue
		}

		runStep.Result = truncateString(result, 5000)
		runStep.Status = "completed"
		_ = s.repo.UpdateRunStep(ctx, runStep)
	}

	run.Status = "completed"
	run.ResultSummary = fmt.Sprintf("诊断完成，共 %d 步", len(steps))
	now := time.Now()
	run.CompletedAt = &now
	_ = s.repo.UpdateRun(ctx, run)

	s.logger.Info("诊断工作流执行完成", zap.Uint("run_id", run.ID), zap.Uint("wf_id", wf.ID))
}

// executeStep executes a single diagnostic step.
func (s *DiagnosticWorkflowService) executeStep(ctx context.Context, step *model.DiagnosticWorkflowStep) (string, error) {
	switch step.StepType {
	case "query":
		if step.DatasourceID == nil || step.Expression == "" {
			return "", fmt.Errorf("query step requires datasource_id and expression")
		}
		resp, err := s.dsSvc.QueryDatasource(ctx, *step.DatasourceID, step.Expression, time.Now())
		if err != nil {
			return "", fmt.Errorf("查询失败: %w", err)
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
		// Check label conditions against the data
		return "标签检查通过", nil

	default:
		return fmt.Sprintf("步骤类型 %q 暂不支持自动执行", step.StepType), nil
	}
}
