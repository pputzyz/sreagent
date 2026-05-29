package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// StatusSubscriptionService wraps StatusSubscriptionRepository to maintain proper layering.
type StatusSubscriptionService struct {
	repo *repository.StatusSubscriptionRepository
}

func NewStatusSubscriptionService(repo *repository.StatusSubscriptionRepository) *StatusSubscriptionService {
	return &StatusSubscriptionService{repo: repo}
}

func (s *StatusSubscriptionService) Subscribe(ctx context.Context, email string) error {
	return s.repo.Subscribe(ctx, email)
}

func (s *StatusSubscriptionService) Unsubscribe(ctx context.Context, email string) error {
	return s.repo.Unsubscribe(ctx, email)
}

func (s *StatusSubscriptionService) List(ctx context.Context) ([]model.StatusSubscription, error) {
	return s.repo.List(ctx)
}
