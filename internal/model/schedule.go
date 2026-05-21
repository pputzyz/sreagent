package model

import "time"

// RotationType defines the rotation strategy.
type RotationType string

const (
	RotationDaily  RotationType = "daily"
	RotationWeekly RotationType = "weekly"
	RotationCustom RotationType = "custom"
)

// Schedule represents an on-call rotation schedule.
type Schedule struct {
	BaseModel
	Name         string       `json:"name" gorm:"size:128;not null"`
	TeamID       *uint        `json:"team_id" gorm:"index"` // optional: which team this schedule belongs to
	Team         *Team        `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	Description  string       `json:"description" gorm:"size:512"`
	RotationType RotationType `json:"rotation_type" gorm:"size:32;not null"`
	// Timezone for schedule calculations
	Timezone string `json:"timezone" gorm:"size:64;default:Asia/Shanghai"`
	// Handoff time (e.g., "09:00" - when rotation happens)
	// These rotation fields are used only for auto-generation of OnCallShift records.
	HandoffTime string `json:"handoff_time" gorm:"size:8;default:09:00"`
	// Handoff day for weekly rotation (0=Sunday, 1=Monday, ...)
	HandoffDay int  `json:"handoff_day" gorm:"default:1"`
	IsEnabled  bool `json:"is_enabled" gorm:"default:true"`
	// Default severity filter for the entire schedule.
	// Empty string = all severities; "critical" = only critical; "critical,warning" = both.
	SeverityFilter string `json:"severity_filter" gorm:"size:128"`
}

func (Schedule) TableName() string {
	return "schedules"
}

// ScheduleParticipant defines the rotation order.
type ScheduleParticipant struct {
	BaseModel
	ScheduleID uint `json:"schedule_id" gorm:"index;not null"`
	UserID     uint `json:"user_id" gorm:"index;not null"`
	User       User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	// Position in rotation order (0-based)
	Position int `json:"position" gorm:"not null"`
}

func (ScheduleParticipant) TableName() string {
	return "schedule_participants"
}

// ScheduleOverride represents a temporary override of the schedule.
type ScheduleOverride struct {
	BaseModel
	ScheduleID uint      `json:"schedule_id" gorm:"index;not null"`
	UserID     uint      `json:"user_id" gorm:"index;not null"` // who takes over
	User       User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	StartTime  time.Time `json:"start_time" gorm:"not null"`
	EndTime    time.Time `json:"end_time" gorm:"not null"`
	Reason     string    `json:"reason" gorm:"size:256"`
}

func (ScheduleOverride) TableName() string {
	return "schedule_overrides"
}

// EscalationPolicy defines what happens when alerts are not acknowledged.
type EscalationPolicy struct {
	BaseModel
	Name      string `json:"name" gorm:"size:128;not null"`
	TeamID    uint   `json:"team_id" gorm:"index;not null"`
	Team      Team   `json:"team,omitempty" gorm:"foreignKey:TeamID"`
	IsEnabled bool   `json:"is_enabled" gorm:"default:true"`
}

func (EscalationPolicy) TableName() string {
	return "escalation_policies"
}

// EscalationStep defines a step in the escalation policy.
type EscalationStep struct {
	BaseModel
	PolicyID        uint   `json:"policy_id" gorm:"index;not null"`
	StepOrder       int    `json:"step_order" gorm:"not null"`          // 1, 2, 3...
	DelayMinutes    int    `json:"delay_minutes" gorm:"not null"`       // wait X minutes before escalating
	TargetType      string `json:"target_type" gorm:"size:32;not null"` // user, schedule, team
	TargetID        uint   `json:"target_id" gorm:"not null"`           // ID of the target
	NotifyChannelID *uint  `json:"notify_channel_id" gorm:"index"`      // optional override channel
}

func (EscalationStep) TableName() string {
	return "escalation_steps"
}

// EscalationStepExecution records that an escalation step has been executed for an event.
// Used for atomic dedup via INSERT IGNORE.
type EscalationStepExecution struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	EventID    uint      `json:"event_id" gorm:"index;not null"`
	StepID     uint      `json:"step_id" gorm:"not null"`
	ExecutedAt time.Time `json:"executed_at" gorm:"not null;default:CURRENT_TIMESTAMP(3)"`
}

func (EscalationStepExecution) TableName() string {
	return "escalation_step_executions"
}

// OnCallShift represents a specific on-call time slot assigned to a person.
// This is the core of the schedule system - each shift has a clear owner and time range.
type OnCallShift struct {
	BaseModel
	ScheduleID uint      `json:"schedule_id" gorm:"index;not null"`
	UserID     uint      `json:"user_id" gorm:"index;not null"`
	User       User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	StartTime  time.Time `json:"start_time" gorm:"not null;index"`
	EndTime    time.Time `json:"end_time" gorm:"not null"`
	// Which severity levels should be dispatched to this person during their shift.
	// Empty string = all severities; "critical" = only critical; "critical,warning" = both.
	SeverityFilter string `json:"severity_filter" gorm:"size:128"`
	// Source: manual | rotation (auto-generated from rotation config)
	Source string `json:"source" gorm:"size:32;default:manual"`
	Note   string `json:"note" gorm:"size:256"`
}

func (OnCallShift) TableName() string { return "oncall_shifts" }
