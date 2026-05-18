package model

import "time"

type LabelRegistry struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	DatasourceID uint      `json:"datasource_id" gorm:"index;not null"`
	LabelKey     string    `json:"label_key" gorm:"size:128;not null;index"`
	LabelValue   string    `json:"label_value" gorm:"size:2048;not null"`
	Source       string    `json:"source" gorm:"size:100;index;default:''"` // "sync", "event", "manual"
	LastSeenAt   time.Time `json:"last_seen_at" gorm:"not null"`
	HitCount     uint      `json:"hit_count" gorm:"not null;default:1"`
}

func (LabelRegistry) TableName() string { return "label_registry" }
