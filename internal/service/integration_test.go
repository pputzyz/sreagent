package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// Test_integration_synthetic_event_has_fingerprint verifies that events created
// by the webhook integration have a non-empty, deterministic fingerprint.
func Test_integration_synthetic_event_has_fingerprint(t *testing.T) {
	integrationID := uint(42)
	labels := map[string]string{"env": "prod", "job": "api-server"}
	title := "HighCPU"

	fp1 := generateIntegrationFingerprint(integrationID, labels, title)
	fp2 := generateIntegrationFingerprint(integrationID, labels, title)

	assert.NotEmpty(t, fp1, "fingerprint must not be empty")
	assert.Len(t, fp1, 32, "md5 hex fingerprint should be 32 chars")
	assert.Equal(t, fp1, fp2, "same inputs must produce identical fingerprints")
}

// Test_integration_synthetic_event_fingerprint_different_inputs verifies that
// different integration IDs, labels, or titles produce distinct fingerprints.
func Test_integration_synthetic_event_fingerprint_different_inputs(t *testing.T) {
	labels := map[string]string{"env": "prod"}

	fpSameIntegration := generateIntegrationFingerprint(1, labels, "AlertA")
	fpDiffIntegration := generateIntegrationFingerprint(2, labels, "AlertA")
	assert.NotEqual(t, fpSameIntegration, fpDiffIntegration,
		"different integration IDs should produce different fingerprints")

	fpDiffTitle := generateIntegrationFingerprint(1, labels, "AlertB")
	assert.NotEqual(t, fpSameIntegration, fpDiffTitle,
		"different titles should produce different fingerprints")

	fpDiffLabels := generateIntegrationFingerprint(1, map[string]string{"env": "staging"}, "AlertA")
	assert.NotEqual(t, fpSameIntegration, fpDiffLabels,
		"different labels should produce different fingerprints")
}

// Test_shared_integration_channel_reset_per_alert verifies that in shared mode
// batch processing, channelID resets to default for each alert. This prevents
// a routing-rule match on one alert from leaking into subsequent alerts.
func Test_shared_integration_channel_reset_per_alert(t *testing.T) {
	// Set up routing rules: one rule matches only severity=critical.
	rules := []model.RoutingRule{
		{
			IsEnabled:       true,
			TargetChannelID: 99,
			Conditions:      marshalFilterConditions(t, []model.FilterCondition{{Field: "severity", Operator: "eq", Value: "critical"}}),
		},
	}

	s := &IntegrationService{}

	// Alert 1: matches the rule → should get channel 99.
	ch1 := s.matchRoutingRule(rules, map[string]string{"job": "api"}, "critical")
	assert.Equal(t, uint(99), ch1, "critical alert should match routing rule")

	// Alert 2: does NOT match → should get 0 (no routing match).
	ch2 := s.matchRoutingRule(rules, map[string]string{"job": "api"}, "warning")
	assert.Equal(t, uint(0), ch2, "warning alert should NOT match critical-only rule")

	// Verify the shared-mode loop pattern: each iteration starts from integration default.
	// This simulates the production code in ReceiveAlerts where iterChannelID is
	// reset to integ.ChannelID each iteration.
	integDefaultChannelID := uint(5)
	type alertResult struct {
		title     string
		severity  string
		channelID uint
	}
	var results []alertResult

	alerts := []NormalizedAlert{
		{Title: "CriticalAlert", Severity: model.SeverityCritical, Labels: map[string]string{"job": "api"}},
		{Title: "WarningAlert", Severity: model.SeverityWarning, Labels: map[string]string{"job": "api"}},
		{Title: "AnotherCritical", Severity: model.SeverityCritical, Labels: map[string]string{"job": "web"}},
	}

	for _, alert := range alerts {
		// Reset to integration default each iteration (mirrors production code).
		iterChannelID := integDefaultChannelID
		targetID := s.matchRoutingRule(rules, alert.Labels, string(alert.Severity))
		if targetID > 0 {
			iterChannelID = targetID
		}
		results = append(results, alertResult{
			title:     alert.Title,
			severity:  string(alert.Severity),
			channelID: iterChannelID,
		})
	}

	require.Len(t, results, 3)
	assert.Equal(t, uint(99), results[0].channelID, "first critical alert → routed to channel 99")
	assert.Equal(t, uint(5), results[1].channelID, "warning alert → falls back to integ default 5")
	assert.Equal(t, uint(99), results[2].channelID, "second critical alert → routed to channel 99, not leaked from previous")
}

// marshalFilterConditions is a test helper to JSON-encode filter conditions.
func marshalFilterConditions(t *testing.T, conds []model.FilterCondition) string {
	t.Helper()
	data, err := json.Marshal(conds)
	require.NoError(t, err)
	return string(data)
}

// ---------------------------------------------------------------------------
// normaliseSeverity tests
// ---------------------------------------------------------------------------

