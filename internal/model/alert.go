package model

import "time"

// AlertStatus defines the status of an alert (deduplicated series).
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// Alert represents a deduplicated alert series identified by alert_key.
// Multiple Events can belong to the same Alert.
// An Alert can be linked to an Incident via incident_id and to a Channel via channel_id.
type Alert struct {
	BaseModel
	// AlertKey is the deduplication key (e.g. hash of rule_id + labels subset).
	AlertKey string      `json:"alert_key" gorm:"size:128;uniqueIndex;not null"`
	Title    string      `json:"title" gorm:"size:512;not null;index"`
	Severity AlertSeverity `json:"severity" gorm:"size:32;not null;index;default:warning"`
	Status   AlertStatus `json:"status" gorm:"size:32;not null;index;default:firing"`

	// Source alert rule (optional — webhook-based alerts may not have a rule)
	RuleID *uint      `json:"rule_id" gorm:"index"`
	Rule   *AlertRule `json:"rule,omitempty" gorm:"foreignKey:RuleID"`

	// Labels & Annotations (from the most recent event)
	Labels      JSONLabels `json:"labels" gorm:"type:json"`
	Annotations JSONLabels `json:"annotations" gorm:"type:json"`

	// Channel association (which 协作空间 this alert belongs to)
	ChannelID *uint    `json:"channel_id" gorm:"index"`
	Channel   *Channel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`

	// Incident association (which 故障 this alert is grouped into)
	IncidentID *uint     `json:"incident_id" gorm:"index"`
	Incident   *Incident `json:"incident,omitempty" gorm:"foreignKey:IncidentID"`

	// Source information
	Source       string `json:"source" gorm:"size:128"`
	GeneratorURL string `json:"generator_url" gorm:"size:512"`

	// Timestamps
	FirstFiredAt time.Time  `json:"first_fired_at" gorm:"not null;index"`
	LastFiredAt  time.Time  `json:"last_fired_at" gorm:"not null;index"`
	ResolvedAt   *time.Time `json:"resolved_at"`

	// Counters
	EventCount int `json:"event_count" gorm:"default:1"`
	FireCount  int `json:"fire_count" gorm:"default:1"`
}

func (Alert) TableName() string {
	return "alerts"
}

// AlertEventV2Status defines the status of a single alert event.
type AlertEventV2Status string

const (
	AlertEventV2StatusFiring   AlertEventV2Status = "firing"
	AlertEventV2StatusResolved AlertEventV2Status = "resolved"
)

// AlertEventV2 represents a single firing or resolution event for an Alert.
// This is the raw event data received from evaluators or webhooks.
type AlertEventV2 struct {
	BaseModel
	AlertID uint   `json:"alert_id" gorm:"index;not null"`
	Alert   *Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID"`

	// Event-level fields
	EventStatus   AlertEventV2Status `json:"event_status" gorm:"size:32;not null;index"`
	EventSeverity AlertSeverity      `json:"event_severity" gorm:"size:32;not null"`
	Labels        JSONLabels         `json:"labels" gorm:"type:json"`
	Annotations   JSONLabels         `json:"annotations" gorm:"type:json"`
	Value         float64            `json:"value"`     // metric value at evaluation time
	Timestamp     time.Time          `json:"timestamp" gorm:"not null;index"` // when the event was generated

	// Fingerprint for deduplication within the alert
	Fingerprint string `json:"fingerprint" gorm:"size:64;index"`
}

func (AlertEventV2) TableName() string {
	return "alert_events_v2"
}
