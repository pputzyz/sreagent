package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/testutil"
)

func seedDispatch(t *testing.T, repo *ScheduledDispatchRepository, d *model.ScheduledDispatch) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), d))
}

// Test_ScheduledDispatch_MarkExpired_DispatchAt verifies MarkExpired keys off
// dispatch_at, not created_at — so an active repeating dispatch (next cycle in the
// future) is NOT force-expired, while a genuinely stuck pending (dispatch_at long
// past) is. Requires SREAGENT_TEST_DSN.
func Test_ScheduledDispatch_MarkExpired_DispatchAt(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { db.Exec("DELETE FROM scheduled_dispatches") })
	repo := NewScheduledDispatchRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Active repeating dispatch: next cycle is in the near future.
	active := &model.ScheduledDispatch{
		IncidentID: 1, EventID: 1, Fingerprint: "fp-active", PolicyID: 1, ChannelID: 1,
		DispatchAt: now.Add(10 * time.Minute), RepeatInterval: 1800, Status: model.ScheduledDispatchPending,
	}
	// Genuinely stuck pending: scheduled time is far in the past.
	stuck := &model.ScheduledDispatch{
		IncidentID: 2, EventID: 2, Fingerprint: "fp-stuck", PolicyID: 1, ChannelID: 1,
		DispatchAt: now.Add(-48 * time.Hour), Status: model.ScheduledDispatchPending,
	}
	seedDispatch(t, repo, active)
	seedDispatch(t, repo, stuck)

	n, err := repo.MarkExpired(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "only the stuck (dispatch_at < cutoff) dispatch should be expired")

	var gotActive model.ScheduledDispatch
	require.NoError(t, db.First(&gotActive, active.ID).Error)
	assert.Equal(t, model.ScheduledDispatchPending, gotActive.Status, "active repeating dispatch must NOT be expired")

	var gotStuck model.ScheduledDispatch
	require.NoError(t, db.First(&gotStuck, stuck.ID).Error)
	assert.Equal(t, model.ScheduledDispatchExpired, gotStuck.Status)
}

// Test_ScheduledDispatch_RescheduleAfterFailure verifies a transient failure on a
// repeating dispatch advances to the next cycle (status back to pending, repeat_count
// incremented, last_error recorded) instead of terminating the chain.
func Test_ScheduledDispatch_RescheduleAfterFailure(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { db.Exec("DELETE FROM scheduled_dispatches") })
	repo := NewScheduledDispatchRepository(db)
	ctx := context.Background()
	now := time.Now()

	d := &model.ScheduledDispatch{
		IncidentID: 3, EventID: 3, Fingerprint: "fp-retry", PolicyID: 1, ChannelID: 1,
		DispatchAt: now, RepeatInterval: 1800, MaxRepeats: 5, RepeatCount: 1,
		Status: model.ScheduledDispatchPending,
	}
	seedDispatch(t, repo, d)

	next := now.Add(30 * time.Minute)
	require.NoError(t, repo.RescheduleAfterFailure(ctx, d.ID, next, "lark 503"))

	var got model.ScheduledDispatch
	require.NoError(t, db.First(&got, d.ID).Error)
	assert.Equal(t, model.ScheduledDispatchPending, got.Status, "must stay pending for the next cycle, not failed")
	assert.Equal(t, 2, got.RepeatCount, "failed cycle counts toward repeat_count to bound retries")
	assert.Equal(t, "lark 503", got.LastError)
	assert.WithinDuration(t, next, got.DispatchAt, time.Second)
}
