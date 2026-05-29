package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// InspectionService wraps InspectionRepository to maintain proper layering.
type InspectionService struct {
	repo *repository.InspectionRepository
}

func NewInspectionService(repo *repository.InspectionRepository) *InspectionService {
	return &InspectionService{repo: repo}
}

func (s *InspectionService) ListTasks(ctx context.Context, enabled *bool) ([]model.InspectionTask, error) {
	return s.repo.ListTasks(ctx, enabled)
}

func (s *InspectionService) GetTask(ctx context.Context, id uint) (*model.InspectionTask, error) {
	return s.repo.GetTask(ctx, id)
}

func (s *InspectionService) CreateTask(ctx context.Context, task *model.InspectionTask) error {
	return s.repo.CreateTask(ctx, task)
}

func (s *InspectionService) UpdateTask(ctx context.Context, task *model.InspectionTask) error {
	return s.repo.UpdateTask(ctx, task)
}

func (s *InspectionService) DeleteTask(ctx context.Context, id uint) error {
	return s.repo.DeleteTask(ctx, id)
}

func (s *InspectionService) ListRuns(ctx context.Context, taskID *uint, page, pageSize int) ([]model.InspectionRun, int64, error) {
	return s.repo.ListRuns(ctx, taskID, page, pageSize)
}

func (s *InspectionService) GetRun(ctx context.Context, id uint) (*model.InspectionRun, error) {
	return s.repo.GetRun(ctx, id)
}
