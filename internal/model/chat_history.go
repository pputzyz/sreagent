package model

// ChatHistory stores a single chat message.
type ChatHistory struct {
	BaseModel
	UserID  uint   `json:"user_id" gorm:"index;not null"`
	Mode    string `json:"mode" gorm:"size:20;not null"` // alert, general, pet
	Role    string `json:"role" gorm:"size:10;not null"` // user, assistant
	Content string `json:"content" gorm:"type:text;not null"`
	Context string `json:"context,omitempty" gorm:"type:text"` // optional JSON context
}
