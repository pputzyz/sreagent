package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// templateRe is the pre-compiled regex for {{key}} placeholders.
var templateRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// DispatchService manages dispatch policies for collaboration channels.
type DispatchService struct {
	repo        *repository.DispatchPolicyRepository
	logRepo     *repository.DispatchLogRepository
	channelRepo *repository.ChannelRepository
	logger      *zap.Logger
}

func NewDispatchService(
	repo *repository.DispatchPolicyRepository,
	logRepo *repository.DispatchLogRepository,
	channelRepo *repository.ChannelRepository,
	logger *zap.Logger,
) *DispatchService {
	return &DispatchService{repo: repo, logRepo: logRepo, channelRepo: channelRepo, logger: logger}
}

// --- CRUD ---

func (s *DispatchService) Create(ctx context.Context, p *model.DispatchPolicy) error {
	if err := s.validatePolicy(p); err != nil {
		return err
	}
	// Validate channel existence.
	if _, err := s.channelRepo.GetByID(ctx, p.ChannelID); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "channel not found")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		s.logger.Error("failed to create dispatch policy", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *DispatchService) GetByID(ctx context.Context, id uint) (*model.DispatchPolicy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return p, nil
}

func (s *DispatchService) ListByChannel(ctx context.Context, channelID uint) ([]model.DispatchPolicy, error) {
	list, err := s.repo.ListByChannel(ctx, channelID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

func (s *DispatchService) Update(ctx context.Context, id uint, updates *model.DispatchPolicy) (*model.DispatchPolicy, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	// Validate the updated policy before saving
	if err := s.validatePolicy(updates); err != nil {
		return nil, err
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	// Description can be cleared to empty string
	existing.Description = updates.Description
	existing.IsEnabled = updates.IsEnabled
	existing.Priority = updates.Priority
	existing.MatchConditions = updates.MatchConditions
	existing.ActiveTimeConfig = updates.ActiveTimeConfig
	existing.DelaySeconds = updates.DelaySeconds
	existing.EscalationPolicyID = updates.EscalationPolicyID
	existing.RepeatIntervalSeconds = updates.RepeatIntervalSeconds
	existing.MaxRepeats = updates.MaxRepeats
	existing.NotifyMode = updates.NotifyMode
	existing.UnifiedMediaID = updates.UnifiedMediaID
	existing.LabelEnhancementRules = updates.LabelEnhancementRules

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return existing, nil
}

// ListLogsByIncident returns all dispatch logs for an incident.
func (s *DispatchService) ListLogsByIncident(ctx context.Context, incidentID uint) ([]model.DispatchLog, error) {
	list, err := s.logRepo.ListByIncident(ctx, incidentID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, nil
}

// CreateLog persists a dispatch log entry.
func (s *DispatchService) CreateLog(ctx context.Context, log *model.DispatchLog) error {
	return s.logRepo.Create(ctx, log)
}

func (s *DispatchService) Delete(ctx context.Context, id uint) error {
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

// --- Matching ---

// FindMatchingPolicy returns the highest-priority enabled policy for a channel
// that matches the given incident/alert labels. Returns nil if none match.
func (s *DispatchService) FindMatchingPolicy(
	ctx context.Context,
	channelID uint,
	labels model.JSONLabels,
	severity string,
) (*model.DispatchPolicy, error) {
	policies, err := s.repo.ListEnabledByChannel(ctx, channelID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	now := time.Now()
	for i := range policies {
		p := &policies[i]
		if !s.matchConditions(p.MatchConditions, labels, severity) {
			continue
		}
		if !s.isActiveNow(p.ActiveTimeConfig, now) {
			continue
		}
		return p, nil
	}
	return nil, nil
}

// matchConditions checks if a set of labels/severity matches the policy conditions.
func (s *DispatchService) matchConditions(condJSON string, labels model.JSONLabels, severity string) bool {
	if condJSON == "" || condJSON == "[]" || condJSON == "null" {
		return true // no conditions = match all
	}
	var conds []model.FilterCondition
	if err := json.Unmarshal([]byte(condJSON), &conds); err != nil {
		zap.L().Warn("dispatch: failed to parse match_conditions JSON, treating as no-match",
			zap.String("conditions", condJSON), zap.Error(err))
		return false // parse error = match none (fail-closed)
	}
	// Build a synthetic event-like map for matching
	for _, c := range conds {
		var actual string
		switch {
		case c.Field == "severity":
			actual = severity
		case strings.HasPrefix(c.Field, "labels."):
			actual = labels[strings.TrimPrefix(c.Field, "labels.")]
		default:
			actual = labels[c.Field]
		}
		if !evalDispatchCondition(c.Operator, actual, c.Value) {
			return false
		}
	}
	return true
}

func evalDispatchCondition(op, actual, expected string) bool {
	switch op {
	case "eq":
		return actual == expected
	case "ne":
		return actual != expected
	case "contains":
		return strings.Contains(actual, expected)
	case "not_contains":
		return !strings.Contains(actual, expected)
	case "regex":
		re, err := labelmatch.CompileRegex(expected)
		if err != nil {
			return false
		}
		return re.MatchString(actual)
	case "in":
		for _, v := range strings.Split(expected, ",") {
			if strings.TrimSpace(v) == actual {
				return true
			}
		}
		return false
	case "not_in":
		for _, v := range strings.Split(expected, ",") {
			if strings.TrimSpace(v) == actual {
				return false
			}
		}
		return true
	}
	return false // fail-closed: unknown operator should not match
}

// isActiveNow checks if a policy is active at the given time based on its time config.
func (s *DispatchService) isActiveNow(cfgJSON string, now time.Time) bool {
	if cfgJSON == "" || cfgJSON == "null" {
		return true // no time restriction
	}
	var cfg model.DispatchActiveTimeConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil || !cfg.Enabled {
		return true
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)

	// Check day of week
	if len(cfg.DaysOfWeek) > 0 {
		wd := int(local.Weekday())
		found := false
		for _, d := range cfg.DaysOfWeek {
			if d == wd {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range (supports overnight ranges like 22:00-06:00)
	if cfg.StartTime != "" && cfg.EndTime != "" {
		hhmm := fmt.Sprintf("%02d:%02d", local.Hour(), local.Minute())
		if cfg.StartTime <= cfg.EndTime {
			// Normal range: e.g. 09:00-18:00
			if hhmm < cfg.StartTime || hhmm >= cfg.EndTime {
				return false
			}
		} else {
			// Overnight range: e.g. 22:00-06:00 → active if >= 22:00 OR < 06:00
			if hhmm < cfg.StartTime && hhmm >= cfg.EndTime {
				return false
			}
		}
	}
	return true
}

// --- Label enhancement ---

// ApplyLabelEnhancements applies label enhancement rules to a copy of the labels.
func (s *DispatchService) ApplyLabelEnhancements(rulesJSON string, labels model.JSONLabels) model.JSONLabels {
	if rulesJSON == "" || rulesJSON == "[]" || rulesJSON == "null" {
		return labels
	}
	var rules []model.LabelEnhancementAction
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return labels
	}

	// Work on a copy
	result := make(model.JSONLabels, len(labels))
	for k, v := range labels {
		result[k] = v
	}

	for _, rule := range rules {
		// Check conditions
		if !s.matchConditions(conditionsFromFilterSlice(rule.Conditions), result, result["severity"]) {
			continue
		}
		switch rule.Type {
		case "set":
			if rule.SetKey != "" && (rule.Overwrite || result[rule.SetKey] == "") {
				result[rule.SetKey] = rule.SetValue
			}
		case "extract":
			if rule.SourceField != "" && rule.Regex != "" && rule.TargetLabel != "" {
				src := fieldValue(result, rule.SourceField)
				re, err := labelmatch.CompileRegex(rule.Regex)
				if err != nil {
					break
				}
				matches := re.FindStringSubmatch(src)
				if len(matches) > 1 && (rule.Overwrite || result[rule.TargetLabel] == "") {
					result[rule.TargetLabel] = matches[1]
				}
			}
		case "combine":
			if rule.TargetLabel != "" && rule.Template != "" {
				val := expandTemplate(rule.Template, result)
				if rule.Overwrite || result[rule.TargetLabel] == "" {
					result[rule.TargetLabel] = val
				}
			}
		case "map":
			if rule.MappingSourceLabel != "" && rule.TargetLabel != "" {
				src := result[rule.MappingSourceLabel]
				if mapped, ok := rule.MappingTable[src]; ok {
					if rule.Overwrite || result[rule.TargetLabel] == "" {
						result[rule.TargetLabel] = mapped
					}
				}
			}
		case "delete":
			if rule.DeleteKey != "" {
				delete(result, rule.DeleteKey)
			}
		}
	}
	return result
}

// helpers

func conditionsFromFilterSlice(conds []model.FilterCondition) string {
	if len(conds) == 0 {
		return ""
	}
	b, _ := json.Marshal(conds)
	return string(b)
}

func fieldValue(labels model.JSONLabels, field string) string {
	if strings.HasPrefix(field, "labels.") {
		return labels[strings.TrimPrefix(field, "labels.")]
	}
	return labels[field]
}

// expandTemplate replaces {{labels.xxx}} or {{xxx}} placeholders with label values.
func expandTemplate(tmpl string, labels model.JSONLabels) string {
	return templateRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := strings.Trim(m, "{}")
		key = strings.TrimPrefix(key, "labels.")
		if v, ok := labels[strings.TrimSpace(key)]; ok {
			return v
		}
		return m
	})
}

func (s *DispatchService) validatePolicy(p *model.DispatchPolicy) error {
	if p.DelaySeconds < 0 || p.DelaySeconds > 3600 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "delay_seconds must be 0-3600")
	}
	if p.RepeatIntervalSeconds < 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "repeat_interval_seconds must be >= 0")
	}
	// B7-13: Validate label_enhancement_rules is valid JSON array if provided.
	if p.LabelEnhancementRules != "" {
		var rules []model.LabelEnhancementAction
		if err := json.Unmarshal([]byte(p.LabelEnhancementRules), &rules); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam,
				fmt.Sprintf("label_enhancement_rules must be a valid JSON array: %v", err))
		}
	}
	return nil
}
