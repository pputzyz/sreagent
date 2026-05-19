package model

// UserNotificationType defines the kind of user notification.
type UserNotificationType string

const (
	UserNotificationAlert    UserNotificationType = "alert"
	UserNotificationIncident UserNotificationType = "incident"
	UserNotificationSystem   UserNotificationType = "system"
	UserNotificationTodo     UserNotificationType = "todo"
)

// UserNotification represents a user-targeted notification for the notification center.
type UserNotification struct {
	BaseModel
	UserID   uint                 `json:"user_id" gorm:"index;not null"`
	Title    string               `json:"title" gorm:"size:256;not null"`
	Content  string               `json:"content" gorm:"size:1024"`
	Type     UserNotificationType `json:"type" gorm:"size:32;not null;default:system"`
	IsRead   bool                 `json:"is_read" gorm:"index;default:false"`
	Link     string               `json:"link" gorm:"size:512"`
	Metadata JSONLabels           `json:"metadata" gorm:"type:json"`
}

func (UserNotification) TableName() string {
	return "user_notifications"
}
