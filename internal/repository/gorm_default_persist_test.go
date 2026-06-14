package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/testutil"
)

// Test_NotifyRule_CreateDisabled_Persists is a regression test for the GORM
// `default:true` bug: creating a resource with IsEnabled=false must persist false.
// Previously the struct tag made GORM omit the zero value on INSERT, so the DB
// DEFAULT 1 silently flipped it to enabled. Requires SREAGENT_TEST_DSN.
func Test_NotifyRule_CreateDisabled_Persists(t *testing.T) {
	db := testutil.TestDB(t)
	t.Cleanup(func() { db.Exec("DELETE FROM notify_rules") })

	repo := NewNotifyRuleRepository(db)
	ctx := context.Background()

	rule := &model.NotifyRule{Name: "regression-disabled", IsEnabled: false}
	require.NoError(t, repo.Create(ctx, rule))
	require.NotZero(t, rule.ID)

	got, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.False(t, got.IsEnabled, "create with IsEnabled=false must persist false, not be flipped to true by the DB default")

	// Sanity: an explicitly-enabled rule still persists true.
	enabled := &model.NotifyRule{Name: "regression-enabled", IsEnabled: true}
	require.NoError(t, repo.Create(ctx, enabled))
	gotEnabled, err := repo.GetByID(ctx, enabled.ID)
	require.NoError(t, err)
	assert.True(t, gotEnabled.IsEnabled)
}
