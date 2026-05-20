package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type AIConversationRepository struct {
	db *gorm.DB
}

func NewAIConversationRepository(db *gorm.DB) *AIConversationRepository {
	return &AIConversationRepository{db: db}
}

func (r *AIConversationRepository) Create(ctx context.Context, conv *model.AIConversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *AIConversationRepository) GetByID(ctx context.Context, id uint) (*model.AIConversation, error) {
	var conv model.AIConversation
	err := r.db.WithContext(ctx).First(&conv, id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *AIConversationRepository) ListByUser(ctx context.Context, userID uint, page, pageSize int) ([]model.AIConversation, int64, error) {
	var list []model.AIConversation
	var total int64

	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if err := q.Model(&model.AIConversation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("updated_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *AIConversationRepository) Update(ctx context.Context, conv *model.AIConversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}

func (r *AIConversationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AIConversation{}, id).Error
}

// Tool calls

func (r *AIConversationRepository) CreateToolCall(ctx context.Context, call *model.AIToolCall) error {
	return r.db.WithContext(ctx).Create(call).Error
}

func (r *AIConversationRepository) ListToolCalls(ctx context.Context, conversationID uint) ([]model.AIToolCall, error) {
	var calls []model.AIToolCall
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("step_index ASC").
		Find(&calls).Error
	return calls, err
}

func (r *AIConversationRepository) UpdateToolCall(ctx context.Context, call *model.AIToolCall) error {
	return r.db.WithContext(ctx).Save(call).Error
}
