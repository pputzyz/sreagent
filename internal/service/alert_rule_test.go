package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
)

func setupAlertRuleService(t *testing.T) (*service.AlertRuleService, *gorm.DB) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)
	repo := repository.NewAlertRuleRepository(db)
	historyRepo := repository.NewAlertRuleHistoryRepository(db)
	dsRepo := repository.NewDataSourceRepository(db)
	svc := service.NewAlertRuleService(repo, historyRepo, dsRepo, testutil.TestLogger())
	return svc, db
}

func seedDataSource(t *testing.T, db *gorm.DB, name string, dsType model.DataSourceType) *model.DataSource {
	t.Helper()
	ds := &model.DataSource{
		Name:     name,
		Type:     dsType,
		Endpoint: "http://localhost:9090",
	}
	require.NoError(t, db.Create(ds).Error)
	return ds
}

// Test_Create_source_ai_defaults_to_draft verifies that AI-generated rules
// are created with status "draft" regardless of the input status.
func Test_Create_source_ai_defaults_to_draft(t *testing.T) {
	svc, db := setupAlertRuleService(t)
	ds := seedDataSource(t, db, "prometheus-ai", model.DSTypePrometheus)

	rule := &model.AlertRule{
		Name:         "ai-generated-rule",
		DataSourceID: &ds.ID,
		Expression:   "up == 0",
		ForDuration:  "5m",
		Severity:     model.SeverityWarning,
		Status:       model.RuleStatusActive, // caller sets enabled, but AI overrides
		Labels:       model.JSONLabels{"severity": "warning", "job": "test"},
	}

	err := svc.Create(context.Background(), rule, "ai")
	require.NoError(t, err)
	assert.NotZero(t, rule.ID)
	assert.Equal(t, model.RuleStatusDraft, rule.Status, "AI-generated rules must start as draft")
}

// Test_Create_source_manual_defaults_to_enabled verifies that manually created
// rules keep their provided status (typically "active").
func Test_Create_source_manual_defaults_to_enabled(t *testing.T) {
	svc, db := setupAlertRuleService(t)
	ds := seedDataSource(t, db, "prometheus-manual", model.DSTypePrometheus)

	rule := &model.AlertRule{
		Name:         "manual-rule",
		DataSourceID: &ds.ID,
		Expression:   "up == 0",
		ForDuration:  "1m",
		Severity:     model.SeverityCritical,
		Status:       model.RuleStatusActive,
		Labels:       model.JSONLabels{"severity": "critical", "job": "test"},
	}

	err := svc.Create(context.Background(), rule, "manual")
	require.NoError(t, err)
	assert.Equal(t, model.RuleStatusActive, rule.Status, "manual rules should keep their status")
}

// Test_Create_missing_datasource_returns_error verifies that creating a rule
// without datasource_id or datasource_type returns an error.
func Test_Create_missing_datasource_returns_error(t *testing.T) {
	svc, _ := setupAlertRuleService(t)

	rule := &model.AlertRule{
		Name:       "no-datasource",
		Expression: "up == 0",
		Severity:   model.SeverityWarning,
	}

	err := svc.Create(context.Background(), rule, "manual")
	assert.Error(t, err, "should fail when no datasource is specified")
}

// Test_Update_rule_fields_persisted verifies that Update correctly persists
// changes to name, expression, and severity.
func Test_Update_rule_fields_persisted(t *testing.T) {
	svc, db := setupAlertRuleService(t)
	ds := seedDataSource(t, db, "prometheus-update", model.DSTypePrometheus)
	rule := testutil.SeedAlertRule(t, db, "original-rule", ds.ID)

	rule.Name = "updated-rule"
	rule.Expression = "cpu_usage > 90"
	rule.Severity = model.SeverityCritical

	err := svc.Update(context.Background(), rule)
	require.NoError(t, err)

	fetched, err := svc.GetByID(context.Background(), rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-rule", fetched.Name)
	assert.Equal(t, "cpu_usage > 90", fetched.Expression)
	assert.Equal(t, model.SeverityCritical, fetched.Severity)
	assert.Equal(t, 2, fetched.Version, "version should increment on update")
}

// Test_List_filter_by_status verifies that List correctly filters rules by status.
func Test_List_filter_by_status(t *testing.T) {
	svc, db := setupAlertRuleService(t)
	ds := seedDataSource(t, db, "prometheus-list", model.DSTypePrometheus)

	r1 := &model.AlertRule{
		Name: "enabled-rule", DataSourceID: &ds.ID,
		Expression: "a", Severity: model.SeverityWarning, Status: model.RuleStatusActive,
	}
	r2 := &model.AlertRule{
		Name: "draft-rule", DataSourceID: &ds.ID,
		Expression: "b", Severity: model.SeverityWarning, Status: model.RuleStatusDraft,
	}
	require.NoError(t, svc.Create(context.Background(), r1, "manual"))
	require.NoError(t, svc.Create(context.Background(), r2, "ai"))

	enabledRules, total, err := svc.List(context.Background(), "", "enabled", "", "", "", nil, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, enabledRules, 1)
	assert.Equal(t, "enabled-rule", enabledRules[0].Name)

	draftRules, total, err := svc.List(context.Background(), "", "draft", "", "", "", nil, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, draftRules, 1)
	assert.Equal(t, "draft-rule", draftRules[0].Name)
}

// Test_GetByID_existing_rule_returns_rule verifies that GetByID returns
// the correct rule for a valid ID.
func Test_GetByID_existing_rule_returns_rule(t *testing.T) {
	svc, db := setupAlertRuleService(t)
	ds := seedDataSource(t, db, "prometheus-getbyid", model.DSTypePrometheus)
	seeded := testutil.SeedAlertRule(t, db, "fetch-me", ds.ID)

	fetched, err := svc.GetByID(context.Background(), seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, seeded.ID, fetched.ID)
	assert.Equal(t, "fetch-me", fetched.Name)
}

// Test_GetByID_nonexistent_returns_error verifies that GetByID returns
// an error for a non-existent rule ID.
func Test_GetByID_nonexistent_returns_error(t *testing.T) {
	svc, _ := setupAlertRuleService(t)

	_, err := svc.GetByID(context.Background(), 999999)
	assert.Error(t, err, "should return error for non-existent rule")
}
