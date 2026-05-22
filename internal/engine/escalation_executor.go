package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
)

// EscalationExecutor periodically checks firing alert events and executes escalation steps
// when the configured delay has elapsed and the alert has not yet been resolved or acknowledged.
type EscalationExecutor struct {
	policyRepo           *repository.EscalationPolicyRepository
	stepRepo             *repository.EscalationStepRepository
	stepExecRepo         *repository.EscalationStepExecutionRepository
	eventRepo            *repository.AlertEventRepository
	ruleRepo             *repository.AlertRuleRepository // optional — used for SLA checks
	timelineRepo         *repository.AlertTimelineRepository
	channelRepo          *repository.NotifyChannelRepository
	userRepo             *repository.UserRepository
	notifyMediaSvc       *service.NotifyMediaService
	userNotifyConfigRepo *repository.UserNotifyConfigRepository
	teamRepo             service.TeamRepository
	onCallShiftRepo      *repository.OnCallShiftRepository
	larkSvc              *service.LarkService          // optional — enables lark_personal DM
	settingSvc           *service.SystemSettingService // optional — enables personal email via global SMTP
	logger               *zap.Logger

	interval  time.Duration
	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewEscalationExecutor creates a new EscalationExecutor.
func NewEscalationExecutor(
	policyRepo *repository.EscalationPolicyRepository,
	stepRepo *repository.EscalationStepRepository,
	stepExecRepo *repository.EscalationStepExecutionRepository,
	eventRepo *repository.AlertEventRepository,
	timelineRepo *repository.AlertTimelineRepository,
	channelRepo *repository.NotifyChannelRepository,
	userRepo *repository.UserRepository,
	notifyMediaSvc *service.NotifyMediaService,
	userNotifyConfigRepo *repository.UserNotifyConfigRepository,
	teamRepo service.TeamRepository,
	onCallShiftRepo *repository.OnCallShiftRepository,
	larkSvc *service.LarkService,
	settingSvc *service.SystemSettingService,
	ruleRepo *repository.AlertRuleRepository,
	logger *zap.Logger,
) *EscalationExecutor {
	return &EscalationExecutor{
		policyRepo:           policyRepo,
		stepRepo:             stepRepo,
		stepExecRepo:         stepExecRepo,
		eventRepo:            eventRepo,
		ruleRepo:             ruleRepo,
		timelineRepo:         timelineRepo,
		channelRepo:          channelRepo,
		userRepo:             userRepo,
		notifyMediaSvc:       notifyMediaSvc,
		userNotifyConfigRepo: userNotifyConfigRepo,
		teamRepo:             teamRepo,
		onCallShiftRepo:      onCallShiftRepo,
		larkSvc:              larkSvc,
		settingSvc:           settingSvc,
		logger:               logger,
		interval:             60 * time.Second,
		stopCh:               make(chan struct{}),
	}
}

// SetInterval overrides the default 60-second check interval.
func (e *EscalationExecutor) SetInterval(d time.Duration) {
	if d > 0 {
		e.interval = d
	}
}

// sendViaChannel adapts a v1 NotifyChannel to the v2 NotifyMediaService dispatch.
func (e *EscalationExecutor) sendViaChannel(ctx context.Context, event *model.AlertEvent, channel *model.NotifyChannel) error {
	mediaType := mapChannelTypeToMediaType(channel.Type)
	media := &model.NotifyMedia{
		Name:      channel.Name,
		Type:      mediaType,
		Config:    channel.Config,
		IsEnabled: true,
	}
	data := &service.TemplateData{
		AlertName:   event.AlertName,
		Severity:    string(event.Severity),
		Status:      string(event.Status),
		Labels:      map[string]string(event.Labels),
		Annotations: map[string]string(event.Annotations),
		FiredAt:     event.FiredAt,
		EventID:     event.ID,
		Source:      event.Source,
	}
	rendered := fmt.Sprintf("[%s] %s - %s", event.Severity, event.AlertName, event.Status)
	return e.notifyMediaSvc.SendNotification(ctx, media, rendered, data)
}

// mapChannelTypeToMediaType maps v1 NotifyChannelType to v2 NotifyMediaType.
func mapChannelTypeToMediaType(ct model.NotifyChannelType) model.NotifyMediaType {
	switch ct {
	case model.ChannelTypeLarkWebhook:
		return model.MediaTypeLarkWebhook
	case model.ChannelTypeEmail:
		return model.MediaTypeEmail
	case model.ChannelTypeCustom:
		return model.MediaTypeHTTP
	default:
		return model.MediaTypeHTTP
	}
}

// Start runs the escalation check loop in a background goroutine.
func (e *EscalationExecutor) Start() {
	e.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(e.interval)
			defer ticker.Stop()
			e.logger.Info("escalation executor started", zap.Duration("interval", e.interval))
			for {
				select {
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					e.runOnce(ctx)
					cancel()
				case <-e.stopCh:
					e.logger.Info("escalation executor stopped")
					return
				}
			}
		}()
	})
}

