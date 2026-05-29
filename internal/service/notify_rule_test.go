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
