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
	Alerts      []InboundAlert `json:"alerts"`
	Receiver    string         `json:"receiver"`
	Status      string         `json:"status"`
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

	// 7. Process each alert
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

// processInboundAlert processes a single inbound alert.
func (s *AlertForwarderService) processInboundAlert(ctx context.Context, forwarder *model.AlertForwarder, alert *InboundAlert, payload *InboundPayload) error {
	// Apply severity mapping if enabled
	severity := ""
	if sev, ok := alert.Labels["severity"]; ok {
		severity = sev
	}

	if forwarder.SeverityMapping != nil && forwarder.SeverityMapping.Enabled {
		shouldMap := forwarder.SeverityMapping.Direction == model.SeverityMappingDirInbound ||
			forwarder.SeverityMapping.Direction == model.SeverityMappingDirBoth

		if shouldMap {
			if mapped, ok := forwarder.SeverityMapping.Mapping[severity]; ok {
				// Store original severity in labels
				alert.Labels["original_severity"] = severity
				alert.Labels["severity"] = mapped
				severity = mapped
			} else if forwarder.SeverityMapping.DefaultSeverity != "" {
				alert.Labels["original_severity"] = severity
				alert.Labels["severity"] = forwarder.SeverityMapping.DefaultSeverity
				severity = forwarder.SeverityMapping.DefaultSeverity
			}
		}
	}

	// Build AlertEvent
	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "Unknown"
	}

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

	// Apply platform capabilities
	caps := forwarder.PlatformCapabilities
	if caps == nil {
		caps = &model.PlatformCapabilitiesConfig{
			EnableNotification: true,
		}
	}

	// Save event to database if event repository is available
	if s.eventRepo != nil {
		if err := s.eventRepo.Create(ctx, event); err != nil {
			s.logger.Error("failed to save inbound alert event",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("fingerprint", alert.Fingerprint),
				zap.Error(err),
			)
			// Continue even if save fails - still try to route
		}
	}

	// Platform capability: Inhibition check
	if caps.EnableInhibition && s.inhibitorSvc != nil {
		// Get currently firing events for inhibition check
		var firingEvents []model.AlertEvent
		if s.eventRepo != nil {
			firingEvents, _, _ = s.eventRepo.List(ctx, "firing", "", 1, 2000)
		}
		if s.inhibitorSvc.IsInhibited(ctx, event, firingEvents) {
			s.logger.Info("inbound alert inhibited",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
				zap.String("fingerprint", alert.Fingerprint),
			)
			return nil
		}
	}

	// Platform capability: Mute check
	if caps.EnableMute && s.muteSvc != nil {
		if s.muteSvc.IsAlertMuted(ctx, event) {
			s.logger.Info("inbound alert muted",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
				zap.String("fingerprint", alert.Fingerprint),
			)
			return nil
		}
	}

	// Platform capability: Notification routing
	if caps.EnableNotification && s.notifySvc != nil {
		if err := s.notifySvc.RouteAlert(ctx, event); err != nil {
			s.logger.Error("failed to route inbound alert notification",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("alert_name", alertName),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("inbound alert processed",
		zap.Uint("forwarder_id", forwarder.ID),
		zap.String("forwarder_name", forwarder.Name),
		zap.String("alert_name", alertName),
		zap.String("severity", severity),
		zap.String("fingerprint", alert.Fingerprint),
		zap.Bool("enable_notification", caps.EnableNotification),
		zap.Bool("enable_mute", caps.EnableMute),
		zap.Bool("enable_inhibition", caps.EnableInhibition),
	)

	return nil
}

// generateFingerprint generates a fingerprint from labels.
func generateFingerprint(labels map[string]string) string {
	// Sort labels and create a hash
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	// Simple fingerprint - in production, use a proper hash
	return strings.Join(parts, ",")
}
