package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/testutil"
)

// ---------------------------------------------------------------------------
// isThrottled integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

// Test_isThrottled_per_fingerprint verifies that the throttle logic is scoped
// per fingerprint. Two alerts with different fingerprints but the same rule+media
// must NOT throttle each other — reaching MaxNotifications on one fingerprint
// should not silence a different alert.
func Test_isThrottled_per_fingerprint(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	recordRepo := repository.NewNotifyRecordRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		recordRepo: recordRepo,
		logger:     logger,
	}

	ctx := context.Background()

	rule := &model.NotifyRule{
		MaxNotifications: 1, // cap at 1 notification per fingerprint
		RepeatInterval:   0, // disable repeat-interval check for this test
	}
	nc := &model.NotifyConfig{MediaID: 10}

	fpA := "fp-alert-aaa-111"
	fpB := "fp-alert-bbb-222"

	// Initially, neither fingerprint should be throttled.
	assert.False(t, svc.isThrottled(ctx, rule, nc, fpA),
		"fingerprint A should NOT be throttled before any sends")
	assert.False(t, svc.isThrottled(ctx, rule, nc, fpB),
		"fingerprint B should NOT be throttled before any sends")

	// Create a "sent" record for fingerprint A.
	require.NoError(t, recordRepo.Create(ctx, &model.NotifyRecord{
		EventID:     1,
		ChannelID:   nc.MediaID,
		PolicyID:    1, // does not need to match rule.ID for this test
		Fingerprint: fpA,
		Status:      "sent",
	}))

	// Fingerprint A should now be throttled (count >= MaxNotifications).
	assert.True(t, svc.isThrottled(ctx, rule, nc, fpA),
		"fingerprint A should be throttled after reaching MaxNotifications")

	// Fingerprint B should still NOT be throttled — throttle is per-fingerprint.
	assert.False(t, svc.isThrottled(ctx, rule, nc, fpB),
		"fingerprint B should NOT be throttled just because fingerprint A hit its cap")
}

// Test_isThrottled_repeat_interval_per_fingerprint verifies that the repeat
// interval throttle is also scoped per fingerprint.
func Test_isThrottled_repeat_interval_per_fingerprint(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	recordRepo := repository.NewNotifyRecordRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		recordRepo: recordRepo,
		logger:     logger,
	}

	ctx := context.Background()

	rule := &model.NotifyRule{
		MaxNotifications: 0,   // no cap
		RepeatInterval:   300, // 5 minutes
	}
	nc := &model.NotifyConfig{MediaID: 20}

	fpX := "fp-repeat-xxx"
	fpY := "fp-repeat-yyy"

	// Create a recently-sent record for fingerprint X.
	// GORM autoCreateTime sets CreatedAt to now, which is within the 5-min window.
	require.NoError(t, recordRepo.Create(ctx, &model.NotifyRecord{
		EventID:     2,
		ChannelID:   nc.MediaID,
		PolicyID:    1,
		Fingerprint: fpX,
		Status:      "sent",
	}))

	// Fingerprint X should be throttled by repeat interval.
	assert.True(t, svc.isThrottled(ctx, rule, nc, fpX),
		"fingerprint X should be throttled within repeat interval")

	// Fingerprint Y should NOT be throttled — no prior send record.
	assert.False(t, svc.isThrottled(ctx, rule, nc, fpY),
		"fingerprint Y should NOT be throttled (no prior send, different fingerprint)")
}

// ---------------------------------------------------------------------------
// FindMatchingRules tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

// Test_FindMatchingRules_ByLabels verifies that only rules whose match_labels
// are a subset of the event labels are returned.
func Test_FindMatchingRules_ByLabels(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	// Create a rule that matches env=prod
	rule := &model.NotifyRule{
		Name:          "prod-rule",
		IsEnabled:     true,
		MatchLabels:   model.JSONLabels{"env": "prod"},
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, rule))

	// Event with matching labels
	event := &model.AlertEvent{
		Severity: model.SeverityWarning,
		Labels:   model.JSONLabels{"env": "prod", "job": "api"},
	}

	matched, err := svc.FindMatchingRules(ctx, event, nil)
	require.NoError(t, err)
	require.Len(t, matched, 1, "should match exactly 1 rule")
	assert.Equal(t, "prod-rule", matched[0].Name)
}

