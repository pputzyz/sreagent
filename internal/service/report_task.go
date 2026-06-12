package service

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// reportCronParser matches the scheduler's parser (cron.WithSeconds): SIX
// fields. Validating with anything else would accept expressions the
// scheduler later rejects silently (e.g. the standard 5-field form).
var reportCronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// validateReportTask rejects tasks the scheduler could not run.
func validateReportTask(task *model.ReportTask) error {
	if task.Name == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "name 不能为空")
	}
	if _, err := reportCronParser.Parse(task.CronExpr); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			fmt.Sprintf("cron 表达式无效（需要 6 段含秒，如 \"0 0 9 * * *\"）: %v", err))
	}
	switch task.ReportType {
	case "", "daily", "weekly", "custom":
	default:
		return apperr.WithMessage(apperr.ErrInvalidParam, "report_type 必须是 daily/weekly/custom")
	}
	return nil
}

// ReportTaskService wraps ReportTaskRepository to maintain proper layering.
type ReportTaskService struct {
	repo *repository.ReportTaskRepository
}

func NewReportTaskService(repo *repository.ReportTaskRepository) *ReportTaskService {
	return &ReportTaskService{repo: repo}
}

func (s *ReportTaskService) ListTasks(ctx context.Context, enabled *bool) ([]model.ReportTask, error) {
	return s.repo.ListTasks(ctx, enabled)
}

func (s *ReportTaskService) GetTask(ctx context.Context, id uint) (*model.ReportTask, error) {
	return s.repo.GetTask(ctx, id)
}

func (s *ReportTaskService) CreateTask(ctx context.Context, task *model.ReportTask) error {
	if err := validateReportTask(task); err != nil {
		return err
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *ReportTaskService) UpdateTask(ctx context.Context, task *model.ReportTask) error {
	if err := validateReportTask(task); err != nil {
		return err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *ReportTaskService) DeleteTask(ctx context.Context, id uint) error {
	return s.repo.DeleteTask(ctx, id)
}

func (s *ReportTaskService) ListRuns(ctx context.Context, taskID *uint, page, pageSize int) ([]model.ReportRun, int64, error) {
	return s.repo.ListRuns(ctx, taskID, page, pageSize)
}

func (s *ReportTaskService) GetRun(ctx context.Context, id uint) (*model.ReportRun, error) {
	return s.repo.GetRun(ctx, id)
}
