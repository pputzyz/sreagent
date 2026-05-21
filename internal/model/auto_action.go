package model

import "time"

// AutoAction defines an automated remediation action with guardrails.
type AutoAction struct {
	BaseModel
	Name             string     `json:"name" gorm:"size:255;not null"`
	Description      string     `json:"description" gorm:"type:text"`
	ActionType       string     `json:"action_type" gorm:"size:50;not null"`
	Level            string     `json:"level" gorm:"size:10;default:L1"`
	TriggerLabels    JSONLabels `json:"trigger_labels" gorm:"type:json"`
	TriggerSeverity  string     `json:"trigger_severity" gorm:"size:20"`
	ActionConfig     JSONLabels `json:"action_config" gorm:"type:json"`
	Enabled          bool       `json:"enabled" gorm:"default:false"`
	DryRun           bool       `json:"dry_run" gorm:"default:true"`
	ApprovalRequired bool       `json:"approval_required" gorm:"default:true"`
	Confidence       int        `json:"confidence" gorm:"default:50"`
	SuccessCount     int        `json:"success_count" gorm:"default:0"`
	FailureCount     int        `json:"failure_count" gorm:"default:0"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	CreatedBy        *uint      `json:"created_by" gorm:"index"`
}

func (AutoAction) TableName() string { return "auto_actions" }

// AutoActionLog records a single execution of an auto action.
type AutoActionLog struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	ActionID     uint       `json:"action_id" gorm:"index;not null"`
	IncidentID   *uint      `json:"incident_id" gorm:"index"`
	UserID       *uint      `json:"user_id"`
	Status       string     `json:"status" gorm:"size:20;default:pending"`
	DryRun       bool       `json:"dry_run" gorm:"default:false"`
	InputContext string     `json:"input_context" gorm:"type:text"`
	OutputResult string     `json:"output_result" gorm:"type:text"`
	Error        string     `json:"error,omitempty" gorm:"type:text"`
	DurationMs   int64      `json:"duration_ms" gorm:"default:0"`
	ApprovedBy   *uint      `json:"approved_by"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (AutoActionLog) TableName() string { return "auto_action_logs" }
