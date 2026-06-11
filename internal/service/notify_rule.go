package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	ppipeline "github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// NotifyRuleService provides CRUD and event processing for notify rules.
type NotifyRuleService struct {
	ruleRepo       *repository.NotifyRuleRepository
	mediaRepo      *repository.NotifyMediaRepository
	templateRepo   *repository.MessageTemplateRepository
	recordRepo     *repository.NotifyRecordRepository
	alertRuleRepo  *repository.AlertRuleRepository  // for loading AlertRule (template enrichment)
	dsRepo         *repository.DataSourceRepository // for loading DataSource name (template enrichment)
	mediaSvc       *NotifyMediaService
	templateSvc    *MessageTemplateService
	pipeline       *AlertPipeline
	pipelineEngine *ppipeline.Engine
	pipelineRepo   *repository.EventPipelineRepository
	dedupSvc       *NotificationDedupService
	// userNotifyConfigRepo and teamRepo enable UserIDs/TeamIDs dispatch in NotifyConfig.
	// Optional — when nil, UserIDs/TeamIDs are logged as warnings and skipped.
	userNotifyConfigRepo *repository.UserNotifyConfigRepository
	teamRepo             *repository.TeamRepository
	// inhibitionSvc is an optional inhibition check before dispatching notifications.
	inhibitionSvc *InhibitionRuleService
	// muteSvc is an optional mute rule check before dispatching notifications (B4-5).
	muteSvc *MuteRuleService
	// eventRepo is required by the inhibition check to fetch currently-firing events.
	eventRepo *repository.AlertEventRepository
	logger    *zap.Logger
}

// SetPipelineEngine injects the event pipeline engine and repository.
func (s *NotifyRuleService) SetPipelineEngine(engine *ppipeline.Engine, repo *repository.EventPipelineRepository) {
	s.pipelineEngine = engine
	s.pipelineRepo = repo
}

// SetDedupService injects the Redis-backed notification dedup service.
func (s *NotifyRuleService) SetDedupService(dedupSvc *NotificationDedupService) {
	s.dedupSvc = dedupSvc
}

// SetUserNotifyConfigRepo injects the user notify config repository for UserIDs/TeamIDs dispatch.
func (s *NotifyRuleService) SetUserNotifyConfigRepo(repo *repository.UserNotifyConfigRepository) {
	s.userNotifyConfigRepo = repo
}

// SetTeamRepo injects the team repository for expanding TeamIDs to members.
func (s *NotifyRuleService) SetTeamRepo(repo *repository.TeamRepository) {
	s.teamRepo = repo
}

// SetInhibitionService injects the inhibition rule service for pre-dispatch checks.
func (s *NotifyRuleService) SetInhibitionService(svc *InhibitionRuleService) {
	s.inhibitionSvc = svc
}

// SetMuteRuleService injects the mute rule service for pre-dispatch mute checks (B4-5).
func (s *NotifyRuleService) SetMuteRuleService(svc *MuteRuleService) {
	s.muteSvc = svc
}

// SetAlertEventRepository injects the event repository for inhibition checks.
func (s *NotifyRuleService) SetAlertEventRepository(repo *repository.AlertEventRepository) {
	s.eventRepo = repo
}

// NewNotifyRuleService creates a new NotifyRuleService.
func NewNotifyRuleService(
	ruleRepo *repository.NotifyRuleRepository,
	mediaRepo *repository.NotifyMediaRepository,
	templateRepo *repository.MessageTemplateRepository,
	recordRepo *repository.NotifyRecordRepository,
	alertRuleRepo *repository.AlertRuleRepository,
	dsRepo *repository.DataSourceRepository,
	mediaSvc *NotifyMediaService,
	templateSvc *MessageTemplateService,
	pipeline *AlertPipeline,
	dedupSvc *NotificationDedupService,
	logger *zap.Logger,
) *NotifyRuleService {
	return &NotifyRuleService{
		ruleRepo:      ruleRepo,
		mediaRepo:     mediaRepo,
		templateRepo:  templateRepo,
		recordRepo:    recordRepo,
		alertRuleRepo: alertRuleRepo,
		dsRepo:        dsRepo,
		mediaSvc:      mediaSvc,
		templateSvc:   templateSvc,
		pipeline:      pipeline,
		dedupSvc:      dedupSvc,
		logger:        logger,
	}
}

// FindMatchingRules returns all enabled notify rules whose match_labels are a
// subset of the event labels and whose severity filter matches.
// dataSourceID is resolved from the event's alert rule (nil = wildcard).
func (s *NotifyRuleService) FindMatchingRules(ctx context.Context, event *model.AlertEvent, dataSourceID *uint) ([]model.NotifyRule, error) {
	return s.ruleRepo.FindMatchingRules(ctx, map[string]string(event.Labels), string(event.Severity), dataSourceID)
}

