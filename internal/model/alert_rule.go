package model

import "time"

// AlertSeverity defines the severity level of an alert.
// Preferred values: critical, warning, info.
// Legacy values (p0–p4) are kept for backward compatibility with historical data.
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"

	// Legacy severity levels — kept for backward compatibility with existing data.
	// New rules should use critical/warning/info instead.
	SeverityP0 AlertSeverity = "p0"
	SeverityP1 AlertSeverity = "p1"
	SeverityP2 AlertSeverity = "p2"
	SeverityP3 AlertSeverity = "p3"
	SeverityP4 AlertSeverity = "p4"
)

// AlertRuleStatus defines the status of an alert rule.
// - draft: AI-generated rule, not yet activated by the user.
// - enabled: actively evaluating.
// - disabled: paused / not evaluating.
type AlertRuleStatus string

const (
	RuleStatusDraft    AlertRuleStatus = "draft"    // AI-generated, not yet activated
	RuleStatusEnabled  AlertRuleStatus = "enabled"
	RuleStatusDisabled AlertRuleStatus = "disabled"
)

// AlertRuleType identifies the evaluation strategy for a rule.
type AlertRuleType string

const (
	RuleTypeThreshold AlertRuleType = "threshold" // default: PromQL/LogQL/Zabbix expression
	RuleTypeHeartbeat AlertRuleType = "heartbeat" // fire when no ping received within interval
)

// AlertRule represents an alerting rule definition.
type AlertRule struct {
	BaseModel
	// RuleType controls the evaluation strategy. Default is "threshold".
	RuleType    AlertRuleType `json:"rule_type" gorm:"size:32;not null;default:threshold"`
	Name         string     `json:"name" gorm:"size:256;not null;index"`
	DisplayName  string     `json:"display_name" gorm:"size:256"`
	Description  string     `json:"description" gorm:"type:text"`
	DataSourceID   *uint          `json:"datasource_id" gorm:"index"`
	DataSource     *DataSource    `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
	DatasourceType DataSourceType `json:"datasource_type" gorm:"size:32;index"`
	// Rule expression (PromQL, LogsQL, Zabbix trigger expression, etc.)
	Expression string `json:"expression" gorm:"type:text;not null"`
	// For duration (e.g., "5m" - alert must be firing for this duration)
	ForDuration string          `json:"for_duration" gorm:"size:32;default:0s"`
	Severity    AlertSeverity   `json:"severity" gorm:"size:32;not null;index"`
	Labels      JSONLabels      `json:"labels" gorm:"type:json"`
	Annotations JSONLabels      `json:"annotations" gorm:"type:json"` // summary, description templates
	Status      AlertRuleStatus `json:"status" gorm:"size:32;default:enabled;index"`
	// Grouping
	GroupName string `json:"group_name" gorm:"size:128;index"`
	Category  string `json:"category" gorm:"size:64;index;default:''"`
	// Version tracking
	Version   int  `json:"version" gorm:"default:1"`
	CreatedBy uint `json:"created_by" gorm:"index"`
	UpdatedBy uint `json:"updated_by"`
	// Evaluation interval in seconds (default 60)
	EvalInterval int `json:"eval_interval" gorm:"default:60"`
	// Recovery hold duration (留观时长) - e.g., "5m"
	RecoveryHold string `json:"recovery_hold" gorm:"size:32;default:0s"`
	// Group notification timing (Alertmanager-style)
	// GroupWaitSeconds: how long to buffer the first alert in a group before sending (0 = disabled)
	GroupWaitSeconds int `json:"group_wait_seconds" gorm:"not null;default:0"`
	// GroupIntervalSeconds: minimum interval between notifications for an ongoing group (0 = disabled)
	GroupIntervalSeconds int `json:"group_interval_seconds" gorm:"not null;default:0"`
	// NoData detection
	NoDataEnabled  bool   `json:"nodata_enabled" gorm:"default:false"`
	NoDataDuration string `json:"nodata_duration" gorm:"size:32;default:5m"` // after this duration of no data, fire nodata alert
	// Level suppression (for rules with multiple severity conditions)
	SuppressEnabled bool `json:"suppress_enabled" gorm:"default:false"`
	// Business group
	BizGroupID *uint `json:"biz_group_id" gorm:"index"`

	// Heartbeat monitoring (only relevant when RuleType="heartbeat")
	// HeartbeatToken is the unique token embedded in the ping URL: POST /heartbeat/:token
	HeartbeatToken    string     `json:"heartbeat_token" gorm:"size:128;not null;default:'';uniqueIndex"`
	// HeartbeatInterval is the expected ping interval in seconds.
	HeartbeatInterval int        `json:"heartbeat_interval" gorm:"not null;default:300"`
	// HeartbeatLastAt is the timestamp of the last received ping.
	HeartbeatLastAt   *time.Time `json:"heartbeat_last_at"`

	// SLA (Service Level Agreement) — 0 means disabled.
	// If AckSlaMinutes > 0 and the event is not acknowledged within this window, an escalation is triggered.
	AckSlaMinutes int `json:"ack_sla_minutes" gorm:"not null;default:0"`

	// ChannelID links this rule to a collaboration channel (v2).
	// When set, alerts fired by this rule are routed to the specified channel
	// instead of the default channel.
	ChannelID *uint    `json:"channel_id" gorm:"index"`
	Channel   *Channel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// MaskHeartbeatToken replaces the HeartbeatToken with a masked value
// (first 8 chars + "...") for safe exposure in API responses.
// Returns a copy so the original is not mutated.
func (r AlertRule) MaskHeartbeatToken() AlertRule {
	if len(r.HeartbeatToken) > 8 {
		r.HeartbeatToken = r.HeartbeatToken[:8] + "..."
	}
	return r
}

// AlertRuleHistory records changes to alert rules for audit trail.
type AlertRuleHistory struct {
	BaseModel
	RuleID     uint   `json:"rule_id" gorm:"index;not null"`
	Version    int    `json:"version" gorm:"not null"`
	ChangeType string `json:"change_type" gorm:"size:32;not null"` // created, updated, deleted
	Snapshot   string `json:"snapshot" gorm:"type:text;not null"`  // JSON snapshot of the rule
	ChangedBy  uint   `json:"changed_by" gorm:"index"`
}

func (AlertRuleHistory) TableName() string {
	return "alert_rule_histories"
}
