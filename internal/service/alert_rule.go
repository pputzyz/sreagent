package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type AlertRuleService struct {
	repo        *repository.AlertRuleRepository
	historyRepo *repository.AlertRuleHistoryRepository
	dsRepo      *repository.DataSourceRepository
	settingSvc  *SystemSettingService
	logger      *zap.Logger
}

func NewAlertRuleService(
	repo *repository.AlertRuleRepository,
	historyRepo *repository.AlertRuleHistoryRepository,
	dsRepo *repository.DataSourceRepository,
	logger *zap.Logger,
) *AlertRuleService {
	return &AlertRuleService{repo: repo, historyRepo: historyRepo, dsRepo: dsRepo, logger: logger}
}

// SetSystemSettingService injects the system setting service (called after construction
// to avoid circular dependency in DI wiring).
func (s *AlertRuleService) SetSystemSettingService(svc *SystemSettingService) {
	s.settingSvc = svc
}

// validLabelSeverities is the set of allowed label severity values.
var validLabelSeverities = map[string]bool{
	"critical": true,
	"warning":  true,
	"info":     true,
	"debug":    true,
}

// validateLabels checks that labels follow semantic conventions:
// - Required labels exist: severity, and either job or instance.
// - Label values are non-empty strings.
// - severity value is one of: critical, warning, info, debug.
// Returns nil if labels are empty (no labels = skip validation) or valid.
func (s *AlertRuleService) validateLabels(labels model.JSONLabels) error {
	if len(labels) == 0 {
		return nil
	}

	// Check that all label values are non-empty
	for k, v := range labels {
		if strings.TrimSpace(v) == "" {
			return apperr.WithMessage(apperr.ErrInvalidParam,
				fmt.Sprintf("label %q has empty value", k))
		}
	}

	// Required label: severity
	if sev, ok := labels["severity"]; ok {
		if !validLabelSeverities[strings.ToLower(strings.TrimSpace(sev))] {
			return apperr.WithMessage(apperr.ErrInvalidParam,
				fmt.Sprintf("label severity value %q is not allowed; must be one of: critical, warning, info, debug", sev))
		}
	} else {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			"label \"severity\" is required")
	}

	// Required label: job or instance (at least one)
	if _, hasJob := labels["job"]; !hasJob {
		if _, hasInstance := labels["instance"]; !hasInstance {
			return apperr.WithMessage(apperr.ErrInvalidParam,
				"label \"job\" or \"instance\" is required")
		}
	}

	return nil
}

func (s *AlertRuleService) Create(ctx context.Context, rule *model.AlertRule) error {
	// Validate labels if enabled
	if s.settingSvc != nil {
		if cfg, err := s.settingSvc.GetLabelValidationConfig(ctx); err == nil && cfg.Enabled {
			if err := s.validateLabels(rule.Labels); err != nil {
				return err
			}
		}
	}

	// Validate datasource: either a specific ID or a type must be provided
	if rule.DataSourceID != nil {
		if _, err := s.dsRepo.GetByID(ctx, *rule.DataSourceID); err != nil {
			return apperr.WithMessage(apperr.ErrDSNotFound, fmt.Sprintf("datasource ID %d not found", *rule.DataSourceID))
		}
	} else if rule.DatasourceType == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "either datasource_id or datasource_type must be provided")
	}

	rule.Version = 1
	if err := s.repo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create alert rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.recordHistory(ctx, rule, "created")
	return nil
}

func (s *AlertRuleService) GetByID(ctx context.Context, id uint) (*model.AlertRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrRuleNotFound
	}
	return rule, nil
}

func (s *AlertRuleService) List(ctx context.Context, severity, status, groupName, category string, page, pageSize int) ([]model.AlertRule, int64, error) {
	return s.repo.List(ctx, severity, status, groupName, category, page, pageSize)
}

// ListCategories returns all distinct non-empty category values.
func (s *AlertRuleService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.ListCategories(ctx)
}

func (s *AlertRuleService) Update(ctx context.Context, rule *model.AlertRule) error {
	// Validate labels if enabled
	if s.settingSvc != nil {
		if cfg, err := s.settingSvc.GetLabelValidationConfig(ctx); err == nil && cfg.Enabled {
			if err := s.validateLabels(rule.Labels); err != nil {
				return err
			}
		}
	}

	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return apperr.ErrRuleNotFound
	}

	existing.Name = rule.Name
	existing.DisplayName = rule.DisplayName
	existing.Description = rule.Description
	existing.DataSourceID = rule.DataSourceID
	existing.DatasourceType = rule.DatasourceType
	existing.Expression = rule.Expression
	existing.ForDuration = rule.ForDuration
	existing.Severity = rule.Severity
	existing.Labels = rule.Labels
	existing.Annotations = rule.Annotations
	existing.GroupName = rule.GroupName
	existing.Category = rule.Category
	existing.GroupWaitSeconds = rule.GroupWaitSeconds
	existing.GroupIntervalSeconds = rule.GroupIntervalSeconds
	existing.UpdatedBy = rule.UpdatedBy
	existing.EvalInterval = rule.EvalInterval
	existing.RecoveryHold = rule.RecoveryHold
	existing.NoDataEnabled = rule.NoDataEnabled
	existing.NoDataDuration = rule.NoDataDuration
	existing.SuppressEnabled = rule.SuppressEnabled
	existing.BizGroupID = rule.BizGroupID
	// Heartbeat / SLA fields
	existing.RuleType = rule.RuleType
	existing.HeartbeatToken = rule.HeartbeatToken
	existing.HeartbeatInterval = rule.HeartbeatInterval
	existing.AckSlaMinutes = rule.AckSlaMinutes
	existing.Version++

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update alert rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.recordHistory(ctx, existing, "updated")
	return nil
}

