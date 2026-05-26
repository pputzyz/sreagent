package model

import "time"

// RuleQuery represents a single query within a multi-query alert rule.
// Each query has a reference label (A, B, C...) and its own PromQL expression.
type RuleQuery struct {
	Ref          string `json:"ref"`            // A, B, C...
	DatasourceID uint   `json:"datasource_id"`  // datasource to query against
	Expr         string `json:"expr"`           // PromQL / LogsQL expression
	Legend       string `json:"legend"`         // display format for the result
}

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

// IsValid returns true if the severity is a recognized value.
func (s AlertSeverity) IsValid() bool {
	switch s {
	case SeverityCritical, SeverityWarning, SeverityInfo,
		SeverityP0, SeverityP1, SeverityP2, SeverityP3, SeverityP4:
		return true
	}
	return false
}

// AlertRuleStatus defines the status of an alert rule.
// - draft: AI-generated rule, not yet activated by the user.
// - active: actively evaluating.
// - disabled: paused / not evaluating.
type AlertRuleStatus string

const (
	RuleStatusDraft    AlertRuleStatus = "draft"    // AI-generated, not yet activated
	RuleStatusActive   AlertRuleStatus = "active"
	RuleStatusDisabled AlertRuleStatus = "disabled"
)

// IsValid returns true if the status is a recognized value.
func (s AlertRuleStatus) IsValid() bool {
	switch s {
	case RuleStatusDraft, RuleStatusActive, RuleStatusDisabled:
		return true
	}
	return false
}

// AlertRuleType identifies the evaluation strategy for a rule.
type AlertRuleType string

const (
	RuleTypeThreshold AlertRuleType = "threshold" // default: PromQL/LogQL/Zabbix expression
	RuleTypeHeartbeat AlertRuleType = "heartbeat" // fire when no ping received within interval
)

// VarConfig defines variable filling configuration for alert rules.
// Inspired by Nightingale's $host/$val variable replacement system.
// When set, the evaluator substitutes variables in the expression with actual values.
type VarConfig struct {
	// Strategy: "before_query" (substitute variables then query) or
	//           "after_query" (query first, then filter by variable values).
	// before_query is required when the expression contains aggregation functions
	// (sum, avg, etc.) that would lose the variable label after grouping.
	Strategy string     `json:"strategy"`
	Params   []VarParam `json:"params"` // variable definitions
}

// VarParam defines a single variable parameter.
type VarParam struct {
	Name   string   `json:"name"`   // variable name (e.g., "host", "device")
	Type   string   `json:"type"`   // "host", "device", "enum"
	Query  string   `json:"query"`  // optional filter query (JSON-encoded, for host/device type)
	Values []string `json:"values"` // explicit values (used when type="enum", or as fallback)
}

// AlertRule represents an alerting rule definition.
type AlertRule struct {
	BaseModel
	// RuleType controls the evaluation strategy. Default is "threshold".
	RuleType    AlertRuleType `json:"rule_type" gorm:"size:32;not null;default:threshold"`
	Name         string     `json:"name" gorm:"size:256;not null;index"`
	TeamID       *uint      `json:"team_id" gorm:"index"` // optional: which team owns this rule
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
	Status      AlertRuleStatus `json:"status" gorm:"size:32;default:active;index"`
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
	// Multi-query support (Nightingale-style)
	// When Queries is non-empty, the rule uses multi-query evaluation:
	//   1. Each query (A, B, C...) is evaluated independently
	//   2. Results are joined according to JoinType and JoinKeys
	//   3. TriggerExp is evaluated against the combined results (referencing $A, $B, etc.)
	// When Queries is empty, the rule falls back to single Expression evaluation (backward compatible).
	Queries    []RuleQuery `json:"queries" gorm:"serializer:json"`    // multiple queries
	TriggerExp string      `json:"trigger_exp" gorm:"size:512"`       // trigger expression referencing $A, $B
	JoinType   string      `json:"join_type" gorm:"size:32"`          // inner_join, left_join, right_join, none
	JoinKeys   []string    `json:"join_keys" gorm:"serializer:json"`  // label keys to join on

	// Business group
	BizGroupID *uint `json:"biz_group_id" gorm:"index"`
	// Variable filling config — enables $var replacement in expression.
	// When nil, the rule evaluates normally without variable substitution.
	VarConfig *VarConfig `json:"var_config" gorm:"column:var_config;type:json;serializer:json"`

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
