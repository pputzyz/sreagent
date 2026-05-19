package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

type UserNotificationService struct {
	repo   *repository.UserNotificationRepository
	logger *zap.Logger
}

func NewUserNotificationService(repo *repository.UserNotificationRepository, logger *zap.Logger) *UserNotificationService {
	return &UserNotificationService{repo: repo, logger: logger}
}

func (s *UserNotificationService) Create(ctx context.Context, userID uint, title, content string, ntype model.UserNotificationType, link string, metadata model.JSONLabels) (*model.UserNotification, error) {
	n := &model.UserNotification{
		UserID:   userID,
		Title:    title,
		Content:  content,
		Type:     ntype,
		Link:     link,
		Metadata: metadata,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *UserNotificationService) List(ctx context.Context, userID uint, isRead *bool, page, pageSize int) ([]model.UserNotification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, userID, isRead, page, pageSize)
}

func (s *UserNotificationService) MarkRead(ctx context.Context, id, userID uint) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *UserNotificationService) MarkAllRead(ctx context.Context, userID uint) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *UserNotificationService) CountUnread(ctx context.Context, userID uint) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *UserNotificationService) Delete(ctx context.Context, id, userID uint) error {
	return s.repo.Delete(ctx, id, userID)
}