// Stop signals the background goroutine to exit.
func (e *EscalationExecutor) Stop() {
	e.stopOnce.Do(func() { close(e.stopCh) })
}

// runOnce performs a single escalation check pass.
// Uses cursor pagination to avoid loading all firing events at once.
func (e *EscalationExecutor) runOnce(ctx context.Context) {
	const pageSize = 500
	var afterID uint

	policies, err := e.policyRepo.ListAllEnabled(ctx)
	if err != nil {
		e.logger.Error("escalation: failed to list enabled policies", zap.Error(err))
		return
	}

	// Build teamID → policies map for H1 matching.
	teamPolicies := make(map[uint][]model.EscalationPolicy)
	var globalPolicies []model.EscalationPolicy
	for _, p := range policies {
		if p.TeamID > 0 {
			teamPolicies[p.TeamID] = append(teamPolicies[p.TeamID], p)
		} else {
			globalPolicies = append(globalPolicies, p)
		}
	}

	// Batch-load all steps for all enabled policies (H3).
	var allSteps map[uint][]model.EscalationStep
	if e.stepRepo != nil {
		policyIDs := make([]uint, 0, len(policies))
		for _, p := range policies {
			policyIDs = append(policyIDs, p.ID)
		}
		allSteps, err = e.stepRepo.BatchLoadByPolicyIDs(ctx, policyIDs)
		if err != nil {
			e.logger.Error("escalation: failed to batch-load steps", zap.Error(err))
			return
		}
	}

	now := time.Now()
	for {
		events, err := e.eventRepo.ListFiringForEscalation(ctx, afterID, pageSize)
		if err != nil {
			e.logger.Error("escalation: failed to list events", zap.Error(err))
			return
		}
		if len(events) == 0 {
			break
		}

		ruleMap := e.batchLoadRules(ctx, events)

		// Group events by team for concurrent processing (H3).
		type teamBatch struct {
			teamID uint
			events []*model.AlertEvent
		}
		teamBatches := make(map[uint]*teamBatch)
		var orphanEvents []*model.AlertEvent // events with no team/rule

		for i := range events {
			ev := &events[i]
			var teamID uint
			if ev.RuleID != nil {
				if rule, ok := ruleMap[*ev.RuleID]; ok && rule.TeamID != nil {
					teamID = *rule.TeamID
				}
			}
			if teamID > 0 {
				tb, ok := teamBatches[teamID]
				if !ok {
					tb = &teamBatch{teamID: teamID}
					teamBatches[teamID] = tb
				}
				tb.events = append(tb.events, ev)
			} else {
				orphanEvents = append(orphanEvents, ev)
			}
		}

		// Process each team's events concurrently with errgroup (H3).
		var eg errgroup.Group
		eg.SetLimit(8)

		for _, tb := range teamBatches {
			batch := tb
			eg.Go(func() error {
				matched := append(append([]model.EscalationPolicy{}, globalPolicies...), teamPolicies[batch.teamID]...)
				for _, ev := range batch.events {
					rule := e.getRuleFromMap(ruleMap, ev)
					e.escalateEvent(ctx, ev, matched, allSteps, now)
					e.checkSLABreach(ctx, ev, ruleMap, now)
					_ = rule // used by escalateEvent via ruleMap
				}
				return nil
			})
		}

		// Process orphan events (no team match) with global policies only.
		eg.Go(func() error {
			for _, ev := range orphanEvents {
				e.escalateEvent(ctx, ev, globalPolicies, allSteps, now)
				e.checkSLABreach(ctx, ev, ruleMap, now)
			}
			return nil
		})

		eg.Wait()

		afterID = events[len(events)-1].ID
		if len(events) < pageSize {
			break
		}
	}
}

