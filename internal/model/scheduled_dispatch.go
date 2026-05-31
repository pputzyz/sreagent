package model

import "time"

// ScheduledDispatchStatus defines the lifecycle of a scheduled dispatch.
type ScheduledDispatchStatus string

const (
	ScheduledDispatchPending    ScheduledDispatchStatus = "pending"
	ScheduledDispatchDispatched ScheduledDispatchStatus = "dispatched"
	ScheduledDispatchFailed     ScheduledDispatchStatus = "failed"
	ScheduledDispatchCancelled  ScheduledDispatchStatus = "cancelled"
	ScheduledDispatchExpired    ScheduledDispatchStatus = "expired"
)

// ScheduledDispatch represents a deferred or repeating notification dispatch.
// Created when a DispatchPolicy has delay_seconds > 0 or repeat_interval_seconds > 0.
// A background worker polls for due dispatches and sends notifications.
type ScheduledDispatch struct {
	ID             uint                    `gorm:"primaryKey" json:"id"`
	IncidentID     uint                    `gorm:"index;not null" json:"incident_id"`
	EventID        uint                    `gorm:"not null" json:"event_id"`
	Fingerprint    string                  `gorm:"index;size:256;not null" json:"fingerprint"`
	PolicyID       uint                    `gorm:"not null" json:"policy_id"`
	ChannelID      uint                    `gorm:"not null" json:"channel_id"`
	NotifyMode     string                  `gorm:"size:32;default:unified" json:"notify_mode"`
	DispatchAt     time.Time               `gorm:"index;not null" json:"dispatch_at"`
	RepeatCount    int                     `gorm:"default:0" json:"repeat_count"`
	MaxRepeats     int                     `gorm:"default:0" json:"max_repeats"`
	RepeatInterval int                     `gorm:"default:0" json:"repeat_interval"`
	Status         ScheduledDispatchStatus `gorm:"size:32;default:pending;index" json:"status"`
	LastError      string                  `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

func (ScheduledDispatch) TableName() string {
	return "scheduled_dispatches"
}
