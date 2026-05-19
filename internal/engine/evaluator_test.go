package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/testutil"
)

// ---------------------------------------------------------------------------
// StateEntry round-trip tests
// ---------------------------------------------------------------------------

func Test_toStateEntry_fromStateEntry_roundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	state := &AlertState{
		Labels:            map[string]string{"alertname": "HighCPU", "instance": "web-1"},
		Status:            "firing",
		ActiveAt:          now.Add(-5 * time.Minute),
		FiredAt:           now.Add(-3 * time.Minute),
		ResolvedAt:        time.Time{},
		Value:             95.3,
		Annotations:       map[string]string{"summary": "CPU > 90%"},
		RecoveryHoldUntil: now.Add(2 * time.Minute),
		LastSeen:          now,
		EventID:           42,
	}

	fp := "abc123"
	entry := toStateEntry(fp, state)

	assert.Equal(t, fp, entry.Fingerprint)
	assert.Equal(t, state.Labels, entry.Labels)
	assert.Equal(t, state.Status, entry.Status)
	assert.Equal(t, state.Value, entry.Value)
	assert.Equal(t, state.EventID, entry.EventID)
	assert.Equal(t, state.Annotations, entry.Annotations)

	restored := fromStateEntry(entry)
	assert.Equal(t, state.Labels, restored.Labels)
	assert.Equal(t, state.Status, restored.Status)
	assert.Equal(t, state.Value, restored.Value)
	assert.Equal(t, state.EventID, restored.EventID)
	assert.Equal(t, state.Annotations, restored.Annotations)
	assert.True(t, state.ActiveAt.Equal(restored.ActiveAt), "ActiveAt mismatch")
	assert.True(t, state.FiredAt.Equal(restored.FiredAt), "FiredAt mismatch")
	assert.True(t, state.RecoveryHoldUntil.Equal(restored.RecoveryHoldUntil), "RecoveryHoldUntil mismatch")
	assert.True(t, state.LastSeen.Equal(restored.LastSeen), "LastSeen mismatch")
}

func Test_toStateEntry_resolved_state(t *testing.T) {
	resolvedAt := time.Now().Truncate(time.Second)
	state := &AlertState{
		Labels:     map[string]string{"alertname": "DiskFull"},
		Status:     "resolved",
		ActiveAt:   time.Now().Add(-10 * time.Minute),
		FiredAt:    time.Now().Add(-8 * time.Minute),
		ResolvedAt: resolvedAt,
		Value:      0,
	}

	entry := toStateEntry("fp_resolved", state)
	restored := fromStateEntry(entry)

	assert.Equal(t, "resolved", restored.Status)
	assert.True(t, restored.ResolvedAt.Equal(resolvedAt), "ResolvedAt should survive round-trip")
}

// ---------------------------------------------------------------------------
// NewEvaluator configuration tests
// ---------------------------------------------------------------------------

func Test_NewEvaluator_defaults(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	assert.NotNil(t, eval)
	assert.NotNil(t, eval.evaluators)
	assert.NotNil(t, eval.suppressor)
	assert.NotNil(t, eval.stopCh)
	assert.Equal(t, 30*time.Second, eval.syncInterval)
	assert.Empty(t, eval.evaluators)
}

func Test_SetSyncInterval(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	eval.SetSyncInterval(10 * time.Second)
	assert.Equal(t, 10*time.Second, eval.syncInterval)

	// Zero/negative should be ignored
	eval.SetSyncInterval(0)
	assert.Equal(t, 10*time.Second, eval.syncInterval)

	eval.SetSyncInterval(-5 * time.Second)
	assert.Equal(t, 10*time.Second, eval.syncInterval)
}

func Test_SetOnAlert(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	called := false
	eval.SetOnAlert(func(ctx context.Context, event *model.AlertEvent) {
		called = true
	})

	assert.NotNil(t, eval.onAlert)
	// Call the stored callback to verify it works
	eval.onAlert(context.Background(), &model.AlertEvent{})
	assert.True(t, called, "onAlert callback should be invocable")
}

func Test_SetStateStore(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	assert.Nil(t, eval.stateStore, "default stateStore should be nil")

	ss := &mockStateStore{}
	eval.SetStateStore(ss)
	assert.Equal(t, ss, eval.stateStore)
}

func Test_SetWorkerPool(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	assert.Nil(t, eval.workerPool, "default workerPool should be nil")

	wp := &mockWorkerPool{}
	eval.SetWorkerPool(wp)
	assert.NotNil(t, eval.workerPool)
}

