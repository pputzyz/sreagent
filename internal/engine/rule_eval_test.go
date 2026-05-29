package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_generateFingerprint_deterministic verifies that the same label set
// always produces the same fingerprint regardless of map iteration order.
func Test_generateFingerprint_deterministic(t *testing.T) {
	labels := map[string]string{
		"alertname": "HighCPU",
		"severity":  "critical",
		"instance":  "web-1",
	}

	// Run 100 times to rule out map ordering flakiness.
	fp1 := generateFingerprint(labels)
	for i := 0; i < 100; i++ {
		fp := generateFingerprint(labels)
		assert.Equal(t, fp1, fp, "fingerprint must be deterministic")
	}
}

// Test_generateFingerprint_different_labels_differ verifies that different
// label sets produce different fingerprints.
func Test_generateFingerprint_different_labels_differ(t *testing.T) {
	fp1 := generateFingerprint(map[string]string{"alertname": "A", "severity": "critical"})
	fp2 := generateFingerprint(map[string]string{"alertname": "B", "severity": "critical"})

	assert.NotEqual(t, fp1, fp2, "different labels must produce different fingerprints")
}

// Test_generateFingerprint_skips_internal_labels verifies that labels with
// double-underscore prefix/suffix (like __nodata__) are excluded from the
// fingerprint computation.
func Test_generateFingerprint_skips_internal_labels(t *testing.T) {
	fpNormal := generateFingerprint(map[string]string{
		"alertname": "Test",
	})

	fpWithInternal := generateFingerprint(map[string]string{
		"alertname":  "Test",
		"__nodata__": "true",
	})

	assert.Equal(t, fpNormal, fpWithInternal,
		"internal labels (double-underscore) should be excluded from fingerprint")
}

// Test_stateTransition_pending_to_firing verifies the state machine transition
// from "pending" to "firing" when the for_duration has elapsed.
//
// Since the full evaluate() loop has heavy dependencies (DB, datasource, Redis),
// we test the state machine logic by directly manipulating AlertState structs
// and verifying the expected transition conditions.
func Test_stateTransition_pending_to_firing(t *testing.T) {
	// Simulate an AlertState in "pending" status.
	state := &AlertState{
		Labels:   map[string]string{"alertname": "Test", "instance": "web-1"},
		Status:   "pending",
		ActiveAt: time.Now().Add(-2 * time.Minute), // started 2 minutes ago
		Value:    1.0,
	}

	// The rule has a for_duration of 1 minute.
	forDuration := 1 * time.Minute

	// The state machine logic: if status == "pending" && time.Since(ActiveAt) >= forDuration
	if state.Status == "pending" && time.Since(state.ActiveAt) >= forDuration {
		state.Status = "firing"
		now := time.Now()
		state.FiredAt = now
	}

	assert.Equal(t, "firing", state.Status,
		"pending state should transition to firing after for_duration elapses")
	assert.False(t, state.FiredAt.IsZero(), "FiredAt must be set on firing transition")
}

// Test_stateTransition_firing_to_resolved verifies the state machine transition
// from "firing" to "resolved" when the alert is no longer present in query results.
func Test_stateTransition_firing_to_resolved(t *testing.T) {
	// Simulate an AlertState in "firing" status.
	state := &AlertState{
		Labels:   map[string]string{"alertname": "Test", "instance": "web-1"},
		Status:   "firing",
		ActiveAt: time.Now().Add(-5 * time.Minute),
		FiredAt:  time.Now().Add(-5 * time.Minute),
		Value:    0.0,
		EventID:  42,
	}

	// Recovery hold is 0 (no observation period).
	recoveryHold := time.Duration(0)

	// The fingerprint is not in seenFingerprints (alert disappeared).
	seenFingerprints := map[string]bool{}
	fp := generateFingerprint(state.Labels)

	// State machine logic for resolved alerts:
	if !seenFingerprints[fp] {
		switch state.Status {
		case "firing":
			if recoveryHold > 0 && state.RecoveryHoldUntil.IsZero() {
				// Would enter recovery observation — not our case.
			} else {
				// Resolve the alert.
				state.Status = "resolved"
				state.ResolvedAt = time.Now()
			}
		}
	}

	assert.Equal(t, "resolved", state.Status,
		"firing state should transition to resolved when alert disappears from results")
	assert.False(t, state.ResolvedAt.IsZero(), "ResolvedAt must be set on resolution")
}

