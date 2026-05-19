package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type TodoItemRepository struct {
	db *gorm.DB
}

func NewTodoItemRepository(db *gorm.DB) *TodoItemRepository {
	return &TodoItemRepository{db: db}
}

func (r *TodoItemRepository) Create(ctx context.Context, item *model.TodoItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *TodoItemRepository) GetByID(ctx context.Context, id uint) (*model.TodoItem, error) {
	var item model.TodoItem
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TodoItemRepository) List(ctx context.Context, userID uint, status string, page, pageSize int) ([]model.TodoItem, int64, error) {
	var items []model.TodoItem
	var total int64

	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Model(&model.TodoItem{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("FIELD(priority, 'high', 'medium', 'low'), created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *TodoItemRepository) Update(ctx context.Context, item *model.TodoItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *TodoItemRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.TodoItem{}).Error
}

func (r *TodoItemRepository) CountPending(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.TodoItem{}).
		Where("user_id = ? AND status = 'pending'", userID).
		Count(&count).Error
	return count, err
}