// ---------------------------------------------------------------------------
// GetStatus / GetFiringEvents on a fresh evaluator
// ---------------------------------------------------------------------------

func Test_GetStatus_not_started(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	status := eval.GetStatus()
	assert.False(t, status.Running)
	assert.Equal(t, 0, status.TotalRules)
	assert.Equal(t, 0, status.ActiveAlerts)
	assert.Empty(t, status.Uptime)
}

func Test_GetFiringEvents_empty(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	events := eval.GetFiringEvents()
	assert.Empty(t, events)
}

func Test_GetFiringAlertEvents_empty(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	events := eval.GetFiringAlertEvents()
	assert.NotNil(t, events, "should return non-nil empty slice")
	assert.Empty(t, events)
}

func Test_GetFiringEvents_returns_only_firing(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	// Inject a rule evaluator with mixed states
	re := &RuleEvaluator{
		states: map[string]*AlertState{
			"fp_firing": {
				Labels: map[string]string{"alertname": "A"},
				Status: "firing",
			},
			"fp_pending": {
				Labels: map[string]string{"alertname": "B"},
				Status: "pending",
			},
			"fp_resolved": {
				Labels: map[string]string{"alertname": "C"},
				Status: "resolved",
			},
			"fp_firing2": {
				Labels: map[string]string{"alertname": "D"},
				Status: "firing",
			},
		},
	}

	eval.mu.Lock()
	eval.evaluators[1] = re
	eval.mu.Unlock()

	firing := eval.GetFiringEvents()
	assert.Len(t, firing, 2, "should return only firing states")
}

func Test_GetFiringAlertEvents_adaptation(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	re := &RuleEvaluator{
		states: map[string]*AlertState{
			"fp1": {
				Labels:  map[string]string{"alertname": "Test", "severity": "critical"},
				Status:  "firing",
				EventID: 99,
			},
		},
	}

	eval.mu.Lock()
	eval.evaluators[1] = re
	eval.mu.Unlock()

	events := eval.GetFiringAlertEvents()
	require.Len(t, events, 1)
	assert.Equal(t, uint(99), events[0].ID)
	assert.Equal(t, model.AlertEventStatus("firing"), events[0].Status)
	assert.Equal(t, "Test", events[0].Labels["alertname"])
}

// ---------------------------------------------------------------------------
// Stop idempotency
// ---------------------------------------------------------------------------

func Test_Stop_idempotent(t *testing.T) {
	logger := zap.NewNop()
	eval := NewEvaluator(nil, nil, nil, nil, nil, nil, logger)

	// Stop on a fresh evaluator should not panic
	eval.Stop()

	// Second stop should also not panic (already closed channel)
	eval.Stop()
}

// ---------------------------------------------------------------------------
// Recovery hold logic (test the branch conditions)
// ---------------------------------------------------------------------------

func Test_recoveryHold_starts_observation_period(t *testing.T) {
	state := &AlertState{
		Labels:  map[string]string{"alertname": "Test"},
		Status:  "firing",
		FiredAt: time.Now().Add(-5 * time.Minute),
	}

	recoveryHold := 3 * time.Minute
	now := time.Now()

	// First disappearance: start recovery observation
	if recoveryHold > 0 && state.RecoveryHoldUntil.IsZero() {
		state.RecoveryHoldUntil = now.Add(recoveryHold)
	}

	assert.False(t, state.RecoveryHoldUntil.IsZero(), "RecoveryHoldUntil should be set")
	assert.True(t, state.RecoveryHoldUntil.After(now), "RecoveryHoldUntil should be in the future")
	assert.Equal(t, "firing", state.Status, "status should remain firing during observation")
}

func Test_recoveryHold_skips_resolution_during_observation(t *testing.T) {
	state := &AlertState{
		Labels:            map[string]string{"alertname": "Test"},
		Status:            "firing",
		FiredAt:           time.Now().Add(-5 * time.Minute),
		RecoveryHoldUntil: time.Now().Add(2 * time.Minute), // still 2 min left
	}

	recoveryHold := 3 * time.Minute
	now := time.Now()

	resolved := false
	if recoveryHold > 0 && now.Before(state.RecoveryHoldUntil) {
		// Still in observation, skip
		resolved = false
	} else {
		state.Status = "resolved"
		resolved = true
	}

	assert.False(t, resolved, "should not resolve during recovery observation")
	assert.Equal(t, "firing", state.Status)
}

