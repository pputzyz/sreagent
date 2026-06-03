package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
	"github.com/sreagent/sreagent/internal/repository"
)

// InhibitionRuleService handles inhibition rule business logic.
type InhibitionRuleService struct {
	repo   *repository.InhibitionRuleRepository
	logger *zap.Logger

	// Cache for enabled inhibition rules — avoids DB round-trip on every alert evaluation.
	cacheMu  sync.RWMutex
	cache    []model.InhibitionRule
	cacheAt  time.Time
	cacheTTL time.Duration
}

// NewInhibitionRuleService creates a new InhibitionRuleService.
func NewInhibitionRuleService(repo *repository.InhibitionRuleRepository, logger *zap.Logger) *InhibitionRuleService {
	return &InhibitionRuleService{
		repo:     repo,
		logger:   logger,
		cacheTTL: 30 * time.Second,
	}
}

// listEnabledCached returns enabled inhibition rules from cache if fresh, otherwise reloads from DB.
func (s *InhibitionRuleService) listEnabledCached(ctx context.Context) ([]model.InhibitionRule, error) {
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cacheAt) < s.cacheTTL {
		result := make([]model.InhibitionRule, len(s.cache))
		copy(result, s.cache)
		s.cacheMu.RUnlock()
		return result, nil
	}
	s.cacheMu.RUnlock()

	// Double-check after acquiring write lock.
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache != nil && time.Since(s.cacheAt) < s.cacheTTL {
		result := make([]model.InhibitionRule, len(s.cache))
		copy(result, s.cache)
		return result, nil
	}

	rules, err := s.repo.FindAllEnabled(ctx)
	if err != nil {
		return nil, err
	}
	s.cache = rules
	s.cacheAt = time.Now()
	result := make([]model.InhibitionRule, len(rules))
	copy(result, rules)
	return result, nil
}

// InvalidateCache forces a reload on next access.
func (s *InhibitionRuleService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()
}

// validateLabelMap checks that a label map has non-empty keys and values.
func validateLabelMap(labels model.JSONLabels, fieldName string) error {
	if len(labels) == 0 {
		return nil
	}
	for k, v := range labels {
		if strings.TrimSpace(k) == "" {
			return apperr.WithMessage(apperr.ErrInvalidParam, fieldName+": label key must not be empty")
		}
		if strings.TrimSpace(v) == "" {
			return apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("%s: label %q has empty value", fieldName, k))
		}
	}
	return nil
}

// Create inserts a new inhibition rule.
func (s *InhibitionRuleService) Create(ctx context.Context, rule *model.InhibitionRule) error {
	if err := validateLabelMap(rule.SourceMatch, "source_match"); err != nil {
		return err
	}
	if err := validateLabelMap(rule.TargetMatch, "target_match"); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create inhibition rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.InvalidateCache()
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
	if err := validateLabelMap(rule.SourceMatch, "source_match"); err != nil {
		return err
	}
	if err := validateLabelMap(rule.TargetMatch, "target_match"); err != nil {
		return err
	}
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
	s.InvalidateCache()
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
	s.InvalidateCache()
	return nil
}

// IsInhibited returns true if event is suppressed by any enabled inhibition rule,
// given the set of currently-firing alert events (firingEvents).
func (s *InhibitionRuleService) IsInhibited(
	ctx context.Context,
	event *model.AlertEvent,
	firingEvents []model.AlertEvent,
) bool {
	rules, err := s.listEnabledCached(ctx)
	if err != nil {
		s.logger.Error("inhibition: failed to load rules", zap.Error(err))
		return false
	}
	for i := range rules {
		if matchesInhibition(&rules[i], event, firingEvents) {
			s.logger.Debug("alert inhibited",
				zap.String("alert_name", event.AlertName),
				zap.String("inhibition_rule", rules[i].Name),
			)
			return true
		}
	}
	return false
}

// MatchesInhibition returns true if the given rule would suppress the target event
// given the set of currently-firing events. Exported for handler-level preview use.
func (s *InhibitionRuleService) MatchesInhibition(rule *model.InhibitionRule, target *model.AlertEvent, firingEvents []model.AlertEvent) bool {
	return matchesInhibition(rule, target, firingEvents)
}

