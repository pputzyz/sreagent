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
// Pure unit tests (no DB required)
// ---------------------------------------------------------------------------

func Test_parseLarkPersonalConfig_user_id(t *testing.T) {
	rt, id, err := parseLarkPersonalConfig(`{"user_id":"uid_123"}`)
	require.NoError(t, err)
	assert.Equal(t, "user_id", rt)
	assert.Equal(t, "uid_123", id)
}

func Test_parseLarkPersonalConfig_open_id(t *testing.T) {
	rt, id, err := parseLarkPersonalConfig(`{"open_id":"ou_xxx"}`)
	require.NoError(t, err)
	assert.Equal(t, "open_id", rt)
	assert.Equal(t, "ou_xxx", id)
}

func Test_parseLarkPersonalConfig_union_id(t *testing.T) {
	rt, id, err := parseLarkPersonalConfig(`{"union_id":"on_xxx"}`)
	require.NoError(t, err)
	assert.Equal(t, "union_id", rt)
	assert.Equal(t, "on_xxx", id)
}

func Test_parseLarkPersonalConfig_lark_user_id_legacy(t *testing.T) {
	rt, id, err := parseLarkPersonalConfig(`{"lark_user_id":"legacy_123"}`)
	require.NoError(t, err)
	assert.Equal(t, "user_id", rt)
	assert.Equal(t, "legacy_123", id)
}

func Test_parseLarkPersonalConfig_empty(t *testing.T) {
	_, _, err := parseLarkPersonalConfig("")
	assert.Error(t, err)
}

func Test_parseLarkPersonalConfig_no_fields(t *testing.T) {
	_, _, err := parseLarkPersonalConfig(`{"foo":"bar"}`)
	assert.Error(t, err)
}

func Test_parseLarkPersonalConfig_invalid_json(t *testing.T) {
	_, _, err := parseLarkPersonalConfig(`not json`)
	assert.Error(t, err)
}

func Test_parseLarkPersonalConfig_priority(t *testing.T) {
	// user_id takes priority over open_id
	rt, id, err := parseLarkPersonalConfig(`{"user_id":"u1","open_id":"o1"}`)
	require.NoError(t, err)
	assert.Equal(t, "user_id", rt)
	assert.Equal(t, "u1", id)
}

func Test_mapChannelTypeToMediaType(t *testing.T) {
	tests := []struct {
		in   model.NotifyChannelType
		want model.NotifyMediaType
	}{
		{model.ChannelTypeLarkWebhook, model.MediaTypeLarkWebhook},
		{model.ChannelTypeEmail, model.MediaTypeEmail},
		{model.ChannelTypeCustom, model.MediaTypeHTTP},
		{"unknown", model.MediaTypeHTTP},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			assert.Equal(t, tt.want, mapChannelTypeToMediaType(tt.in))
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

func TestEscalation_StepExecRepo_InsertIgnore_ThenMarkSuccess(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	repo := repository.NewEscalationStepExecutionRepository(db)
	ctx := context.Background()

	inserted, err := repo.InsertIgnore(ctx, 100, 1)
	require.NoError(t, err)
	assert.True(t, inserted, "first insert should succeed")

	// Second insert should be ignored (dedup).
	inserted, err = repo.InsertIgnore(ctx, 100, 1)
	require.NoError(t, err)
	assert.False(t, inserted, "duplicate insert should be ignored")

	// HasExecuted should be false (status=pending).
	executed, err := repo.HasExecuted(ctx, 100, 1)
	require.NoError(t, err)
	assert.False(t, executed)

	// MarkSuccess.
	require.NoError(t, repo.MarkSuccess(ctx, 100, 1))
	executed, err = repo.HasExecuted(ctx, 100, 1)
	require.NoError(t, err)
	assert.True(t, executed)
}

func TestEscalation_StepExecRepo_MarkFailed_Retry(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	repo := repository.NewEscalationStepExecutionRepository(db)
	ctx := context.Background()

	inserted, err := repo.InsertIgnore(ctx, 200, 2)
	require.NoError(t, err)
	assert.True(t, inserted)

	// Mark as failed.
	require.NoError(t, repo.MarkFailed(ctx, 200, 2))
	executed, err := repo.HasExecuted(ctx, 200, 2)
	require.NoError(t, err)
	assert.False(t, executed, "failed should not count as executed")

	// Delete and re-insert (retry path).
	require.NoError(t, repo.DeleteByEventAndStep(ctx, 200, 2))
	inserted, err = repo.InsertIgnore(ctx, 200, 2)
	require.NoError(t, err)
	assert.True(t, inserted, "re-insert after delete should succeed")

	// Now mark success.
	require.NoError(t, repo.MarkSuccess(ctx, 200, 2))
	executed, err = repo.HasExecuted(ctx, 200, 2)
	require.NoError(t, err)
	assert.True(t, executed)
}

func TestEscalation_StepExecRepo_ConcurrentInsertIgnore(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	repo := repository.NewEscalationStepExecutionRepository(db)
	ctx := context.Background()

	const goroutines = 10
	results := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			inserted, err := repo.InsertIgnore(ctx, 300, 3)
			require.NoError(t, err)
			results <- inserted
		}()
	}

	trueCount := 0
	for i := 0; i < goroutines; i++ {
		if <-results {
			trueCount++
		}
	}
	assert.Equal(t, 1, trueCount, "exactly one goroutine should succeed with INSERT IGNORE")
}

