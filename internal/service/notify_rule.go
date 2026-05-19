package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"
)

// NotifyRuleService provides CRUD and event processing for notify rules.
type NotifyRuleService struct {
	ruleRepo     *repository.NotifyRuleRepository
	mediaRepo    *repository.NotifyMediaRepository
	templateRepo *repository.MessageTemplateRepository
	recordRepo   *repository.NotifyRecordRepository
	mediaSvc     *NotifyMediaService
	templateSvc  *MessageTemplateService
	pipeline     *AlertPipeline
	logger       *zap.Logger
}

// NewNotifyRuleService creates a new NotifyRuleService.
func NewNotifyRuleService(
	ruleRepo *repository.NotifyRuleRepository,
	mediaRepo *repository.NotifyMediaRepository,
	templateRepo *repository.MessageTemplateRepository,
	recordRepo *repository.NotifyRecordRepository,
	mediaSvc *NotifyMediaService,
	templateSvc *MessageTemplateService,
	pipeline *AlertPipeline,
	logger *zap.Logger,
) *NotifyRuleService {
	return &NotifyRuleService{
		ruleRepo:     ruleRepo,
		mediaRepo:    mediaRepo,
		templateRepo: templateRepo,
		recordRepo:   recordRepo,
		mediaSvc:     mediaSvc,
		templateSvc:  templateSvc,
		pipeline:     pipeline,
		logger:       logger,
	}
}

// Create creates a new notify rule.
func (s *NotifyRuleService) Create(ctx context.Context, rule *model.NotifyRule) error {
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
	existing.NotifyConfigs = rule.NotifyConfigs
	existing.RepeatInterval = rule.RepeatInterval
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
	if rule.Pipeline != "" {
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

	// Prepare template data
	templateData := EventToTemplateData(event, analysis)

	for _, nc := range notifyConfigs {
		// Filter by severity if specified in the notify config
		if nc.Severity != "" && nc.Severity != string(event.Severity) {
			continue
		}

		// Check throttle
		if s.isThrottled(ctx, rule, &nc) {
			s.logger.Debug("notification throttled",
				zap.Uint("event_id", event.ID),
				zap.Uint("rule_id", rule.ID),
				zap.Uint("media_id", nc.MediaID),
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
			s.createRecord(ctx, event.ID, media.ID, rule.ID, "failed", err.Error())
			continue
		}

		s.createRecord(ctx, event.ID, media.ID, rule.ID, "sent", "")
	}

	// 4. Fire callback if configured
	if rule.CallbackURL != "" {
		s.fireCallback(ctx, rule.CallbackURL, event, analysis)
	}

	return nil
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

// isThrottled checks whether a notification should be throttled based on the rule's repeat interval.
func (s *NotifyRuleService) isThrottled(ctx context.Context, rule *model.NotifyRule, nc *model.NotifyConfig) bool {
	if rule.RepeatInterval <= 0 {
		return false
	}

	// Check the last sent record for this media + rule combination
	lastRecord, err := s.recordRepo.GetLastSentRecord(ctx, nc.MediaID, rule.ID)
	if err != nil {
		// No previous record found, not throttled
		return false
	}

	elapsed := time.Since(lastRecord.CreatedAt)
	throttleDuration := time.Duration(rule.RepeatInterval) * time.Second
	return elapsed < throttleDuration
}

// createRecord creates a notification record for audit and tracking.
func (s *NotifyRuleService) createRecord(ctx context.Context, eventID, mediaID, ruleID uint, status, response string) {
	record := &model.NotifyRecord{
		EventID:   eventID,
		ChannelID: mediaID, // Reusing ChannelID field to store media ID
		PolicyID:  ruleID,  // Reusing PolicyID field to store rule ID
		Status:    status,
		Response:  response,
	}
	if err := s.recordRepo.Create(ctx, record); err != nil {
		s.logger.Error("failed to create notify record",
			zap.Uint("event_id", eventID),
			zap.Error(err),
		)
	}
}

// fireCallback sends a POST request to the configured callback URL.
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

	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, strings.NewReader(string(body)))
	if err != nil {
		s.logger.Error("failed to create callback request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := safehttp.NewSafeClient(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("callback request failed",
			zap.String("url", callbackURL),
			zap.Error(err),
		)
		return
	}
	defer resp.Body.Close()

	s.logger.Info("callback fired",
		zap.String("url", callbackURL),
		zap.Int("status", resp.StatusCode),
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
