package service

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// ExclusionRuleService manages channel exclusion rules (排除规则).
type ExclusionRuleService struct {
	repo   *repository.ExclusionRuleRepository
	logger *zap.Logger
}

func NewExclusionRuleService(repo *repository.ExclusionRuleRepository, logger *zap.Logger) *ExclusionRuleService {
	return &ExclusionRuleService{repo: repo, logger: logger}
}

func (s *ExclusionRuleService) Create(ctx context.Context, rule *model.ChannelExclusionRule) error {
	// Validate conditions JSON
	if rule.Conditions != "" {
		var conds []model.FilterCondition
		if err := json.Unmarshal([]byte(rule.Conditions), &conds); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid conditions JSON: "+err.Error())
		}
	}
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create exclusion rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *ExclusionRuleService) GetByID(ctx context.Context, id uint) (*model.ChannelExclusionRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return rule, nil
}

func (s *ExclusionRuleService) ListByChannel(ctx context.Context, channelID uint) ([]model.ChannelExclusionRule, error) {
	list, err := s.repo.ListByChannel(ctx, channelID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

func (s *ExclusionRuleService) Update(ctx context.Context, id uint, updates *model.ChannelExclusionRule) (*model.ChannelExclusionRule, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Conditions != "" {
		var conds []model.FilterCondition
		if err := json.Unmarshal([]byte(updates.Conditions), &conds); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid conditions JSON")
		}
		existing.Conditions = updates.Conditions
	}
	existing.IsEnabled = updates.IsEnabled
	existing.Priority = updates.Priority

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return existing, nil
}

func (s *ExclusionRuleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}