func TestEscalation_BatchLoadByPolicyIDs(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	stepRepo := repository.NewEscalationStepRepository(db)
	ctx := context.Background()

	// Create two policies with steps.
	p1 := &model.EscalationPolicy{Name: "p1", TeamID: 1, IsEnabled: true}
	p2 := &model.EscalationPolicy{Name: "p2", TeamID: 2, IsEnabled: true}
	require.NoError(t, db.Create(p1).Error)
	require.NoError(t, db.Create(p2).Error)

	require.NoError(t, stepRepo.Create(ctx, &model.EscalationStep{PolicyID: p1.ID, StepOrder: 1, DelayMinutes: 5, TargetType: "user", TargetID: 1}))
	require.NoError(t, stepRepo.Create(ctx, &model.EscalationStep{PolicyID: p1.ID, StepOrder: 2, DelayMinutes: 15, TargetType: "user", TargetID: 2}))
	require.NoError(t, stepRepo.Create(ctx, &model.EscalationStep{PolicyID: p2.ID, StepOrder: 1, DelayMinutes: 10, TargetType: "user", TargetID: 3}))

	// Batch load.
	m, err := stepRepo.BatchLoadByPolicyIDs(ctx, []uint{p1.ID, p2.ID})
	require.NoError(t, err)
	assert.Len(t, m, 2)
	assert.Len(t, m[p1.ID], 2)
	assert.Len(t, m[p2.ID], 1)
	assert.Equal(t, 1, m[p1.ID][0].StepOrder)
	assert.Equal(t, 2, m[p1.ID][1].StepOrder)

	// Empty IDs.
	m, err = stepRepo.BatchLoadByPolicyIDs(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, m)
}

func TestEscalation_ListAllEnabled(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	policyRepo := repository.NewEscalationPolicyRepository(db)
	ctx := context.Background()

	// Create enabled and disabled policies.
	require.NoError(t, db.Create(&model.EscalationPolicy{Name: "enabled1", TeamID: 1, IsEnabled: true}).Error)
	require.NoError(t, db.Create(&model.EscalationPolicy{Name: "disabled1", TeamID: 1, IsEnabled: false}).Error)
	require.NoError(t, db.Create(&model.EscalationPolicy{Name: "enabled2", TeamID: 2, IsEnabled: true}).Error)

	list, err := policyRepo.ListAllEnabled(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	for _, p := range list {
		assert.True(t, p.IsEnabled)
	}
}

func TestEscalation_ListFiringForEscalation_CursorPagination(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { testutil.CleanupDB(t, db) })
	eventRepo := repository.NewAlertEventRepository(db)
	ctx := context.Background()

	// Seed 5 firing events.
	for i := 0; i < 5; i++ {
		require.NoError(t, eventRepo.Create(ctx, &model.AlertEvent{
			Fingerprint: "fp-cursor-" + time.Now().Format("150405.000") + "-" + string(rune('a'+i)),
			AlertName:   "test-alert",
			Severity:    model.SeverityWarning,
			Status:      model.EventStatusFiring,
			FiredAt:     time.Now().Add(-time.Duration(i) * time.Minute),
		}))
	}

	// First page.
	page1, err := eventRepo.ListFiringForEscalation(ctx, 0, 3)
	require.NoError(t, err)
	assert.Len(t, page1, 3)

	// Second page using cursor.
	page2, err := eventRepo.ListFiringForEscalation(ctx, page1[len(page1)-1].ID, 3)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Third page should be empty.
	page3, err := eventRepo.ListFiringForEscalation(ctx, page2[len(page2)-1].ID, 3)
	require.NoError(t, err)
	assert.Len(t, page3, 0)
}
