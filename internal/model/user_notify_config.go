package model

// UserNotifyConfig stores a user's personal notification contact info.
// A user can have multiple configs, one per media_type.
// When an alert is dispatched to a user (via on-call), the system uses
// all enabled configs to notify them.
type UserNotifyConfig struct {
	BaseModel
	UserID    uint   `json:"user_id" gorm:"uniqueIndex:udx_user_media;not null"`
	MediaType string `json:"media_type" gorm:"size:32;uniqueIndex:udx_user_media"` // "lark_personal", "email", "webhook"
	// lark_personal: {"lark_user_id": "xxx"}
	// email: {"email": "user@example.com"}
	// webhook: {"url": "https://..."}
	Config    string `json:"config" gorm:"type:text"`
	IsEnabled bool   `json:"is_enabled"` // create handler defaults to true; DB column keeps DEFAULT 1 for seeds
}

func (UserNotifyConfig) TableName() string { return "user_notify_configs" }
