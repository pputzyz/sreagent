package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// newCardStateSvcForTest builds a service good enough for buildEventCardJSON
// (no repo/cardkit calls happen during card rendering; userRepo is nil-safe).
func newCardStateSvcForTest() *LarkCardStateService {
	svc := NewLarkCardStateService(nil, nil, nil, zap.NewNop())
	svc.SetExternalURL("https://sre.example.com")
	return svc
}

func testEvent(status model.AlertEventStatus) *model.AlertEvent {
	now := time.Now()
	e := &model.AlertEvent{
		AlertName:   "HighCPU",
		Severity:    model.SeverityCritical,
		Status:      status,
		Labels:      model.JSONLabels{"host": "web-01", "env": "prod"},
		Annotations: model.JSONLabels{"summary": "CPU above 90%"},
		FiredAt:     now.Add(-10 * time.Minute),
		FireCount:   3,
	}
	e.ID = 42
	return e
}

// Test_buildEventCardJSON_SixStates_RenderDistinctContent calls the REAL
// service builder for every platform state and asserts state-specific content
// — the previous version of this test re-implemented the builder inside the
// test body and asserted its own output, which verified nothing.
func Test_buildEventCardJSON_SixStates_RenderDistinctContent(t *testing.T) {
	svc := newCardStateSvcForTest()
	cfg := LarkConfig{CardInteractionMode: "callback_http"}
	ctx := context.Background()

	until := time.Now().Add(2 * time.Hour)
	resolvedAt := time.Now()

	cases := []struct {
		status   model.AlertEventStatus
		mutate   func(e *model.AlertEvent)
		contains []string
	}{
		{model.EventStatusFiring, nil, []string{"告警中", `"action":"ack"`, `"action":"resolve"`, `"action":"silence"`}},
		{model.EventStatusAcknowledged, nil, []string{"已认领", `"action":"resolve"`}},
		{model.EventStatusAssigned, nil, []string{"已指派", `"action":"resolve"`}},
		{model.EventStatusSilenced, func(e *model.AlertEvent) {
			e.SilencedUntil = &until
			e.SilenceReason = "maintenance window"
		}, []string{"已静默", "maintenance window", `"action":"resolve"`}},
		{model.EventStatusResolved, func(e *model.AlertEvent) { e.ResolvedAt = &resolvedAt }, []string{"已恢复"}},
		{model.EventStatusClosed, nil, []string{"已关闭"}},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			event := testEvent(tc.status)
			if tc.mutate != nil {
				tc.mutate(event)
			}
			cardJSON, err := svc.buildEventCardJSON(ctx, event, cfg)
			require.NoError(t, err)
			for _, want := range tc.contains {
				assert.Contains(t, cardJSON, want, "state %s must render %q", tc.status, want)
			}
		})
	}

	// Terminal states must NOT carry operational callback buttons.
	for _, status := range []model.AlertEventStatus{model.EventStatusResolved, model.EventStatusClosed} {
		cardJSON, err := svc.buildEventCardJSON(ctx, testEvent(status), cfg)
		require.NoError(t, err)
		assert.NotContains(t, cardJSON, `"action":"ack"`, "terminal state %s must not offer ack", status)
		assert.NotContains(t, cardJSON, `"action":"silence"`, "terminal state %s must not offer silence", status)
	}
}

// Test_buildEventCardJSON_OpenURLMode_NoCallbackButtons: the open_url
// interaction mode (the zero-callback fallback) must not generate callback
// buttons — those would be dead clicks without a callback channel.
func Test_buildEventCardJSON_OpenURLMode_NoCallbackButtons(t *testing.T) {
	svc := newCardStateSvcForTest()
	ctx := context.Background()

	cardJSON, err := svc.buildEventCardJSON(ctx, testEvent(model.EventStatusFiring), LarkConfig{CardInteractionMode: "open_url"})
	require.NoError(t, err)
	assert.NotContains(t, cardJSON, `"callback"`, "open_url mode must not emit callback behaviors")
	assert.Contains(t, cardJSON, "https://sre.example.com/alerts/events/42", "open_url mode must link to the platform")

	// Empty mode defaults to open_url (the safe zero-callback default).
	cardJSON, err = svc.buildEventCardJSON(ctx, testEvent(model.EventStatusFiring), LarkConfig{})
	require.NoError(t, err)
	assert.NotContains(t, cardJSON, `"callback"`)
}

// Test_buildEventCardJSON_LabelsFolded_AISummaryShown verifies labels land in
// a collapsible panel and a pipeline ai_summary annotation is surfaced.
func Test_buildEventCardJSON_LabelsFolded_AISummaryShown(t *testing.T) {
	svc := newCardStateSvcForTest()
	event := testEvent(model.EventStatusFiring)
	event.Annotations["ai_summary"] = "likely memory leak in java process"

	cardJSON, err := svc.buildEventCardJSON(context.Background(), event, LarkConfig{})
	require.NoError(t, err)
	assert.Contains(t, cardJSON, "collapsible_panel")
	assert.Contains(t, cardJSON, "web-01")
	assert.Contains(t, cardJSON, "likely memory leak")
	// ai_summary must not be duplicated into the annotations panel.
	assert.Equal(t, 1, strings.Count(cardJSON, "likely memory leak"))
}

// Test_SyncCardStatus_AfterStop_NoTimerScheduled: Stop() must prevent new
// debounce timers (shutdown path) — otherwise timers fire after dependencies
// are torn down.
func Test_SyncCardStatus_AfterStop_NoTimerScheduled(t *testing.T) {
	svc := newCardStateSvcForTest()
	svc.Stop()

	err := svc.SyncCardStatus(context.Background(), testEvent(model.EventStatusFiring))
	require.NoError(t, err)

	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.Empty(t, svc.debounceTimers, "no debounce timer may be scheduled after Stop")
}

// Test_statusLabel_CoversAllSixStates guards the human-readable mapping.
func Test_statusLabel_CoversAllSixStates(t *testing.T) {
	for status, want := range map[model.AlertEventStatus]string{
		model.EventStatusFiring:       "告警中",
		model.EventStatusAcknowledged: "已认领",
		model.EventStatusAssigned:     "已指派",
		model.EventStatusSilenced:     "已静默",
		model.EventStatusResolved:     "已恢复",
		model.EventStatusClosed:       "已关闭",
	} {
		assert.Contains(t, statusLabel(status), want)
	}
}
