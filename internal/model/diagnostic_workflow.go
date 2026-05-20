package model

import "time"

// DiagnosticWorkflow defines a diagnostic SOP template.
type DiagnosticWorkflow struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"size:255;not null"`
	Description     string     `json:"description" gorm:"type:text"`
	TriggerLabels   JSONLabels `json:"trigger_labels" gorm:"type:json"`
	TriggerSeverity string     `json:"trigger_severity" gorm:"size:20"`
	Category        string     `json:"category" gorm:"size:50;default:general"`
	Enabled         bool       `json:"enabled" gorm:"default:true"`
	MaxSteps        int        `json:"max_steps" gorm:"default:10"`
	RequireApproval bool       `json:"require_approval" gorm:"default:true"`
	CreatedBy       *uint      `json:"created_by" gorm:"index"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (DiagnosticWorkflow) TableName() string { return "diagnostic_workflows" }

// DiagnosticWorkflowStep defines a single step in a diagnostic workflow.
type DiagnosticWorkflowStep struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	WorkflowID     uint   `json:"workflow_id" gorm:"index;not null"`
	StepOrder      int    `json:"step_order" gorm:"default:0"`
	Name           string `json:"name" gorm:"size:255;not null"`
	StepType       string `json:"step_type" gorm:"size:20;default:query"`
	DatasourceID   *uint  `json:"datasource_id"`
	Expression     string `json:"expression" gorm:"type:text"`
	ConditionExpr  string `json:"condition_expr" gorm:"size:500"`
	AutoAdvance    bool   `json:"auto_advance" gorm:"default:true"`
	TimeoutSeconds int    `json:"timeout_seconds" gorm:"default:30"`
	OnFailure      string `json:"on_failure" gorm:"size:20;default:continue"`
	CreatedAt      time.Time `json:"created_at"`
}

func (DiagnosticWorkflowStep) TableName() string { return "diagnostic_workflow_steps" }

// DiagnosticRun records an execution of a diagnostic workflow.
type DiagnosticRun struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	WorkflowID    uint       `json:"workflow_id" gorm:"index;not null"`
	IncidentID    *uint      `json:"incident_id" gorm:"index"`
	UserID        *uint      `json:"user_id"`
	Status        string     `json:"status" gorm:"size:20;default:pending"`
	CurrentStep   int        `json:"current_step" gorm:"default:0"`
	ResultSummary string     `json:"result_summary" gorm:"type:text"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (DiagnosticRun) TableName() string { return "diagnostic_runs" }

// DiagnosticRunStep records the result of a single step execution.
type DiagnosticRunStep struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	RunID       uint       `json:"run_id" gorm:"index;not null"`
	StepOrder   int        `json:"step_order" gorm:"default:0"`
	StepName    string     `json:"step_name" gorm:"size:255"`
	StepType    string     `json:"step_type" gorm:"size:20"`
	Expression  string     `json:"expression" gorm:"type:text"`
	Result      string     `json:"result" gorm:"type:text"`
	Status      string     `json:"status" gorm:"size:20;default:pending"`
	DurationMs  int64      `json:"duration_ms" gorm:"default:0"`
	Error       string     `json:"error,omitempty" gorm:"type:text"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (DiagnosticRunStep) TableName() string { return "diagnostic_run_steps" }
