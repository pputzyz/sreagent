package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/testutil"
)

// ---------------------------------------------------------------------------
// NotificationService unit tests
// ---------------------------------------------------------------------------

func Test_RouteAlert_skips_silenced_event(t *testing.T) {
	logger := testutil.TestLogger()

	svc := NewNotificationService(nil, nil, nil, logger)

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

func Test_NewNotificationService_returns_non_nil(t *testing.T) {
	logger := testutil.TestLogger()
	svc := NewNotificationService(nil, nil, nil, logger)
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

	rule := &model.NotifyRule{
		Name:           "match-env-prod",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{"env": "prod", "severity": "critical"},
		Severities:     "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	matched, err := ruleRepo.FindMatchingRules(context.Background(),
		map[string]string{"env": "prod", "severity": "critical", "instance": "web-1"}, "critical", nil)
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

	rule := &model.NotifyRule{
		Name:           "critical-warning-only",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{},
		Severities:     "critical,warning",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	matched, err := ruleRepo.FindMatchingRules(context.Background(),
		map[string]string{"alertname": "TestAlert"}, "info", nil)
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

	r1 := &model.NotifyRule{Name: "batch-rule-1", IsEnabled: true, RepeatInterval: 3600}
	r2 := &model.NotifyRule{Name: "batch-rule-2", IsEnabled: true, RepeatInterval: 3600}
	r3 := &model.NotifyRule{Name: "batch-rule-3", IsEnabled: true, RepeatInterval: 3600}
	require.NoError(t, ruleRepo.Create(context.Background(), r1))
	require.NoError(t, ruleRepo.Create(context.Background(), r2))
	require.NoError(t, ruleRepo.Create(context.Background(), r3))

	err := ruleRepo.BatchUpdateEnabled(context.Background(), []uint{r1.ID, r2.ID}, false)
	require.NoError(t, err)

	fetched1, err := ruleRepo.GetByID(context.Background(), r1.ID)
	require.NoError(t, err)
	assert.False(t, fetched1.IsEnabled, "r1 should be disabled")

	fetched2, err := ruleRepo.GetByID(context.Background(), r2.ID)
	require.NoError(t, err)
	assert.False(t, fetched2.IsEnabled, "r2 should be disabled")

	fetched3, err := ruleRepo.GetByID(context.Background(), r3.ID)
	require.NoError(t, err)
	assert.True(t, fetched3.IsEnabled, "r3 should still be enabled")
}

func Test_NotifyRule_FindMatchingRules_LabelSubset_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	rule := &model.NotifyRule{
		Name:           "subset-match-rule",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{"env": "prod", "team": "infra"},
		Severities:     "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	eventLabels := map[string]string{
		"env":      "prod",
		"team":     "infra",
		"instance": "web-1",
		"region":   "us-east-1",
	}
	matched, err := ruleRepo.FindMatchingRules(context.Background(), eventLabels, "critical", nil)
	require.NoError(t, err)
	require.Len(t, matched, 1, "rule should match when event labels are a superset")
	assert.Equal(t, rule.ID, matched[0].ID)
	assert.Equal(t, "subset-match-rule", matched[0].Name)
}

func Test_NotifyRule_FindMatchingRules_NoMatch_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ruleRepo := repository.NewNotifyRuleRepository(db)

	rule := &model.NotifyRule{
		Name:           "prod-only-rule",
		IsEnabled:      true,
		MatchLabels:    model.JSONLabels{"env": "prod"},
		Severities:     "",
		RepeatInterval: 3600,
	}
	require.NoError(t, ruleRepo.Create(context.Background(), rule))

	eventLabels := map[string]string{
		"env":      "staging",
		"instance": "web-2",
	}
	matched, err := ruleRepo.FindMatchingRules(context.Background(), eventLabels, "warning", nil)
	require.NoError(t, err)
	assert.Empty(t, matched, "rule with env=prod should not match env=staging event")
}
