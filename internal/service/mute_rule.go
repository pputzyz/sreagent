package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/muterule"
	"github.com/sreagent/sreagent/internal/repository"
)

// MuteRuleService handles mute rule business logic.
type MuteRuleService struct {
	repo   *repository.MuteRuleRepository
	logger *zap.Logger

	// Cache for enabled mute rules — avoids DB round-trip on every alert evaluation.
	cacheMu  sync.RWMutex
	cache    []model.MuteRule
	cacheAt  time.Time
	cacheTTL time.Duration
}

// NewMuteRuleService creates a new MuteRuleService.
func NewMuteRuleService(repo *repository.MuteRuleRepository, logger *zap.Logger) *MuteRuleService {
	return &MuteRuleService{
		repo:     repo,
		logger:   logger,
		cacheTTL: 30 * time.Second,
	}
}

// listEnabledCached returns enabled mute rules from cache if fresh, otherwise reloads from DB.
func (s *MuteRuleService) listEnabledCached(ctx context.Context) ([]model.MuteRule, error) {
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cacheAt) < s.cacheTTL {
		result := make([]model.MuteRule, len(s.cache))
		copy(result, s.cache)
		s.cacheMu.RUnlock()
		return result, nil
	}
	s.cacheMu.RUnlock()

	// Double-check after acquiring write lock.
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache != nil && time.Since(s.cacheAt) < s.cacheTTL {
		result := make([]model.MuteRule, len(s.cache))
		copy(result, s.cache)
		return result, nil
	}

	rules, err := s.repo.FindAllEnabled(ctx)
	if err != nil {
		return nil, err
	}
	s.cache = rules
	s.cacheAt = time.Now()
	result := make([]model.MuteRule, len(rules))
	copy(result, rules)
	return result, nil
}

// InvalidateCache forces a reload on next access.
func (s *MuteRuleService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()
}

// LoadMuteTimezone loads a timezone with a consistent fallback to Asia/Shanghai.
// Delegates to muterule.LoadMuteTimezone — kept here for backward compatibility.
func LoadMuteTimezone(name string) *time.Location {
	return muterule.LoadMuteTimezone(name)
}

// validateDaysOfWeek checks that the DaysOfWeek CSV contains only values 1-7.
func validateDaysOfWeek(csv string) error {
	for _, d := range strings.Split(csv, ",") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		day, err := strconv.Atoi(d)
		if err != nil || day < 1 || day > 7 {
			return apperr.WithMessage(apperr.ErrInvalidParam, "days_of_week must be 1-7, got: "+d)
		}
	}
	return nil
}

// validateRuleIDs checks that the CSV contains only valid positive integers.
func validateRuleIDs(csv string) error {
	ids := strings.Split(csv, ",")
	for _, idStr := range ids {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id == 0 {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid rule_id: "+idStr)
		}
	}
	return nil
}

// Create creates a new mute rule.
func (s *MuteRuleService) Create(ctx context.Context, rule *model.MuteRule) error {
	if rule.RuleIDs != "" {
		if err := validateRuleIDs(rule.RuleIDs); err != nil {
			return err
		}
	}
	if rule.DaysOfWeek != "" {
		if err := validateDaysOfWeek(rule.DaysOfWeek); err != nil {
			return err
		}
	}
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create mute rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.InvalidateCache()
	return nil
}

// GetByID returns a mute rule by ID.
func (s *MuteRuleService) GetByID(ctx context.Context, id uint) (*model.MuteRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return rule, nil
}

// List returns a paginated list of mute rules.
func (s *MuteRuleService) List(ctx context.Context, page, pageSize int) ([]model.MuteRule, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}

// Update updates an existing mute rule.
func (s *MuteRuleService) Update(ctx context.Context, rule *model.MuteRule) error {
	if rule.RuleIDs != "" {
		if err := validateRuleIDs(rule.RuleIDs); err != nil {
			return err
		}
	}
	if rule.DaysOfWeek != "" {
		if err := validateDaysOfWeek(rule.DaysOfWeek); err != nil {
			return err
		}
	}
	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return apperr.ErrNotFound
	}

	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.MatchLabels = rule.MatchLabels
	existing.Severities = rule.Severities
	existing.StartTime = rule.StartTime
	existing.EndTime = rule.EndTime
	existing.PeriodicStart = rule.PeriodicStart
	existing.PeriodicEnd = rule.PeriodicEnd
	existing.DaysOfWeek = rule.DaysOfWeek
	existing.Timezone = rule.Timezone
	existing.IsEnabled = rule.IsEnabled
	existing.RuleIDs = rule.RuleIDs

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update mute rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.InvalidateCache()
	return nil
}

// Delete deletes a mute rule by ID.
func (s *MuteRuleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete mute rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.InvalidateCache()
	return nil
}

// IsAlertMuted checks whether an alert event should be muted based on active mute rules.
// It loads all enabled mute rules, checks label matching, time window, and severity filter.
// Returns true if ANY mute rule matches.
func (s *MuteRuleService) IsAlertMuted(ctx context.Context, event *model.AlertEvent) bool {
	rules, err := s.listEnabledCached(ctx)
	if err != nil {
		s.logger.Error("failed to load mute rules", zap.Error(err))
		return false
	}

	now := time.Now()

	for _, rule := range rules {
		if s.matchesRule(&rule, event, now) {
			s.logger.Info("alert muted by rule",
				zap.Uint("event_id", event.ID),
				zap.String("alert_name", event.AlertName),
				zap.Uint("mute_rule_id", rule.ID),
				zap.String("mute_rule_name", rule.Name),
			)
			return true
		}
	}

	return false
}

// MatchesRule checks if a single mute rule matches an alert event.
// Delegates to muterule.IsMutedByRule (shared with engine).
func (s *MuteRuleService) MatchesRule(rule *model.MuteRule, event *model.AlertEvent, now time.Time) bool {
	return muterule.IsMutedByRule(rule, map[string]string(event.Labels), string(event.Severity), event.RuleID, now)
}

// matchesRule is the internal alias kept for readability in IsAlertMuted.
func (s *MuteRuleService) matchesRule(rule *model.MuteRule, event *model.AlertEvent, now time.Time) bool {
	return s.MatchesRule(rule, event, now)
}

// BatchEnable enables multiple mute rules.
func (s *MuteRuleService) BatchEnable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.repo.BatchUpdateEnabled(ctx, ids, true); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// BatchDisable disables multiple mute rules.
func (s *MuteRuleService) BatchDisable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.repo.BatchUpdateEnabled(ctx, ids, false); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// BatchDelete soft-deletes multiple mute rules.
func (s *MuteRuleService) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}
