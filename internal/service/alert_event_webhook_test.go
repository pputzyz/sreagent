package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// testWebhookDB creates an in-memory SQLite database with the tables needed
// for alert_event_webhook tests.
func testWebhookDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open sqlite")
	require.NoError(t, db.AutoMigrate(
		&model.AlertEvent{},
		&model.AlertTimeline{},
	))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// newTestWebhookService creates a minimal AlertEventService wired to real
// SQLite-backed repos but with nil optional services (notify, oncall, lark).
func newTestWebhookService(t *testing.T, db *gorm.DB) *AlertEventService {
	t.Helper()
	return &AlertEventService{
		repo:         repository.NewAlertEventRepository(db),
		timelineRepo: repository.NewAlertTimelineRepository(db),
		notifySvc:    nil,
		onCallSvc:    nil,
		larkSvc:      nil,
		workerPool:   nil,
		dispatchSem:  make(chan struct{}, defaultDispatchConcurrency),
		logger:       zap.NewNop(),
	}
}


func Test_HandleWebhook_ResolvedAlertRefires_ReopensAndNotifies(t *testing.T) {
	db := testWebhookDB(t)

	// Seed an existing resolved event
	now := time.Now()
	resolvedAt := now.Add(-5 * time.Minute)
	existing := &model.AlertEvent{
		Fingerprint: "abc123",
		AlertName:   "HighCPU",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusResolved,
		Labels:      model.JSONLabels{"alertname": "HighCPU", "severity": "warning"},
		Annotations: model.JSONLabels{"summary": "CPU is high"},
		Source:      "test",
		FiredAt:     now.Add(-10 * time.Minute),
		ResolvedAt:  &resolvedAt,
		FireCount:   1,
	}
	require.NoError(t, db.Create(existing).Error)
	originalFiredAt := existing.FiredAt

	// Build service (nil notifySvc — we verify the code path doesn't panic)
	svc := newTestWebhookService(t, db)

	// Build a firing webhook payload with the same fingerprint
	payload := &model.AlertManagerPayload{
		Status:   "firing",
		Receiver: "test-webhook",
		Alerts: []model.AlertManagerAlert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "HighCPU", "severity": "warning"},
				Annotations: map[string]string{"summary": "CPU is high"},
				StartsAt:    now,
				Fingerprint: "abc123",
			},
		},
	}

	// Process the webhook
	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	// Reload the event from DB
	var updated model.AlertEvent
	require.NoError(t, db.First(&updated, existing.ID).Error)

	// Assert: status transitioned back to firing
	assert.Equal(t, string(model.EventStatusFiring), string(updated.Status),
		"resolved alert that re-fires should transition to firing")

	// Assert: FiredAt was updated to the new firing time
	assert.True(t, updated.FiredAt.After(originalFiredAt),
		"FiredAt should be updated to the new firing time")

	// Assert: ResolvedAt was cleared
	assert.Nil(t, updated.ResolvedAt,
		"ResolvedAt should be cleared when re-firing")

	// Assert: FireCount was incremented
	assert.Equal(t, 2, updated.FireCount,
		"FireCount should be incremented on re-fire")

	// Assert: timeline has a "reopened" entry
	var timelines []model.AlertTimeline
	require.NoError(t, db.Where("event_id = ?", existing.ID).Order("id ASC").Find(&timelines).Error)

	var hasReopenedEntry bool
	for _, tl := range timelines {
		if tl.Action == model.TimelineActionReopened {
			hasReopenedEntry = true
			assert.Contains(t, tl.Note, "re-fired after resolve",
				"timeline note should mention re-fire")
			break
		}
	}
	assert.True(t, hasReopenedEntry,
		"timeline should have a reopened entry for re-fired alert")
}

