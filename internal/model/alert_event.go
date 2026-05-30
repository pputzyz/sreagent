package model

import "time"

// AlertEventStatus defines the lifecycle status of an alert event.
type AlertEventStatus string

const (
	EventStatusFiring       AlertEventStatus = "firing"
	EventStatusAcknowledged AlertEventStatus = "acknowledged"
	EventStatusAssigned     AlertEventStatus = "assigned"
	EventStatusSilenced     AlertEventStatus = "silenced"
	EventStatusResolved     AlertEventStatus = "resolved"
	EventStatusClosed       AlertEventStatus = "closed"
)

// IsValid returns true if the status is a recognized value.
func (s AlertEventStatus) IsValid() bool {
	switch s {
	case EventStatusFiring, EventStatusAcknowledged, EventStatusAssigned,
		EventStatusSilenced, EventStatusResolved, EventStatusClosed:
		return true
	}
	return false
}

// AlertEvent represents an instance of an alert firing.
// This is the unified model that serves both the v1 engine lifecycle
// (firing → acknowledged → assigned → silenced → resolved → closed)
// and the v2 pipeline snapshot events (linked to an Alert via AlertID).
type AlertEvent struct {
	BaseModel
	// Fingerprint for deduplication (hash of labels + rule)
	Fingerprint string           `json:"fingerprint" gorm:"size:64;index;not null"`
	RuleID      *uint            `json:"rule_id" gorm:"index"`
	Rule        *AlertRule       `json:"rule,omitempty" gorm:"foreignKey:RuleID"`
	AlertName   string           `json:"alert_name" gorm:"size:256;not null;index"`
	Severity    AlertSeverity    `json:"severity" gorm:"size:32;not null;index"`
	Status      AlertEventStatus `json:"status" gorm:"size:32;not null;index;default:firing"`
	Labels      JSONLabels       `json:"labels" gorm:"type:json"`
	Annotations JSONLabels       `json:"annotations" gorm:"type:json"`
	// Source information
	Source       string `json:"source" gorm:"size:128"` // datasource name or external
	DataSourceID *uint  `json:"datasource_id" gorm:"index"` // which datasource triggered this event
	GeneratorURL string `json:"generator_url" gorm:"size:512"`
	// Timestamps
	FiredAt    time.Time  `json:"fired_at" gorm:"not null;index"`
	AckedAt    *time.Time `json:"acked_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	ClosedAt   *time.Time `json:"closed_at"`
	// Assignment
	AckedBy      *uint `json:"acked_by" gorm:"index"`
	AckedByUser  *User `json:"acked_by_user,omitempty" gorm:"foreignKey:AckedBy"`
	AssignedTo   *uint `json:"assigned_to" gorm:"index"`
	AssignedUser *User `json:"assigned_to_user,omitempty" gorm:"foreignKey:AssignedTo"`
	// Silence
	SilencedUntil *time.Time `json:"silenced_until" gorm:"index"`
	SilenceReason string     `json:"silence_reason" gorm:"size:512"`
	// Resolution
	Resolution string `json:"resolution" gorm:"type:text"`
	// Count of occurrences (for dedup grouping)
	FireCount int `json:"fire_count" gorm:"default:1"`
	// OnCall dispatch fields
	OnCallUserID *uint `json:"oncall_user_id" gorm:"index"`        // user assigned via on-call
	IsDispatched bool  `json:"is_dispatched" gorm:"default:false"` // whether an on-call assignment was made
	// Lark Bot API message ID — set when the alert card was sent via Bot API (not Incoming Webhook).
	// Non-empty value enables in-place card updates on status change.
	LarkMessageID string `json:"lark_message_id" gorm:"size:128;default:''"`
	// SlaEscalatedAt records when the SLA breach escalation was fired for this event.
	// Nil means no SLA escalation has been triggered yet.
	SlaEscalatedAt *time.Time `json:"sla_escalated_at"`
	// EscalationPolicyID is the specific escalation policy assigned to this event
	// via dispatch policy matching. When set, the escalation executor will prefer
	// this policy over team/global matching.
	EscalationPolicyID *uint `json:"escalation_policy_id,omitempty" gorm:"index"`
	// --- v2 pipeline fields (unified from alert_events_v2) ---
	// AlertID links this event to a v2 Alert record. Nil for pure v1 engine events.
	AlertID *uint  `json:"alert_id,omitempty" gorm:"index"`
	Alert   *Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
	// Value is the metric value at evaluation time (v2 pipeline events).
	Value float64 `json:"value" gorm:"default:0"`
}

func (AlertEvent) TableName() string {
	return "alert_events"
}

// AlertTimelineAction defines actions tracked in the timeline.
type AlertTimelineAction string

const (
	TimelineActionCreated      AlertTimelineAction = "created"
	TimelineActionAcknowledged AlertTimelineAction = "acknowledged"
	TimelineActionAssigned     AlertTimelineAction = "assigned"
	TimelineActionCommented    AlertTimelineAction = "commented"
	TimelineActionEscalated    AlertTimelineAction = "escalated"
	TimelineActionResolved     AlertTimelineAction = "resolved"
	TimelineActionClosed       AlertTimelineAction = "closed"
	TimelineActionReopened     AlertTimelineAction = "reopened"
	TimelineActionNotified     AlertTimelineAction = "notified"
	TimelineActionSilenced     AlertTimelineAction = "silenced"
	TimelineActionUnsilenced   AlertTimelineAction = "unsilenced"
	TimelineActionDispatched   AlertTimelineAction = "dispatched" // auto-assigned via on-call
)

// AlertTimeline records the lifecycle events of an alert.
type AlertTimeline struct {
	BaseModel
	EventID          uint                `json:"event_id" gorm:"index;not null"`
	Action           AlertTimelineAction `json:"action" gorm:"size:32;not null"`
	OperatorID       *uint               `json:"operator_id" gorm:"index"`
	Operator         *User               `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
	Note             string              `json:"note" gorm:"type:text"`
	Extra            string              `json:"extra" gorm:"type:json"`                          // additional context as JSON
	EscalationStepID *uint               `json:"escalation_step_id,omitempty" gorm:"index"`       // links to escalation_steps.id for dedup
}