func (e *EscalationExecutor) getRuleFromMap(ruleMap map[uint]*model.AlertRule, event *model.AlertEvent) *model.AlertRule {
	if ruleMap == nil || event.RuleID == nil {
		return nil
	}
	return ruleMap[*event.RuleID]
}

// batchLoadRules collects unique rule IDs from events and loads them in a single query.
// Returns nil if ruleRepo is not configured or on error.
func (e *EscalationExecutor) batchLoadRules(ctx context.Context, events []model.AlertEvent) map[uint]*model.AlertRule {
	if e.ruleRepo == nil {
		return nil
	}
	seen := make(map[uint]struct{})
	ruleIDs := make([]uint, 0, len(events))
	for i := range events {
		if events[i].RuleID != nil {
			if _, dup := seen[*events[i].RuleID]; !dup {
				seen[*events[i].RuleID] = struct{}{}
				ruleIDs = append(ruleIDs, *events[i].RuleID)
			}
		}
	}
	if len(ruleIDs) == 0 {
		return nil
	}
	rules, err := e.ruleRepo.GetByIDs(ctx, ruleIDs)
	if err != nil {
		e.logger.Error("escalation: failed to batch-load rules", zap.Error(err))
		return nil
	}
	m := make(map[uint]*model.AlertRule, len(rules))
	for i := range rules {
		m[rules[i].ID] = &rules[i]
	}
	return m
}

// checkSLABreach fires an SLA escalation when an unacknowledged firing alert
// exceeds the rule's AckSlaMinutes threshold. Only fires once per event.
// ruleMap is the pre-loaded map from batchLoadRules; may be nil.
func (e *EscalationExecutor) checkSLABreach(ctx context.Context, event *model.AlertEvent, ruleMap map[uint]*model.AlertRule, now time.Time) {
	if ruleMap == nil || event.RuleID == nil {
		return
	}
	rule, ok := ruleMap[*event.RuleID]
	if !ok || rule.AckSlaMinutes <= 0 {
		return
	}

	// If the event has already been SLA-escalated, skip.
	if event.SlaEscalatedAt != nil {
		return
	}

	// SLA window starts from FiredAt.
	slaDeadline := event.FiredAt.Add(time.Duration(rule.AckSlaMinutes) * time.Minute)
	if now.Before(slaDeadline) {
		return // still within SLA
	}

	// Record SLA escalation timestamp to prevent repeat fires.
	slaAt := now
	if err := e.eventRepo.UpdateSLAEscalated(ctx, event.ID, slaAt); err != nil {
		e.logger.Error("sla: failed to mark sla_escalated_at",
			zap.Uint("event_id", event.ID), zap.Error(err))
		return
	}

	note := fmt.Sprintf("SLA breach: event not acknowledged within %d minutes (rule: %s)",
		rule.AckSlaMinutes, rule.Name)
	_ = e.recordTimeline(ctx, event.ID, note, nil) // nil stepID — SLA breach is not an escalation step

	e.logger.Warn("SLA breach detected",
		zap.Uint("event_id", event.ID),
		zap.String("alert_name", event.AlertName),
		zap.Int("sla_minutes", rule.AckSlaMinutes),
	)
}

