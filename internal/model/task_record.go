package model

// TaskRecord records a single task execution (manual or alert-triggered).
// Ported from Nightingale TaskRecord.
type TaskRecord struct {
	BaseModel
	TplID     uint   `gorm:"index" json:"tpl_id"`
	EventID   uint   `gorm:"index" json:"event_id"` // 0 if manual
	Title     string `gorm:"size:256" json:"title"`
	Account   string `gorm:"size:64" json:"account"`
	Password  string `gorm:"size:256" json:"-"` // SSH password (encrypted)
	Batch     int    `json:"batch"`
	Tolerance int    `json:"tolerance"`
	Timeout   int    `json:"timeout"`
	Script    string `gorm:"type:longtext" json:"script"`
	Args      string `gorm:"size:512" json:"args"`
	Hosts     string `gorm:"type:text" json:"hosts"`  // JSON array
	Status    int    `gorm:"default:0" json:"status"` // 0=pending 1=running 2=success 3=fail
	CreateBy  string `gorm:"size:64" json:"create_by"`
}

func (TaskRecord) TableName() string { return "task_records" }

// Task status constants.
const (
	TaskStatusPending = 0
	TaskStatusRunning = 1
	TaskStatusSuccess = 2
	TaskStatusFail    = 3
)
