package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// ChatHistoryRepository handles persistence for chat messages.
type ChatHistoryRepository struct {
	db *gorm.DB
}

// NewChatHistoryRepository creates a new ChatHistoryRepository.
func NewChatHistoryRepository(db *gorm.DB) *ChatHistoryRepository {
	return &ChatHistoryRepository{db: db}
}

// Create saves a new chat message.
func (r *ChatHistoryRepository) Create(ctx context.Context, msg *model.ChatHistory) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// ListByUserAndMode returns recent chat messages for a user and mode,
// in chronological order (oldest first).
func (r *ChatHistoryRepository) ListByUserAndMode(ctx context.Context, userID uint, mode string, limit int) ([]model.ChatHistory, error) {
	var msgs []model.ChatHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND mode = ?", userID, mode).
		Order("created_at ASC").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}

// DeleteByUserAndMode soft-deletes all chat messages for a user and mode.
func (r *ChatHistoryRepository) DeleteByUserAndMode(ctx context.Context, userID uint, mode string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND mode = ?", userID, mode).
		Delete(&model.ChatHistory{}).Error
}