// escalateEvent evaluates escalation policies and executes any due steps for the given event.
// policies is the pre-filtered list (team-matched + global) from runOnce.
// stepsMap is the batch-loaded steps from runOnce; may be nil (falls back to per-policy query).
func (e *EscalationExecutor) escalateEvent(ctx context.Context, event *model.AlertEvent, policies []model.EscalationPolicy, stepsMap map[uint][]model.EscalationStep, now time.Time) {
	// Fallback: timeline-based dedup when stepExecRepo is not configured (M4 — skip when repo available).
	var executedSteps map[string]bool
	if e.stepExecRepo == nil {
		executedSteps = e.executedStepOrders(ctx, event.ID)
	}

	for _, policy := range policies {
		steps, ok := stepsMap[policy.ID]
		if !ok {
			// Fallback: load individually if not in batch.
			var err error
			steps, err = e.stepRepo.ListByPolicyID(ctx, policy.ID)
			if err != nil {
				e.logger.Warn("escalation: failed to list steps",
					zap.Uint("policy_id", policy.ID), zap.Error(err))
				continue
			}
		}

		for _, step := range steps {
			// Check if enough time has passed since the alert fired.
			dueAt := event.FiredAt.Add(time.Duration(step.DelayMinutes) * time.Minute)
			if now.Before(dueAt) {
				continue
			}

			// M5: Recheck event status — may have been resolved/ack'd since we fetched.
			if e.stepExecRepo != nil {
				fresh, err := e.eventRepo.GetByID(ctx, event.ID)
				if err != nil {
					e.logger.Error("escalation: failed to recheck event status",
						zap.Uint("event_id", event.ID), zap.Error(err))
					continue
				}
				if fresh.Status != model.EventStatusFiring {
					return // event no longer firing — skip all remaining steps
				}
			}

			// Atomic dedup: INSERT IGNORE ensures only one goroutine executes this step.
			if e.stepExecRepo != nil {
				inserted, err := e.stepExecRepo.InsertIgnore(ctx, event.ID, step.ID)
				if err != nil {
					e.logger.Error("escalation: failed to check step execution",
						zap.Uint("event_id", event.ID),
						zap.Uint("step_id", step.ID),
						zap.Error(err),
					)
					continue
				}
				if !inserted {
					// Already executed or in-progress — check if it failed and needs retry (H2).
					if e.stepExecRepo.HasExecuted(ctx, event.ID, step.ID) {
						continue // successfully done
					}
					// Status is 'pending' or 'failed' — allow retry by deleting the old record.
					if err := e.stepExecRepo.DeleteByEventAndStep(ctx, event.ID, step.ID); err != nil {
						e.logger.Error("escalation: failed to delete stale step exec",
							zap.Uint("event_id", event.ID), zap.Uint("step_id", step.ID), zap.Error(err))
						continue
					}
					// Re-insert with fresh 'pending' status.
					inserted, err = e.stepExecRepo.InsertIgnore(ctx, event.ID, step.ID)
					if err != nil || !inserted {
						continue
					}
				}
			} else {
				// Fallback: timeline-based dedup when stepExecRepo is not configured.
				stepKey := fmt.Sprintf("step:%d", step.ID)
				if executedSteps[stepKey] {
					continue
				}
			}

			// Execute this step.
			policyIDStr := strconv.FormatUint(uint64(policy.ID), 10)
			if err := e.executeStep(ctx, event, &policy, &step); err != nil {
				e.logger.Error("escalation: failed to execute step",
					zap.Uint("event_id", event.ID),
					zap.Uint("policy_id", policy.ID),
					zap.Int("step_order", step.StepOrder),
					zap.Error(err),
				)
				// H2: Mark as failed so it can be retried next cycle.
				if e.stepExecRepo != nil {
					_ = e.stepExecRepo.MarkFailed(ctx, event.ID, step.ID)
				}
				// Record failure in timeline.
				_ = e.recordTimeline(ctx, event.ID, fmt.Sprintf(
					"escalation step %d (policy %s) failed: %v", step.StepOrder, policy.Name, err,
				), &step.ID)
				metrics.IncEscalationSteps(policyIDStr, "failure")
			} else {
				// H2: Mark as success.
				if e.stepExecRepo != nil {
					_ = e.stepExecRepo.MarkSuccess(ctx, event.ID, step.ID)
				}
				metrics.IncEscalationSteps(policyIDStr, "success")
			}
		}
	}
}

