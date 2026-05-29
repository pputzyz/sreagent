package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// TaskRecordService wraps TaskRecordRepository to maintain proper layering.
type TaskRecordService struct {
	repo *repository.TaskRecordRepository
}

func NewTaskRecordService(repo *repository.TaskRecordRepository) *TaskRecordService {
	return &TaskRecordService{repo: repo}
}

func (s *TaskRecordService) ListRecords(ctx context.Context, tplID, eventID *uint, status *int, page, pageSize int) ([]model.TaskRecord, int64, error) {
	return s.repo.ListRecords(ctx, tplID, eventID, status, page, pageSize)
}

func (s *TaskRecordService) GetRecordByID(ctx context.Context, id uint) (*model.TaskRecord, error) {
	return s.repo.GetRecordByID(ctx, id)
}

func (s *TaskRecordService) ListHostRecords(ctx context.Context, recordID uint) ([]model.TaskHostRecord, error) {
	return s.repo.ListHostRecords(ctx, recordID)
}

func (s *TaskRecordService) GetHostRecordByID(ctx context.Context, id uint) (*model.TaskHostRecord, error) {
	return s.repo.GetHostRecordByID(ctx, id)
}
