package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// SubscribeRuleService provides CRUD and subscription matching for subscribe rules.
type SubscribeRuleService struct {
	repo   *repository.SubscribeRuleRepository
	logger *zap.Logger
}

// NewSubscribeRuleService creates a new SubscribeRuleService.
func NewSubscribeRuleService(
	repo *repository.SubscribeRuleRepository,
	logger *zap.Logger,
) *SubscribeRuleService {
	return &SubscribeRuleService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new subscribe rule.
func (s *SubscribeRuleService) Create(ctx context.Context, rule *model.SubscribeRule) error {
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create subscribe rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a subscribe rule by its ID.
func (s *SubscribeRuleService) GetByID(ctx context.Context, id uint) (*model.SubscribeRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrSubscribeRuleNotFound
	}
	return rule, nil
}

// List returns a paginated list of subscribe rules.
func (s *SubscribeRuleService) List(ctx context.Context, page, pageSize int) ([]model.SubscribeRule, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list subscribe rules", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing subscribe rule.
func (s *SubscribeRuleService) Update(ctx context.Context, rule *model.SubscribeRule) error {
	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return apperr.ErrSubscribeRuleNotFound
	}

	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.IsEnabled = rule.IsEnabled
	existing.MatchLabels = rule.MatchLabels
	existing.Severities = rule.Severities
	existing.TagFilters = rule.TagFilters
	existing.DatasourceIDs = rule.DatasourceIDs
	existing.RuleIDs = rule.RuleIDs
	existing.ForDuration = rule.ForDuration
	existing.NotifyRuleID = rule.NotifyRuleID

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update subscribe rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a subscribe rule by ID.
func (s *SubscribeRuleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrSubscribeRuleNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete subscribe rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// FindSubscriptions returns all enabled subscribe rules that match the given
// alert event across all filter dimensions. This is used to find which
// users/teams should receive notifications for a specific alert event.
func (s *SubscribeRuleService) FindSubscriptions(ctx context.Context, event *model.AlertEvent) ([]model.SubscribeRule, error) {
	matched, err := s.repo.FindMatchingSubscriptions(ctx, event)
	if err != nil {
		s.logger.Error("failed to find matching subscriptions",
			zap.Uint("event_id", event.ID),
			zap.Error(err),
		)
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return matched, nil
}
