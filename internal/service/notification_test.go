package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/testutil"
)

// ---------------------------------------------------------------------------
// buildEmailBody tests (pure function)
// ---------------------------------------------------------------------------

func Test_buildEmailBody_basic_fields(t *testing.T) {
	event := &model.AlertEvent{
		AlertName: "HighCPU",
		Severity:  model.SeverityCritical,
		Status:    model.EventStatusFiring,
		FiredAt:   time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Labels:    model.JSONLabels{"instance": "web-1", "env": "prod"},
		Annotations: model.JSONLabels{"summary": "CPU usage > 90%"},
	}

	body := buildEmailBody(event, nil)

	assert.Contains(t, body, "Alert: HighCPU")
	assert.Contains(t, body, "Severity: critical")
	assert.Contains(t, body, "Status: firing")
	assert.Contains(t, body, "instance: web-1")
	assert.Contains(t, body, "env: prod")
	assert.Contains(t, body, "summary: CPU usage > 90%")
	assert.NotContains(t, body, "AI Analysis")
}

func Test_buildEmailBody_with_analysis(t *testing.T) {
	event := &model.AlertEvent{
		AlertName: "DiskFull",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusFiring,
		FiredAt:   time.Now(),
	}

	analysis := &AlertAnalysis{
		Summary: "Disk /data is 95% full. Log rotation may have failed.",
	}

	body := buildEmailBody(event, analysis)

	assert.Contains(t, body, "AI Analysis:")
	assert.Contains(t, body, "Disk /data is 95% full")
}

func Test_buildEmailBody_empty_labels_and_annotations(t *testing.T) {
	event := &model.AlertEvent{
		AlertName: "SimpleAlert",
		Severity:  model.SeverityInfo,
		Status:    model.EventStatusResolved,
		FiredAt:   time.Now(),
	}

	body := buildEmailBody(event, nil)

	assert.Contains(t, body, "Alert: SimpleAlert")
	assert.NotContains(t, body, "Labels:")
	assert.NotContains(t, body, "Annotations:")
}

func Test_buildEmailBody_resolved_status(t *testing.T) {
	event := &model.AlertEvent{
		AlertName: "ResolvedAlert",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusResolved,
		FiredAt:   time.Now(),
	}

	body := buildEmailBody(event, nil)

	assert.Contains(t, body, "Status: resolved")
}

// ---------------------------------------------------------------------------
// extractWebhookURL tests (pure function)
// ---------------------------------------------------------------------------

func Test_extractWebhookURL_valid(t *testing.T) {
	config := `{"webhook_url": "https://hooks.example.com/alert"}`
	url, err := extractWebhookURL(config)

	require.NoError(t, err)
	assert.Equal(t, "https://hooks.example.com/alert", url)
}

