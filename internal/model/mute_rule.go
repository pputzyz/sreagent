package model

import "time"

// MuteRule defines a rule for suppressing alert notifications.
type MuteRule struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	// Label matchers - alert must match ALL these labels to be muted
	MatchLabels JSONLabels `json:"match_labels" gorm:"type:json"`
	// Severity filter (empty = all severities)
	Severities string `json:"severities" gorm:"size:128"`
	// Time-based muting (one-time window)
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	// Periodic muting (e.g., every day 02:00-06:00)
	PeriodicStart string `json:"periodic_start" gorm:"size:8"` // "02:00"
	PeriodicEnd   string `json:"periodic_end" gorm:"size:8"`   // "06:00"
	DaysOfWeek    string `json:"days_of_week" gorm:"size:32"`  // "1,2,3,4,5" (Mon-Fri)
	Timezone      string `json:"timezone" gorm:"size:64;default:Asia/Shanghai"`
	// Who created it
	CreatedBy uint `json:"created_by" gorm:"index"`
	IsEnabled bool `json:"is_enabled"` // create handler defaults to true; DB column keeps DEFAULT 1 for seeds
	// Optional: specific rule IDs to mute (empty = match by labels)
	RuleIDs string `json:"rule_ids" gorm:"size:512"` // comma-separated
}

func (MuteRule) TableName() string {
	return "mute_rules"
}