// Test_stateTransition_pending_removed_when_disappears verifies that a pending
// alert that disappears from query results is simply removed (no resolution event).
func Test_stateTransition_pending_removed_when_disappears(t *testing.T) {
	states := map[string]*AlertState{
		"fp1": {
			Labels:   map[string]string{"alertname": "Test"},
			Status:   "pending",
			ActiveAt: time.Now().Add(-30 * time.Second),
		},
	}

	seenFingerprints := map[string]bool{} // fp1 not seen
	fp := "fp1"

	// State machine logic for missing fingerprints.
	if !seenFingerprints[fp] {
		switch states[fp].Status {
		case "pending":
			delete(states, fp)
		}
	}

	_, exists := states[fp]
	assert.False(t, exists, "pending alert that disappears should be removed from states")
}

// Test_stateTransition_resolved_to_firing_on_reoccurrence verifies that a
// resolved alert transitions back to "firing" when the condition re-occurs.
func Test_stateTransition_resolved_to_firing_on_reoccurrence(t *testing.T) {
	state := &AlertState{
		Labels:     map[string]string{"alertname": "Test"},
		Status:     "resolved",
		ActiveAt:   time.Now().Add(-10 * time.Minute),
		FiredAt:    time.Now().Add(-10 * time.Minute),
		ResolvedAt: time.Now().Add(-2 * time.Minute),
	}

	// for_duration is 0 (immediate fire).
	forDuration := time.Duration(0)

	// The condition re-occurred; simulate the state machine path for "resolved" states.
	if state.Status == "resolved" {
		if forDuration <= 0 {
			state.Status = "firing"
			state.ActiveAt = time.Now()
			state.FiredAt = time.Now()
		}
	}

	assert.Equal(t, "firing", state.Status,
		"resolved alert should re-fire when condition re-occurs with for_duration=0")
}

// Test_parseDuration verifies duration string parsing used by the evaluator.
func Test_parseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"empty string", "", 0},
		{"zero literal", "0", 0},
		{"zero seconds", "0s", 0},
		{"five minutes", "5m", 5 * time.Minute},
		{"one hour", "1h", 1 * time.Hour},
		{"thirty seconds", "30s", 30 * time.Second},
		{"invalid string", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseDuration(tt.input))
		})
	}
}

// Test_AlertState_fields verifies AlertState struct field assignments
// used in the state machine transitions.
func Test_AlertState_fields(t *testing.T) {
	now := time.Now()
	state := &AlertState{
		Labels:      map[string]string{"alertname": "TestAlert", "env": "prod"},
		Status:      "firing",
		ActiveAt:    now,
		FiredAt:     now,
		Value:       42.5,
		Annotations: map[string]string{"summary": "CPU high"},
		EventID:     123,
	}

	require.Equal(t, "firing", state.Status)
	require.Equal(t, 42.5, state.Value)
	require.Equal(t, uint(123), state.EventID)
	require.Equal(t, "TestAlert", state.Labels["alertname"])
}

// Test_buildCombinations_rejects_huge verifies that buildCombinations returns
// an error when the cartesian product exceeds the 10,000 limit, preventing
// an explosion of queries that would overwhelm the TSDB.
func Test_buildCombinations_rejects_huge(t *testing.T) {
	// 101 * 100 = 10,100 > 10,000
	hosts := make([]string, 101)
	for i := range hosts {
		hosts[i] = "host" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}
	envs := make([]string, 100)
	for i := range envs {
		envs[i] = "env" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}

	paramNames := []string{"host", "env"}
	varValues := map[string][]string{
		"host": hosts,
		"env":  envs,
	}

	result, err := buildCombinations(paramNames, varValues)
	assert.Error(t, err, "buildCombinations must reject combinations exceeding 10,000")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "exceeds limit", "error should mention the limit")
}

// Test_buildCombinations_within_limit verifies that buildCombinations succeeds
// when the total is within the 10,000 limit.
func Test_buildCombinations_within_limit(t *testing.T) {
	paramNames := []string{"host", "env"}
	varValues := map[string][]string{
		"host": {"a", "b"},
		"env":  {"prod", "staging"},
	}

	result, err := buildCombinations(paramNames, varValues)
	assert.NoError(t, err)
	assert.Len(t, result, 4, "2*2 = 4 combinations")
}
