package model

import "time"

// ReportTask defines a scheduled report generation job.
type ReportTask struct {
	BaseModel
	Name           string `json:"name" gorm:"size:128;not null"`
	Description    string `json:"description" gorm:"type:text;not null"`
	CronExpr       string `json:"cron_expr" gorm:"size:64;not null"`
	ReportType     string `json:"report_type" gorm:"size:32;not null;default:daily"`
	Scope          string `json:"scope" gorm:"type:json"`
	PromptTemplate string `json:"prompt_template" gorm:"type:text;not null"`
	AllowedTools   string `json:"allowed_tools" gorm:"type:json"`
	OutputChannels string `json:"output_channels" gorm:"type:json;not null"`
	Enabled        bool   `json:"enabled" gorm:"default:true"`
	CreatedBy      uint   `json:"created_by" gorm:"not null"`
}

func (ReportTask) TableName() string { return "report_tasks" }

// ReportRun records a single execution of a report task.
type ReportRun struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	TaskID           uint       `json:"task_id" gorm:"index;not null"`
	Status           string     `json:"status" gorm:"size:20;not null;default:running"` // running/success/failed
	StartedAt        time.Time  `json:"started_at" gorm:"not null"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	ReportMarkdown   string     `json:"report_markdown" gorm:"type:longtext"`
	ReportSummary    string     `json:"report_summary" gorm:"size:500"`
	FindingsJSON     string     `json:"findings_json" gorm:"type:json"`
	ErrorMsg         string     `json:"error_msg" gorm:"type:text"`
	AIConversationID *uint      `json:"ai_conversation_id"`
}

func (ReportRun) TableName() string { return "report_runs" }
