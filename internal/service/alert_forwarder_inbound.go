package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// InboundPayload represents the parsed inbound alert payload.
type InboundPayload struct {
	Alerts         []InboundAlert    `json:"alerts"`
	Receiver       string            `json:"receiver"`
	Status         string            `json:"status"`
	ExternalLabels map[string]string `json:"external_labels,omitempty"`
}

// InboundAlert represents a single alert in the inbound payload.
type InboundAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
}

// ProcessInbound processes an inbound alert payload for a specific forwarder.
func (s *AlertForwarderService) ProcessInbound(ctx context.Context, forwarderID uint, r *http.Request) error {
	// 1. Load forwarder
	forwarder, err := s.forwarderRepo.GetByID(ctx, forwarderID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// 2. Check enabled
	if !forwarder.Enabled {
		return apperr.WithMessage(apperr.ErrInvalidParam, "forwarder is disabled")
	}

	// 3. Check direction
	if forwarder.Direction != model.ForwarderDirectionInbound && forwarder.Direction != model.ForwarderDirectionBidirectional {
		return apperr.WithMessage(apperr.ErrInvalidParam, "forwarder does not support inbound direction")
	}

	// 4. Authenticate
	if forwarder.InboundConfig != nil {
		if err := s.authenticateInbound(r, forwarder.InboundConfig); err != nil {
			return err
		}
	}

	// 5. Parse payload
	payload, err := s.parseInboundPayload(r, forwarder)
	if err != nil {
		return err
	}

	// 6. Match labels
	if len(forwarder.MatchLabels) > 0 {
		matched := false
		for _, alert := range payload.Alerts {
			if labelmatch.Match(map[string]string(alert.Labels), map[string]string(forwarder.MatchLabels)) {
				matched = true
				break
			}
		}
		if !matched {
			s.logger.Debug("inbound payload does not match forwarder labels",
				zap.Uint("forwarder_id", forwarderID),
			)
			return nil // Silently skip
		}
	}

	// 7. Process each alert based on mode
	for _, alert := range payload.Alerts {
		if err := s.processInboundAlert(ctx, forwarder, &alert, payload); err != nil {
			s.logger.Error("failed to process inbound alert",
				zap.Uint("forwarder_id", forwarderID),
				zap.String("fingerprint", alert.Fingerprint),
				zap.Error(err),
			)
			// Continue processing remaining alerts
		}
	}

	return nil
}

// authenticateInbound validates the inbound request authentication.
func (s *AlertForwarderService) authenticateInbound(r *http.Request, config *model.InboundConfig) error {
	if config.AuthType == model.ForwarderAuthNone {
		return nil
	}

	if config.AuthConfig == nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "auth_config is required when auth_type is not none")
	}

	switch config.AuthType {
	case model.ForwarderAuthBearer:
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			return apperr.WithMessage(apperr.ErrUnauthorized, "missing or invalid Bearer token")
		}
		if strings.TrimPrefix(token, "Bearer ") != config.AuthConfig.Token {
			return apperr.WithMessage(apperr.ErrUnauthorized, "invalid Bearer token")
		}

	case model.ForwarderAuthBasic:
		username, password, ok := r.BasicAuth()
		if !ok {
			return apperr.WithMessage(apperr.ErrUnauthorized, "missing Basic auth credentials")
		}
		if username != config.AuthConfig.Username || password != config.AuthConfig.Password {
			return apperr.WithMessage(apperr.ErrUnauthorized, "invalid Basic auth credentials")
		}

	case model.ForwarderAuthHMAC:
		signature := r.Header.Get(config.AuthConfig.HMACHeader)
		if signature == "" {
			return apperr.WithMessage(apperr.ErrUnauthorized, "missing HMAC signature header")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, "failed to read request body")
		}
		// Reset body for later reading
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		var expectedSig string
		switch config.AuthConfig.HMACAlgorithm {
		case "sha1":
			mac := hmac.New(sha1.New, []byte(config.AuthConfig.HMACSecret))
			mac.Write(body)
			expectedSig = hex.EncodeToString(mac.Sum(nil))
		default: // sha256
			mac := hmac.New(sha256.New, []byte(config.AuthConfig.HMACSecret))
			mac.Write(body)
			expectedSig = hex.EncodeToString(mac.Sum(nil))
		}

		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			return apperr.WithMessage(apperr.ErrUnauthorized, "invalid HMAC signature")
		}
	}

	return nil
}

