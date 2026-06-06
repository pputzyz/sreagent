package service

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// AlertRuleOperator is the interface consumed by cross-cutting services
// (ai_tools, rule_generator, heartbeat handler) to decouple from the concrete type.
type AlertRuleOperator interface {
	Create(ctx context.Context, rule *model.AlertRule, source string) error
	GetByID(ctx context.Context, id uint) (*model.AlertRule, error)
	List(ctx context.Context, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error)
	ListScoped(ctx context.Context, isAdmin bool, teamIDs []uint, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error)
	Update(ctx context.Context, rule *model.AlertRule) error
	Delete(ctx context.Context, id uint) error
	UpdateStatus(ctx context.Context, id uint, status model.AlertRuleStatus) error
	RecordHeartbeatPing(ctx context.Context, token string) error
}

// Compile-time check: *AlertRuleService satisfies AlertRuleOperator.
var _ AlertRuleOperator = (*AlertRuleService)(nil)

// generateSecureToken generates a cryptographically secure random token
// encoded as a URL-safe base64 string (n bytes of entropy).
func generateSecureToken(n int) string {
	b := make([]byte, n)
	if _, err := crypto_rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

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

func (s *AlertRuleService) Create(ctx context.Context, rule *model.AlertRule, source string) error {
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

	// Validate multi-query: max 2 queries (N-way join not yet implemented)
	if len(rule.Queries) > 2 {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			"multi-query currently supports max 2 queries (N-way join not yet implemented)")
	}

	// AI-generated rules start as draft and disabled until the user activates them.
	if source == "ai" {
		rule.Status = model.RuleStatusDraft
	}

	// Auto-generate a unique heartbeat token for all rules (required by unique index)
	if rule.HeartbeatToken == "" {
		rule.HeartbeatToken = generateSecureToken(32)
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

func (s *AlertRuleService) List(ctx context.Context, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error) {
	return s.repo.List(ctx, severity, status, groupName, category, keyword, datasourceID, page, pageSize)
}

// ListScoped returns paginated alert rules with team-level data isolation.
// If isAdmin is true, the regular List is called (no filtering).
// Otherwise, only rules belonging to the given teamIDs are returned.
// When teamIDs is empty for a non-admin user, an empty result is returned.
func (s *AlertRuleService) ListScoped(ctx context.Context, isAdmin bool, teamIDs []uint, severity, status, groupName, category, keyword string, datasourceID *uint, page, pageSize int) ([]model.AlertRule, int64, error) {
	if isAdmin {
		return s.repo.List(ctx, severity, status, groupName, category, keyword, datasourceID, page, pageSize)
	}
	if len(teamIDs) == 0 {
		// Non-admin user with no team membership — return empty result.
		return []model.AlertRule{}, 0, nil
	}
	return s.repo.ListByTeamIDs(ctx, teamIDs, severity, status, groupName, category, keyword, datasourceID, page, pageSize)
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

	// Validate datasource_id if it changed
	if rule.DataSourceID != nil && (existing.DataSourceID == nil || *rule.DataSourceID != *existing.DataSourceID) {
		if _, err := s.dsRepo.GetByID(ctx, *rule.DataSourceID); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("datasource ID %d not found", *rule.DataSourceID))
		}
	}

	// Validate multi-query: max 2 queries (N-way join not yet implemented)
	if len(rule.Queries) > 2 {
		return apperr.WithMessage(apperr.ErrInvalidParam,
			"multi-query currently supports max 2 queries (N-way join not yet implemented)")
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
	// Multi-query fields
	existing.Queries = rule.Queries
	existing.TriggerExp = rule.TriggerExp
	existing.JoinType = rule.JoinType
	existing.JoinKeys = rule.JoinKeys
	// Variable filling
	existing.VarConfig = rule.VarConfig
	// Ownership / Channel
	existing.TeamID = rule.TeamID
	existing.ChannelID = rule.ChannelID

	oldVersion := existing.Version
	existing.Version++

	ok, err := s.repo.UpdateVersion(ctx, existing, oldVersion)
	if err != nil {
		s.logger.Error("failed to update alert rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		s.logger.Warn("version conflict on alert rule update", zap.Uint("rule_id", existing.ID), zap.Int("expected_version", oldVersion))
		return apperr.ErrVersionConflict
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

// validateImportRule checks that an imported alert rule has required fields.
func validateImportRule(rule *model.AlertRule, index int) error {
	if rule.Name == "" {
		return fmt.Errorf("rule #%d: name is required", index)
	}
	if rule.Expression == "" {
		return fmt.Errorf("rule #%d (%s): expression is required", index, rule.Name)
	}
	if rule.Severity == "" {
		return fmt.Errorf("rule #%d (%s): severity is required", index, rule.Name)
	}
	if rule.DataSourceID == nil && rule.DatasourceType == "" {
		return fmt.Errorf("rule #%d (%s): either datasource_id or datasource_type is required", index, rule.Name)
	}
	return nil
}

// ImportRules batch-creates alert rules, returning success/failed counts and error details.
func (s *AlertRuleService) ImportRules(ctx context.Context, rules []model.AlertRule) (success, failed int, errors []string) {
	for i, rule := range rules {
		if err := validateImportRule(&rule, i+1); err != nil {
			failed++
			errors = append(errors, err.Error())
			continue
		}
		rule.Version = 1
		// Generate unique heartbeat_token if not set
		if rule.HeartbeatToken == "" {
			rule.HeartbeatToken = generateSecureToken(32)
		}
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

	oldVersion := rule.Version
	rule.Status = status
	rule.Version++

	ok, err := s.repo.UpdateVersion(ctx, rule, oldVersion)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !ok {
		return apperr.ErrVersionConflict
	}

	s.recordHistory(ctx, rule, "updated")
	return nil
}

// BatchEnable enables all rules in ids.
func (s *AlertRuleService) BatchEnable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ids must not be empty")
	}
	if err := s.repo.BatchUpdateStatus(ctx, ids, model.RuleStatusActive); err != nil {
		s.logger.Error("failed to batch enable alert rules", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	// Record history for each affected rule
	rules, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		s.logger.Error("failed to fetch rules for history recording", zap.Error(err))
		return nil // batch succeeded; history is best-effort
	}
	for i := range rules {
		s.recordHistory(ctx, &rules[i], "updated")
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
	// Record history for each affected rule
	rules, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		s.logger.Error("failed to fetch rules for history recording", zap.Error(err))
		return nil // batch succeeded; history is best-effort
	}
	for i := range rules {
		s.recordHistory(ctx, &rules[i], "updated")
	}
	return nil
}

// BatchDelete soft-deletes all rules in ids.
func (s *AlertRuleService) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return apperr.WithMessage(apperr.ErrInvalidParam, "ids must not be empty")
	}
	// Record history before deletion (snapshot captured while rule still exists)
	rules, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		s.logger.Error("failed to fetch rules for history recording before delete", zap.Error(err))
		// continue with deletion even if history fetch fails
	} else {
		for i := range rules {
			s.recordHistory(ctx, &rules[i], "deleted")
		}
	}
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		s.logger.Error("failed to batch delete alert rules", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// RecordHeartbeatPing is called when a valid heartbeat token is received via
// POST /heartbeat/:token. It looks up the rule and updates HeartbeatLastAt.
// Uses a targeted column update to avoid overwriting concurrent UI edits.
func (s *AlertRuleService) RecordHeartbeatPing(ctx context.Context, token string) error {
	rule, err := s.repo.GetByHeartbeatToken(ctx, token)
	if err != nil {
		return apperr.ErrNotFound
	}
	now := time.Now()
	if err := s.repo.UpdateHeartbeatLastAt(ctx, rule.ID, now); err != nil {
		s.logger.Error("failed to update heartbeat_last_at", zap.Uint("rule_id", rule.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.logger.Debug("heartbeat ping recorded", zap.String("rule_name", rule.Name), zap.Uint("rule_id", rule.ID))
	return nil
}

// GetHeartbeatToken returns the full heartbeat token for the given rule (admin-only).
// Returns an error if the rule is not found or is not a heartbeat-type rule.
func (s *AlertRuleService) GetHeartbeatToken(ctx context.Context, id uint) (string, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", apperr.ErrRuleNotFound
	}
	if rule.RuleType != model.RuleTypeHeartbeat {
		return "", apperr.WithMessage(apperr.ErrInvalidParam, "rule is not a heartbeat-type rule")
	}
	return rule.HeartbeatToken, nil
}

// recordHistory creates an audit trail entry for an alert rule change.
// The HeartbeatToken is masked in the snapshot to avoid leaking secrets.
func (s *AlertRuleService) recordHistory(ctx context.Context, rule *model.AlertRule, changeType string) {
	if s.historyRepo == nil {
		return
	}

	maskedRule := rule.MaskHeartbeatToken()
	snapshot, err := json.Marshal(maskedRule)
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

// LabelValidationResult holds preview results for the dry-run endpoint.
type LabelValidationResult struct {
	Total   int                     `json:"total"`
	Passing int                     `json:"passing"`
	Failing int                     `json:"failing"`
	Samples []LabelValidationSample `json:"samples"`
}

// LabelValidationSample describes a single rule's validation outcome.
type LabelValidationSample struct {
	RuleID   uint     `json:"rule_id"`
	RuleName string   `json:"rule_name"`
	Pass     bool     `json:"pass"`
	Issues   []string `json:"issues,omitempty"`
}

// PreviewLabelValidation checks all alert rules against label validation rules
// without modifying anything. Returns aggregate counts and up to `limit` failing samples.
// Fetches rules in batches of 1000 to avoid loading the entire table into memory at once.
func (s *AlertRuleService) PreviewLabelValidation(ctx context.Context, limit int) (*LabelValidationResult, error) {
	// If label validation is disabled, return empty result
	if s.settingSvc != nil {
		cfg, err := s.settingSvc.GetLabelValidationConfig(ctx)
		if err == nil && !cfg.Enabled {
			return &LabelValidationResult{Samples: []LabelValidationSample{}}, nil
		}
	}

	const batchSize = 1000
	result := &LabelValidationResult{
		Samples: []LabelValidationSample{},
	}

	for page := 1; ; page++ {
		rules, total, err := s.repo.List(ctx, "", "", "", "", "", nil, page, batchSize)
		if err != nil {
			s.logger.Error("failed to list alert rules for label validation preview", zap.Error(err))
			return nil, apperr.Wrap(apperr.ErrDatabase, err)
		}

		if page == 1 {
			result.Total = int(total)
		}

		for _, rule := range rules {
			err := s.validateLabels(rule.Labels)
			if err == nil {
				result.Passing++
			} else {
				result.Failing++
				if len(result.Samples) < limit {
					sample := LabelValidationSample{
						RuleID:   rule.ID,
						RuleName: rule.Name,
						Pass:     false,
						Issues:   []string{err.Error()},
					}
					result.Samples = append(result.Samples, sample)
				}
			}
		}

		// Stop when we've fetched all rules
		if len(rules) < batchSize {
			break
		}
	}

	return result, nil
}

// ListHistory returns paginated history records for a given rule.
func (s *AlertRuleService) ListHistory(ctx context.Context, ruleID uint, page, pageSize int) ([]model.AlertRuleHistory, int64, error) {
	if s.historyRepo == nil {
		return nil, 0, nil
	}
	return s.historyRepo.ListByRuleID(ctx, ruleID, page, pageSize)
}