// executeStep dispatches a notification for a single escalation step.
func (e *EscalationExecutor) executeStep(ctx context.Context, event *model.AlertEvent, policy *model.EscalationPolicy, step *model.EscalationStep) error {
	// Per-step timeout: a single slow webhook must not consume the entire escalation budget.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// This note is also used as the dedup key in executedStepOrders — keep format in sync.
	note := fmt.Sprintf("escalation policy '%s' step %d triggered (delay: %dm)",
		policy.Name, step.StepOrder, step.DelayMinutes)

	// Resolve the notification channel: prefer the step's override channel, then fall
	// back to notifying the target user/team directly via a system message.
	if step.NotifyChannelID != nil {
		channel, err := e.channelRepo.GetByID(ctx, *step.NotifyChannelID)
		if err != nil {
			return fmt.Errorf("channel %d not found: %w", *step.NotifyChannelID, err)
		}
		if err := e.sendViaChannel(ctx, event, channel); err != nil {
			return fmt.Errorf("send notification via channel %d: %w", *step.NotifyChannelID, err)
		}
	} else {
		// No channel override — dispatch directly to the target via personal notify configs.
		if err := e.dispatchToTarget(ctx, event, step); err != nil {
			return fmt.Errorf("dispatch to target %s/%d: %w", step.TargetType, step.TargetID, err)
		}
	}

	// Record the escalation in the timeline so we don't repeat this step.
	_ = e.recordTimeline(ctx, event.ID, note, &step.ID)

	e.logger.Info("escalation step executed",
		zap.Uint("event_id", event.ID),
		zap.String("policy", policy.Name),
		zap.Int("step_order", step.StepOrder),
	)
	return nil
}

// executedStepOrders returns a set of step keys already recorded in the
// event's timeline with action=escalated.
// Primary dedup: EscalationStepID → "step:<id>" (stable across note format changes).
// Fallback: Note text (for records created before migration 000037).
func (e *EscalationExecutor) executedStepOrders(ctx context.Context, eventID uint) map[string]bool {
	timelines, err := e.timelineRepo.ListByEventID(ctx, eventID)
	if err != nil {
		return map[string]bool{}
	}
	result := make(map[string]bool)
	for _, t := range timelines {
		if t.Action == model.TimelineActionEscalated {
			if t.EscalationStepID != nil {
				// Primary: stable step ID dedup.
				result[fmt.Sprintf("step:%d", *t.EscalationStepID)] = true
			} else {
				// Fallback: legacy records without EscalationStepID — use note text.
				result[t.Note] = true
			}
		}
	}
	return result
}

// recordTimeline appends an escalation action to the event timeline.
// stepID links the record to a specific EscalationStep for reliable dedup; may be nil for non-step events.
func (e *EscalationExecutor) recordTimeline(ctx context.Context, eventID uint, note string, stepID *uint) error {
	t := &model.AlertTimeline{
		EventID:          eventID,
		Action:           model.TimelineActionEscalated,
		Note:             note,
		EscalationStepID: stepID,
	}
	if err := e.timelineRepo.Create(ctx, t); err != nil {
		e.logger.Error("escalation: failed to record timeline",
			zap.Uint("event_id", eventID), zap.Error(err))
		return err
	}
	return nil
}

