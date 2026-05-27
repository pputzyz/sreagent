package model

import "time"

// TeamNotifyChannel links a team to a notification media channel.
type TeamNotifyChannel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TeamID    uint      `json:"team_id" gorm:"index;not null"`
	MediaID   uint      `json:"media_id" gorm:"not null"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TeamNotifyChannel) TableName() string { return "team_notify_channels" }