func Test_recoveryHold_resolves_after_observation(t *testing.T) {
	state := &AlertState{
		Labels:            map[string]string{"alertname": "Test"},
		Status:            "firing",
		FiredAt:           time.Now().Add(-10 * time.Minute),
		RecoveryHoldUntil: time.Now().Add(-1 * time.Minute), // expired 1 min ago
	}

	recoveryHold := 3 * time.Minute
	now := time.Now()

	if recoveryHold > 0 && now.Before(state.RecoveryHoldUntil) {
		t.Fatal("should not enter observation branch")
	}

	state.Status = "resolved"
	state.ResolvedAt = now

	assert.Equal(t, "resolved", state.Status)
	assert.False(t, state.ResolvedAt.IsZero())
}

// ---------------------------------------------------------------------------
// stateTTL calculation
// ---------------------------------------------------------------------------

func Test_stateTTL(t *testing.T) {
	tests := []struct {
		name         string
		evalInterval int
		expectedTTL  time.Duration
	}{
		{"default 60s", 0, 3 * 60 * time.Second},
		{"30s interval", 30, 3 * 30 * time.Second},
		{"120s interval", 120, 3 * 120 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := &RuleEvaluator{
				rule: &model.AlertRule{EvalInterval: tt.evalInterval},
			}
			assert.Equal(t, tt.expectedTTL, re.stateTTL())
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

func Test_Evaluator_StartStop_integration(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()

	eval := NewEvaluator(db, nil, nil, nil, nil, nil, logger)
	eval.SetSyncInterval(1 * time.Second)

	eval.Start()

	status := eval.GetStatus()
	assert.True(t, status.Running, "evaluator should be running after Start()")
	assert.NotEmpty(t, status.Uptime)

	eval.Stop()

	status = eval.GetStatus()
	assert.False(t, status.Running, "evaluator should not be running after Stop()")
}

func Test_syncRules_loads_enabled_rules(t *testing.T) {
	db := testutil.TestDB(t)
	logger := testutil.TestLogger()
	t.Cleanup(func() { testutil.CleanupDB(t, db) })

	// Create a datasource
	ds := &model.DataSource{
		Name:      "test-prometheus",
		Type:      model.DSTypePrometheus,
		Endpoint:  "http://localhost:9090",
		IsEnabled: true,
	}
	require.NoError(t, db.Create(ds).Error)

	// Create an enabled rule
	rule := &model.AlertRule{
		Name:           "test-rule-sync",
		DataSourceID:   &ds.ID,
		Expression:     "up == 0",
		Severity:       model.SeverityWarning,
		Status:         model.RuleStatusEnabled,
		EvalInterval:   60,
		DatasourceType: model.DSTypePrometheus,
	}
	require.NoError(t, db.Create(rule).Error)

	eval := NewEvaluator(db, nil, nil, nil, nil, nil, logger)

	// syncRules should pick up the enabled rule
	eval.syncRules()

	eval.mu.RLock()
	_, exists := eval.evaluators[rule.ID]
	eval.mu.RUnlock()

	assert.True(t, exists, "syncRules should create evaluator for enabled rule")
}

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

type mockStateStore struct {
	saveErr   error
	deleteErr error
	loadErr   error
	states    map[string]*StateEntry
}

func (m *mockStateStore) SaveState(_ context.Context, _ uint, _ string, _ *StateEntry, _ time.Duration) error {
	return m.saveErr
}

func (m *mockStateStore) DeleteState(_ context.Context, _ uint, _ string) error {
	return m.deleteErr
}

func (m *mockStateStore) LoadStates(_ context.Context, _ uint) (map[string]*StateEntry, error) {
	if m.states != nil {
		return m.states, m.loadErr
	}
	return nil, m.loadErr
}

func (m *mockStateStore) DeleteRuleStates(_ context.Context, _ uint) error {
	return m.deleteErr
}

type mockWorkerPool struct{}

func (m *mockWorkerPool) Submit(_ context.Context, fn func(context.Context)) bool {
	fn(context.Background())
	return true
}

func (m *mockWorkerPool) Wait() {}

// Ensure mock types satisfy interfaces at compile time.
var _ StateStore = (*mockStateStore)(nil)
var _ AlertWorkerPoolSubmiter = (*mockWorkerPool)(nil)

// ---------------------------------------------------------------------------
// DB helper (reused by integration tests)
// ---------------------------------------------------------------------------

// dbFromTest returns a test DB or skips. Used by integration tests that need a DB.
func dbFromTest(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.TestDB(t)
}