// validateNotifyConfigs checks that the NotifyConfigs JSON is valid (if provided).
func validateNotifyConfigs(raw string) error {
	if raw == "" {
		return nil
	}
	var configs []model.NotifyConfig
	if err := json.Unmarshal([]byte(raw), &configs); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "notify_configs is not valid JSON: "+err.Error())
	}
	for i, nc := range configs {
		if nc.MediaID == 0 {
			return apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("notify_configs[%d]: media_id is required", i))
		}
	}
	return nil
}

// Create creates a new notify rule.
func (s *NotifyRuleService) Create(ctx context.Context, rule *model.NotifyRule) error {
	if err := validateNotifyConfigs(rule.NotifyConfigs); err != nil {
		return err
	}
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create notify rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a notify rule by its ID.
func (s *NotifyRuleService) GetByID(ctx context.Context, id uint) (*model.NotifyRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotifyRuleNotFound
	}
	return rule, nil
}

// List returns a paginated list of notify rules.
func (s *NotifyRuleService) List(ctx context.Context, page, pageSize int) ([]model.NotifyRule, int64, error) {
	list, total, err := s.ruleRepo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list notify rules", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Update updates an existing notify rule.
func (s *NotifyRuleService) Update(ctx context.Context, rule *model.NotifyRule) error {
	if err := validateNotifyConfigs(rule.NotifyConfigs); err != nil {
		return err
	}
	existing, err := s.ruleRepo.GetByID(ctx, rule.ID)
	if err != nil {
		return apperr.ErrNotifyRuleNotFound
	}

	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.IsEnabled = rule.IsEnabled
	existing.Severities = rule.Severities
	existing.MatchLabels = rule.MatchLabels
	existing.Pipeline = rule.Pipeline
	existing.PipelineID = rule.PipelineID
	existing.NotifyConfigs = rule.NotifyConfigs
	existing.RepeatInterval = rule.RepeatInterval
	existing.MaxNotifications = rule.MaxNotifications
	existing.CallbackURL = rule.CallbackURL

	if err := s.ruleRepo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update notify rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a notify rule by ID.
func (s *NotifyRuleService) Delete(ctx context.Context, id uint) error {
	if _, err := s.ruleRepo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotifyRuleNotFound
	}

	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete notify rule", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// ProcessEvent processes an alert event through a notify rule's pipeline
// and dispatches notifications via the configured media.
func (s *NotifyRuleService) ProcessEvent(ctx context.Context, event *model.AlertEvent, notifyRuleID uint) error {
	// NOTE: Inhibition check is performed upstream in NotificationService.RouteAlert
	// before calling ProcessEvent, so it is not duplicated here.

	// B4-5: Mute rule check — suppress notification if any active mute rule matches.
	if s.muteSvc != nil && s.muteSvc.IsAlertMuted(ctx, event) {
		s.logger.Info("notification suppressed by mute rule (notify_rule)",
			zap.Uint("event_id", event.ID),
			zap.String("alert_name", event.AlertName),
			zap.Uint("rule_id", notifyRuleID),
		)
		return nil
	}

	// 1. Load the notify rule
	rule, err := s.ruleRepo.GetByID(ctx, notifyRuleID)
	if err != nil {
		return apperr.ErrNotifyRuleNotFound
	}

	if !rule.IsEnabled {
		s.logger.Debug("skipping disabled notify rule",
			zap.Uint("rule_id", rule.ID),
			zap.String("rule_name", rule.Name),
		)
		return nil
	}

	// Check severity match
	if rule.Severities != "" {
		sevs := strings.Split(rule.Severities, ",")
		matched := false
		for _, sev := range sevs {
			if strings.TrimSpace(sev) == string(event.Severity) {
				matched = true
				break
			}
		}
		if !matched {
			s.logger.Debug("event severity does not match notify rule",
				zap.Uint("rule_id", rule.ID),
				zap.String("event_severity", string(event.Severity)),
				zap.String("rule_severities", rule.Severities),
			)
			return nil
		}
	}

	s.logger.Info("processing event through notify rule",
		zap.Uint("event_id", event.ID),
		zap.Uint("rule_id", rule.ID),
		zap.String("rule_name", rule.Name),
	)

	// 2. Run the event pipeline (relabel, AI summary, etc.)
	var analysis *AlertAnalysis

	// Priority: PipelineID (reusable pipeline) > inline Pipeline > default AI pipeline
	if rule.PipelineID != nil && *rule.PipelineID > 0 && s.pipelineEngine != nil && s.pipelineRepo != nil {
		pipelineObj, err := s.pipelineRepo.GetByID(ctx, *rule.PipelineID)
		if err != nil {
			s.logger.Warn("failed to load event pipeline, falling back to inline",
				zap.Uint("pipeline_id", *rule.PipelineID),
				zap.Error(err),
			)
		} else if !pipelineObj.Disabled {
			processedEvent, exec, execErr := s.pipelineEngine.Execute(ctx, pipelineObj, event, rule.Name)
			if execErr != nil {
				s.logger.Error("pipeline execution failed",
					zap.String("exec_id", exec.ID),
					zap.Error(execErr),
				)
			}
			if processedEvent == nil {
				s.logger.Info("event dropped by pipeline",
					zap.Uint("event_id", event.ID),
					zap.String("pipeline", pipelineObj.Name),
				)
				return nil // event dropped, skip notification
			}
			*event = *processedEvent
		}
	} else if rule.Pipeline != "" {
		analysis = s.runPipeline(ctx, event, rule.Pipeline)
	} else if s.pipeline != nil {
		// Default: run the standard AI pipeline
		analysis = s.pipeline.AnalyzeAlert(ctx, event)
	}

	// 3. Parse notify configs and dispatch notifications
	var notifyConfigs []model.NotifyConfig
	if rule.NotifyConfigs != "" {
		if err := json.Unmarshal([]byte(rule.NotifyConfigs), &notifyConfigs); err != nil {
			s.logger.Error("failed to parse notify_configs", zap.Error(err), zap.Uint("rule_id", rule.ID))
			return fmt.Errorf("invalid notify_configs: %w", err)
		}
	}

	if len(notifyConfigs) == 0 {
		s.logger.Debug("no notify configs defined for rule", zap.Uint("rule_id", rule.ID))
		return nil
	}

	// Prepare template data — load AlertRule and DataSource for richer template variables
	var alertRule *model.AlertRule
	var ds *model.DataSource
	if event.RuleID != nil && *event.RuleID > 0 && s.alertRuleRepo != nil {
		if r, err := s.alertRuleRepo.GetByID(ctx, *event.RuleID); err == nil {
			alertRule = r
			// Load datasource if the rule references one
			if r.DataSourceID != nil && *r.DataSourceID > 0 && s.dsRepo != nil {
				if d, err := s.dsRepo.GetByID(ctx, *r.DataSourceID); err == nil {
					ds = d
				}
			}
		} else {
			s.logger.Debug("failed to load alert rule for template enrichment",
				zap.Uint("rule_id", *event.RuleID),
				zap.Error(err),
			)
		}
	}
	templateData := EventToTemplateData(event, analysis, alertRule, ds)

	for _, nc := range notifyConfigs {
		// Filter by severity if specified in the notify config
		if nc.Severity != "" && nc.Severity != string(event.Severity) {
			continue
		}

		// Filter by time range (Nightingale-compatible)
		if !isInTimeRanges(time.Now(), nc.TimeRanges) {
			s.logger.Debug("notification outside time range",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
			)
			continue
		}

		// Check throttle
		if s.isThrottled(ctx, rule, &nc, event.Fingerprint) {
			s.logger.Debug("notification throttled",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
				zap.String("fingerprint", event.Fingerprint),
			)
			continue
		}

		// Dedup: skip if this event+media was already sent via either pipeline.
		// Include event.Status so firing and resolved are not deduped against each other (M1).
		// Include FireCount so a genuine re-fire (firing->resolved->firing) gets a new dedup key
		// and is not silently blocked by the 4h TTL of the previous fire cycle.
		dedupKey := BuildNotifyDedupKeyV2(event.ID, nc.MediaID, event.Fingerprint, string(event.Status), event.FireCount)
		if s.dedupSvc != nil && !s.dedupSvc.TrySend(ctx, dedupKey) {
			s.logger.Debug("v2 notification deduped (redis)",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
				zap.String("fingerprint", event.Fingerprint),
			)
			continue
		} else if s.dedupSvc == nil && !routeDedup.TrySend(dedupKey) {
			s.logger.Debug("v2 notification deduped (in-memory)",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
				zap.String("fingerprint", event.Fingerprint),
			)
			continue
		}

		// Load media
		media, err := s.mediaRepo.GetByID(ctx, nc.MediaID)
		if err != nil {
			s.logger.Error("failed to load notify media",
				zap.Uint("media_id", nc.MediaID),
				zap.Error(err),
			)
			continue
		}

		// Render template
		var renderedContent string
		if nc.TemplateID > 0 {
			rendered, err := s.templateSvc.RenderTemplate(ctx, nc.TemplateID, templateData)
			if err != nil {
				s.logger.Error("failed to render template",
					zap.Uint("template_id", nc.TemplateID),
					zap.Error(err),
				)
				// Fall back to a basic message
				renderedContent = fmt.Sprintf("[%s] %s - %s", event.Severity, event.AlertName, event.Status)
			} else {
				renderedContent = rendered
			}
		} else {
			// No template specified - use a basic message
			renderedContent = fmt.Sprintf("[%s] %s - %s", event.Severity, event.AlertName, event.Status)
		}

		// Send notification
		if err := s.mediaSvc.SendNotification(ctx, media, renderedContent, templateData); err != nil {
			s.logger.Error("failed to send notification via media",
				zap.Uint("event_id", event.ID),
				zap.Uint("media_id", media.ID),
				zap.String("media_name", media.Name),
				zap.Error(err),
			)
			s.createRecord(ctx, event.ID, media.ID, rule.ID, event.Fingerprint, "failed", err.Error())
			continue
		}

		s.createRecord(ctx, event.ID, media.ID, rule.ID, event.Fingerprint, "sent", "")

		// Dispatch to UserIDs and TeamIDs personal notification channels.
		if len(nc.UserIDs) > 0 || len(nc.TeamIDs) > 0 {
			s.dispatchToUsersAndTeams(ctx, event, rule, &nc, renderedContent, templateData)
		}
	}

	// 4. Fire callback if configured
	if rule.CallbackURL != "" {
		s.fireCallback(ctx, rule.CallbackURL, event, analysis)
	}

	return nil
}

// dispatchToUsersAndTeams sends notifications to users specified by UserIDs and
// to all members of teams specified by TeamIDs. Each user's personal
// UserNotifyConfig entries are used to determine which channels to send through.
// If the required repositories are not injected, a warning is logged and the
// dispatch is skipped.
//
// NOTE: TeamNotifyChannel (team-level notification media) is configured via
// TeamNotifyChannelService but is NOT consulted here. Currently, team dispatch
// expands TeamIDs to members and sends via each member's personal UserNotifyConfig.
// To honor team-level channels, query TeamNotifyChannelService.ListByTeam for each
// team and send through the team's configured media (especially IsDefault=true).
// This requires injecting TeamNotifyChannelService into NotifyRuleService.
func (s *NotifyRuleService) dispatchToUsersAndTeams(
	ctx context.Context,
	event *model.AlertEvent,
	rule *model.NotifyRule,
	nc *model.NotifyConfig,
	renderedContent string,
	templateData *TemplateData,
) {
	// Collect all target user IDs.
	allUserIDs := make(map[uint]struct{})
	for _, uid := range nc.UserIDs {
		allUserIDs[uid] = struct{}{}
	}

	// Expand TeamIDs to member user IDs.
	if len(nc.TeamIDs) > 0 && s.teamRepo != nil {
		for _, tid := range nc.TeamIDs {
			members, err := s.teamRepo.ListMembers(ctx, tid)
			if err != nil {
				s.logger.Warn("failed to list team members for UserIDs/TeamIDs dispatch",
					zap.Uint("team_id", tid), zap.Error(err))
				continue
			}
			for _, m := range members {
				allUserIDs[m.UserID] = struct{}{}
			}
		}
	} else if len(nc.TeamIDs) > 0 && s.teamRepo == nil {
		s.logger.Warn("TeamIDs specified but team repository not injected, skipping team expansion",
			zap.Uints("team_ids", nc.TeamIDs))
	}

	if len(allUserIDs) == 0 {
		return
	}

	// Look up each user's personal notification configs and send.
	if s.userNotifyConfigRepo == nil {
		s.logger.Warn("UserIDs/TeamIDs specified but user_notify_config repository not injected, skipping personal dispatch",
			zap.Any("user_ids", nc.UserIDs), zap.Any("team_ids", nc.TeamIDs))
		return
	}

	for uid := range allUserIDs {
		cfgs, err := s.userNotifyConfigRepo.ListByUserID(ctx, uid)
		if err != nil {
			s.logger.Warn("failed to load user notify config",
				zap.Uint("user_id", uid), zap.Error(err))
			continue
		}
		for _, cfg := range cfgs {
			if !cfg.IsEnabled {
				continue
			}
			// Send via the user's configured channel.
			// cfg.MediaType is like "lark_personal", "email", etc.
			// We send the rendered content directly — personal channels use the
			// same template already rendered for the rule's notify config.
			if err := s.mediaSvc.SendByUserConfig(ctx, &cfg, renderedContent, templateData); err != nil {
				s.logger.Error("failed to send personal notification",
					zap.Uint("user_id", uid),
					zap.String("media_type", cfg.MediaType),
					zap.Error(err),
				)
				s.createRecord(ctx, event.ID, 0, rule.ID, event.Fingerprint, "failed", err.Error())
			} else {
				s.createRecord(ctx, event.ID, 0, rule.ID, event.Fingerprint, "sent", "")
			}
		}
	}
}

// runPipeline executes the event processing pipeline defined in the rule.
func (s *NotifyRuleService) runPipeline(ctx context.Context, event *model.AlertEvent, pipelineJSON string) *AlertAnalysis {
	var steps []model.PipelineStep
	if err := json.Unmarshal([]byte(pipelineJSON), &steps); err != nil {
		s.logger.Error("failed to parse pipeline config", zap.Error(err))
		return nil
	}

	var analysis *AlertAnalysis
	for _, step := range steps {
		switch step.Type {
		case "ai_summary":
			if s.pipeline != nil {
				analysis = s.pipeline.AnalyzeAlert(ctx, event)
			}
		case "relabel":
			// Relabel step: modify event labels based on config
			s.applyRelabel(event, step.Config)
		default:
			s.logger.Warn("unknown pipeline step type", zap.String("type", step.Type))
		}
	}

	return analysis
}

// applyRelabel modifies event labels based on the relabel configuration.
func (s *NotifyRuleService) applyRelabel(event *model.AlertEvent, config map[string]interface{}) {
	if event.Labels == nil {
		event.Labels = make(model.JSONLabels)
	}

	// Support simple key-value additions/overrides
	if add, ok := config["add"].(map[string]interface{}); ok {
		for k, v := range add {
			if sv, ok := v.(string); ok {
				event.Labels[k] = sv
			}
		}
	}

	// Support label removal
	if remove, ok := config["remove"].([]interface{}); ok {
		for _, v := range remove {
			if sv, ok := v.(string); ok {
				delete(event.Labels, sv)
			}
		}
	}
}

// isInTimeRange checks whether the current time falls within any of the
// configured time ranges. If no time ranges are configured, it returns true
// (always allowed). This implements Nightingale-compatible time-range filtering.
func isInTimeRanges(now time.Time, ranges []model.TimeRange) bool {
	if len(ranges) == 0 {
		return true
	}

	weekday := int(now.Weekday()) // 0=Sunday in Go
	if weekday == 0 {
		weekday = 7 // convert to ISO: 7=Sunday
	}

	hour, min := now.Hour(), now.Minute()
	currentMinutes := hour*60 + min

	for _, tr := range ranges {
		// Check day-of-week filter
		if len(tr.Week) > 0 {
			dayMatch := false
			for _, d := range tr.Week {
				if d == weekday {
					dayMatch = true
					break
				}
			}
			if !dayMatch {
				continue
			}
		}

		// Parse start and end times
		startMin := parseHHMM(tr.Start)
		endMin := parseHHMM(tr.End)

		if startMin < 0 || endMin < 0 {
			continue // invalid time format, skip
		}

		// Handle same-day range (e.g. 09:00 - 18:00)
		if startMin <= endMin {
			if currentMinutes >= startMin && currentMinutes < endMin {
				return true
			}
		} else {
			// Overnight range (e.g. 22:00 - 06:00)
			if currentMinutes >= startMin || currentMinutes < endMin {
				return true
			}
		}
	}

	return false
}

// parseHHMM parses a "HH:MM" string into total minutes since midnight.
// Returns -1 if the format is invalid.
func parseHHMM(s string) int {
	if len(s) < 4 || len(s) > 5 {
		return -1
	}
	parts := splitHHMM(s)
	if len(parts) != 2 {
		return -1
	}
	h, m := parts[0], parts[1]
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

// splitHHMM splits "HH:MM" into [hour, minute].
func splitHHMM(s string) [2]int {
	idx := -1
	for i, c := range s {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx <= 0 || idx >= len(s)-1 {
		return [2]int{-1, -1}
	}
	h := parseIntSafe(s[:idx])
	m := parseIntSafe(s[idx+1:])
	return [2]int{h, m}
}

// parseIntSafe parses a string to int, returning -1 on error.
func parseIntSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// isThrottled checks whether a notification should be throttled based on the
// rule's max notification cap and repeat interval.
// fingerprint scopes both checks to the specific alert instance, preventing
// different alerts matching the same rule+media from blocking each other.
func (s *NotifyRuleService) isThrottled(ctx context.Context, rule *model.NotifyRule, nc *model.NotifyConfig, fingerprint string) bool {
	// Check max notification cap first (Nightingale NotifyMaxNumber pattern)
	// Scoped per fingerprint so one alert reaching its cap doesn't silence the rule for all alerts.
	if rule.MaxNotifications > 0 {
		count, err := s.recordRepo.CountSentByFingerprint(ctx, fingerprint, nc.MediaID, rule.ID)
		if err == nil && count >= rule.MaxNotifications {
			s.logger.Debug("notification throttled by max cap",
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
				zap.String("fingerprint", fingerprint),
				zap.Int("sent_count", count),
				zap.Int("max_notifications", rule.MaxNotifications),
			)
			return true
		}
	}

	// Then check repeat interval
	if rule.RepeatInterval <= 0 {
		return false
	}

	// Check the last sent record for this fingerprint + media + rule combination.
	// Scoped per fingerprint so the repeat interval only applies to the same alert,
	// not to different alerts that share the same rule+media.
	lastRecord, err := s.recordRepo.GetLastSentByFingerprint(ctx, fingerprint, nc.MediaID, rule.ID)
	if err != nil {
		// No previous record found, not throttled
		return false
	}

	elapsed := time.Since(lastRecord.CreatedAt)
	throttleDuration := time.Duration(rule.RepeatInterval) * time.Second
	return elapsed < throttleDuration
}

// createRecord creates a notification record for audit and tracking.
func (s *NotifyRuleService) createRecord(ctx context.Context, eventID, mediaID, ruleID uint, fingerprint, status, response string) {
	record := &model.NotifyRecord{
		EventID:     eventID,
		ChannelID:   mediaID, // NOTE: column named channel_id but stores the notify media ID
		PolicyID:    ruleID,  // NOTE: column named policy_id but stores the notify rule ID
		Fingerprint: fingerprint,
		Status:      status,
		Response:    response,
	}
	if err := s.recordRepo.Create(ctx, record); err != nil {
		s.logger.Error("failed to create notify record",
			zap.Uint("event_id", eventID),
			zap.Error(err),
		)
	}
}

// callbackMaxRetries is the number of retry attempts for callback HTTP POSTs.
// Total attempts = 1 (initial) + callbackMaxRetries. Backoff: 1s, 2s, 4s.
const callbackMaxRetries = 3

// fireCallback sends a POST request to the configured callback URL.
// Retries up to callbackMaxRetries times with exponential backoff on failure.
func (s *NotifyRuleService) fireCallback(ctx context.Context, callbackURL string, event *model.AlertEvent, analysis *AlertAnalysis) {
	payload := map[string]interface{}{
		"event_id":   event.ID,
		"alert_name": event.AlertName,
		"severity":   event.Severity,
		"status":     event.Status,
		"labels":     event.Labels,
		"fired_at":   event.FiredAt,
	}
	if analysis != nil {
		payload["ai_analysis"] = analysis
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal callback payload", zap.Error(err))
		return
	}

	// B5-8: HMAC-SHA256 signing for callback payload integrity verification.
	// If SREAGENT_WEBHOOK_SECRET is set, compute HMAC and attach X-Signature-256 header.
	// Callers should verify this header before processing the callback.
	computeSignature := func() string {
		if secret := os.Getenv("SREAGENT_WEBHOOK_SECRET"); secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			return "sha256=" + hex.EncodeToString(mac.Sum(nil))
		}
		return ""
	}

	client := safehttp.NewSafeClient(10 * time.Second)
	backoff := 1 * time.Second

	var lastErr error
	for attempt := 0; attempt <= callbackMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				s.logger.Warn("callback retry aborted: context cancelled",
					zap.String("url", callbackURL),
					zap.Int("attempt", attempt),
				)
				return
			case <-time.After(backoff):
				backoff *= 2
			}
			s.logger.Info("retrying callback",
				zap.String("url", callbackURL),
				zap.Int("attempt", attempt),
			)
		}

		req, reqErr := http.NewRequestWithContext(ctx, "POST", callbackURL, strings.NewReader(string(body)))
		if reqErr != nil {
			s.logger.Error("failed to create callback request", zap.Error(reqErr))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if sig := computeSignature(); sig != "" {
			req.Header.Set("X-Signature-256", sig)
		}

		resp, doErr := client.Do(req)
		if doErr != nil {
			lastErr = doErr
			s.logger.Error("callback request failed",
				zap.String("url", callbackURL),
				zap.Int("attempt", attempt),
				zap.Error(doErr),
			)
			continue
		}
		// Drain body to allow connection reuse (M6).
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server returned %d", resp.StatusCode)
			s.logger.Warn("callback returned server error, will retry",
				zap.String("url", callbackURL),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt),
			)
			continue
		}

		if resp.StatusCode >= 400 {
			s.logger.Warn("callback returned client error (not retried)",
				zap.String("url", callbackURL),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt),
			)
		}

		s.logger.Info("callback fired",
			zap.String("url", callbackURL),
			zap.Int("status", resp.StatusCode),
			zap.Int("attempt", attempt),
		)
		return
	}

	s.logger.Error("callback failed after all retries",
		zap.String("url", callbackURL),
		zap.Int("max_retries", callbackMaxRetries),
		zap.Error(lastErr),
	)
}

