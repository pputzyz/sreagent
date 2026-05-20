package model

import "time"

// ChangeEvent represents a CI/CD or infrastructure change event.
type ChangeEvent struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Source      string     `json:"source" gorm:"size:50;not null"`
	ChangeType  string     `json:"change_type" gorm:"size:50;not null;default:deploy"`
	Service     string     `json:"service" gorm:"size:255;default:''"`
	Environment string     `json:"environment" gorm:"size:50;default:''"`
	CommitSHA   string     `json:"commit_sha" gorm:"size:64;default:''"`
	Author      string     `json:"author" gorm:"size:255;default:''"`
	Description string     `json:"description" gorm:"type:text"`
	RiskLevel   string     `json:"risk_level" gorm:"size:20;default:low"`
	Metadata    JSONLabels `json:"metadata" gorm:"type:json"`
	Timestamp   time.Time  `json:"timestamp"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (ChangeEvent) TableName() string { return "change_events" }