func (AlertTimeline) TableName() string {
	return "alert_timelines"
}

// ViewAlertEvent is the API response type for the v2 alert events endpoint.
// It mirrors the old AlertEventV2 JSON shape so the frontend does not need changes.
type ViewAlertEvent struct {
	ID            uint              `json:"id"`
	AlertID       uint              `json:"alert_id"`
	EventStatus   AlertEventV2Status `json:"event_status"`
	EventSeverity AlertSeverity     `json:"event_severity"`
	Labels        JSONLabels        `json:"labels"`
	Annotations   JSONLabels        `json:"annotations"`
	Value         float64           `json:"value"`
	Timestamp     time.Time         `json:"timestamp"`
	Fingerprint   string            `json:"fingerprint"`
	CreatedAt     time.Time         `json:"created_at"`
}

// ToViewAlertEvent converts an AlertEvent to the ViewAlertEvent API response format.
func (e *AlertEvent) ToViewAlertEvent() ViewAlertEvent {
	v := ViewAlertEvent{
		ID:            e.ID,
		EventSeverity: e.Severity,
		Labels:        e.Labels,
		Annotations:   e.Annotations,
		Value:         e.Value,
		Timestamp:     e.FiredAt,
		Fingerprint:   e.Fingerprint,
		CreatedAt:     e.CreatedAt,
	}
	if e.AlertID != nil {
		v.AlertID = *e.AlertID
	}
	// Map lifecycle status to v2 firing/resolved
	switch e.Status {
	case EventStatusResolved, EventStatusClosed:
		v.EventStatus = AlertEventV2StatusResolved
	default:
		v.EventStatus = AlertEventV2StatusFiring
	}
	return v
}
