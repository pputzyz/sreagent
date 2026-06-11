// Package model — schema migration strategy.
//
// SREAgent uses a dual migration system:
//
//  1. SQL migrations (authoritative): Located in internal/pkg/dbmigrate/migrations/.
//     These are ordered, versioned .up.sql / .down.sql files applied sequentially
//     by the dbmigrate runner at startup. They handle column additions, data
//     backfills, index creation, and destructive schema changes that AutoMigrate
//     cannot safely perform (e.g. column renames, type changes).
//
//  2. AutoMigrate (safety net): The functions below list models whose tables
//     should exist. GORM's AutoMigrate will CREATE TABLE IF NOT EXISTS for
//     each, but will NOT modify existing columns or drop unused ones.
//     This ensures new deployments work even if the SQL migration runner has
//     not yet executed, and that the application always starts with a valid schema.
//
// The two systems are complementary: SQL migrations handle incremental upgrades
// on existing databases; AutoMigrate handles greenfield deployments.
// When adding a new model, always create both a SQL migration AND list it in the
// appropriate AutoMigrate function below.
//
// IMPORTANT CAVEATS (B12-2):
//   - AutoMigrate only creates tables and adds missing columns. It NEVER modifies
//     existing column types, drops columns, or renames columns. Those require SQL migrations.
//   - On existing databases, SQL migrations run first (via golang-migrate version tracking).
//     AutoMigrate runs after and is a no-op for existing tables.
//   - On fresh databases, AutoMigrate creates tables immediately. SQL migrations then
//     run against the already-created tables (column additions are idempotent with IF NOT EXISTS).
//   - If a SQL migration and AutoMigrate both touch the same table, the SQL migration
//     is authoritative. AutoMigrate is purely a safety net for table existence.
package model

// LarkCardModels returns models for the CardKit card entity/message system.
func LarkCardModels() []interface{} {
	return []interface{}{
		&LarkCardEntity{},
		&LarkCardMessage{},
	}
}

// NotificationV2Models returns all new notification system v2 models
// that need to be auto-migrated. This function is called by main.go
// during database initialization.
func NotificationV2Models() []interface{} {
	return []interface{}{
		&NotifyRule{},
		&NotifyMedia{},
		&MessageTemplate{},
		&SubscribeRule{},
		&BizGroup{},
		&BizGroupMember{},
	}
}

// DispatchModels returns models for the alert channel and user notify config system.
func DispatchModels() []interface{} {
	return []interface{}{
		&AlertChannel{},
		&UserNotifyConfig{},
	}
}

// V2Models returns all v2 feature models. These are primarily managed by SQL
// migrations (000019–000033), but listing them here ensures AutoMigrate can
// create them as a safety net when migrations haven't been run yet.
func V2Models() []interface{} {
	return []interface{}{
		// Alerts v2
		&Alert{},
		// AlertEventV2 removed — unified into alert_events (migration 000108)
		// Channels (collaboration spaces)
		&Channel{},
		&ChannelExclusionRule{},
		&ChannelStar{},
		// Incidents
		&Incident{},
		&IncidentAssignee{},
		&IncidentTimeline{},
		&PostMortem{},
		// Integrations
		&Integration{},
		&RoutingRule{},
		// Dispatch
		&DispatchPolicy{},
		&DispatchLog{},
		&ScheduledDispatch{},
		// Alert rule templates
		&AlertRuleTemplate{},
	}
}