func Test_extractWebhookURL_empty_url(t *testing.T) {
	config := `{"webhook_url": ""}`
	_, err := extractWebhookURL(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_url is empty")
}

func Test_extractWebhookURL_missing_field(t *testing.T) {
	config := `{"other_field": "value"}`
	_, err := extractWebhookURL(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook_url is empty")
}

func Test_extractWebhookURL_invalid_json(t *testing.T) {
	config := `{invalid json`
	_, err := extractWebhookURL(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse channel config")
}

// ---------------------------------------------------------------------------
// customWebhookPayload serialization tests
// ---------------------------------------------------------------------------

func Test_customWebhookPayload_serialization(t *testing.T) {
	payload := customWebhookPayload{
		EventID:   42,
		AlertName: "TestAlert",
		Severity:  "critical",
		Status:    "firing",
		Labels:    map[string]string{"instance": "web-1"},
		Annotations: map[string]string{"summary": "Test"},
		FiredAt:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Source:    "prometheus",
		AISummary: "Root cause: memory leak",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded customWebhookPayload
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, uint(42), decoded.EventID)
	assert.Equal(t, "TestAlert", decoded.AlertName)
	assert.Equal(t, "critical", decoded.Severity)
	assert.Equal(t, "Root cause: memory leak", decoded.AISummary)
}

func Test_customWebhookPayload_omitempty_ai_summary(t *testing.T) {
	payload := customWebhookPayload{
		EventID:   1,
		AlertName: "Test",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "ai_summary",
		"empty AISummary should be omitted from JSON")
}

// ---------------------------------------------------------------------------
// emailChannelConfig deserialization tests
// ---------------------------------------------------------------------------

func Test_emailChannelConfig_deserialization(t *testing.T) {
	configJSON := `{
		"smtp_host": "smtp.example.com",
		"smtp_port": 587,
		"smtp_tls": true,
		"username": "alert@example.com",
		"password": "secret",
		"from": "SREAgent <alert@example.com>",
		"recipients": ["ops@example.com", "team@example.com"]
	}`

	var cfg emailChannelConfig
	require.NoError(t, json.Unmarshal([]byte(configJSON), &cfg))

	assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
	assert.Equal(t, 587, cfg.SMTPPort)
	assert.True(t, cfg.SMTPTLS)
	assert.Equal(t, "alert@example.com", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, "SREAgent <alert@example.com>", cfg.From)
	assert.Len(t, cfg.Recipients, 2)
	assert.Contains(t, cfg.Recipients, "ops@example.com")
}

func Test_emailChannelConfig_defaults(t *testing.T) {
	configJSON := `{"smtp_host": "smtp.test.com", "recipients": ["a@b.com"]}`

	var cfg emailChannelConfig
	require.NoError(t, json.Unmarshal([]byte(configJSON), &cfg))

	assert.Equal(t, "smtp.test.com", cfg.SMTPHost)
	assert.Equal(t, 0, cfg.SMTPPort, "default port should be 0 (caller applies 587)")
	assert.False(t, cfg.SMTPTLS, "default TLS should be false")
}

// ---------------------------------------------------------------------------
// customWebhookConfig deserialization tests
// ---------------------------------------------------------------------------

func Test_customWebhookConfig_deserialization(t *testing.T) {
	configJSON := `{
		"url": "https://hooks.example.com/alert",
		"method": "PUT",
		"headers": {"Authorization": "Bearer xxx", "X-Custom": "value"},
		"timeout_seconds": 15
	}`

	var cfg customWebhookConfig
	require.NoError(t, json.Unmarshal([]byte(configJSON), &cfg))

	assert.Equal(t, "https://hooks.example.com/alert", cfg.URL)
	assert.Equal(t, "PUT", cfg.Method)
	assert.Equal(t, "Bearer xxx", cfg.Headers["Authorization"])
	assert.Equal(t, 15, cfg.TimeoutSeconds)
}

func Test_customWebhookConfig_defaults(t *testing.T) {
	configJSON := `{"url": "https://example.com"}`

	var cfg customWebhookConfig
	require.NoError(t, json.Unmarshal([]byte(configJSON), &cfg))

	assert.Equal(t, "https://example.com", cfg.URL)
	assert.Empty(t, cfg.Method, "default method should be empty (caller applies POST)")
	assert.Equal(t, 0, cfg.TimeoutSeconds, "default timeout should be 0 (caller applies 10)")
}

// ---------------------------------------------------------------------------
// Integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

func Test_RouteAlert_skips_silenced_event(t *testing.T) {
	_ = testutil.TestDB(t) // skip if no test DB
	logger := testutil.TestLogger()

	svc := NewNotificationService(
		nil, nil, nil, nil, nil, nil, nil, nil, logger,
	)

	futureTime := time.Now().Add(1 * time.Hour)
	event := &model.AlertEvent{
		BaseModel:     model.BaseModel{ID: 1},
		AlertName:     "SilencedAlert",
		Severity:      model.SeverityWarning,
		Status:        model.EventStatusSilenced,
		SilencedUntil: &futureTime,
		Labels:        model.JSONLabels{"alertname": "SilencedAlert"},
	}

	err := svc.RouteAlert(context.Background(), event)
	assert.NoError(t, err, "silenced alerts should be skipped without error")
}

func Test_RouteAlert_no_matching_policies(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	policyRepo := repository.NewNotifyPolicyRepository(db)
	recordRepo := repository.NewNotifyRecordRepository(db)

	svc := NewNotificationService(
		nil, policyRepo, recordRepo, nil, nil, nil, nil, nil, logger,
	)

	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 999},
		AlertName: "NoMatch",
		Severity:  model.SeverityInfo,
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"alertname": "NoMatch", "env": "staging"},
	}

	err := svc.RouteAlert(context.Background(), event)
	assert.NoError(t, err, "no matching policies should return nil error")
}

