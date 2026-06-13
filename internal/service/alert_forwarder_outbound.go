package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/labelmatch"
)

// OutboundRenderData is the data context for outbound body template rendering.
type OutboundRenderData struct {
	AlertName   string            `json:"alert_name"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Source      string            `json:"source"`
	EventID     uint              `json:"event_id"`
	FiredAt     string            `json:"fired_at"`
	RuleName    string            `json:"rule_name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Value       string            `json:"value"`
	Duration    string            `json:"duration"`
	FireCount   int               `json:"fire_count"`
	// Forwarder-specific
	ForwarderName string `json:"forwarder_name"`
	ForwarderID   uint   `json:"forwarder_id"`
}

// ProcessOutbound processes outbound forwarding for an alert event.
// This should be called when an alert event is created or updated.
func (s *AlertForwarderService) ProcessOutbound(ctx context.Context, event *model.AlertEvent) error {
	// Get all enabled outbound forwarders
	forwarders, err := s.forwarderRepo.ListEnabledByDirection(ctx, model.ForwarderDirectionOutbound)
	if err != nil {
		return fmt.Errorf("failed to list outbound forwarders: %w", err)
	}

	if len(forwarders) == 0 {
		return nil
	}

	// Process each forwarder
	for _, forwarder := range forwarders {
		// Check label match
		if len(forwarder.MatchLabels) > 0 {
			if !labelmatch.Match(map[string]string(event.Labels), map[string]string(forwarder.MatchLabels)) {
				continue
			}
		}

		// Process outbound for this forwarder
		if err := s.processOutboundForForwarder(ctx, &forwarder, event); err != nil {
			s.logger.Error("failed to process outbound forwarding",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("forwarder_name", forwarder.Name),
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
			// Continue with other forwarders
		}
	}

	return nil
}

// processOutboundForForwarder processes outbound forwarding for a specific forwarder.
func (s *AlertForwarderService) processOutboundForForwarder(ctx context.Context, forwarder *model.AlertForwarder, event *model.AlertEvent) error {
	if forwarder.OutboundConfig == nil {
		return fmt.Errorf("outbound config is nil")
	}

	// Apply outbound severity mapping
	severity := string(event.Severity)
	originalSeverity := severity

	if mapped, applied := forwarder.OutboundSeverityMapping.ApplySeverityMapping(severity); applied {
		severity = mapped
	}

	// Build render data
	data := OutboundRenderData{
		AlertName:     event.AlertName,
		Severity:      severity,
		Status:        string(event.Status),
		Source:        event.Source,
		EventID:       event.ID,
		FiredAt:       event.FiredAt.Format(time.RFC3339),
		Labels:        map[string]string(event.Labels),
		Annotations:   map[string]string(event.Annotations),
		FireCount:     event.FireCount,
		ForwarderName: forwarder.Name,
		ForwarderID:   forwarder.ID,
	}

	// Get value from annotations
	if val, ok := event.Annotations["value"]; ok {
		data.Value = val
	}
	if dur, ok := event.Annotations["duration"]; ok {
		data.Duration = dur
	}

	// Determine target
	if forwarder.OutboundConfig.TargetMediaID != nil {
		// Send via NotifyMedia
		return s.sendViaMedia(ctx, forwarder, &data, originalSeverity, severity)
	}

	if forwarder.OutboundConfig.TargetURL != "" {
		// Send via direct HTTP
		return s.sendViaHTTP(ctx, forwarder, &data)
	}

	return fmt.Errorf("no target configured")
}

// sendViaMedia sends the alert via a configured NotifyMedia.
func (s *AlertForwarderService) sendViaMedia(ctx context.Context, forwarder *model.AlertForwarder, data *OutboundRenderData, originalSeverity, mappedSeverity string) error {
	media, err := s.mediaRepo.GetByID(ctx, *forwarder.OutboundConfig.TargetMediaID)
	if err != nil {
		return fmt.Errorf("failed to load target media: %w", err)
	}

	// Build content
	content := fmt.Sprintf("[%s] %s - %s", data.Severity, data.AlertName, data.Status)

	// Build template data for media service
	// Parse FiredAt string to time.Time
	var firedAtTime time.Time
	if data.FiredAt != "" {
		if t, err := time.Parse(time.RFC3339, data.FiredAt); err == nil {
			firedAtTime = t
		} else {
			firedAtTime = time.Now()
		}
	} else {
		firedAtTime = time.Now()
	}

	templateData := &TemplateData{
		AlertName:   data.AlertName,
		Severity:    data.Severity,
		Status:      data.Status,
		Source:      data.Source,
		EventID:     data.EventID,
		FiredAt:     firedAtTime,
		Labels:      data.Labels,
		Annotations: data.Annotations,
		Value:       data.Value,
		Duration:    data.Duration,
		FireCount:   data.FireCount,
	}

	// Send notification
	if s.mediaSvc != nil {
		if err := s.mediaSvc.SendNotification(ctx, media, content, templateData); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
	}

	s.logger.Info("outbound alert forwarded via media",
		zap.Uint("forwarder_id", forwarder.ID),
		zap.String("forwarder_name", forwarder.Name),
		zap.String("media_name", media.Name),
		zap.String("original_severity", originalSeverity),
		zap.String("mapped_severity", mappedSeverity),
	)

	return nil
}

// sendViaHTTP sends the alert via direct HTTP POST.
func (s *AlertForwarderService) sendViaHTTP(ctx context.Context, forwarder *model.AlertForwarder, data *OutboundRenderData) error {
	config := forwarder.OutboundConfig

	// SSRF protection: validate target URL
	if err := validateEndpoint(ctx, config.TargetURL); err != nil {
		return fmt.Errorf("outbound target URL blocked by SSRF policy: %w", err)
	}

	// Render body template
	var bodyStr string
	if config.BodyTemplate != "" {
		// Use safe template with restricted function set and missing key error
		tmpl, err := template.New("body").Option("missingkey=error").Parse(config.BodyTemplate)
		if err != nil {
			return fmt.Errorf("invalid body template: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to render body template: %w", err)
		}
		bodyStr = buf.String()
	} else {
		// Default: JSON payload
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		bodyStr = string(jsonData)
	}

	// Build request settings
	method := config.Method
	if method == "" {
		method = "POST"
	}

	// Set timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30000
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	// Send with retry
	retryTimes := config.RetryTimes
	if retryTimes == 0 {
		retryTimes = 3
	}
	retryInterval := config.RetryInterval
	if retryInterval == 0 {
		retryInterval = 100
	}

	var lastErr error
	for i := 0; i <= retryTimes; i++ {
		if i > 0 {
			time.Sleep(time.Duration(retryInterval) * time.Millisecond)
		}

		// Create new request for each retry (body is consumed after each send)
		req, err := http.NewRequestWithContext(ctx, method, config.TargetURL, bytes.NewBufferString(bodyStr))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		if config.Headers != nil {
			for k, v := range config.Headers {
				req.Header.Set(k, v)
			}
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.logger.Info("outbound alert forwarded via HTTP",
				zap.Uint("forwarder_id", forwarder.ID),
				zap.String("forwarder_name", forwarder.Name),
				zap.String("target_url", config.TargetURL),
				zap.Int("status_code", resp.StatusCode),
			)
			return nil
		}

		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return fmt.Errorf("failed after %d retries: %w", retryTimes, lastErr)
}
