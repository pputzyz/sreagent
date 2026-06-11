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

// seedAlertEvent creates a test alert event in the given status and returns it.
func seedAlertEvent(t *testing.T, repo *repository.AlertEventRepository, status model.AlertEventStatus) *model.AlertEvent {
	t.Helper()
	event := &model.AlertEvent{
		Fingerprint: "test-fp-" + string(status),
		AlertName:   "TestAlert",
		Severity:    model.SeverityWarning,
		Status:      status,
		Labels:      model.JSONLabels{"alertname": "TestAlert"},
		Source:      "test",
		FiredAt:     time.Now(),
		FireCount:   1,
	}
	require.NoError(t, repo.Create(context.Background(), event))
	return event
}

// ---------------------------------------------------------------------------
// P0-2 regression: assigned/silenced states can be resolved/closed
// ---------------------------------------------------------------------------

func Test_Resolve_FromAssigned_Succeeds(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed an event in "assigned" state.
	event := seedAlertEvent(t, eventRepo, model.EventStatusAssigned)

	// Resolve should succeed from assigned state.
	err := svc.Resolve(ctx, event.ID, 0, "auto-resolved")
	require.NoError(t, err, "Resolve from assigned state should succeed")

	// Verify the event is now resolved.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusResolved, updated.Status,
		"event should be in resolved status after Resolve from assigned")
}

func Test_Close_FromSilenced_Succeeds(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed an event in "silenced" state.
	event := seedAlertEvent(t, eventRepo, model.EventStatusSilenced)

	// Close should succeed from silenced state.
	err := svc.Close(ctx, event.ID, 0, "manual close")
	require.NoError(t, err, "Close from silenced state should succeed")

	// Verify the event is now closed.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusClosed, updated.Status,
		"event should be in closed status after Close from silenced")
}

func Test_Acknowledge_FromAssigned_Succeeds(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed an event in "assigned" state.
	event := seedAlertEvent(t, eventRepo, model.EventStatusAssigned)

	// Acknowledge should succeed from assigned state.
	err := svc.Acknowledge(ctx, event.ID, 0)
	require.NoError(t, err, "Acknowledge from assigned state should succeed")

	// Verify the event is now acknowledged.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusAcknowledged, updated.Status,
		"event should be in acknowledged status after Acknowledge from assigned")
}

func Test_Resolve_FromSilenced_Succeeds(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed an event in "silenced" state.
	event := seedAlertEvent(t, eventRepo, model.EventStatusSilenced)

	// Resolve should succeed from silenced state.
	err := svc.Resolve(ctx, event.ID, 0, "auto-resolved from silence")
	require.NoError(t, err, "Resolve from silenced state should succeed")

	// Verify the event is now resolved.
	updated, err := eventRepo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EventStatusResolved, updated.Status,
		"event should be in resolved status after Resolve from silenced")
}

// ---------------------------------------------------------------------------
// P1-11 regression: BatchAcknowledge on already-acked events must not duplicate timeline
// ---------------------------------------------------------------------------

func Test_BatchAcknowledge_AlreadyAcked_NoDuplicateTimeline(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed one firing event and one already-acknowledged event.
	firingEvent := seedAlertEvent(t, eventRepo, model.EventStatusFiring)
	ackedEvent := seedAlertEvent(t, eventRepo, model.EventStatusAcknowledged)

	// First batch ack — should transition the firing event, skip the acked one.
	success, failed, err := svc.BatchAcknowledge(ctx, []uint{firingEvent.ID, ackedEvent.ID}, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, success, "only the firing event should be acknowledged")
	assert.Equal(t, 1, failed, "the already-acked event should count as failed")

	// Verify exactly 1 acknowledge timeline entry for the firing event.
	timeline, err := timelineRepo.ListByEventID(ctx, firingEvent.ID)
	require.NoError(t, err)
	ackCount := 0
	for _, entry := range timeline {
		if entry.Action == model.TimelineActionAcknowledged {
			ackCount++
		}
	}
	assert.Equal(t, 1, ackCount, "firing event should have exactly 1 acknowledge timeline entry")

	// Verify no timeline entries were written for the already-acked event.
	ackedTimeline, err := timelineRepo.ListByEventID(ctx, ackedEvent.ID)
	require.NoError(t, err)
	assert.Empty(t, ackedTimeline, "already-acked event should have no new timeline entries")
}

// ---------------------------------------------------------------------------
// P1-11 regression: BatchClose on already-closed events must not duplicate timeline
// ---------------------------------------------------------------------------

func Test_BatchClose_AlreadyClosed_NoDuplicateTimeline(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	ctx := context.Background()
	eventRepo := repository.NewAlertEventRepository(db)
	timelineRepo := repository.NewAlertTimelineRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := testutil.TestLogger()

	svc := NewAlertEventService(eventRepo, timelineRepo, userRepo, nil, nil, nil, nil, logger)

	// Seed one firing event and one already-closed event.
	firingEvent := seedAlertEvent(t, eventRepo, model.EventStatusFiring)
	closedEvent := seedAlertEvent(t, eventRepo, model.EventStatusClosed)

	// Batch close — should transition the firing event, skip the closed one.
	success, failed, err := svc.BatchClose(ctx, []uint{firingEvent.ID, closedEvent.ID}, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, success, "only the firing event should be closed")
	assert.Equal(t, 1, failed, "the already-closed event should count as failed")

	// Verify exactly 1 close timeline entry for the firing event.
	timeline, err := timelineRepo.ListByEventID(ctx, firingEvent.ID)
	require.NoError(t, err)
	closeCount := 0
	for _, entry := range timeline {
		if entry.Action == model.TimelineActionClosed {
			closeCount++
		}
	}
	assert.Equal(t, 1, closeCount, "firing event should have exactly 1 close timeline entry")

	// Verify no timeline entries were written for the already-closed event.
	closedTimeline, err := timelineRepo.ListByEventID(ctx, closedEvent.ID)
	require.NoError(t, err)
	assert.Empty(t, closedTimeline, "already-closed event should have no new timeline entries")
}