// BatchEnable enables multiple notify rules.
func (s *NotifyRuleService) BatchEnable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.ruleRepo.BatchUpdateEnabled(ctx, ids, true)
}

// BatchDisable disables multiple notify rules.
func (s *NotifyRuleService) BatchDisable(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.ruleRepo.BatchUpdateEnabled(ctx, ids, false)
}

// BatchDelete soft-deletes multiple notify rules.
func (s *NotifyRuleService) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.ruleRepo.BatchDelete(ctx, ids)
}

// TestRuleRequest is the payload for testing a notify rule.
type TestRuleRequest struct {
	// MediaID tests a specific media within the rule (0 = test all configured media).
	MediaID uint `json:"media_id"`
	// Custom test message (optional, defaults to a synthetic alert).
	AlertName string `json:"alert_name"`
	Severity  string `json:"severity"`
}

// TestRuleResult is the result of testing a single media within a rule.
type TestRuleResult struct {
	MediaID   uint   `json:"media_id"`
	MediaName string `json:"media_name"`
	Status    string `json:"status"` // "sent" or "failed"
	Error     string `json:"error,omitempty"`
}

// TestRule sends a test notification through a notify rule's configured media.
// If mediaID is specified, only that media is tested; otherwise all configured media are tested.
func (s *NotifyRuleService) TestRule(ctx context.Context, ruleID uint, req TestRuleRequest) ([]TestRuleResult, error) {
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, apperr.ErrNotifyRuleNotFound
	}

	// Parse notify configs
	var notifyConfigs []model.NotifyConfig
	if rule.NotifyConfigs != "" {
		if err := json.Unmarshal([]byte(rule.NotifyConfigs), &notifyConfigs); err != nil {
			return nil, fmt.Errorf("invalid notify_configs: %w", err)
		}
	}

	if len(notifyConfigs) == 0 {
		return nil, fmt.Errorf("no notify configs defined for rule %d", ruleID)
	}

	// Build a synthetic test event
	alertName := req.AlertName
	if alertName == "" {
		alertName = "SREAgent Test Alert"
	}
	severity := req.Severity
	if severity == "" {
		severity = "warning"
	}

	now := time.Now()
	testEvent := &model.AlertEvent{
		AlertName: alertName,
		Severity:  model.AlertSeverity(severity),
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"test": "true", "source": "sreagent-test"},
		FiredAt:   now,
		Source:    "sreagent-test",
	}

	templateData := EventToTemplateData(testEvent, nil, nil, nil)

	var results []TestRuleResult
	for _, nc := range notifyConfigs {
		// Filter by specific media if requested
		if req.MediaID > 0 && nc.MediaID != req.MediaID {
			continue
		}

		// Load media
		media, err := s.mediaRepo.GetByID(ctx, nc.MediaID)
		if err != nil {
			results = append(results, TestRuleResult{
				MediaID: nc.MediaID,
				Status:  "failed",
				Error:   fmt.Sprintf("media not found: %v", err),
			})
			continue
		}

		// Render template
		var renderedContent string
		if nc.TemplateID > 0 {
			rendered, err := s.templateSvc.RenderTemplate(ctx, nc.TemplateID, templateData)
			if err != nil {
				renderedContent = fmt.Sprintf("[%s] %s - test (template error: %v)", severity, alertName, err)
			} else {
				renderedContent = rendered
			}
		} else {
			renderedContent = fmt.Sprintf("[%s] %s - test notification", strings.ToUpper(severity), alertName)
		}

		// Send
		if err := s.mediaSvc.SendNotification(ctx, media, renderedContent, templateData); err != nil {
			results = append(results, TestRuleResult{
				MediaID:   nc.MediaID,
				MediaName: media.Name,
				Status:    "failed",
				Error:     err.Error(),
			})
		} else {
			results = append(results, TestRuleResult{
				MediaID:   nc.MediaID,
				MediaName: media.Name,
				Status:    "sent",
			})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no matching media found for test")
	}

	return results, nil
}

