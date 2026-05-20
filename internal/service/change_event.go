package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// ChangeEventService manages CI/CD change events.
type ChangeEventService struct {
	repo   *repository.ChangeEventRepository
	logger *zap.Logger
}

func NewChangeEventService(repo *repository.ChangeEventRepository, logger *zap.Logger) *ChangeEventService {
	return &ChangeEventService{repo: repo, logger: logger}
}

func (s *ChangeEventService) Ingest(ctx context.Context, event *model.ChangeEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	return s.repo.Create(ctx, event)
}

func (s *ChangeEventService) GetByID(ctx context.Context, id uint) (*model.ChangeEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ChangeEventService) List(ctx context.Context, service, environment, source string, page, pageSize int) ([]model.ChangeEvent, int64, error) {
	return s.repo.List(ctx, service, environment, source, page, pageSize)
}

func (s *ChangeEventService) FindByTimeWindow(ctx context.Context, svc string, start, end time.Time) ([]model.ChangeEvent, error) {
	return s.repo.FindByTimeWindow(ctx, svc, start, end)
}

func (s *ChangeEventService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