// parseInboundPayload parses the inbound request body based on source format.
func (s *AlertForwarderService) parseInboundPayload(r *http.Request, forwarder *model.AlertForwarder) (*InboundPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read request body")
	}

	var payload InboundPayload

	if forwarder.InboundConfig == nil {
		// Default to alertmanager format
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid JSON payload: "+err.Error())
		}
		return &payload, nil
	}

	switch forwarder.InboundConfig.SourceFormat {
	case model.SourceFormatAlertmanager, model.SourceFormatGrafana:
		// Alertmanager/Grafana use the same format
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid Alertmanager payload: "+err.Error())
		}

	case model.SourceFormatPrometheus:
		// Prometheus remote write format (simplified)
		var promPayload struct {
			Alerts []struct {
				Labels      map[string]string `json:"labels"`
				Annotations map[string]string `json:"annotations"`
				StartsAt    time.Time         `json:"startsAt"`
				EndsAt      time.Time         `json:"endsAt"`
			} `json:"alerts"`
		}
		if err := json.Unmarshal(body, &promPayload); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid Prometheus payload: "+err.Error())
		}
		for _, a := range promPayload.Alerts {
			status := "firing"
			if !a.EndsAt.IsZero() {
				status = "resolved"
			}
			fingerprint := generateFingerprint(a.Labels)
			payload.Alerts = append(payload.Alerts, InboundAlert{
				Status:      status,
				Labels:      a.Labels,
				Annotations: a.Annotations,
				StartsAt:    a.StartsAt,
				EndsAt:      a.EndsAt,
				Fingerprint: fingerprint,
			})
		}

	case model.SourceFormatGeneric:
		// Generic format - try to parse as-is
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "invalid JSON payload: "+err.Error())
		}

	default:
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "unsupported source format")
	}

	return &payload, nil
}

// processInboundAlert processes a single inbound alert based on the forwarder's mode.
func (s *AlertForwarderService) processInboundAlert(ctx context.Context, forwarder *model.AlertForwarder, alert *InboundAlert, payload *InboundPayload) error {
	// Get severity from labels
	severity := ""
	if sev, ok := alert.Labels["severity"]; ok {
		severity = sev
	}

	// Apply inbound severity mapping
	if mapped, applied := forwarder.InboundSeverityMapping.ApplySeverityMapping(severity); applied {
		alert.Labels["original_severity"] = severity
		alert.Labels["severity"] = mapped
		severity = mapped
	}

	// Build alert name
	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "Unknown"
	}

	// Route based on mode
	if forwarder.InboundConfig != nil && forwarder.InboundConfig.Mode == model.InboundModeProxy {
		return s.processProxyAlert(ctx, forwarder, alert, alertName, severity)
	}

	// Default: integrate mode
	return s.processIntegrateAlert(ctx, forwarder, alert, alertName, severity)
}

