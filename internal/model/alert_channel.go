package model

// AlertChannel is a virtual receiver that gets alerts matching specific labels.
// Example: "Payment Team Lark Group" receives all critical alerts with business_line=payment
type AlertChannel struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description" gorm:"size:512"`
	// Label matchers - alert must match ALL these labels
	MatchLabels JSONLabels `json:"match_labels" gorm:"type:json"`
	// Datasource filter (nil = wildcard, matches any datasource)
	DataSourceID *uint       `json:"datasource_id" gorm:"index"`
	DataSource   *DataSource `json:"datasource,omitempty" gorm:"foreignKey:DataSourceID"`
	// Severity filter (empty = all)
	Severities string `json:"severities" gorm:"size:128"` // "critical,warning"
	// Notification target
	MediaID    uint  `json:"media_id" gorm:"index;not null"` // which NotifyMedia to use
	TemplateID *uint `json:"template_id" gorm:"index"`       // optional template override
	// Throttle in minutes (0 = no throttle)
	ThrottleMin int  `json:"throttle_min" gorm:"default:5"`
	IsEnabled   bool `json:"is_enabled"` // create handler defaults to true; DB column keeps DEFAULT 1 for seeds
	CreatedBy   uint `json:"created_by" gorm:"index"`
}

func (AlertChannel) TableName() string { return "alert_channels" }
