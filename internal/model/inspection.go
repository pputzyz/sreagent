package model

import "time"

// InspectionTask defines a scheduled inspection job.
type InspectionTask struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Name           string     `json:"name" gorm:"size:128;not null"`
	Description    string     `json:"description" gorm:"type:text;not null"`
	CronExpr       string     `json:"cron_expr" gorm:"size:64;not null"`
	TargetType     string     `json:"target_type" gorm:"size:32;not null;default:global"` // global / biz_group
	TargetIDs      string     `json:"target_ids" gorm:"type:json"`                        // [1,2,3]
	AllowedTools   string     `json:"allowed_tools" gorm:"type:json"`                     // ["tool_a","tool_b"]
	OutputChannels string     `json:"output_channels" gorm:"type:json;not null"`          // channel config array
	Enabled        bool       `json:"enabled" gorm:"default:true"`
	CreatedBy      uint       `json:"created_by" gorm:"not null"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (InspectionTask) TableName() string { return "inspection_tasks" }

// InspectionRun records a single execution of an inspection task.
type InspectionRun struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	TaskID           uint       `json:"task_id" gorm:"index;not null"`
	Status           string     `json:"status" gorm:"size:20;not null;default:running"` // running/success/failed
	StartedAt        time.Time  `json:"started_at" gorm:"not null"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	ReportMarkdown   string     `json:"report_markdown" gorm:"type:longtext"`
	ReportSummary    string     `json:"report_summary" gorm:"size:500"`
	FindingsJSON     string     `json:"findings_json" gorm:"type:json"` // [{severity,category,object,detail}]
	ErrorMsg         string     `json:"error_msg" gorm:"type:text"`
	AIConversationID *uint      `json:"ai_conversation_id"`
}

func (InspectionRun) TableName() string { return "inspection_runs" }
