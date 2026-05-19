package service

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// InhibitionRuleService handles inhibition rule business logic.
type InhibitionRuleService struct {
	repo   *repository.InhibitionRuleRepository
	logger *zap.Logger
}

// NewInhibitionRuleService creates a new InhibitionRuleService.
func NewInhibitionRuleService(repo *repository.InhibitionRuleRepository, logger *zap.Logger) *InhibitionRuleService {
	return &InhibitionRuleService{repo: repo, logger: logger}
}

// Create inserts a new inhibition rule.
func (s *InhibitionRuleService) Create(ctx context.Context, rule *model.InhibitionRule) error {
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create inhibition rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an inhibition rule by ID.
func (s *InhibitionRuleService) GetByID(ctx context.Context, id uint) (*model.InhibitionRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return rule, nil
}

// List returns a paginated list of inhibition rules.
func (s *InhibitionRuleService) List(ctx context.Context, page, pageSize int) ([]model.InhibitionRule, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}

// Update updates an existing inhibition rule.
func (s *InhibitionRuleService) Update(ctx context.Context, rule *model.InhibitionRule) error {
	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return apperr.ErrNotFound
	}
	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.SourceMatch = rule.SourceMatch
	existing.TargetMatch = rule.TargetMatch
	existing.EqualLabels = rule.EqualLabels
	existing.IsEnabled = rule.IsEnabled
	if err := s.repo.Update(ctx, existing); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete soft-deletes an inhibition rule.
func (s *InhibitionRuleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete inhibition rule", zap.Error(err), zap.Uint("id", id))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// IsInhibited returns true if event is suppressed by any enabled inhibition rule,
// given the set of currently-firing alert events (firingEvents).
func (s *InhibitionRuleService) IsInhibited(
	ctx context.Context,
	event *model.AlertEvent,
	firingEvents []model.AlertEvent,
) bool {
	rules, err := s.repo.FindAllEnabled(ctx)
	if err != nil {
		s.logger.Error("inhibition: failed to load rules", zap.Error(err))
		return false
	}
	for i := range rules {
		if matchesInhibition(&rules[i], event, firingEvents) {
			s.logger.Info("alert inhibited",
				zap.String("alert_name", event.AlertName),
				zap.String("inhibition_rule", rules[i].Name),
			)
			return true
		}
	}
	return false
}

// matchesInhibition returns true when the given inhibition rule causes event to be suppressed.
func matchesInhibition(rule *model.InhibitionRule, target *model.AlertEvent, firingEvents []model.AlertEvent) bool {
	// The target alert must match TargetMatch labels.
	if !inhibitionLabelsMatch(rule.TargetMatch, target.Labels) {
		return false
	}

	equalFields := parseEqualLabels(rule.EqualLabels)

	// Look for at least one firing source alert that matches SourceMatch.
	for i := range firingEvents {
		src := &firingEvents[i]
		if src.ID == target.ID {
			continue
		}
		if src.Status == model.EventStatusResolved || src.Status == model.EventStatusClosed {
			continue
		}
		if !inhibitionLabelsMatch(rule.SourceMatch, src.Labels) {
			continue
		}
		// If EqualLabels is specified, both source and target must have the same value for each.
		if len(equalFields) > 0 {
			allEqual := true
			for _, lbl := range equalFields {
				if src.Labels[lbl] != target.Labels[lbl] {
					allEqual = false
					break
				}
			}
			if !allEqual {
				continue
			}
		}
		return true
	}
	return false
}

// inhibitionLabelsMatch returns true when all entries in matchers are found in labels.
func inhibitionLabelsMatch(matchers model.JSONLabels, labels model.JSONLabels) bool {
	for k, v := range matchers {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// parseEqualLabels splits a comma-separated label name list.
func parseEqualLabels(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
