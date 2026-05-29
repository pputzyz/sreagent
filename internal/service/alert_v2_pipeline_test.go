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

// testSQLiteDB creates an in-memory SQLite database with the required tables migrated.
func testSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open sqlite")
	require.NoError(t, db.AutoMigrate(
		&model.Incident{},
		&model.IncidentTimeline{},
		&model.AlertEvent{},
	))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	return db
}

func Test_buildAlertKey_CrossDatasource_NoCollision(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(42)
	ds1 := uint(1)
	ds2 := uint(2)
	labels := model.JSONLabels{"job": "api", "instance": "host1:9090"}

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &ds1,
		Labels:       labels,
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &ds2,
		Labels:       labels,
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)

	assert.NotEqual(t, key1, key2, "same rule+labels but different datasource should produce different keys")
	assert.Len(t, key1, 32, "md5 hex should be 32 chars")
	assert.Len(t, key2, 32)
}

func Test_buildAlertKey_SameInput_Stable(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)
	labels := model.JSONLabels{"env": "prod", "job": "web"}

	event := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       labels,
	}

	key1 := p.buildAlertKey(event)
	key2 := p.buildAlertKey(event)
	assert.Equal(t, key1, key2, "same input should produce stable key")
}

func Test_buildAlertKey_LabelOrder_Stable(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"b": "2", "a": "1"},
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"a": "1", "b": "2"},
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)
	assert.Equal(t, key1, key2, "label order should not affect key")
}

func Test_buildAlertKey_DifferentLabels_DifferentKey(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	ruleID := uint(1)
	dsID := uint(1)

	event1 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"job": "api"},
	}
	event2 := &model.AlertEvent{
		RuleID:       &ruleID,
		DataSourceID: &dsID,
		Labels:       model.JSONLabels{"job": "web"},
	}

	key1 := p.buildAlertKey(event1)
	key2 := p.buildAlertKey(event2)
	assert.NotEqual(t, key1, key2)
}

func Test_buildAlertKey_NilIDs(t *testing.T) {
	p := &AlertV2Pipeline{logger: zap.NewNop()}

	event := &model.AlertEvent{
		Labels: model.JSONLabels{"job": "api"},
	}

	key := p.buildAlertKey(event)
	assert.Len(t, key, 32, "should handle nil ruleID and datasourceID")
}

// Test_aggregator_same_fingerprint_single_incident verifies that when the same
// fingerprint fires multiple times, only ONE incident is created (not one per
// firing). Subsequent firings increment EventCount on the existing incident.
func Test_aggregator_same_fingerprint_single_incident(t *testing.T) {
	db := testSQLiteDB(t)
	logger := zap.NewNop()

	incidentRepo := repository.NewIncidentRepository(db)
	alertEventRepo := repository.NewAlertEventRepository(db)
	incidentSvc := NewIncidentService(incidentRepo, nil, logger)

	agg := NewIncidentAggregator(incidentSvc, alertEventRepo, incidentRepo, 1, logger)

	fp := "test-fingerprint-abc123"

	// First firing — should create one incident.
	event1 := &model.AlertEvent{
		AlertName:   "HighCPU",
		Severity:    model.SeverityCritical,
		Status:      model.EventStatusFiring,
		Fingerprint: fp,
		Labels:      model.JSONLabels{"job": "api"},
		FiredAt:     time.Now(),
	}
	agg.OnEventFired(context.Background(), event1)

	var incidents []model.Incident
	err := db.Where("fingerprint = ?", fp).Find(&incidents).Error
	assert.NoError(t, err)
	assert.Len(t, incidents, 1, "first firing must create exactly one incident")
	assert.Equal(t, 1, incidents[0].AlertCount, "AlertCount should be 1 after first firing")
	assert.Equal(t, 1, incidents[0].EventCount, "EventCount should be 1 after first firing")
	assert.Equal(t, fp, incidents[0].Fingerprint, "Fingerprint must be set on the incident")

	// Second firing with the same fingerprint — should NOT create a new incident.
	event2 := &model.AlertEvent{
		AlertName:   "HighCPU",
		Severity:    model.SeverityCritical,
		Status:      model.EventStatusFiring,
		Fingerprint: fp,
		Labels:      model.JSONLabels{"job": "api"},
		FiredAt:     time.Now(),
	}
	agg.OnEventFired(context.Background(), event2)

	var incidents2 []model.Incident
	err = db.Where("fingerprint = ?", fp).Find(&incidents2).Error
	assert.NoError(t, err)
	assert.Len(t, incidents2, 1, "second firing must NOT create a second incident")
	assert.Equal(t, 2, incidents2[0].EventCount, "EventCount should be 2 after second firing")
}
