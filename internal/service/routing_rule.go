package service

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// RoutingRuleService wraps RoutingRuleRepository to maintain proper layering.
// Handlers should depend on this service, not directly on the repository.
type RoutingRuleService struct {
	repo *repository.RoutingRuleRepository
}

func NewRoutingRuleService(repo *repository.RoutingRuleRepository) *RoutingRuleService {
	return &RoutingRuleService{repo: repo}
}

func (s *RoutingRuleService) ListByIntegration(ctx context.Context, integrationID uint) ([]model.RoutingRule, error) {
	return s.repo.ListByIntegration(ctx, integrationID)
}

func (s *RoutingRuleService) GetByID(ctx context.Context, id uint) (*model.RoutingRule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RoutingRuleService) Create(ctx context.Context, rule *model.RoutingRule) error {
	return s.repo.Create(ctx, rule)
}

func (s *RoutingRuleService) Update(ctx context.Context, rule *model.RoutingRule) error {
	return s.repo.Update(ctx, rule)
}

func (s *RoutingRuleService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
