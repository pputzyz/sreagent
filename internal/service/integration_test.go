package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		title      string
		severity   string
		channelID  uint
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
