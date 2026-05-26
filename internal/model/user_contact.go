package model

// UserContact stores a user's notification contact (email, phone, feishu, etc.).
type UserContact struct {
	BaseModel
	UserID    uint   `gorm:"index" json:"user_id"`
	Type      string `gorm:"size:32" json:"type"`     // email, phone, feishu, wecom, dingtalk, webhook
	Value     string `gorm:"size:256" json:"value"`   // the contact value
	Name      string `gorm:"size:64" json:"name"`     // user-friendly name
	IsDefault bool   `gorm:"default:false" json:"is_default"`
}

func (UserContact) TableName() string {
	return "user_contacts"
}
