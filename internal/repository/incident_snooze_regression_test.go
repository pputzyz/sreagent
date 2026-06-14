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

// Test_Incident_ListExpiredSnoozed verifies the query that drives snooze-wake:
// snoozed incidents whose snoozed_until has elapsed are returned, while a still-
// snoozed incident (future snoozed_until) and a non-snoozed incident are not.
// This is the source for WakeExpiredSnoozed, which returns such incidents to
// "triggered" so escalation/notify resume. Requires SREAGENT_TEST_DSN.
func Test_Incident_ListExpiredSnoozed(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { db.Exec("DELETE FROM incidents") })
	repo := NewIncidentRepository(db)
	ctx := context.Background()
	now := time.Now()

	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	expiredSnooze := &model.Incident{
		Title: "expired-snooze", ChannelID: 1, Severity: model.IncidentSeverityWarning,
		Status: model.IncidentStatusSnoozed, TriggeredAt: now.Add(-2 * time.Hour), SnoozedUntil: &past,
	}
	activeSnooze := &model.Incident{
		Title: "active-snooze", ChannelID: 1, Severity: model.IncidentSeverityWarning,
		Status: model.IncidentStatusSnoozed, TriggeredAt: now.Add(-2 * time.Hour), SnoozedUntil: &future,
	}
	triggered := &model.Incident{
		Title: "triggered", ChannelID: 1, Severity: model.IncidentSeverityWarning,
		Status: model.IncidentStatusTriggered, TriggeredAt: now,
	}
	require.NoError(t, db.Create(expiredSnooze).Error)
	require.NoError(t, db.Create(activeSnooze).Error)
	require.NoError(t, db.Create(triggered).Error)

	list, err := repo.ListExpiredSnoozed(ctx, now)
	require.NoError(t, err)

	ids := map[uint]bool{}
	for _, inc := range list {
		ids[inc.ID] = true
	}
	assert.True(t, ids[expiredSnooze.ID], "expired-snooze incident must be returned for wake-up")
	assert.False(t, ids[activeSnooze.ID], "still-snoozed (future) incident must NOT be returned")
	assert.False(t, ids[triggered.ID], "non-snoozed incident must NOT be returned")
}
