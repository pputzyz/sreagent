package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

const defaultChatHistoryLimit = 50

// ChatHistoryService provides business logic for chat history management.
type ChatHistoryService struct {
	repo *repository.ChatHistoryRepository
}

// NewChatHistoryService creates a new ChatHistoryService.
func NewChatHistoryService(repo *repository.ChatHistoryRepository) *ChatHistoryService {
	return &ChatHistoryService{repo: repo}
}

// Save persists a chat message.
func (s *ChatHistoryService) Save(ctx context.Context, msg *model.ChatHistory) error {
	return s.repo.Create(ctx, msg)
}

// GetHistory returns recent chat messages for a user and mode.
func (s *ChatHistoryService) GetHistory(ctx context.Context, userID uint, mode string) ([]model.ChatHistory, error) {
	return s.repo.ListByUserAndMode(ctx, userID, mode, defaultChatHistoryLimit)
}

// ClearHistory deletes all chat messages for a user and mode.
func (s *ChatHistoryService) ClearHistory(ctx context.Context, userID uint, mode string) error {
	return s.repo.DeleteByUserAndMode(ctx, userID, mode)
}