// BatchCreate creates multiple notify rules at once (Nightingale parity).
func (s *NotifyRuleService) BatchCreate(ctx context.Context, rules []*model.NotifyRule) error {
	for _, rule := range rules {
		if err := s.ruleRepo.Create(ctx, rule); err != nil {
			s.logger.Error("failed to create notify rule in batch", zap.Error(err), zap.String("name", rule.Name))
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
	}
	return nil
}

// ProcessEventBatch processes a batch of events through matching notify rules.
// For rules with GroupAggregate=true, it sends a single aggregated notification per media.
// For rules with GroupAggregate=false, it falls back to per-event processing.
// This is called by the AlertGroupManager when flushing a notification group.
func (s *NotifyRuleService) ProcessEventBatch(ctx context.Context, events []*model.AlertEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use the first event to find matching rules (grouped events share similar labels).
	representative := events[0]

	// Resolve datasource_id from the representative event's alert rule.
	var dataSourceID *uint
	if representative.RuleID != nil && *representative.RuleID > 0 && s.alertRuleRepo != nil {
		if r, err := s.alertRuleRepo.GetByID(ctx, *representative.RuleID); err == nil {
			dataSourceID = r.DataSourceID
		}
	}

	rules, err := s.FindMatchingRules(ctx, representative, dataSourceID)
	if err != nil {
		return fmt.Errorf("failed to find matching notify rules for batch: %w", err)
	}

	for _, rule := range rules {
		if rule.GroupAggregate {
			// Aggregated path: send one notification for the whole batch.
			if err := s.processAggregatedRule(ctx, &rule, events); err != nil {
				s.logger.Error("failed to process aggregated rule",
					zap.Uint("rule_id", rule.ID),
					zap.Int("event_count", len(events)),
					zap.Error(err),
				)
			}
		} else {
			// Per-event path: delegate to existing ProcessEvent for each event.
			for _, event := range events {
				eventCopy := shallowCopyEvent(event)
				if err := s.ProcessEvent(ctx, eventCopy, rule.ID); err != nil {
					s.logger.Error("failed to process event in batch (non-aggregated)",
						zap.Uint("event_id", event.ID),
						zap.Uint("rule_id", rule.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

// processAggregatedRule dispatches a single aggregated notification for all events
// in the batch through the given notify rule's configured media.
func (s *NotifyRuleService) processAggregatedRule(ctx context.Context, rule *model.NotifyRule, events []*model.AlertEvent) error {
	if !rule.IsEnabled {
		return nil
	}

	var notifyConfigs []model.NotifyConfig
	if rule.NotifyConfigs != "" {
		if err := json.Unmarshal([]byte(rule.NotifyConfigs), &notifyConfigs); err != nil {
			return fmt.Errorf("invalid notify_configs: %w", err)
		}
	}
	if len(notifyConfigs) == 0 {
		return nil
	}

	// Collect unique media IDs from the notify configs.
	mediaMap := make(map[uint]*model.NotifyMedia)
	for _, nc := range notifyConfigs {
		if nc.Severity != "" {
			// Check if any event matches this severity filter.
			matched := false
			for _, ev := range events {
				if nc.Severity == string(ev.Severity) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if !isInTimeRanges(time.Now(), nc.TimeRanges) {
			continue
		}

		if _, exists := mediaMap[nc.MediaID]; !exists {
			media, err := s.mediaRepo.GetByID(ctx, nc.MediaID)
			if err != nil {
				s.logger.Error("failed to load notify media for aggregated batch",
					zap.Uint("media_id", nc.MediaID),
					zap.Error(err),
				)
				continue
			}
			mediaMap[nc.MediaID] = media
		}
	}

	// Send one aggregated notification per unique media.
	for mediaID, media := range mediaMap {
		if err := s.mediaSvc.SendAggregatedLarkCard(ctx, media, events); err != nil {
			s.logger.Error("failed to send aggregated notification",
				zap.Uint("media_id", mediaID),
				zap.String("media_name", media.Name),
				zap.Int("event_count", len(events)),
				zap.Error(err),
			)
			// Create failure records for each event.
			for _, ev := range events {
				s.createRecord(ctx, ev.ID, mediaID, rule.ID, ev.Fingerprint, "failed", err.Error())
			}
			continue
		}

		s.logger.Info("aggregated notification sent",
			zap.Uint("media_id", mediaID),
			zap.String("media_name", media.Name),
			zap.Int("event_count", len(events)),
			zap.Uint("rule_id", rule.ID),
		)

		// Create success records for each event.
		for _, ev := range events {
			s.createRecord(ctx, ev.ID, mediaID, rule.ID, ev.Fingerprint, "sent", "")
		}
	}

	return nil
}