func TestNormaliseSeverity_MapsCorrectly(t *testing.T) {
	tests := []struct {
		input    string
		expected model.AlertSeverity
	}{
		{"p0", model.SeverityCritical},
		{"critical", model.SeverityCritical},
		{"crit", model.SeverityCritical},
		{"error", model.SeverityCritical},
		{"high", model.SeverityCritical},
		{"p1", model.SeverityWarning},
		{"p2", model.SeverityWarning},
		{"warning", model.SeverityWarning},
		{"warn", model.SeverityWarning},
		{"medium", model.SeverityWarning},
		{"info", model.SeverityInfo},
		{"p3", model.SeverityInfo},
		{"p4", model.SeverityInfo},
		{"unknown", model.SeverityInfo},
		{"", model.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normaliseSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// normaliseStatus tests
// ---------------------------------------------------------------------------

func TestNormaliseStatus_MapsCorrectly(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"firing", "firing"},
		{"resolved", "resolved"},
		{"ok", "resolved"},
		{"normal", "resolved"},
		{"good", "resolved"},
		{"pending", "firing"},
		{"unknown", "firing"},
		{"", "firing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normaliseStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// normalise (dispatch) tests
// ---------------------------------------------------------------------------

func TestNormalise_StandardFormat_SingleAlert(t *testing.T) {
	s := &IntegrationService{logger: zap.NewNop()}
	body := []byte(`{"title":"HighCPU","severity":"critical","status":"firing","labels":{"env":"prod"}}`)

	alerts, err := s.normalise(model.IntegrationTypeStandard, body)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "HighCPU", alerts[0].Title)
	assert.Equal(t, model.SeverityCritical, alerts[0].Severity)
	assert.Equal(t, "firing", alerts[0].Status)
	assert.Equal(t, "prod", alerts[0].Labels["env"])
}

func TestNormalise_AlertManagerFormat(t *testing.T) {
	s := &IntegrationService{logger: zap.NewNop()}
	body := []byte(`{"alerts":[{"status":"firing","labels":{"alertname":"DiskFull","severity":"warning","instance":"host1"},"annotations":{"description":"Disk is full"},"startsAt":"2026-05-30T12:00:00Z"}]}`)

	alerts, err := s.normalise(model.IntegrationTypeAlertManager, body)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "DiskFull", alerts[0].Title)
	assert.Equal(t, model.SeverityWarning, alerts[0].Severity)
	assert.Equal(t, "firing", alerts[0].Status)
	assert.Equal(t, "Disk is full", alerts[0].Description)
}

func TestNormalise_GrafanaFormat(t *testing.T) {
	s := &IntegrationService{logger: zap.NewNop()}
	body := []byte(`{"alerts":[{"title":"CPU Alert","state":"alerting","labels":{"severity":"critical"},"annotations":{"summary":"CPU high"}}]}`)

	alerts, err := s.normalise(model.IntegrationTypeGrafana, body)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "CPU Alert", alerts[0].Title)
	assert.Equal(t, model.SeverityCritical, alerts[0].Severity)
	assert.Equal(t, "firing", alerts[0].Status)
}

func TestNormalise_GrafanaResolvedState(t *testing.T) {
	s := &IntegrationService{logger: zap.NewNop()}
	body := []byte(`{"alerts":[{"title":"CPU Alert","state":"ok","labels":{}}]}`)

	alerts, err := s.normalise(model.IntegrationTypeGrafana, body)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "resolved", alerts[0].Status)
}

func TestNormalise_InvalidJSON_ReturnsError(t *testing.T) {
	s := &IntegrationService{logger: zap.NewNop()}
	body := []byte(`not valid json`)

	_, err := s.normalise(model.IntegrationTypeStandard, body)
	assert.Error(t, err, "invalid JSON should return error")
}

// ---------------------------------------------------------------------------
// expandPipelineTemplate tests
// ---------------------------------------------------------------------------

func TestExpandPipelineTemplate_ReplacesPlaceholders(t *testing.T) {
	alert := NormalizedAlert{
		Title:       "HighCPU",
		Severity:    model.SeverityCritical,
		Description: "CPU at 99%",
		Labels:      map[string]string{"env": "prod", "service": "api"},
	}

	tests := []struct {
		name     string
		tmpl     string
		expected string
	}{
		{"title", "{{title}}", "HighCPU"},
		{"severity", "{{severity}}", "critical"},
		{"description", "{{description}}", "CPU at 99%"},
		{"label", "{{labels.env}}", "prod"},
		{"mixed", "{{title}} [{{severity}}] on {{labels.service}}", "HighCPU [critical] on api"},
		{"no_placeholder", "static text", "static text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPipelineTemplate(tt.tmpl, alert)
			assert.Equal(t, tt.expected, result)
		})
	}
}
