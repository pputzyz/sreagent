package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type UserNotificationRepository struct {
	db *gorm.DB
}

func NewUserNotificationRepository(db *gorm.DB) *UserNotificationRepository {
	return &UserNotificationRepository{db: db}
}

func (r *UserNotificationRepository) Create(ctx context.Context, n *model.UserNotification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *UserNotificationRepository) GetByID(ctx context.Context, id uint) (*model.UserNotification, error) {
	var n model.UserNotification
	if err := r.db.WithContext(ctx).First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *UserNotificationRepository) List(ctx context.Context, userID uint, isRead *bool, page, pageSize int) ([]model.UserNotification, int64, error) {
	var items []model.UserNotification
	var total int64

	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if isRead != nil {
		q = q.Where("is_read = ?", *isRead)
	}

	if err := q.Model(&model.UserNotification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *UserNotificationRepository) MarkRead(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (r *UserNotificationRepository) MarkAllRead(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.UserNotification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (r *UserNotificationRepository) CountUnread(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserNotification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

func (r *UserNotificationRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.UserNotification{}).Error
}
