package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

type TodoItemService struct {
	repo   *repository.TodoItemRepository
	logger *zap.Logger
}

func NewTodoItemService(repo *repository.TodoItemRepository, logger *zap.Logger) *TodoItemService {
	return &TodoItemService{repo: repo, logger: logger}
}

type CreateTodoRequest struct {
	Title       string             `json:"title" binding:"required"`
	Description string             `json:"description"`
	Type        string             `json:"type"`
	Priority    model.TodoPriority `json:"priority"`
	Link        string             `json:"link"`
	DueAt       *time.Time         `json:"due_at"`
}

func (s *TodoItemService) Create(ctx context.Context, userID uint, req *CreateTodoRequest) (*model.TodoItem, error) {
	item := &model.TodoItem{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Status:      model.TodoStatusPending,
		Priority:    req.Priority,
		Link:        req.Link,
		DueAt:       req.DueAt,
	}
	if item.Type == "" {
		item.Type = "manual"
	}
	if item.Priority == "" {
		item.Priority = model.TodoPriorityMedium
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TodoItemService) List(ctx context.Context, userID uint, status string, page, pageSize int) ([]model.TodoItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, userID, status, page, pageSize)
}

func (s *TodoItemService) Complete(ctx context.Context, id, userID uint) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	item.Status = model.TodoStatusCompleted
	now := time.Now()
	item.CompletedAt = &now
	return s.repo.Update(ctx, item)
}

func (s *TodoItemService) Update(ctx context.Context, id, userID uint, req *CreateTodoRequest) (*model.TodoItem, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Title = req.Title
	item.Description = req.Description
	item.Priority = req.Priority
	item.Link = req.Link
	item.DueAt = req.DueAt
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TodoItemService) Delete(ctx context.Context, id, userID uint) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *TodoItemService) CountPending(ctx context.Context, userID uint) (int64, error) {
	return s.repo.CountPending(ctx, userID)
}
