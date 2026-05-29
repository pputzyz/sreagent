package service

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// integrationDropped counts async tasks dropped due to full semaphore.
var integrationDropped int64

// NormalizedAlert is a canonical representation of an inbound alert,
// regardless of whether it came from standard/AlertManager/Grafana format.
type NormalizedAlert struct {
	Title       string
	Description string
	Severity    model.AlertSeverity
	Status      string // "firing" | "resolved"
	Labels      map[string]string
	Annotations map[string]string
	GeneratorURL string
	StartsAt    time.Time
	EndsAt      *time.Time
}

// ---- Rate limiter (fixed-window counter, per integration) ----
// Note: current implementation uses separate fixed-window counters for
// per-second and per-minute limits. Window boundaries may allow ~2x burst
// (e.g. a burst at t=0.99s + t=1.01s consumes 2x per-second tokens).
// For strict rate limiting, consider golang.org/x/time/rate (token bucket).

type rateLimiter struct {
	mu           sync.Mutex
	perSecTokens int
	perMinTokens int
	secBucket    int
	minBucket    int
	lastSecReset time.Time
	lastMinReset time.Time
}

func newRateLimiter(perSec, perMin int) *rateLimiter {
	return &rateLimiter{
		perSecTokens: perSec,
		perMinTokens: perMin,
		secBucket:    perSec,
		minBucket:    perMin,
		lastSecReset: time.Now(),
		lastMinReset: time.Now(),
	}
}

// Allow checks whether a request is within the per-second and per-minute
// rate limits. Uses independent fixed-window counters: each counter resets
// when its window expires, then decrements on every call. Returns false if
// either budget is exhausted.
func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if now.Sub(rl.lastSecReset) >= time.Second {
		rl.secBucket = rl.perSecTokens
		rl.lastSecReset = now
	}
	if now.Sub(rl.lastMinReset) >= time.Minute {
		rl.minBucket = rl.perMinTokens
		rl.lastMinReset = now
	}
	if rl.secBucket <= 0 || rl.minBucket <= 0 {
		return false
	}
	rl.secBucket--
	rl.minBucket--
	return true
}

// ---- IntegrationService ----

// maxAsyncIntegrationTasks caps concurrent async goroutines in the integration service.
const maxAsyncIntegrationTasks = 100

type IntegrationService struct {
	repo        *repository.IntegrationRepository
	routingRepo *repository.RoutingRuleRepository
	pipeline    *AlertV2Pipeline
	logger      *zap.Logger
	dispatchSem chan struct{}

	// Per-integration rate limiters
	limitersMu sync.RWMutex
	limiters   map[uint]*rateLimiter
}

func NewIntegrationService(
	repo *repository.IntegrationRepository,
	routingRepo *repository.RoutingRuleRepository,
	pipeline *AlertV2Pipeline,
	logger *zap.Logger,
) *IntegrationService {
	return &IntegrationService{
		repo:        repo,
		routingRepo: routingRepo,
		pipeline:    pipeline,
		logger:      logger,
		dispatchSem: make(chan struct{}, maxAsyncIntegrationTasks),
		limiters:    make(map[uint]*rateLimiter),
	}
}

// --- CRUD ---