// dispatchToTarget routes the escalation to the correct target based on step.TargetType.
func (e *EscalationExecutor) dispatchToTarget(ctx context.Context, event *model.AlertEvent, step *model.EscalationStep) error {
	switch step.TargetType {
	case "user":
		return e.notifyUserPersonal(ctx, event, step.TargetID)

	case "team":
		if e.teamRepo == nil {
			e.logger.Warn("escalation: teamRepo not configured, skipping team dispatch",
				zap.Uint("event_id", event.ID))
			return nil
		}
		members, err := e.teamRepo.ListMembers(ctx, step.TargetID)
		if err != nil {
			return fmt.Errorf("list team members: %w", err)
		}
		var lastErr error
		for _, m := range members {
			if err := e.notifyUserPersonal(ctx, event, m.UserID); err != nil {
				e.logger.Warn("escalation: failed to notify team member",
					zap.Uint("user_id", m.UserID), zap.Error(err))
				lastErr = err
			}
		}
		return lastErr

	case "schedule":
		if e.onCallShiftRepo == nil {
			e.logger.Warn("escalation: onCallShiftRepo not configured, skipping schedule dispatch",
				zap.Uint("event_id", event.ID))
			return nil
		}
		user, err := e.onCallShiftRepo.GetCurrentOnCallUser(ctx, step.TargetID)
		if err != nil {
			return fmt.Errorf("get current on-call user: %w", err)
		}
		if user == nil {
			e.logger.Info("escalation: no one currently on call for schedule",
				zap.Uint("schedule_id", step.TargetID))
			return nil
		}
		return e.notifyUserPersonal(ctx, event, user.ID)

	default:
		e.logger.Warn("escalation: unknown target type, skipping",
			zap.String("target_type", step.TargetType),
			zap.Uint("event_id", event.ID))
		return nil
	}
}

