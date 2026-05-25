package model

// NotifyRule defines a notification rule that specifies how events should be
// processed and which media/templates to use for notifications.
// It replaces the legacy NotifyPolicy with a more flexible event pipeline model.
type NotifyRule struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	IsEnabled   bool   `json:"is_enabled" gorm:"default:true"`
	// Which severities this rule applies to (empty = all)
	Severities string `json:"severities" gorm:"size:128"` // "critical,warning"
	// Label matchers (optional - if empty, matches all events routed to this rule)
	MatchLabels JSONLabels `json:"match_labels" gorm:"type:json"`
	// Datasource filter (nil = wildcard, matches any datasource)
	DataSourceID *uint       `json:"datasource_id" gorm:"index"`
	DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
	// Event Pipeline config (JSON array of processor configs)
	// e.g., [{"type":"relabel","config":{...}}, {"type":"ai_summary","config":{"only_critical":true}}]
	Pipeline string `json:"pipeline" gorm:"type:text"`
	// PipelineID references a reusable EventPipeline by ID (takes precedence over inline Pipeline)
	PipelineID *uint `json:"pipeline_id" gorm:"index"`
	// Notification configs - which media to use for which severity
	// JSON: [{"severity":"critical","media_id":1,"template_id":1,"user_ids":[1,2],"team_ids":[1]},...]
	NotifyConfigs string `json:"notify_configs" gorm:"type:text"`
	// Throttle
	RepeatInterval int `json:"repeat_interval" gorm:"default:3600"` // seconds between repeated notifications
	// Callback URL (optional, called when event is processed)
	CallbackURL string `json:"callback_url" gorm:"size:512"`
	CreatedBy   uint   `json:"created_by" gorm:"index"`
}

func (NotifyRule) TableName() string {
	return "notify_rules"
}

// NotifyConfig represents a single notification configuration within a NotifyRule.
// This is the deserialized form of one element in NotifyRule.NotifyConfigs.
type NotifyConfig struct {
	Severity   string `json:"severity"`
	MediaID    uint   `json:"media_id"`
	TemplateID uint   `json:"template_id"`
	UserIDs    []uint `json:"user_ids,omitempty"`
	TeamIDs    []uint `json:"team_ids,omitempty"`
}

// PipelineStep represents a single step in the event processing pipeline.
// This is the deserialized form of one element in NotifyRule.Pipeline.
type PipelineStep struct {
	Type   string                 `json:"type"`   // "relabel", "ai_summary", etc.
	Config map[string]interface{} `json:"config"` // step-specific configuration
}
