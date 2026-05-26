package model

// TaskHostRecord records the execution result of a task on a single host.
type TaskHostRecord struct {
	BaseModel
	TaskID     uint   `gorm:"index" json:"task_id"`
	Host       string `gorm:"size:256" json:"host"`
	Status     int    `gorm:"default:0" json:"status"` // 0=pending 1=running 2=success 3=fail
	Stdout     string `gorm:"type:longtext" json:"stdout"`
	Stderr     string `gorm:"type:longtext" json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

func (TaskHostRecord) TableName() string { return "task_host_records" }
