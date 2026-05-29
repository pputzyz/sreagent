package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// Incident aggregation architecture:
// The AlertV2Pipeline uses fingerprint-level incident aggregation via IncidentAggregator.
// Each unique alert fingerprint maps to exactly one Incident.
// The previous channel-level ensureIncident (which merged all alerts into one "bucket" incident)
// was removed in v4.52.0 because it caused unrelated alerts to be merged into a single incident.
//
// Key behaviors:
// - Same fingerprint -> same Incident (AlertCount incremented)
// - Different fingerprint -> different Incident
// - Resolution: when all events for a fingerprint resolve, Incident is closed
// - ChannelID: determined by the alert's _channel_id label or defaultChannelID

// v2PipelineDropped counts async tasks dropped due to full semaphore.
var v2PipelineDropped int64

// AlertV2Pipeline is the bridge between the legacy alert engine and the v2
// Alert → Incident data model. It is designed as a non-invasive hook:
// the existing engine continues to write AlertEvent records unchanged, while
// this pipeline simultaneously maintains the v2 tables.
//
// Call WrapOnAlert to wrap the existing onAlert callback:
//
//	wrapped := pipeline.WrapOnAlert(existingOnAlert)
//	evaluator.SetOnAlert(wrapped)
// maxAsyncPipelineTasks caps concurrent async v2 pipeline goroutines.
const maxAsyncPipelineTasks = 100

type AlertV2Pipeline struct {
	alertRepo    *repository.AlertRepository
	eventRepo    *repository.AlertEventRepository // for persisting EscalationPolicyID on AlertEvent
	incidentRepo *repository.IncidentRepository
	channelRepo  *repository.ChannelRepository
	logger       *zap.Logger
	dispatchSem  chan struct{}

	// defaultChannelID is the ID of the "default" collaboration channel.
	// All engine-fired alerts go here unless a specific channel is configured.
	defaultChannelID uint

	// noiseReducer applies noise-reduction rules before alert ingestion.
	// May be nil if not configured (degraded mode: no noise reduction).
	noiseReducer *NoiseReducer

	// dispatchSvc applies dispatch policy label enhancements.
	dispatchSvc *DispatchService

	// incidentAggregator bridges AlertEvent lifecycle to Incident management.
	// Optional — when set, called on firing/resolved events.
	incidentAggregator *IncidentAggregator
}

// NewAlertV2Pipeline creates a new pipeline. Call SetDefaultChannelID before use.
func NewAlertV2Pipeline(
	alertRepo *repository.AlertRepository,
	eventRepo *repository.AlertEventRepository,
	incidentRepo *repository.IncidentRepository,
	channelRepo *repository.ChannelRepository,
	logger *zap.Logger,
) *AlertV2Pipeline {
	return &AlertV2Pipeline{
		alertRepo:    alertRepo,
		eventRepo:    eventRepo,
		incidentRepo: incidentRepo,
		channelRepo:  channelRepo,
		logger:       logger,
		dispatchSem:  make(chan struct{}, maxAsyncPipelineTasks),
	}
}

// SetNoiseReducer attaches a NoiseReducer to the pipeline.
func (p *AlertV2Pipeline) SetNoiseReducer(nr *NoiseReducer) {
	p.noiseReducer = nr
}

// SetDispatchService attaches a DispatchService for label enhancement.
func (p *AlertV2Pipeline) SetDispatchService(svc *DispatchService) {
	p.dispatchSvc = svc
}

// SetIncidentAggregator attaches an IncidentAggregator for fingerprint-based incident tracking.
func (p *AlertV2Pipeline) SetIncidentAggregator(agg *IncidentAggregator) {
	p.incidentAggregator = agg
}

// SetDefaultChannelID sets the collaboration channel ID to route alerts to.
func (p *AlertV2Pipeline) SetDefaultChannelID(id uint) {
	p.defaultChannelID = id
}

// GetDefaultChannelID returns the default channel ID.
func (p *AlertV2Pipeline) GetDefaultChannelID() uint {
	return p.defaultChannelID
}

