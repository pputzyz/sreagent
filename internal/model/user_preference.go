package model

// UserPreference stores per-user UI and notification preferences.
type UserPreference struct {
	BaseModel
	UserID                 uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	Theme                  string `json:"theme" gorm:"size:16;default:auto"`                   // auto | light | dark
	Language               string `json:"language" gorm:"size:16;default:zh-CN"`               // zh-CN | en
	Timezone               string `json:"timezone" gorm:"size:64;default:Asia/Shanghai"`       // IANA timezone
	DefaultTimeRange       string `json:"default_time_range" gorm:"size:16;default:24h"`       // 1h | 6h | 24h | 7d | 30d
	NotificationSeverities string `json:"notification_severities" gorm:"type:json"` // JSON array
	AIChatMode             string `json:"ai_chat_mode" gorm:"size:16;default:sidebar"`         // sidebar | modal | inline
}

func (UserPreference) TableName() string {
	return "user_preferences"
}