func Test_HandleWebhook_AlreadyFiring_IncrementsFireCount(t *testing.T) {
	db := testWebhookDB(t)

	// Seed an existing firing event
	existing := &model.AlertEvent{
		Fingerprint: "def456",
		AlertName:   "DiskFull",
		Severity:    model.SeverityCritical,
		Status:      model.EventStatusFiring,
		Labels:      model.JSONLabels{"alertname": "DiskFull"},
		Annotations: model.JSONLabels{},
		Source:      "test",
		FiredAt:     time.Now().Add(-5 * time.Minute),
		FireCount:   3,
	}
	require.NoError(t, db.Create(existing).Error)

	svc := newTestWebhookService(t, db)

	payload := &model.AlertManagerPayload{
		Status:   "firing",
		Receiver: "test-webhook",
		Alerts: []model.AlertManagerAlert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "DiskFull"},
				Annotations: map[string]string{},
				StartsAt:    time.Now(),
				Fingerprint: "def456",
			},
		},
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	var updated model.AlertEvent
	require.NoError(t, db.First(&updated, existing.ID).Error)

	assert.Equal(t, string(model.EventStatusFiring), string(updated.Status),
		"already-firing alert should stay firing")
	assert.Equal(t, 4, updated.FireCount,
		"FireCount should be incremented for dedup")
}

func Test_HandleWebhook_ResolvedAlertRefires_NotifySvcNil_NoPanic(t *testing.T) {
	db := testWebhookDB(t)

	existing := &model.AlertEvent{
		Fingerprint: "ghi789",
		AlertName:   "NetLatency",
		Severity:    model.SeverityWarning,
		Status:      model.EventStatusResolved,
		Labels:      model.JSONLabels{"alertname": "NetLatency"},
		Annotations: model.JSONLabels{},
		Source:      "test",
		FiredAt:     time.Now().Add(-10 * time.Minute),
		ResolvedAt:  ptrTime(time.Now().Add(-5 * time.Minute)),
		FireCount:   1,
	}
	require.NoError(t, db.Create(existing).Error)

	svc := newTestWebhookService(t, db)

	payload := &model.AlertManagerPayload{
		Status:   "firing",
		Receiver: "test-webhook",
		Alerts: []model.AlertManagerAlert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "NetLatency"},
				Annotations: map[string]string{},
				StartsAt:    time.Now(),
				Fingerprint: "ghi789",
			},
		},
	}

	// Should not panic even with nil notifySvc
	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)

	var updated model.AlertEvent
	require.NoError(t, db.First(&updated, existing.ID).Error)
	assert.Equal(t, string(model.EventStatusFiring), string(updated.Status))
}

func Test_HandleWebhook_ClosedAlert_IgnoredByGetByFingerprint(t *testing.T) {
	db := testWebhookDB(t)

	// Seed a closed event — GetByFingerprint filters out closed events
	closed := &model.AlertEvent{
		Fingerprint: "closed001",
		AlertName:   "OldAlert",
		Severity:    model.SeverityInfo,
		Status:      model.EventStatusClosed,
		Labels:      model.JSONLabels{"alertname": "OldAlert"},
		Annotations: model.JSONLabels{},
		Source:      "test",
		FiredAt:     time.Now().Add(-1 * time.Hour),
		FireCount:   1,
	}
	require.NoError(t, db.Create(closed).Error)

	svc := newTestWebhookService(t, db)

	payload := &model.AlertManagerPayload{
		Status:   "firing",
		Receiver: "test-webhook",
		Alerts: []model.AlertManagerAlert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "OldAlert"},
				Annotations: map[string]string{},
				StartsAt:    time.Now(),
				Fingerprint: "closed001",
			},
		},
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	// A new event should be created (the closed one is invisible to GetByFingerprint)
	var events []model.AlertEvent
	require.NoError(t, db.Where("fingerprint = ?", "closed001").Find(&events).Error)
	assert.Len(t, events, 2, "should have the closed event and a new firing event")

	var newEvent *model.AlertEvent
	for i := range events {
		if events[i].Status == model.EventStatusFiring {
			newEvent = &events[i]
			break
		}
	}
	require.NotNil(t, newEvent, "new firing event should exist")
	assert.Equal(t, 1, newEvent.FireCount, "new event should have FireCount=1")
}

func ptrTime(t time.Time) *time.Time { return &t }