// InhibitionPreviewResult holds the preview result for a single inhibition rule.
type InhibitionPreviewResult struct {
	RuleID       uint              `json:"rule_id"`
	RuleName     string            `json:"rule_name"`
	TargetEvents []model.AlertEvent `json:"target_events"`
	SourceEvents []model.AlertEvent `json:"source_events"`
}

// Preview returns a list of enabled inhibition rules paired with the target events
// that would be suppressed and the source events that trigger the suppression.
func (s *InhibitionRuleService) Preview(ctx context.Context, firingEvents []model.AlertEvent) ([]InhibitionPreviewResult, error) {
	rules, err := s.listEnabledCached(ctx)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	var results []InhibitionPreviewResult
	for _, rule := range rules {
		result := InhibitionPreviewResult{
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			TargetEvents: []model.AlertEvent{},
			SourceEvents: []model.AlertEvent{},
		}
		seen := make(map[uint]bool) // avoid duplicate target entries
		for j := range firingEvents {
			target := &firingEvents[j]
			if seen[target.ID] {
				continue
			}
			src := findMatchingSource(&rule, target, firingEvents)
			if src != nil {
				result.TargetEvents = append(result.TargetEvents, *target)
				result.SourceEvents = append(result.SourceEvents, *src)
				seen[target.ID] = true
			}
		}
		if len(result.TargetEvents) > 0 {
			results = append(results, result)
		}
	}
	return results, nil
}

// findMatchingSource returns the first source event from firingEvents that causes
// the target to be suppressed by the given inhibition rule, or nil if no match.
// This is used by Preview to show the correct source event.
func findMatchingSource(rule *model.InhibitionRule, target *model.AlertEvent, firingEvents []model.AlertEvent) *model.AlertEvent {
	// Convert target labels to map[string]string for labelmatch.
	tgtLabels := make(map[string]string, len(target.Labels))
	for k, v := range target.Labels {
		tgtLabels[k] = v
	}

	// The target alert must match TargetMatch labels.
	if !labelmatch.Match(tgtLabels, rule.TargetMatch) {
		return nil
	}

	equalFields := parseEqualLabels(rule.EqualLabels)

	for i := range firingEvents {
		src := &firingEvents[i]
		if src.ID == target.ID {
			continue
		}
		if src.Status == model.EventStatusResolved || src.Status == model.EventStatusClosed {
			continue
		}

		srcLabels := make(map[string]string, len(src.Labels))
		for k, v := range src.Labels {
			srcLabels[k] = v
		}
		if !labelmatch.Match(srcLabels, rule.SourceMatch) {
			continue
		}

		if len(equalFields) > 0 {
			allEqual := true
			for _, lbl := range equalFields {
				srcVal, srcOK := src.Labels[lbl]
				tgtVal, tgtOK := target.Labels[lbl]
				if !srcOK || !tgtOK || srcVal != tgtVal {
					allEqual = false
					break
				}
			}
			if !allEqual {
				continue
			}
		}
		return src
	}
	return nil
}

// matchesInhibition returns true when the given inhibition rule causes event to be suppressed.
// Uses labelmatch.Match for full operator support (exact, !=, =~, !~).
func matchesInhibition(rule *model.InhibitionRule, target *model.AlertEvent, firingEvents []model.AlertEvent) bool {
	// Convert target labels to map[string]string for labelmatch.
	tgtLabels := make(map[string]string, len(target.Labels))
	for k, v := range target.Labels {
		tgtLabels[k] = v
	}

	// The target alert must match TargetMatch labels.
	if !labelmatch.Match(tgtLabels, rule.TargetMatch) {
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

		// Convert source labels to map[string]string for labelmatch.
		srcLabels := make(map[string]string, len(src.Labels))
		for k, v := range src.Labels {
			srcLabels[k] = v
		}
		if !labelmatch.Match(srcLabels, rule.SourceMatch) {
			continue
		}

		// If EqualLabels is specified, both source and target must have the label present
		// and have the same value. Labels missing on either side do NOT count as equal.
		if len(equalFields) > 0 {
			allEqual := true
			for _, lbl := range equalFields {
				srcVal, srcOK := src.Labels[lbl]
				tgtVal, tgtOK := target.Labels[lbl]
				if !srcOK || !tgtOK || srcVal != tgtVal {
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
