package model

// DispatchPolicy represents a channel-level dispatch configuration.
// It controls how incidents in a channel are routed, notified, and escalated.
// Multiple policies can exist per channel with priority ordering.
type DispatchPolicy struct {
	BaseModel
	// Channel association
	ChannelID uint     `json:"channel_id" gorm:"index;not null"`
	Channel   *Channel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`

	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	IsEnabled   bool   `json:"is_enabled"` // explicit value persists on create now; DB column keeps DEFAULT 1 for seeds

	// Priority: lower number = higher priority (evaluated first among all channel policies)
	Priority int `json:"priority" gorm:"default:0;index"`

	// --- Trigger conditions ---
	// MatchConditions: JSON []FilterCondition — only apply this policy when all conditions match
	// e.g. [{"field":"severity","operator":"in","value":"critical,p0"}]
	MatchConditions string `json:"match_conditions" gorm:"type:json"`
	// Datasource filter (nil = wildcard, matches any datasource)
	DataSourceID *uint       `json:"datasource_id" gorm:"index"`
	DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`

	// ActiveTimeConfig: JSON DispatchActiveTimeConfig — restrict when the policy is active
	// null/empty = always active
	ActiveTimeConfig string `json:"active_time_config" gorm:"type:json"`

	// --- Delay window ---
	// DelaySeconds: wait N seconds before dispatching (0 = immediate)
	// During the delay window, if the incident is acknowledged, dispatch is skipped.
	DelaySeconds int `json:"delay_seconds" gorm:"default:0"`

	// --- Escalation policy reference ---
	// EscalationPolicyID: which escalation policy to use for this dispatch.
	// Nil = no escalation, just single-shot notification.
	EscalationPolicyID *uint             `json:"escalation_policy_id" gorm:"index"`
	EscalationPolicy   *EscalationPolicy `json:"escalation_policy,omitempty" gorm:"foreignKey:EscalationPolicyID"`

	// --- Repeat notification ---
	// RepeatIntervalSeconds: re-notify every N seconds if not acknowledged (0 = no repeat)
	RepeatIntervalSeconds int `json:"repeat_interval_seconds" gorm:"default:0"`
	// MaxRepeats: max number of repeat notifications (0 = unlimited)
	MaxRepeats int `json:"max_repeats" gorm:"default:0"`

	// --- Notification mode ---
	// NotifyMode: "personal_preference" (respect user notify_configs) | "unified" (use media below)
	NotifyMode string `json:"notify_mode" gorm:"size:32;default:personal_preference"`
	// UnifiedMediaID: if NotifyMode="unified", which notify media to use
	UnifiedMediaID *uint `json:"unified_media_id" gorm:"index"`
	// UnifiedTemplateID: if NotifyMode="unified", which message template to use
	UnifiedTemplateID *uint            `json:"unified_template_id" gorm:"index"`
	UnifiedTemplate   *MessageTemplate `json:"unified_template,omitempty" gorm:"foreignKey:UnifiedTemplateID"`

	// --- Label enhancement rules ---
	// LabelEnhancementRules: JSON []LabelEnhancementAction
	// Applied to alert labels before dispatch/notification
	LabelEnhancementRules string `json:"label_enhancement_rules" gorm:"type:json"`
}

func (DispatchPolicy) TableName() string {
	return "dispatch_policies"
}

// DispatchActiveTimeConfig restricts when a dispatch policy is active.
type DispatchActiveTimeConfig struct {
	// Enabled: false = always active (ignore time config)
	Enabled bool `json:"enabled"`
	// Timezone: IANA timezone string, default "Asia/Shanghai"
	Timezone string `json:"timezone"`
	// DaysOfWeek: 0=Sunday ... 6=Saturday, empty = all days
	DaysOfWeek []int `json:"days_of_week,omitempty"`
	// StartTime / EndTime: "HH:MM" format (24h), empty = all day
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// LabelEnhancementAction defines a single label enrichment operation.
type LabelEnhancementAction struct {
	// Type: "extract" | "combine" | "map" | "delete" | "set"
	Type string `json:"type"`
	// Conditions: only apply when alert matches (nil = always apply)
	Conditions []FilterCondition `json:"conditions,omitempty"`

	// --- For "set": directly set a label ---
	SetKey   string `json:"set_key,omitempty"`
	SetValue string `json:"set_value,omitempty"`

	// --- For "extract": regex capture from source field ---
	SourceField string `json:"source_field,omitempty"` // alertname, labels.xxx
	Regex       string `json:"regex,omitempty"`        // first capture group used
	TargetLabel string `json:"target_label,omitempty"`

	// --- For "combine": template string ---
	Template string `json:"template,omitempty"` // e.g. "{{labels.env}}-{{labels.service}}"

	// --- For "map": lookup table ---
	MappingSourceLabel string            `json:"mapping_source_label,omitempty"`
	MappingTable       map[string]string `json:"mapping_table,omitempty"`

	// --- For "delete": label key to remove ---
	DeleteKey string `json:"delete_key,omitempty"`

	// Overwrite: whether to overwrite existing label value
	Overwrite bool `json:"overwrite"`
}

// DispatchLog records each dispatch attempt for an incident.
type DispatchLog struct {
	BaseModel
	IncidentID       uint   `json:"incident_id" gorm:"index;not null"`
	DispatchPolicyID uint   `json:"dispatch_policy_id" gorm:"index"`
	Status           string `json:"status" gorm:"size:32"` // pending | sent | skipped | failed
	Attempt          int    `json:"attempt" gorm:"default:1"`
	NextAttemptAt    *int64 `json:"next_attempt_at"` // unix timestamp; ideally *time.Time but kept as *int64 for backward compat
	Note             string `json:"note" gorm:"type:text"`
}

func (DispatchLog) TableName() string {
	return "dispatch_logs"
}
