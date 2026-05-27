package model

import "time"

// UserTeamNotifyPref stores a user's notification preference override for a team channel.
type UserTeamNotifyPref struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	TeamID    uint      `json:"team_id" gorm:"index;not null"`
	MediaID   uint      `json:"media_id" gorm:"not null"`
	IsMuted   bool      `json:"is_muted" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserTeamNotifyPref) TableName() string { return "user_team_notify_prefs" }
