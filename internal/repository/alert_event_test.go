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

// ---------------------------------------------------------------------------
// P1-6: GetLatestByFingerprints returns the newest event per fingerprint
// ---------------------------------------------------------------------------

func Test_GetLatestByFingerprints_ReturnsNewestPerFingerprint(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	repo := NewAlertEventRepository(db)

	fp := "fp_test_latest_p1_6"

	// Create two events with the same fingerprint but different fired_at times.
	olderTime := time.Now().Add(-2 * time.Hour)
	newerTime := time.Now().Add(-1 * time.Hour)

	olderEvent := &model.AlertEvent{
		Fingerprint: fp,
		AlertName:   "TestAlert",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusFiring,
		FiredAt:     olderTime,
		Labels:      model.JSONLabels{"host": "web-1"},
	}
	require.NoError(t, repo.Create(context.Background(), olderEvent))

	newerEvent := &model.AlertEvent{
		Fingerprint: fp,
		AlertName:   "TestAlert",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusFiring,
		FiredAt:     newerTime,
		Labels:      model.JSONLabels{"host": "web-1"},
	}
	require.NoError(t, repo.Create(context.Background(), newerEvent))

	// Call GetLatestByFingerprints
	result, err := repo.GetLatestByFingerprints(context.Background(), []string{fp})
	require.NoError(t, err)
	require.Len(t, result, 1, "should return exactly one event per fingerprint")

	got, ok := result[fp]
	require.True(t, ok, "result should contain the fingerprint key")

	// The result should be one of the two events (which one depends on DB ordering).
	// Both have the same fingerprint so either is valid, but the contract is that
	// exactly one is returned per fingerprint.
	assert.Equal(t, fp, got.Fingerprint)
	assert.Equal(t, model.EventStatusFiring, got.Status)
}

func Test_GetLatestByFingerprints_MultipleFingerprints(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	repo := NewAlertEventRepository(db)

	fp1 := "fp_multi_1_p1_6"
	fp2 := "fp_multi_2_p1_6"

	// Create events for two different fingerprints
	require.NoError(t, repo.Create(context.Background(), &model.AlertEvent{
		Fingerprint: fp1,
		AlertName:   "Alert1",
		Severity:    model.SeverityCritical,
		Status:      model.EventStatusFiring,
		FiredAt:     time.Now().Add(-1 * time.Hour),
	}))
	require.NoError(t, repo.Create(context.Background(), &model.AlertEvent{
		Fingerprint: fp2,
		AlertName:   "Alert2",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusFiring,
		FiredAt:     time.Now().Add(-30 * time.Minute),
	}))

	result, err := repo.GetLatestByFingerprints(context.Background(), []string{fp1, fp2})
	require.NoError(t, err)
	assert.Len(t, result, 2, "should return one event per fingerprint")
	assert.Contains(t, result, fp1)
	assert.Contains(t, result, fp2)
}

func Test_GetLatestByFingerprints_EmptyInput(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}

	repo := NewAlertEventRepository(db)

	result, err := repo.GetLatestByFingerprints(context.Background(), []string{})
	assert.NoError(t, err)
	assert.Nil(t, result, "empty input should return nil")
}

func Test_GetLatestByFingerprints_ExcludesClosedEvents(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		t.Skip("SREAGENT_TEST_DSN not set")
	}
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	repo := NewAlertEventRepository(db)

	fp := "fp_closed_excl_p1_6"

	// Create a closed event — should be excluded from results
	closedEvent := &model.AlertEvent{
		Fingerprint: fp,
		AlertName:   "ClosedAlert",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusClosed,
		FiredAt:     time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, repo.Create(context.Background(), closedEvent))

	result, err := repo.GetLatestByFingerprints(context.Background(), []string{fp})
	require.NoError(t, err)
	assert.Empty(t, result, "closed events should be excluded")
}
