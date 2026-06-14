package model

import "time"

type StatusSubscription struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"size:255;uniqueIndex"`
	IsActive  bool      `json:"is_active"` // repo.Subscribe sets true explicitly; DB column keeps DEFAULT 1 for seeds
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (StatusSubscription) TableName() string { return "status_subscriptions" }