func (s *IntegrationService) Create(ctx context.Context, integ *model.Integration) error {
	// Generate unique webhook token
	token, err := generateToken(32)
	if err != nil {
		return apperr.Wrap(apperr.ErrInternal, err)
	}
	integ.WebhookToken = token

	if err := s.repo.Create(ctx, integ); err != nil {
		s.logger.Error("failed to create integration", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	maskedToken := token
	if len(token) > 8 {
		maskedToken = token[:8] + "..."
	}
	s.logger.Info("integration created", zap.Uint("id", integ.ID), zap.String("token", maskedToken))
	return nil
}

func (s *IntegrationService) GetByID(ctx context.Context, id uint) (*model.Integration, error) {
	integ, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return integ, nil
}

func (s *IntegrationService) List(ctx context.Context, channelID uint, page, pageSize int) ([]model.Integration, int64, error) {
	list, total, err := s.repo.List(ctx, channelID, page, pageSize)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

func (s *IntegrationService) Update(ctx context.Context, id uint, updates *model.Integration) (*model.Integration, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	existing.IsEnabled = updates.IsEnabled
	if updates.PipelineConfig != "" {
		existing.PipelineConfig = updates.PipelineConfig
	}
	if updates.LabelEnhancementConfig != "" {
		existing.LabelEnhancementConfig = updates.LabelEnhancementConfig
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return existing, nil
}

func (s *IntegrationService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.ErrNotFound
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return s.repo.Delete(ctx, id)
}

// --- Webhook receive ---

// ReceiveAlerts is the main entry point for inbound webhook alerts.
// It: (1) looks up the integration, (2) rate-limits, (3) normalises the payload,
// (4) applies the pipeline, (5) routes to the v2 pipeline.
func (s *IntegrationService) ReceiveAlerts(ctx context.Context, token string, rawBody []byte) error {
	integ, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.WithMessage(apperr.ErrNotFound, "integration not found")
		}
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if !integ.IsEnabled {
		return apperr.WithMessage(apperr.ErrBadRequest, "integration is disabled")
	}

	// Rate limiting (4.8): 100/s 1000/min per integration
	limiter := s.getLimiter(integ.ID)
	if !limiter.Allow() {
		return apperr.WithMessage(apperr.ErrBadRequest, "rate limit exceeded (100/s or 1000/min)")
	}

	// Normalise payload based on integration type (4.5/4.6)
	alerts, err := s.normalise(integ.Type, rawBody)
	if err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "failed to parse payload: "+err.Error())
	}

	// Apply pipeline steps (4.7)
	alerts = s.applyPipeline(integ.PipelineConfig, alerts)

	// Resolve target channel
	channelID := uint(0)
	if integ.ChannelID != nil {
		channelID = *integ.ChannelID
	}

	// For shared integrations: use routing rules to determine channel
	if integ.Mode == model.IntegrationModeShared {
		rules, err := s.routingRepo.ListByIntegration(ctx, integ.ID)
		if err != nil {
			s.logger.Error("failed to list routing rules for shared integration", zap.Error(err), zap.Uint("integration_id", integ.ID))
			return fmt.Errorf("list routing rules: %w", err)
		}
		for _, alert := range alerts {
			// Reset to integration default each iteration to prevent stale channelID leaking
			iterChannelID := uint(0)
			if integ.ChannelID != nil {
				iterChannelID = *integ.ChannelID
			}
			targetID := s.matchRoutingRule(rules, alert.Labels, string(alert.Severity))
			if targetID > 0 {
				iterChannelID = targetID
			}
			s.injectAndRoute(ctx, integ, alert, iterChannelID)
		}
	} else {
		for _, alert := range alerts {
			s.injectAndRoute(ctx, integ, alert, channelID)
		}
	}

	// Increment counter (fire-and-forget)
	select {
	case s.dispatchSem <- struct{}{}:
		go func() {
			defer func() { <-s.dispatchSem }()
			_ = s.repo.IncrTotalAlerts(context.Background(), integ.ID)
		}()
	default:
		total := atomic.AddInt64(&integrationDropped, 1)
		s.logger.Error("dropping async integration counter increment, too many in flight",
			zap.Int64("total_dropped", total))
	}

	return nil
}

// injectAndRoute converts a NormalizedAlert into a synthetic AlertEvent and
// feeds it into the v2 pipeline.
func (s *IntegrationService) injectAndRoute(ctx context.Context, integ *model.Integration, alert NormalizedAlert, channelID uint) {
	if s.pipeline == nil {
		return
	}

	labels := make(model.JSONLabels)
	for k, v := range alert.Labels {
		labels[k] = v
	}
	// Inject routing hints
	if channelID > 0 {
		labels["_channel_id"] = fmt.Sprintf("%d", channelID)
	}
	labels["_integration_id"] = fmt.Sprintf("%d", integ.ID)
	labels["_integration_type"] = string(integ.Type)

	var status model.AlertEventStatus
	if alert.Status == "resolved" {
		status = model.EventStatusResolved
	} else {
		status = model.EventStatusFiring
	}

	annotations := make(model.JSONLabels)
	for k, v := range alert.Annotations {
		annotations[k] = v
	}
	if alert.Description != "" {
		annotations["description"] = alert.Description
	}

	syntheticEvent := &model.AlertEvent{
		AlertName:   alert.Title,
		Severity:    alert.Severity,
		Status:      status,
		Labels:      labels,
		Annotations: annotations,
		Source:      string(integ.Type),
		Fingerprint: generateIntegrationFingerprint(integ.ID, alert.Labels, alert.Title),
		GeneratorURL: alert.GeneratorURL,
		FiredAt:     alert.StartsAt,
	}

	select {
	case s.dispatchSem <- struct{}{}:
		go func() {
			defer func() { <-s.dispatchSem }()
			pCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := s.pipeline.process(pCtx, syntheticEvent); err != nil {
				s.logger.Error("integration: v2 pipeline failed",
					zap.Uint("integration_id", integ.ID),
					zap.String("alert", alert.Title),
					zap.Error(err),
				)
			}
		}()
	default:
		total := atomic.AddInt64(&integrationDropped, 1)
		s.logger.Error("dropping async integration pipeline task, too many in flight",
			zap.Int("capacity", maxAsyncIntegrationTasks),
			zap.Int64("total_dropped", total),
			zap.Uint("integration_id", integ.ID))
	}
}

// --- Normalisation ---

// normalise converts a raw webhook payload to a slice of NormalizedAlert.
func (s *IntegrationService) normalise(typ model.IntegrationType, body []byte) ([]NormalizedAlert, error) {
	switch typ {
	case model.IntegrationTypeAlertManager:
		return s.normaliseAlertManager(body)
	case model.IntegrationTypeGrafana:
		return s.normaliseGrafana(body)
	default:
		return s.normaliseStandard(body)
	}
}

// Standard format: {"alerts":[{title,description,severity,status,labels,annotations,generator_url,starts_at}]}
// or a single alert object.
func (s *IntegrationService) normaliseStandard(body []byte) ([]NormalizedAlert, error) {
	// Try array wrapper first
	var wrapper struct {
		Alerts []struct {
			Title        string            `json:"title"`
			Description  string            `json:"description"`
			Severity     string            `json:"severity"`
			Status       string            `json:"status"`
			Labels       map[string]string `json:"labels"`
			Annotations  map[string]string `json:"annotations"`
			GeneratorURL string            `json:"generator_url"`
			StartsAt     string            `json:"starts_at"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.Alerts) > 0 {
		result := make([]NormalizedAlert, 0, len(wrapper.Alerts))
		for _, a := range wrapper.Alerts {
			na := NormalizedAlert{
				Title:        a.Title,
				Description:  a.Description,
				Severity:     normaliseSeverity(a.Severity),
				Status:       normaliseStatus(a.Status),
				Labels:       a.Labels,
				Annotations:  a.Annotations,
				GeneratorURL: a.GeneratorURL,
				StartsAt:     parseTime(a.StartsAt),
			}
			if na.Labels == nil {
				na.Labels = make(map[string]string)
			}
			result = append(result, na)
		}
		return result, nil
	}

	// Try single object
	var single struct {
		Title        string            `json:"title"`
		Description  string            `json:"description"`
		Severity     string            `json:"severity"`
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		GeneratorURL string            `json:"generator_url"`
		StartsAt     string            `json:"starts_at"`
	}
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	labels := single.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	return []NormalizedAlert{{
		Title:        single.Title,
		Description:  single.Description,
		Severity:     normaliseSeverity(single.Severity),
		Status:       normaliseStatus(single.Status),
		Labels:       labels,
		Annotations:  single.Annotations,
		GeneratorURL: single.GeneratorURL,
		StartsAt:     parseTime(single.StartsAt),
	}}, nil
}

// AlertManager format: {"alerts":[{status,labels,annotations,startsAt,endsAt,generatorURL}]}
func (s *IntegrationService) normaliseAlertManager(body []byte) ([]NormalizedAlert, error) {
	var payload struct {
		Alerts []struct {
			Status string `json:"status"` // "firing" | "resolved"
			Labels map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			StartsAt    string `json:"startsAt"`
			EndsAt      string `json:"endsAt"`
			GeneratorURL string `json:"generatorURL"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	result := make([]NormalizedAlert, 0, len(payload.Alerts))
	for _, a := range payload.Alerts {
		labels := a.Labels
		if labels == nil {
			labels = make(map[string]string)
		}
		title := labels["alertname"]
		if title == "" {
			title = "Unknown Alert"
		}
		desc := ""
		if a.Annotations != nil {
			desc = a.Annotations["description"]
			if desc == "" {
				desc = a.Annotations["summary"]
			}
		}
		result = append(result, NormalizedAlert{
			Title:        title,
			Description:  desc,
			Severity:     normaliseSeverity(labels["severity"]),
			Status:       normaliseStatus(a.Status),
			Labels:       labels,
			Annotations:  a.Annotations,
			GeneratorURL: a.GeneratorURL,
			StartsAt:     parseTime(a.StartsAt),
		})
	}
	return result, nil
}

// Grafana format: {"alerts":[{title,state,labels,annotations,generatorURL,startsAt}]}
// Grafana uses "state": "alerting" | "ok" | "no_data"
func (s *IntegrationService) normaliseGrafana(body []byte) ([]NormalizedAlert, error) {
	var payload struct {
		Alerts []struct {
			Title       string            `json:"title"`
			State       string            `json:"state"` // alerting | ok | no_data | pending
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			GeneratorURL string           `json:"generatorURL"`
			StartsAt    string            `json:"startsAt"`
			ValueString string            `json:"valueString"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	result := make([]NormalizedAlert, 0, len(payload.Alerts))
	for _, a := range payload.Alerts {
		labels := a.Labels
		if labels == nil {
			labels = make(map[string]string)
		}
		status := "firing"
		if a.State == "ok" || a.State == "normal" {
			status = "resolved"
		}
		desc := ""
		if a.Annotations != nil {
			desc = a.Annotations["description"]
			if desc == "" {
				desc = a.Annotations["summary"]
			}
		}
		result = append(result, NormalizedAlert{
			Title:        a.Title,
			Description:  desc,
			Severity:     normaliseSeverity(labels["severity"]),
			Status:       status,
			Labels:       labels,
			Annotations:  a.Annotations,
			GeneratorURL: a.GeneratorURL,
			StartsAt:     parseTime(a.StartsAt),
		})
	}
	return result, nil
}

// --- Pipeline (4.7) ---

func (s *IntegrationService) applyPipeline(pipelineJSON string, alerts []NormalizedAlert) []NormalizedAlert {
	if pipelineJSON == "" || pipelineJSON == "[]" || pipelineJSON == "null" {
		return alerts
	}
	var steps []model.AlertPipelineStep
	if err := json.Unmarshal([]byte(pipelineJSON), &steps); err != nil {
		return alerts
	}

	result := make([]NormalizedAlert, 0, len(alerts))
	for _, alert := range alerts {
		drop := false
		for _, step := range steps {
			// Evaluate step conditions
			if !pipelineConditionsMatch(step.Conditions, alert) {
				continue
			}
			switch step.Action {
			case "rewrite_severity":
				alert.Severity = normaliseSeverity(step.TargetValue)
			case "rewrite_title":
				alert.Title = expandPipelineTemplate(step.TargetValue, alert)
			case "rewrite_description":
				alert.Description = expandPipelineTemplate(step.TargetValue, alert)
			case "drop":
				drop = true
			}
		}
		if !drop {
			result = append(result, alert)
		}
	}
	return result
}

func pipelineConditionsMatch(conds []model.FilterCondition, alert NormalizedAlert) bool {
	for _, c := range conds {
		var actual string
		switch {
		case c.Field == "severity":
			actual = string(alert.Severity)
		case c.Field == "title", c.Field == "alertname":
			actual = alert.Title
		case strings.HasPrefix(c.Field, "labels."):
			actual = alert.Labels[strings.TrimPrefix(c.Field, "labels.")]
		default:
			actual = alert.Labels[c.Field]
		}
		if !evalDispatchCondition(c.Operator, actual, c.Value) {
			return false
		}
	}
	return true
}

func expandPipelineTemplate(tmpl string, alert NormalizedAlert) string {
	result := tmpl
	result = strings.ReplaceAll(result, "{{title}}", alert.Title)
	result = strings.ReplaceAll(result, "{{severity}}", string(alert.Severity))
	result = strings.ReplaceAll(result, "{{description}}", alert.Description)
	for k, v := range alert.Labels {
		result = strings.ReplaceAll(result, "{{labels."+k+"}}", v)
	}
	return result
}

// --- Routing rules ---

func (s *IntegrationService) matchRoutingRule(rules []model.RoutingRule, labels map[string]string, severity string) uint {
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		if matchIntegrationConditions(rule.Conditions, labels, severity) {
			return rule.TargetChannelID
		}
	}
	return 0
}

func matchIntegrationConditions(condJSON string, labels map[string]string, severity string) bool {
	if condJSON == "" || condJSON == "[]" {
		return true
	}
	var conds []model.FilterCondition
	if err := json.Unmarshal([]byte(condJSON), &conds); err != nil {
		return true
	}
	for _, c := range conds {
		var actual string
		if c.Field == "severity" {
			actual = severity
		} else {
			actual = labels[strings.TrimPrefix(c.Field, "labels.")]
		}
		if !evalDispatchCondition(c.Operator, actual, c.Value) {
			return false
		}
	}
	return true
}

// --- Rate limiter management ---

func (s *IntegrationService) getLimiter(integID uint) *rateLimiter {
	s.limitersMu.RLock()
	l, ok := s.limiters[integID]
	s.limitersMu.RUnlock()
	if ok {
		return l
	}
	s.limitersMu.Lock()
	defer s.limitersMu.Unlock()
	if l, ok = s.limiters[integID]; ok {
		return l
	}
	l = newRateLimiter(100, 1000)
	s.limiters[integID] = l
	return l
}

// --- Helpers ---

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func normaliseSeverity(s string) model.AlertSeverity {
	switch strings.ToLower(s) {
	case "p0", "critical", "crit", "error", "err", "high":
		return model.SeverityCritical
	case "p1", "p2", "warning", "warn", "medium":
		return model.SeverityWarning
	default:
		return model.SeverityInfo
	}
}

func normaliseStatus(s string) string {
	switch strings.ToLower(s) {
	case "resolved", "ok", "normal", "good":
		return "resolved"
	default:
		return "firing"
	}
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now()
}

// generateIntegrationFingerprint creates a deterministic fingerprint for a
// synthetic alert event so that dedup, incident aggregation, and resolution
// tracking work correctly for webhook-sourced alerts.
//
// Note: This fingerprint intentionally differs from the engine's fingerprint.Compute()
// by including integrationID and title. This prevents webhook-synthesized events from
// colliding with engine-evaluated events.
func generateIntegrationFingerprint(integrationID uint, labels map[string]string, title string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "integ:%d|", integrationID)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s,", k, labels[k])
	}
	b.WriteString(title)

	hash := md5.Sum([]byte(b.String()))
	return fmt.Sprintf("%x", hash)
}