func (s *AlertRuleService) Delete(ctx context.Context, id uint) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrRuleNotFound
	}

	s.recordHistory(ctx, existing, "deleted")

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete alert rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// ImportRules batch-creates alert rules, returning success/failed counts and error details.
func (s *AlertRuleService) ImportRules(ctx context.Context, rules []model.AlertRule) (success, failed int, errors []string) {
	for i, rule := range rules {
		rule.Version = 1
		if err := s.repo.Create(ctx, &rule); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("rule #%d (%s): %v", i+1, rule.Name, err))
			s.logger.Error("failed to import alert rule",
				zap.String("name", rule.Name),
				zap.Error(err),
			)
		} else {
			success++
			s.recordHistory(ctx, &rule, "created")
		}
	}

	return
}

func (s *AlertRuleService) UpdateStatus(ctx context.Context, id uint, status model.AlertRuleStatus) error {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrRuleNotFound
	}

	rule.Status = status
	rule.Version++
	if err := s.repo.Update(ctx, rule); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	s.recordHistory(ctx, rule, "updated")
	return nil
}

// BatchEnable enables all rules in ids.
func (s *AlertRuleService) BatchEnable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ids must not be empty")
	}
	if err := s.repo.BatchUpdateStatus(ctx, ids, model.RuleStatusEnabled); err != nil {
		s.logger.Error("failed to batch enable alert rules", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// BatchDisable disables all rules in ids.
func (s *AlertRuleService) BatchDisable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ids must not be empty")
	}
	if err := s.repo.BatchUpdateStatus(ctx, ids, model.RuleStatusDisabled); err != nil {
		s.logger.Error("failed to batch disable alert rules", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// BatchDelete soft-deletes all rules in ids.
func (s *AlertRuleService) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ids must not be empty")
	}
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		s.logger.Error("failed to batch delete alert rules", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// RecordHeartbeatPing is called when a valid heartbeat token is received via
// POST /heartbeat/:token. It looks up the rule and updates HeartbeatLastAt.
func (s *AlertRuleService) RecordHeartbeatPing(ctx context.Context, token string) error {
	rule, err := s.repo.GetByHeartbeatToken(ctx, token)
	if err != nil {
		return apperr.ErrNotFound
	}
	now := time.Now()
	rule.HeartbeatLastAt = &now
	if err := s.repo.Update(ctx, rule); err != nil {
		s.logger.Error("failed to update heartbeat_last_at", zap.Uint("rule_id", rule.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.logger.Debug("heartbeat ping recorded", zap.String("rule_name", rule.Name), zap.Uint("rule_id", rule.ID))
	return nil
}

// recordHistory creates an audit trail entry for an alert rule change.
func (s *AlertRuleService) recordHistory(ctx context.Context, rule *model.AlertRule, changeType string) {
	if s.historyRepo == nil {
		return
	}

	snapshot, err := json.Marshal(rule)
	if err != nil {
		s.logger.Error("failed to marshal rule snapshot for history",
			zap.Uint("rule_id", rule.ID),
			zap.Error(err),
		)
		return
	}

	h := &model.AlertRuleHistory{
		RuleID:     rule.ID,
		Version:    rule.Version,
		ChangeType: changeType,
		Snapshot:   string(snapshot),
		ChangedBy:  rule.UpdatedBy,
	}
	// For create operations, ChangedBy comes from CreatedBy
	if changeType == "created" {
		h.ChangedBy = rule.CreatedBy
	}

	if err := s.historyRepo.Create(ctx, h); err != nil {
		s.logger.Error("failed to record alert rule history",
			zap.Uint("rule_id", rule.ID),
			zap.String("change_type", changeType),
			zap.Error(err),
		)
	}
}

// ListHistory returns paginated history records for a given rule.
func (s *AlertRuleService) ListHistory(ctx context.Context, ruleID uint, page, pageSize int) ([]model.AlertRuleHistory, int64, error) {
	if s.historyRepo == nil {
		return nil, 0, nil
	}
	return s.historyRepo.ListByRuleID(ctx, ruleID, page, pageSize)
}