func Test_RouteAlert_matching_policy_records_failure(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	// Create a channel pointing to a dead server
	channel := &model.NotifyChannel{
		Name:      "test-webhook-channel",
		Type:      model.ChannelTypeCustom,
		Config:    `{"url": "http://127.0.0.1:19999/test"}`,
		IsEnabled: true,
	}
	require.NoError(t, db.Create(channel).Error)

	// Create a policy matching all alerts
	policy := &model.NotifyPolicy{
		Name:         "test-all-policy",
		MatchLabels:  model.JSONLabels{},
		Severities:   "",
		ChannelID:    channel.ID,
		IsEnabled:    true,
		TemplateName: "default",
	}
	require.NoError(t, db.Create(policy).Error)

	policyRepo := repository.NewNotifyPolicyRepository(db)
	recordRepo := repository.NewNotifyRecordRepository(db)

	svc := NewNotificationService(
		nil, policyRepo, recordRepo, nil, nil, nil, nil, nil, logger,
	)

	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		AlertName: "TestAlert",
		Severity:  model.SeverityCritical,
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"alertname": "TestAlert"},
		FiredAt:   time.Now(),
	}

	// RouteAlert will try to send — the webhook will fail (no server listening)
	err := svc.RouteAlert(context.Background(), event)
	assert.NoError(t, err, "RouteAlert itself should not return error even if send fails")

	// Verify a record was created
	var records []model.NotifyRecord
	db.Where("event_id = ?", 100).Find(&records)
	require.Len(t, records, 1, "should create exactly one notify record")
	assert.Equal(t, "failed", records[0].Status, "webhook to dead server should be recorded as failed")
}

func Test_RouteAlert_throttled_policy(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	channel := &model.NotifyChannel{
		Name:      "throttle-test-channel",
		Type:      model.ChannelTypeCustom,
		Config:    `{"url": "http://127.0.0.1:19999/test"}`,
		IsEnabled: true,
	}
	require.NoError(t, db.Create(channel).Error)

	policy := &model.NotifyPolicy{
		Name:           "throttle-policy",
		MatchLabels:    model.JSONLabels{},
		ChannelID:      channel.ID,
		IsEnabled:      true,
		ThrottleMinutes: 60,
		TemplateName:   "default",
	}
	require.NoError(t, db.Create(policy).Error)

	policyRepo := repository.NewNotifyPolicyRepository(db)
	recordRepo := repository.NewNotifyRecordRepository(db)

	svc := NewNotificationService(
		nil, policyRepo, recordRepo, nil, nil, nil, nil, nil, logger,
	)

	// Insert a recent "sent" record to trigger throttle
	recentRecord := &model.NotifyRecord{
		EventID:   50,
		ChannelID: channel.ID,
		PolicyID:  policy.ID,
		Status:    "sent",
	}
	require.NoError(t, db.Create(recentRecord).Error)

	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 101},
		AlertName: "ThrottledAlert",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"alertname": "ThrottledAlert"},
	}

	err := svc.RouteAlert(context.Background(), event)
	assert.NoError(t, err)

	// Should have a "throttled" record for event 101
	var records []model.NotifyRecord
	db.Where("event_id = ? AND status = ?", 101, "throttled").Find(&records)
	assert.Len(t, records, 1, "should record the throttled notification")
}

func Test_RouteAlert_severity_filtered_policy(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	channel := &model.NotifyChannel{
		Name:      "critical-only-channel",
		Type:      model.ChannelTypeCustom,
		Config:    `{"url": "http://127.0.0.1:19999/test"}`,
		IsEnabled: true,
	}
	require.NoError(t, db.Create(channel).Error)

	// Policy only matches critical alerts
	policy := &model.NotifyPolicy{
		Name:         "critical-only-policy",
		MatchLabels:  model.JSONLabels{},
		Severities:   "critical",
		ChannelID:    channel.ID,
		IsEnabled:    true,
		TemplateName: "default",
	}
	require.NoError(t, db.Create(policy).Error)

	policyRepo := repository.NewNotifyPolicyRepository(db)
	recordRepo := repository.NewNotifyRecordRepository(db)

	svc := NewNotificationService(
		nil, policyRepo, recordRepo, nil, nil, nil, nil, nil, logger,
	)

	// Send a warning event — should NOT match the critical-only policy
	event := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 102},
		AlertName: "WarningAlert",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"alertname": "WarningAlert"},
	}

	err := svc.RouteAlert(context.Background(), event)
	assert.NoError(t, err)

	var records []model.NotifyRecord
	db.Where("event_id = ?", 102).Find(&records)
	assert.Empty(t, records, "warning event should not match critical-only policy")
}

// ---------------------------------------------------------------------------
// NewNotificationService constructor test
// ---------------------------------------------------------------------------

