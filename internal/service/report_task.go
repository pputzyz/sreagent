package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

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
	return s.repo.CreateTask(ctx, task)
}

func (s *ReportTaskService) UpdateTask(ctx context.Context, task *model.ReportTask) error {
	return s.repo.UpdateTask(ctx, task)
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
