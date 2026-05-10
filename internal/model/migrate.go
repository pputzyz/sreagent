package model

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
		&AlertEventV2{},
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
		// Alert rule templates
		&AlertRuleTemplate{},
	}
}