func Test_NewNotificationService_returns_non_nil(t *testing.T) {
	logger := testutil.TestLogger()
	svc := NewNotificationService(nil, nil, nil, nil, nil, nil, nil, nil, logger)
	assert.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// NotifyRule DB integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

func Test_NotifyRule_MatchByLabels_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	// Create a NotifyRule with match_labels
	rule := &model.NotifyRule{
		Name:         "match-env-prod",
		IsEnabled:    true,
		MatchLabels:  model.JSONLabels{"env": "prod", "severity": "critical"},
		Severities:   "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	// Call FindMatchingRules with labels that include the required ones
	matched, err := ruleRepo.FindMatchingRules(context.Background(),
		map[string]string{"env": "prod", "severity": "critical", "instance": "web-1"}, "critical")
	require.NoError(t, err)
	require.Len(t, matched, 1, "should match the rule")
	assert.Equal(t, rule.ID, matched[0].ID)
}

func Test_NotifyRule_SeverityFilter_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	// Create a NotifyRule that only matches critical and warning
	rule := &model.NotifyRule{
		Name:           "critical-warning-only",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{},
		Severities:     "critical,warning",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	// Call FindMatchingRules with severity "info" — should NOT match
	matched, err := ruleRepo.FindMatchingRules(context.Background(),
		map[string]string{"alertname": "TestAlert"}, "info")
	require.NoError(t, err)
	assert.Empty(t, matched, "info severity should not match critical,warning filter")
}

func Test_NotifyRule_BatchUpdateEnabled_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	// Create 3 rules (all enabled by default)
	r1 := &model.NotifyRule{Name: "batch-rule-1", IsEnabled: true, RepeatInterval: 3600}
	r2 := &model.NotifyRule{Name: "batch-rule-2", IsEnabled: true, RepeatInterval: 3600}
	r3 := &model.NotifyRule{Name: "batch-rule-3", IsEnabled: true, RepeatInterval: 3600}
	require.NoError(t, ruleRepo.Create(context.Background(), r1))
	require.NoError(t, ruleRepo.Create(context.Background(), r2))
	require.NoError(t, ruleRepo.Create(context.Background(), r3))

	// Batch disable r1 and r2
	err := ruleRepo.BatchUpdateEnabled(context.Background(), []uint{r1.ID, r2.ID}, false)
	require.NoError(t, err)

	// Assert r1 and r2 are disabled
	fetched1, err := ruleRepo.GetByID(context.Background(), r1.ID)
	require.NoError(t, err)
	assert.False(t, fetched1.IsEnabled, "r1 should be disabled")

	fetched2, err := ruleRepo.GetByID(context.Background(), r2.ID)
	require.NoError(t, err)
	assert.False(t, fetched2.IsEnabled, "r2 should be disabled")

	// Assert r3 is still enabled
	fetched3, err := ruleRepo.GetByID(context.Background(), r3.ID)
	require.NoError(t, err)
	assert.True(t, fetched3.IsEnabled, "r3 should still be enabled")
}

// ---------------------------------------------------------------------------
// NotifyRule FindMatchingRules DB integration tests (require SREAGENT_TEST_DSN)
// Run with: SREAGENT_TEST_DSN="user:pass@tcp(host:port)/db" go test -run DB
// ---------------------------------------------------------------------------

// Test_NotifyRule_FindMatchingRules_LabelSubset_DB verifies that a rule with
// match_labels={"env":"prod","team":"infra"} matches event labels that are a
// superset (i.e., the rule's labels are a subset of the event labels).
func Test_NotifyRule_FindMatchingRules_LabelSubset_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	// Create a rule with specific match_labels
	rule := &model.NotifyRule{
		Name:           "subset-match-rule",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{"env": "prod", "team": "infra"},
		Severities:     "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	// Call FindMatchingRules with labels that are a superset of the rule's match_labels
	eventLabels := map[string]string{
		"env":      "prod",
		"team":     "infra",
		"instance": "web-1",
		"region":   "us-east-1",
	}
	matched, err := ruleRepo.FindMatchingRules(context.Background(), eventLabels, "critical")
	require.NoError(t, err)
	require.Len(t, matched, 1, "rule should match when event labels are a superset")
	assert.Equal(t, rule.ID, matched[0].ID)
	assert.Equal(t, "subset-match-rule", matched[0].Name)
}

// Test_NotifyRule_FindMatchingRules_NoMatch_DB verifies that a rule with
// match_labels={"env":"prod"} does NOT match event labels={"env":"staging"}.
func Test_NotifyRule_FindMatchingRules_NoMatch_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	// Create a rule that only matches env=prod
	rule := &model.NotifyRule{
		Name:           "prod-only-rule",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{"env": "prod"},
		Severities:     "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	// Call FindMatchingRules with env=staging — should NOT match
	eventLabels := map[string]string{
		"env":      "staging",
		"instance": "web-2",
	}
	matched, err := ruleRepo.FindMatchingRules(context.Background(), eventLabels, "warning")
	require.NoError(t, err)
	assert.Empty(t, matched, "rule with env=prod should not match env=staging event")
}
