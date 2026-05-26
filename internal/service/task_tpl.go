package service

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// TaskTplService provides business logic for task template management.
type TaskTplService struct {
	repo   *repository.TaskTplRepository
	logger *zap.Logger
}

// NewTaskTplService creates a new TaskTplService.
func NewTaskTplService(repo *repository.TaskTplRepository, logger *zap.Logger) *TaskTplService {
	return &TaskTplService{repo: repo, logger: logger}
}

// Create validates and creates a new task template.
func (s *TaskTplService) Create(ctx context.Context, tpl *model.TaskTpl) error {
	if err := s.validate(tpl); err != nil {
		return err
	}

	// Check name uniqueness
	existing, _ := s.repo.GetByName(ctx, tpl.Name)
	if existing != nil {
		return apperr.WithMessage(apperr.ErrConflict, "task template name already exists")
	}

	return s.repo.Create(ctx, tpl)
}

// GetByID retrieves a task template by ID.
func (s *TaskTplService) GetByID(ctx context.Context, id uint) (*model.TaskTpl, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return tpl, nil
}

// Update validates and updates a task template.
func (s *TaskTplService) Update(ctx context.Context, tpl *model.TaskTpl) error {
	if err := s.validate(tpl); err != nil {
		return err
	}

	// Check name uniqueness (excluding self)
	existing, _ := s.repo.GetByName(ctx, tpl.Name)
	if existing != nil && existing.ID != tpl.ID {
		return apperr.WithMessage(apperr.ErrConflict, "task template name already exists")
	}

	return s.repo.Update(ctx, tpl)
}

// Delete soft-deletes a task template.
func (s *TaskTplService) Delete(ctx context.Context, id uint) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.Wrap(apperr.ErrNotFound, err)
	}
	return s.repo.Delete(ctx, id)
}

// List returns a paginated list of task templates.
func (s *TaskTplService) List(ctx context.Context, keyword string, page, pageSize int) ([]model.TaskTpl, int64, error) {
	return s.repo.List(ctx, keyword, page, pageSize)
}

// validate checks the task template fields.
func (s *TaskTplService) validate(tpl *model.TaskTpl) error {
	if strings.TrimSpace(tpl.Name) == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "name is required")
	}
	if strings.TrimSpace(tpl.Script) == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "script is required")
	}
	if tpl.Batch < 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "batch must be non-negative")
	}
	if tpl.Tolerance < 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "tolerance must be non-negative")
	}
	if tpl.Timeout < 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "timeout must be non-negative")
	}
	if tpl.Timeout > 3600*24*5 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "timeout cannot exceed 5 days")
	}
	if tpl.Timeout == 0 {
		tpl.Timeout = 60
	}

	// Normalize line endings
	tpl.Script = strings.ReplaceAll(tpl.Script, "\r\n", "\n")

	// Validate hosts JSON if provided
	if tpl.Hosts != "" {
		var hosts []string
		if err := json.Unmarshal([]byte(tpl.Hosts), &hosts); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "hosts must be a valid JSON array")
		}
	}

	// Validate tags JSON if provided
	if tpl.Tags != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tpl.Tags), &tags); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "tags must be a valid JSON array")
		}
	}

	return nil
}