// notifyUserPersonal sends a personal notification to a user via their UserNotifyConfig entries.
// Supports "webhook", "lark_personal" (Lark Bot DM), and "email" (global SMTP) media types.
func (e *EscalationExecutor) notifyUserPersonal(ctx context.Context, event *model.AlertEvent, userID uint) error {
	if e.userNotifyConfigRepo == nil {
		e.logger.Warn("escalation: userNotifyConfigRepo not configured, skipping personal notify",
			zap.Uint("user_id", userID))
		return nil
	}

	configs, err := e.userNotifyConfigRepo.ListByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list user notify configs: %w", err)
	}

	if len(configs) == 0 {
		e.logger.Info("escalation: user has no personal notify configs",
			zap.Uint("user_id", userID))
		return nil
	}

	var lastErr error
	for _, cfg := range configs {
		if !cfg.IsEnabled {
			continue
		}
		switch cfg.MediaType {
		case "webhook":
			// UserNotifyConfig webhook config: {"url": "https://..."}
			// custom_webhook channel accepts the same url field; method defaults to POST.
			syntheticChannel := &model.NotifyChannel{
				Type:   model.ChannelTypeCustom,
				Config: cfg.Config,
			}
			if err := e.sendViaChannel(ctx, event, syntheticChannel); err != nil {
				e.logger.Warn("escalation: personal webhook notify failed",
					zap.Uint("user_id", userID), zap.Error(err))
				lastErr = err
			}
		case "lark_personal":
			if e.larkSvc == nil {
				e.logger.Warn("escalation: larkSvc not configured, cannot send lark_personal DM",
					zap.Uint("user_id", userID))
				continue
			}
			receiveIDType, receiveID, perr := parseLarkPersonalConfig(cfg.Config)
			if perr != nil {
				e.logger.Warn("escalation: invalid lark_personal config",
					zap.Uint("user_id", userID), zap.Error(perr))
				lastErr = perr
				continue
			}
			if _, err := e.larkSvc.SendAlertCardToUser(ctx, event, nil, receiveIDType, receiveID); err != nil {
				e.logger.Warn("escalation: lark_personal DM failed",
					zap.Uint("user_id", userID),
					zap.String("receive_id_type", receiveIDType),
					zap.Error(err))
				lastErr = err
			}
		case "email":
			// UserNotifyConfig email config: {"email": "user@example.com"}
			// Route via global SMTP if configured; otherwise skip with a log.
			if e.settingSvc == nil {
				e.logger.Info("escalation: personal email skipped (setting service not injected)",
					zap.Uint("user_id", userID))
				continue
			}
			smtpCfg, sErr := e.settingSvc.GetSMTPConfig(ctx)
			if sErr != nil || !smtpCfg.Enabled || smtpCfg.SMTPHost == "" {
				e.logger.Info("escalation: personal email skipped (global SMTP not configured)",
					zap.Uint("user_id", userID))
				continue
			}
			var emailCfg struct {
				Email string `json:"email"`
			}
			if jErr := json.Unmarshal([]byte(cfg.Config), &emailCfg); jErr != nil || emailCfg.Email == "" {
				e.logger.Warn("escalation: invalid personal email config", zap.Uint("user_id", userID))
				continue
			}
			port := smtpCfg.SMTPPort
			if port == 0 {
				port = 587
			}
			from := smtpCfg.From
			if from == "" {
				from = smtpCfg.Username
			}
			// Build a synthetic email channel using global SMTP + user's email as recipient
			type emailChanCfg struct {
				SMTPHost   string   `json:"smtp_host"`
				SMTPPort   int      `json:"smtp_port"`
				SMTPTLS    bool     `json:"smtp_tls"`
				Username   string   `json:"username"`
				Password   string   `json:"password"`
				From       string   `json:"from"`
				Recipients []string `json:"recipients"`
			}
			chanBytes, _ := json.Marshal(emailChanCfg{
				SMTPHost:   smtpCfg.SMTPHost,
				SMTPPort:   port,
				SMTPTLS:    smtpCfg.SMTPTLS,
				Username:   smtpCfg.Username,
				Password:   smtpCfg.Password,
				From:       from,
				Recipients: []string{emailCfg.Email},
			})
			syntheticEmailChannel := &model.NotifyChannel{
				Type:   model.ChannelTypeEmail,
				Config: string(chanBytes),
			}
			if err := e.sendViaChannel(ctx, event, syntheticEmailChannel); err != nil {
				e.logger.Warn("escalation: personal email failed",
					zap.Uint("user_id", userID), zap.String("to", emailCfg.Email), zap.Error(err))
				lastErr = err
			}
		default:
			e.logger.Warn("escalation: unsupported personal notify media type",
				zap.String("media_type", cfg.MediaType), zap.Uint("user_id", userID))
		}
	}

	return lastErr
}

// parseLarkPersonalConfig extracts the Lark DM recipient from a UserNotifyConfig
// `lark_personal` record. Accepts any of these JSON shapes (in order of preference):
//
//	{"user_id":"xxx"}       → receive_id_type=user_id
//	{"open_id":"ou_xxx"}    → receive_id_type=open_id
//	{"union_id":"on_xxx"}   → receive_id_type=union_id
//	{"lark_user_id":"xxx"}  → receive_id_type=user_id (legacy alias)
func parseLarkPersonalConfig(raw string) (receiveIDType, receiveID string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("lark_personal config is empty")
	}
	var c struct {
		UserID      string `json:"user_id"`
		OpenID      string `json:"open_id"`
		UnionID     string `json:"union_id"`
		LarkUserID  string `json:"lark_user_id"`
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return "", "", fmt.Errorf("parse lark_personal config: %w", err)
	}
	switch {
	case c.UserID != "":
		return "user_id", c.UserID, nil
	case c.LarkUserID != "":
		return "user_id", c.LarkUserID, nil
	case c.OpenID != "":
		return "open_id", c.OpenID, nil
	case c.UnionID != "":
		return "union_id", c.UnionID, nil
	default:
		return "", "", fmt.Errorf("lark_personal config missing user_id/open_id/union_id")
	}
}