// Test_FindMatchingRules_BySeverity verifies that the severity filter works
// correctly: only rules whose severities include the event severity match.
func Test_FindMatchingRules_BySeverity(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	// Rule that only matches critical alerts
	criticalRule := &model.NotifyRule{
		Name:          "critical-only",
		IsEnabled:     true,
		Severities:    "critical",
		MatchLabels:   model.JSONLabels{"env": "prod"},
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, criticalRule))

	// Rule that matches all severities (empty = all)
	allRule := &model.NotifyRule{
		Name:          "all-severities",
		IsEnabled:     true,
		Severities:    "",
		MatchLabels:   model.JSONLabels{"env": "prod"},
		NotifyConfigs: `[{"media_id":2}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, allRule))

	// Event with warning severity
	event := &model.AlertEvent{
		Severity: model.SeverityWarning,
		Labels:   model.JSONLabels{"env": "prod"},
	}

	matched, err := svc.FindMatchingRules(ctx, event, nil)
	require.NoError(t, err)

	// Should match "all-severities" (empty = all) but NOT "critical-only"
	names := make([]string, len(matched))
	for i, r := range matched {
		names[i] = r.Name
	}
	assert.Contains(t, names, "all-severities", "empty severities should match all")
	assert.NotContains(t, names, "critical-only", "critical-only should not match warning events")
}

// Test_FindMatchingRules_DisabledRule_Excluded verifies that disabled rules
// are not returned by FindMatchingRules.
func Test_FindMatchingRules_DisabledRule_Excluded(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	disabledRule := &model.NotifyRule{
		Name:          "disabled-rule",
		IsEnabled:     false,
		MatchLabels:   model.JSONLabels{"env": "prod"},
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, disabledRule))

	event := &model.AlertEvent{
		Severity: model.SeverityWarning,
		Labels:   model.JSONLabels{"env": "prod"},
	}

	matched, err := svc.FindMatchingRules(ctx, event, nil)
	require.NoError(t, err)

	for _, r := range matched {
		assert.NotEqual(t, "disabled-rule", r.Name, "disabled rules should not be returned")
	}
}

// Test_FindMatchingRules_NoMatch verifies that an empty slice is returned
// when no rules match the event.
func Test_FindMatchingRules_NoMatch(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	rule := &model.NotifyRule{
		Name:          "prod-only",
		IsEnabled:     true,
		MatchLabels:   model.JSONLabels{"env": "prod"},
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, rule))

	// Event with non-matching labels
	event := &model.AlertEvent{
		Severity: model.SeverityWarning,
		Labels:   model.JSONLabels{"env": "staging"},
	}

	matched, err := svc.FindMatchingRules(ctx, event, nil)
	require.NoError(t, err)
	assert.Empty(t, matched, "should return empty when no rules match")
}

// Test_ProcessEvent_DisabledRule_Skips verifies that processing an event
// against a disabled rule returns nil without sending notifications.
func Test_ProcessEvent_DisabledRule_Skips(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	// Create a disabled rule
	rule := &model.NotifyRule{
		Name:          "disabled-for-process",
		IsEnabled:     false,
		Severities:    "critical,warning",
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, rule))

	event := &model.AlertEvent{
		AlertName: "TestAlert",
		Severity:  model.SeverityWarning,
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"env": "prod"},
	}

	err := svc.ProcessEvent(ctx, event, rule.ID)
	assert.NoError(t, err, "disabled rule should return nil without error")
}

// TestProcessEvent_SeverityMismatch_Skips verifies that an event whose severity
// does not match the rule's severity filter is silently skipped.
func TestProcessEvent_SeverityMismatch_Skips(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)
	logger := zap.NewNop()

	svc := &NotifyRuleService{
		ruleRepo: ruleRepo,
		logger:   logger,
	}

	ctx := context.Background()

	// Rule only accepts critical
	rule := &model.NotifyRule{
		Name:          "critical-process",
		IsEnabled:     true,
		Severities:    "critical",
		NotifyConfigs: `[{"media_id":1}]`,
	}
	require.NoError(t, ruleRepo.Create(ctx, rule))

	event := &model.AlertEvent{
		AlertName: "TestAlert",
		Severity:  model.SeverityWarning, // mismatch
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"env": "prod"},
	}

	err := svc.ProcessEvent(ctx, event, rule.ID)
	assert.NoError(t, err, "severity mismatch should be silently skipped")
}
