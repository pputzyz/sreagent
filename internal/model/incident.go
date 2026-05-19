package model

import "time"

// IncidentStatus defines the processing progress of an incident.
type IncidentStatus string

const (
	IncidentStatusTriggered  IncidentStatus = "triggered"  // 待处理 (open)
	IncidentStatusProcessing IncidentStatus = "processing" // 处理中 (acknowledged)
	IncidentStatusClosed     IncidentStatus = "closed"     // 已关闭
	IncidentStatusSnoozed    IncidentStatus = "snoozed"    // 暂缓中
)

// IncidentSeverity uses the same severity as alerts but limited to 3 levels.
type IncidentSeverity string

const (
	IncidentSeverityCritical IncidentSeverity = "critical"
	IncidentSeverityWarning  IncidentSeverity = "warning"
	IncidentSeverityInfo     IncidentSeverity = "info"
)

// Incident represents a fault/problem that may aggregate multiple alerts.
// It is the primary unit of work for on-call responders.
// Modeled after FlashCat's "故障" concept.
type Incident struct {
	BaseModel
	// Title can be auto-generated from first alert or manually set.
	Title       string           `json:"title" gorm:"size:512;not null;index"`
	Description string           `json:"description" gorm:"type:text"`
	Severity    IncidentSeverity `json:"severity" gorm:"size:32;not null;index;default:warning"`
	Status      IncidentStatus   `json:"status" gorm:"size:32;not null;index;default:triggered"`

	// Channel association
	ChannelID uint     `json:"channel_id" gorm:"index;not null"`
	Channel   *Channel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`

	// Fingerprint links this incident to alert events with the same fingerprint.
	Fingerprint string `json:"fingerprint" gorm:"size:64;index"`

	// Labels inherited from the first alert
	Labels JSONLabels `json:"labels" gorm:"type:json"`

	// --- Assignment ---
	// AssignedTo stores comma-separated user IDs or a JSON array (future).
	// For now we use a simple nullable uint for primary assignee.
	AssignedTo   *uint `json:"assigned_to" gorm:"index"`
	AssignedUser *User `json:"assigned_user,omitempty" gorm:"foreignKey:AssignedTo"`

	// --- Timestamps ---
	TriggeredAt    time.Time  `json:"triggered_at" gorm:"not null"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"` // first acknowledgement
	ResolvedAt     *time.Time `json:"resolved_at"`     // all alerts recovered
	ClosedAt       *time.Time `json:"closed_at"`       // manually or auto closed

	// --- Snooze (暂缓) ---
	SnoozedUntil *time.Time `json:"snoozed_until"`

	// --- Counters (denormalized) ---
	AlertCount int `json:"alert_count" gorm:"default:0"`
	EventCount int `json:"event_count" gorm:"default:0"`

	// --- Recovery state ---
	// IsRecovered indicates all associated alerts have recovered.
	IsRecovered bool `json:"is_recovered" gorm:"default:false"`

	// --- Dispatch tracking ---
	// EscalationPolicyID: which policy dispatched this incident.
	EscalationPolicyID *uint             `json:"escalation_policy_id" gorm:"index"`
	EscalationPolicy   *EscalationPolicy `json:"escalation_policy,omitempty" gorm:"foreignKey:EscalationPolicyID"`
	// CurrentEscalationStep: current step index in the escalation (0-based).
	CurrentEscalationStep int `json:"current_escalation_step" gorm:"default:0"`

	// --- Merge ---
	// MergedIntoID: if this incident was merged into another, points to the target.
	MergedIntoID *uint     `json:"merged_into_id" gorm:"index"`
	MergedInto   *Incident `json:"merged_into,omitempty" gorm:"foreignKey:MergedIntoID"`
}

func (Incident) TableName() string {
	return "incidents"
}

// IncidentAssignee tracks all people assigned to an incident and their ack status.
type IncidentAssignee struct {
	BaseModel
	IncidentID     uint       `json:"incident_id" gorm:"uniqueIndex:idx_incident_user;not null"`
	UserID         uint       `json:"user_id" gorm:"uniqueIndex:idx_incident_user;not null"`
	User           *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	IsAcknowledged bool       `json:"is_acknowledged" gorm:"default:false"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	AssignedAt     time.Time  `json:"assigned_at" gorm:"not null"`
	// Source: "policy" (from escalation) or "manual" (direct assign/reassign)
	Source string `json:"source" gorm:"size:32;default:policy"`
}

func (IncidentAssignee) TableName() string {
	return "incident_assignees"
}

// IncidentTimelineAction defines actions tracked in incident timeline.
type IncidentTimelineAction string

const (
	IncidentActionTriggered    IncidentTimelineAction = "triggered"
	IncidentActionAcknowledged IncidentTimelineAction = "acknowledged"
	IncidentActionUnacked      IncidentTimelineAction = "unacknowledged" // cancel ack
	IncidentActionSnoozed      IncidentTimelineAction = "snoozed"
	IncidentActionSnoozeExpired IncidentTimelineAction = "snooze_expired"
	IncidentActionEscalated    IncidentTimelineAction = "escalated"
	IncidentActionReassigned   IncidentTimelineAction = "reassigned"
	IncidentActionAddedAssignee IncidentTimelineAction = "added_assignee"
	IncidentActionResolved     IncidentTimelineAction = "resolved"
	IncidentActionClosed       IncidentTimelineAction = "closed"
	IncidentActionReopened     IncidentTimelineAction = "reopened"
	IncidentActionMerged       IncidentTimelineAction = "merged"
	IncidentActionCommented    IncidentTimelineAction = "commented"
	IncidentActionNotified     IncidentTimelineAction = "notified"
	IncidentActionAlertMerged  IncidentTimelineAction = "alert_merged"  // new alert merged in
	IncidentActionStormWarning IncidentTimelineAction = "storm_warning" // storm threshold hit
	IncidentActionSeverityChanged IncidentTimelineAction = "severity_changed"
	IncidentActionTitleChanged    IncidentTimelineAction = "title_changed"
)

// IncidentTimeline records the lifecycle events of an incident.
type IncidentTimeline struct {
	BaseModel
	IncidentID uint                   `json:"incident_id" gorm:"index;not null"`
	Action     IncidentTimelineAction `json:"action" gorm:"size:32;not null"`
	ActorID    *uint                  `json:"actor_id" gorm:"index"`
	Actor      *User                  `json:"actor,omitempty" gorm:"foreignKey:ActorID"`
	Content    string                 `json:"content" gorm:"type:text"` // human-readable description or comment text
	Extra      string                 `json:"extra" gorm:"type:json"`  // additional structured data
}

func (IncidentTimeline) TableName() string {
	return "incident_timelines"
}

// PostMortem represents a post-incident review/retrospective report.
type PostMortem struct {
	BaseModel
	IncidentID uint      `json:"incident_id" gorm:"uniqueIndex;not null"`
	Incident   *Incident `json:"incident,omitempty" gorm:"foreignKey:IncidentID"`
	Title      string    `json:"title" gorm:"size:256;not null"`
	Content    string    `json:"content" gorm:"type:longtext"` // Markdown content
	// Status: draft / published
	Status    string `json:"status" gorm:"size:32;default:draft"`
	AuthorID  *uint  `json:"author_id" gorm:"index"`
	Author    *User  `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	PublishedAt *time.Time `json:"published_at"`
}

func (PostMortem) TableName() string {
	return "post_mortems"
}
