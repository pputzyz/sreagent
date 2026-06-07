package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// EventPipelineService wraps EventPipelineRepository to maintain proper layering.
type EventPipelineService struct {
	repo *repository.EventPipelineRepository
}

func NewEventPipelineService(repo *repository.EventPipelineRepository) *EventPipelineService {
	return &EventPipelineService{repo: repo}
}

func (s *EventPipelineService) List(ctx context.Context, page, pageSize int, disabled *bool, query string) ([]model.EventPipeline, int64, error) {
	return s.repo.List(ctx, page, pageSize, disabled, query)
}

func (s *EventPipelineService) GetByID(ctx context.Context, id uint) (*model.EventPipeline, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EventPipelineService) Create(ctx context.Context, p *model.EventPipeline) error {
	// Ensure JSON fields have valid values (MySQL JSON columns don't accept empty strings)
	if p.NodesJSON == "" {
		p.NodesJSON = "[]"
	}
	if p.ProcessorsJSON == "" {
		p.ProcessorsJSON = "[]"
	}
	if p.LabelFiltersJSON == "" {
		p.LabelFiltersJSON = "{}"
	}
	return s.repo.Create(ctx, p)
}

func (s *EventPipelineService) Update(ctx context.Context, p *model.EventPipeline) error {
	return s.repo.Update(ctx, p)
}

func (s *EventPipelineService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// EventPipelineExecutionService wraps EventPipelineExecutionRepository.
type EventPipelineExecutionService struct {
	repo *repository.EventPipelineExecutionRepository
}

func NewEventPipelineExecutionService(repo *repository.EventPipelineExecutionRepository) *EventPipelineExecutionService {
	return &EventPipelineExecutionService{repo: repo}
}

func (s *EventPipelineExecutionService) ListByPipelineID(ctx context.Context, pipelineID uint, page, pageSize int) ([]model.EventPipelineExecution, int64, error) {
	return s.repo.ListByPipelineID(ctx, pipelineID, page, pageSize)
}

func (s *EventPipelineExecutionService) GetByID(ctx context.Context, id string) (*model.EventPipelineExecution, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EventPipelineExecutionService) CleanOlderThan(ctx context.Context, days int) (int64, error) {
	return s.repo.CleanOlderThan(ctx, days)
}