// processIntegrateAlert processes an inbound alert in integrate mode.
// The alert enters the platform's full lifecycle management.
func (s *AlertForwarderService) processIntegrateAlert(ctx context.Context, forwarder *model.AlertForwarder, alert *InboundAlert, alertName, severity string) error {
	event := &model.AlertEvent{
		Fingerprint:  alert.Fingerprint,
		AlertName:    alertName,
		Severity:     model.AlertSeverity(severity),
		Status:       model.EventStatusFiring,
		Labels:       model.JSONLabels(alert.Labels),
		Annotations:  model.JSONLabels(alert.Annotations),
		Source:       fmt.Sprintf("forwarder:%s", forwarder.Name),
		GeneratorURL: alert.GeneratorURL,
		FiredAt:      alert.StartsAt,
		FireCount:    1,
	}

	if alert.Status == "resolved" {
		event.Status = model.EventStatusResolved
	}

	// Save event to database
	if s.eventRepo != nil {
		if err := s.eventRepo.Create(ctx, event); err != nil {
			s.logger.Error("failed to save inbound alert event",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("fingerprint", alert.Fingerprint),
				zap.Error(err),
			)
			// Continue even if save fails
		}
	}

	// Apply platform capabilities
	caps := forwarder.PlatformCapabilities
	if caps == nil {
		caps = &model.PlatformCapabilitiesConfig{
			EnableNotification: true,
		}
	}

	// Inhibition check
	if caps.EnableInhibition && s.inhibitorSvc != nil {
		var firingEvents []model.AlertEvent
		if s.eventRepo != nil {
			firingEvents, _, _ = s.eventRepo.List(ctx, "firing", "", 1, 2000)
		}
		if s.inhibitorSvc.IsInhibited(ctx, event, firingEvents) {
			s.logger.Info("inbound alert inhibited",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
			)
			return nil
		}
	}

	// Mute check
	if caps.EnableMute && s.muteSvc != nil {
		if s.muteSvc.IsAlertMuted(ctx, event) {
			s.logger.Info("inbound alert muted",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
			)
			return nil
		}
	}

	// Notification routing
	if caps.EnableNotification && s.notifySvc != nil {
		if err := s.notifySvc.RouteAlert(ctx, event); err != nil {
			s.logger.Error("failed to route inbound alert notification",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("inbound alert integrated into platform",
		zap.Uint("forwarder_id", forwarder.ID),
		zap.String("forwarder_name", forwarder.Name),
		zap.String("alert_name", alertName),
		zap.String("severity", severity),
		zap.String("fingerprint", alert.Fingerprint),
	)

	return nil
}

// processProxyAlert processes an inbound alert in proxy mode.
// The alert is forwarded to the configured proxy target without entering platform lifecycle.
func (s *AlertForwarderService) processProxyAlert(ctx context.Context, forwarder *model.AlertForwarder, alert *InboundAlert, alertName, severity string) error {
	if forwarder.InboundConfig.ProxyTarget == nil {
		return fmt.Errorf("proxy target not configured")
	}

	target := forwarder.InboundConfig.ProxyTarget

	// Build outbound payload (Alertmanager format)
	outPayload := InboundPayload{
		Alerts: []InboundAlert{
			{
				Status:       alert.Status,
				Labels:       alert.Labels,
				Annotations:  alert.Annotations,
				StartsAt:     alert.StartsAt,
				EndsAt:       alert.EndsAt,
				Fingerprint:  alert.Fingerprint,
				GeneratorURL: alert.GeneratorURL,
			},
		},
		Receiver: fmt.Sprintf("proxy:%s", forwarder.Name),
		Status:   alert.Status,
	}

	// Send to proxy target
	if err := s.sendToProxyTarget(ctx, target, &outPayload); err != nil {
		s.logger.Error("failed to forward alert to proxy target",
			zap.Uint("forwarder_id", forwarder.ID),
			zap.String("alert_name", alertName),
			zap.String("target_url", target.TargetURL),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("inbound alert proxied to external target",
		zap.Uint("forwarder_id", forwarder.ID),
		zap.String("forwarder_name", forwarder.Name),
		zap.String("alert_name", alertName),
		zap.String("severity", severity),
		zap.String("target_url", target.TargetURL),
	)

	return nil
}

// sendToProxyTarget sends the payload to the proxy target.
func (s *AlertForwarderService) sendToProxyTarget(ctx context.Context, target *model.OutboundConfig, payload *InboundPayload) error {
	// Serialize payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal proxy payload: %w", err)
	}

	// Build request
	method := target.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequestWithContext(ctx, method, target.TargetURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Set headers
	if target.Headers != nil {
		for k, v := range target.Headers {
			req.Header.Set(k, v)
		}
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set timeout
	timeout := target.Timeout
	if timeout == 0 {
		timeout = 30000
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	// Send with retry
	retryTimes := target.RetryTimes
	if retryTimes == 0 {
		retryTimes = 3
	}
	retryInterval := target.RetryInterval
	if retryInterval == 0 {
		retryInterval = 100
	}

	var lastErr error
	for i := 0; i <= retryTimes; i++ {
		if i > 0 {
			time.Sleep(time.Duration(retryInterval) * time.Millisecond)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return fmt.Errorf("proxy failed after %d retries: %w", retryTimes, lastErr)
}

// generateFingerprint generates a fingerprint from labels.
func generateFingerprint(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
