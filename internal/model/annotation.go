package model

import "time"

// Annotation represents a time-range note attached to a dashboard panel,
// similar to Grafana annotations. Used for marking deployments, incidents,
// or any noteworthy events on dashboard timelines.
type Annotation struct {
	BaseModel
	DashboardID uint       `json:"dashboard_id" gorm:"index"`
	Time        time.Time  `json:"time" gorm:"index"`
	EndTime     *time.Time `json:"end_time"`
	Text        string     `json:"text" gorm:"size:1024"`
	Tags        JSONLabels `json:"tags" gorm:"type:json"`
	Source      string     `json:"source" gorm:"size:64"`  // 'user' | 'alert' | 'system'
	CreatedBy   uint       `json:"created_by"`
}

func (Annotation) TableName() string {
	return "annotations"
}
