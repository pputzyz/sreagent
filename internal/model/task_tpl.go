package model

// TaskTpl defines a reusable task template for self-healing script execution.
// Ported from Nightingale TaskTpl.
type TaskTpl struct {
	BaseModel
	Name      string `gorm:"size:128;uniqueIndex" json:"name"`
	Script    string `gorm:"type:longtext" json:"script"`
	Args      string `gorm:"size:512" json:"args"`
	Batch     int    `gorm:"default:0" json:"batch"`     // 0=all at once
	Tolerance int    `gorm:"default:0" json:"tolerance"` // allowed failures
	Timeout   int    `gorm:"default:60" json:"timeout"`  // seconds
	Account   string `gorm:"size:64" json:"account"`     // SSH account (username)
	Password  string `gorm:"size:256" json:"-"`          // SSH password (encrypted)
	Pause     string `gorm:"size:256" json:"pause"`      // pause between batches
	Hosts     string `gorm:"type:text" json:"hosts"`     // JSON array of host endpoints
	Tags      string `gorm:"type:text" json:"tags"`      // JSON array
	Note      string `gorm:"size:512" json:"note"`
	CreateBy  string `gorm:"size:64" json:"create_by"`
	UpdateBy  string `gorm:"size:64" json:"update_by"`
}

func (TaskTpl) TableName() string { return "task_tpls" }