// InitDefaultChannel looks up (or creates) the default channel and caches its ID.
func (p *AlertV2Pipeline) InitDefaultChannel(ctx context.Context) {
	channels, err := p.channelRepo.ListActive(ctx)
	if err != nil {
		p.logger.Warn("alert_v2_pipeline: failed to list active channels", zap.Error(err))
		return
	}
	for _, ch := range channels {
		if ch.Name == "default" {
			p.defaultChannelID = ch.ID
			p.logger.Info("alert_v2_pipeline: using default channel",
				zap.Uint("channel_id", ch.ID))
			return
		}
	}
	p.logger.Warn("alert_v2_pipeline: 'default' channel not found, v2 incidents will not be created")
}

// WrapOnAlert wraps the existing onAlert callback with v2 pipeline logic.
// The original callback is still called first; v2 processing runs after.
func (p *AlertV2Pipeline) WrapOnAlert(
	original func(ctx context.Context, event *model.AlertEvent),
) func(ctx context.Context, event *model.AlertEvent) {
	return func(ctx context.Context, event *model.AlertEvent) {
		// 1. Run original callback (notification routing, mute check, etc.)
		if original != nil {
			original(ctx, event)
		}

		// 2. Drive v2 pipeline asynchronously — never block the original path
		select {
		case p.dispatchSem <- struct{}{}:
			go func() {
				defer func() { <-p.dispatchSem }()
				defer func() {
					if r := recover(); r != nil {
						p.logger.Error("alert_v2_pipeline: panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
					}
				}()
				pipeCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := p.process(pipeCtx, event); err != nil {
					p.logger.Error("alert_v2_pipeline: processing failed",
						zap.Uint("event_id", event.ID),
						zap.Error(err),
					)
				}
			}()
		default:
			total := atomic.AddInt64(&v2PipelineDropped, 1)
			p.logger.Error("dropping async v2 pipeline task, too many in flight",
				zap.Int("capacity", maxAsyncPipelineTasks),
				zap.Int64("total_dropped", total),
				zap.Uint("event_id", event.ID))
		}
	}
}

// process handles one AlertEvent: runs noise reduction, upserts Alert,
// then drives Incident lifecycle.
func (p *AlertV2Pipeline) process(ctx context.Context, event *model.AlertEvent) error {
	alertKey := p.buildAlertKey(event)
	severity := mapSeverity(event.Severity)

	// Determine target channel: prefer rule-level channel, fall back to default.
	channelID := p.defaultChannelID
	if event.RuleID != nil {
		// We check the Alert repo for a previously-created alert that already has channel set,
		// or fall back to default. Rule channel resolution happens at upsert time via ruleChannelID.
		// Here we pass the rule's potential channel via the alert labels["_channel_id"] hint
		// (set by the engine adapter below) or rely on default.
		if chStr, ok := event.Labels["_channel_id"]; ok && chStr != "" {
			var chID uint
			if _, err := fmt.Sscanf(chStr, "%d", &chID); err == nil && chID > 0 {
				channelID = chID
			}
		}
	}

	var eventStatus model.AlertEventV2Status
	if event.Status == model.EventStatusResolved || event.Status == model.EventStatusClosed {
		eventStatus = model.AlertEventV2StatusResolved
	} else {
		eventStatus = model.AlertEventV2StatusFiring
	}

	// 0. Noise reduction: exclusion rules, flapping, storm warning
	if p.noiseReducer != nil && channelID > 0 {
		nr := p.noiseReducer.Evaluate(ctx, channelID, alertKey, event)

		if nr.Excluded {
			p.logger.Info("alert_v2_pipeline: alert excluded by rule",
				zap.String("alert_key", alertKey),
				zap.String("reason", nr.ExcludeReason),
			)
			return nil // drop — do not upsert or create incident
		}

		if nr.StormWarning {
			p.logger.Warn("alert_v2_pipeline: storm warning",
				zap.Uint("channel_id", channelID),
				zap.Int("storm_level", nr.StormLevel),
			)
			// Storm warning is informational — still process the alert,
			// but the caller (future: notification service) can act on this.
		}

		if nr.Flapping {
			p.logger.Info("alert_v2_pipeline: alert is flapping",
				zap.String("alert_key", alertKey),
				zap.String("mode", nr.FlappingMode),
			)
			if nr.FlappingMode == "notify_then_silence" {
				// Silenced — drop from incident creation but still track the alert
				_, err := p.upsertAlert(ctx, alertKey, event, severity, eventStatus, channelID)
				if err != nil {
					return fmt.Errorf("upsert flapping alert: %w", err)
				}
				return nil // skip incident
			}
			// notify_only: fall through and process normally
		}

		// Record resolution for flap tracking
		if eventStatus == model.AlertEventV2StatusResolved {
			p.noiseReducer.RecordResolution(channelID, alertKey)
		}
	}

	// 1. Apply dispatch policy label enhancements (3.6)
	if p.dispatchSvc != nil && channelID > 0 && len(event.Labels) > 0 {
		policy, err := p.dispatchSvc.FindMatchingPolicy(ctx, channelID, model.JSONLabels(event.Labels), string(event.Severity))
		if err != nil {
			p.logger.Warn("failed to find matching dispatch policy", zap.Error(err), zap.Uint("channel_id", channelID))
		}
		if policy != nil {
			// Warn about unimplemented dispatch fields (delay/repeat/notify_mode)
			if policy.DelaySeconds > 0 {
				p.logger.Warn("dispatch policy delay_seconds not yet implemented, dispatching immediately",
					zap.Uint("policy_id", policy.ID), zap.Int("delay", policy.DelaySeconds))
			}
			if policy.RepeatIntervalSeconds > 0 {
				p.logger.Warn("dispatch policy repeat_interval not yet implemented",
					zap.Uint("policy_id", policy.ID), zap.Int("repeat_interval", policy.RepeatIntervalSeconds))
			}
			if policy.MaxRepeats > 0 {
				p.logger.Warn("dispatch policy max_repeats not yet implemented",
					zap.Uint("policy_id", policy.ID), zap.Int("max_repeats", policy.MaxRepeats))
			}
			if policy.NotifyMode == "unified" {
				p.logger.Warn("dispatch policy unified notify_mode not yet implemented, using personal_preference",
					zap.Uint("policy_id", policy.ID))
			}

			// Propagate the dispatch policy's escalation policy to the event
			// so the escalation executor can use it directly.
			if policy.EscalationPolicyID != nil {
				event.EscalationPolicyID = policy.EscalationPolicyID
				if p.eventRepo != nil {
					if err := p.eventRepo.UpdateEscalationPolicyID(ctx, event.ID, *policy.EscalationPolicyID); err != nil {
						p.logger.Warn("failed to persist escalation_policy_id on event",
							zap.Uint("event_id", event.ID), zap.Error(err))
					}
				}
			}
			if policy.LabelEnhancementRules != "" {
				enhanced := p.dispatchSvc.ApplyLabelEnhancements(policy.LabelEnhancementRules, model.JSONLabels(event.Labels))
				// Merge enhanced labels back into event (non-destructive to existing labels)
				if event.Labels == nil {
					event.Labels = make(model.JSONLabels)
				}
				for k, v := range enhanced {
					event.Labels[k] = v
				}
			}

			// Write dispatch log entry (Bug 4)
			if err := p.dispatchSvc.CreateLog(ctx, &model.DispatchLog{
				DispatchPolicyID: policy.ID,
				Status:           "applied",
				Attempt:          1,
				Note:             fmt.Sprintf("applied to event %d, channel %d", event.ID, channelID),
			}); err != nil {
				p.logger.Warn("failed to create dispatch log", zap.Error(err))
			}
		}
	}

	// 2. Upsert Alert record
	if _, err := p.upsertAlert(ctx, alertKey, event, severity, eventStatus, channelID); err != nil {
		return fmt.Errorf("upsert alert: %w", err)
	}

	// 3. Drive Incident lifecycle via fingerprint-level aggregator (canonical path).
	// Channel-level ensureIncident is removed — the aggregator handles all
	// incident creation, aggregation, and closing.
	if p.incidentAggregator != nil {
		if eventStatus == model.AlertEventV2StatusFiring {
			p.incidentAggregator.OnEventFired(ctx, event)
		} else {
			p.incidentAggregator.OnEventResolved(ctx, event)
		}
	}

	return nil
}

// upsertAlert finds or creates an Alert, then appends an AlertEventV2 record.
func (p *AlertV2Pipeline) upsertAlert(
	ctx context.Context,
	alertKey string,
	event *model.AlertEvent,
	severity model.AlertSeverity,
	status model.AlertEventV2Status,
	channelID uint,
) (*model.Alert, error) {
	now := time.Now()

	existing, err := p.alertRepo.GetByAlertKey(ctx, alertKey)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		// Update existing
		if status == model.AlertEventV2StatusFiring {
			if err := p.alertRepo.IncrementFireCount(ctx, existing.ID, now); err != nil {
				return nil, err
			}
			existing.Status = model.AlertStatusFiring
			existing.LastFiredAt = now
		} else {
			resolvedAt := now
			if err := p.alertRepo.UpdateStatus(ctx, existing.ID, model.AlertStatusResolved, &resolvedAt); err != nil {
				return nil, err
			}
			existing.Status = model.AlertStatusResolved
			existing.ResolvedAt = &resolvedAt
		}
	} else {
		// Create new alert
		ruleID := event.RuleID
		chID := channelID
		newAlert := &model.Alert{
			AlertKey:     alertKey,
			Title:        event.AlertName,
			Severity:     severity,
			Status:       model.AlertStatusFiring,
			RuleID:       ruleID,
			Labels:       event.Labels,
			Annotations:  event.Annotations,
			Source:       event.Source,
			GeneratorURL: event.GeneratorURL,
			FirstFiredAt: now,
			LastFiredAt:  now,
			EventCount:   1,
			FireCount:    1,
		}
		if chID > 0 {
			newAlert.ChannelID = &chID
		}
		if status == model.AlertEventV2StatusResolved {
			newAlert.Status = model.AlertStatusResolved
			newAlert.ResolvedAt = &now
		}
		if err := p.alertRepo.Create(ctx, newAlert); err != nil {
			return nil, err
		}
		existing = newAlert
	}

	// Append event record
	ev := &model.AlertEventV2{
		AlertID:       existing.ID,
		EventStatus:   status,
		EventSeverity: severity,
		Labels:        event.Labels,
		Annotations:   event.Annotations,
		Timestamp:     now,
		Fingerprint:   event.Fingerprint,
	}
	if err := p.alertRepo.CreateEvent(ctx, ev); err != nil {
		p.logger.Error("alert_v2_pipeline: failed to create event record",
			zap.Uint("alert_id", existing.ID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create event record: %w", err)
	}

	return existing, nil
}

// buildAlertKey generates a stable deduplication key for an alert event.
// Key = md5(rule_id + datasource_id + sorted(labels))
// Includes datasource_id to prevent cross-datasource key collisions
// when the same rule evaluates against multiple Prometheus instances.
func (p *AlertV2Pipeline) buildAlertKey(event *model.AlertEvent) string {
	var rulePrefix string
	if event.RuleID != nil {
		rulePrefix = fmt.Sprintf("rule:%d|", *event.RuleID)
	}
	if event.DataSourceID != nil {
		rulePrefix += fmt.Sprintf("ds:%d|", *event.DataSourceID)
	}

	keys := make([]string, 0, len(event.Labels))
	for k := range event.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(rulePrefix)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(event.Labels[k])
		b.WriteByte(',')
	}

	hash := md5.Sum([]byte(b.String()))
	return fmt.Sprintf("%x", hash)
}

// mapSeverity converts AlertSeverity to the v2 scale (critical/warning/info).
func mapSeverity(s model.AlertSeverity) model.AlertSeverity {
	switch s {
	case model.SeverityP0, model.SeverityP1, model.SeverityCritical:
		return model.SeverityCritical
	case model.SeverityP2, model.SeverityP3, model.SeverityWarning:
		return model.SeverityWarning
	default:
		return model.SeverityInfo
	}
}
