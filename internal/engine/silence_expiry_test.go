package engine

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
// P0-2 regression: engine auto-recovery can resolve assigned/silenced events
// ---------------------------------------------------------------------------

func Test_TransitionStatus_ResolveFromAssigned_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	repo := repository.NewAlertEventRepository(db)

	// Seed an assigned event.
	event := &model.AlertEvent{
		Fingerprint: "test-fp-assigned",
		AlertName:   "TestAlert",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusAssigned,
		Labels:      model.JSONLabels{"alertname": "TestAlert"},
		Source:      "test",
		FiredAt:     time.Now(),
		FireCount:   1,
	}
	require.NoError(t, repo.Create(ctx, event))

	// Engine auto-recovery uses TransitionStatus with firing+assigned+silenced.
	now := time.Now()
	ok, err := repo.TransitionStatus(ctx, event.ID,
		[]model.AlertEventStatus{
			model.EventStatusFiring,
			model.EventStatusAssigned,
			model.EventStatusSilenced,
		},
		map[string]interface{}{
			"status":      model.EventStatusResolved,
			"resolved_at": now,
		},
	)
	require.NoError(t, err)
	assert.True(t, ok, "TransitionStatus should succeed for assigned -> resolved")

	updated, err := repo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusResolved, updated.Status)
}

func Test_TransitionStatus_ResolveFromSilenced_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	repo := repository.NewAlertEventRepository(db)

	// Seed a silenced event.
	event := &model.AlertEvent{
		Fingerprint: "test-fp-silenced",
		AlertName:   "TestAlert",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusSilenced,
		Labels:      model.JSONLabels{"alertname": "TestAlert"},
		Source:      "test",
		FiredAt:     time.Now(),
		FireCount:   1,
	}
	require.NoError(t, repo.Create(ctx, event))

	// Engine auto-recovery uses TransitionStatus with firing+assigned+silenced.
	now := time.Now()
	ok, err := repo.TransitionStatus(ctx, event.ID,
		[]model.AlertEventStatus{
			model.EventStatusFiring,
			model.EventStatusAssigned,
			model.EventStatusSilenced,
		},
		map[string]interface{}{
			"status":      model.EventStatusResolved,
			"resolved_at": now,
		},
	)
	require.NoError(t, err)
	assert.True(t, ok, "TransitionStatus should succeed for silenced -> resolved")

	updated, err := repo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusResolved, updated.Status)
}

// ---------------------------------------------------------------------------
// P0-2 regression: SilenceExpiryChecker expires silenced events
// ---------------------------------------------------------------------------

func Test_SilenceExpiryChecker_ExpiresSilencedEvents_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	logger := testutil.TestLogger()

	// Seed a silenced event whose silence has already expired.
	pastTime := time.Now().Add(-5 * time.Minute)
	event := &model.AlertEvent{
		Fingerprint:   "test-fp-expired-silence",
		AlertName:     "ExpiredSilenceAlert",
		Severity:      model.SeverityWarning,
		Status:        model.EventStatusSilenced,
		Labels:        model.JSONLabels{"alertname": "ExpiredSilenceAlert"},
		Source:        "test",
		FiredAt:       time.Now().Add(-10 * time.Minute),
		FireCount:     1,
		SilencedUntil: &pastTime,
		SilenceReason: "test silence",
	}
	require.NoError(t, eventRepo.Create(ctx, event))

	// Run the checker.
	checker := NewSilenceExpiryChecker(eventRepo, timelineRepo, logger)
	checker.runOnce(ctx)

	// Verify the event is now firing.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusFiring, updated.Status,
		"silenced event with expired silence should be transitioned to firing")
	assert.Nil(t, updated.SilencedUntil,
		"silenced_until should be cleared after expiry")
}

func Test_SilenceExpiryChecker_IgnoresActiveSilence_DB(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	logger := testutil.TestLogger()

	// Seed a silenced event whose silence is still active (expires in 1 hour).
	futureTime := time.Now().Add(1 * time.Hour)
	event := &model.AlertEvent{
		Fingerprint:   "test-fp-active-silence",
		AlertName:     "ActiveSilenceAlert",
		Severity:      model.SeverityWarning,
		Status:        model.EventStatusSilenced,
		Labels:        model.JSONLabels{"alertname": "ActiveSilenceAlert"},
		Source:        "test",
		FiredAt:       time.Now().Add(-5 * time.Minute),
		FireCount:     1,
		SilencedUntil: &futureTime,
		SilenceReason: "test silence",
	}
	require.NoError(t, eventRepo.Create(ctx, event))

	// Run the checker.
	checker := NewSilenceExpiryChecker(eventRepo, timelineRepo, logger)
	checker.runOnce(ctx)

	// Verify the event is still silenced.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusSilenced, updated.Status,
		"silenced event with active silence should remain silenced")
}
