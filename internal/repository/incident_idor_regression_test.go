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

// Test_Incident_IncidentInTeams verifies the team-scoping primitive that backs the
// per-incident authorization guard (IDOR fix): an incident is "in" a team only when
// its channel's team_id is among the given teams. Requires SREAGENT_TEST_DSN.
func Test_Incident_IncidentInTeams(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() {
		db.Exec("DELETE FROM incidents")
		db.Exec("DELETE FROM channels")
	})
	repo := NewIncidentRepository(db)
	ctx := context.Background()

	teamA := uint(100)
	chA := &model.Channel{Name: "idor-chan-teamA", TeamID: &teamA, Status: model.ChannelStatusActive, AccessLevel: model.ChannelAccessPublic}
	require.NoError(t, db.Create(chA).Error)

	inc := &model.Incident{
		Title: "idor-incident", ChannelID: chA.ID, Severity: model.IncidentSeverityWarning,
		Status: model.IncidentStatusTriggered, TriggeredAt: time.Now(),
	}
	require.NoError(t, db.Create(inc).Error)

	// Member of the owning team → allowed.
	ok, err := repo.IncidentInTeams(ctx, inc.ID, []uint{teamA})
	require.NoError(t, err)
	assert.True(t, ok, "member of the incident's channel team should be authorized")

	// Member of a different team → denied.
	ok, err = repo.IncidentInTeams(ctx, inc.ID, []uint{200})
	require.NoError(t, err)
	assert.False(t, ok, "member of another team must NOT be authorized (IDOR guard)")

	// Empty team list → denied.
	ok, err = repo.IncidentInTeams(ctx, inc.ID, nil)
	require.NoError(t, err)
	assert.False(t, ok, "no teams → not authorized")
}
