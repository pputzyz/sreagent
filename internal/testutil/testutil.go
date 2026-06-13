package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

// TestDB creates a test database connection from SREAGENT_TEST_DSN env var.
// Skips the test if the env var is not set.
func TestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("SREAGENT_TEST_DSN")
	if dsn == "" {
		t.Skip("SREAGENT_TEST_DSN not set, skipping integration test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to connect to test database")
	return db
}

// TestLogger returns a nop logger for tests.
func TestLogger() *zap.Logger {
	return zap.NewNop()
}

// SeedUser creates a test user and returns it.
func SeedUser(t *testing.T, db *gorm.DB, username string, role model.Role) *model.User {
	t.Helper()
	user := &model.User{
		Username: username,
		Password: "hashed",
		Role:     role,
		IsActive: true,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// SeedAlertRule creates a test alert rule.
func SeedAlertRule(t *testing.T, db *gorm.DB, name string, dsID uint) *model.AlertRule {
	t.Helper()
	rule := &model.AlertRule{
		Name:         name,
		DataSourceID: &dsID,
		Expression:   "up == 0",
		ForDuration:  "60s",
		Severity:     model.SeverityWarning,
		Status:       model.RuleStatusActive,
	}
	require.NoError(t, db.Create(rule).Error)
	return rule
}

// CleanupDB truncates all test tables. Call in t.Cleanup.
func CleanupDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"diagnostic_run_steps", "diagnostic_runs", "diagnostic_workflow_steps", "diagnostic_workflows",
		"alert_timelines", "alert_events", "alert_rule_histories",
		"notify_records", "subscribe_rules", "notify_rules",
		"mute_rules", "inhibition_rules", "alert_channels",
		"escalation_steps", "escalation_policies", "oncall_shifts",
		"schedule_overrides", "schedule_participants", "schedules",
		"biz_group_members", "biz_groups",
		"alert_rules", "datasources",
		"team_members", "teams", "users",
		"label_registry", "audit_logs", "system_settings",
		"alert_forwarders",
	}
	for _, table := range tables {
		db.Exec("DELETE FROM `" + table + "`")
	}
}
