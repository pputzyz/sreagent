package model

// Pet represents a user's virtual pet.
type Pet struct {
	BaseModel
	UserID  uint   `json:"user_id" gorm:"not null"`
	Name    string `json:"name" gorm:"size:50;not null;default:'小狐'"`
	Species string `json:"species" gorm:"size:20;not null;default:'fox'"`
	Level   int    `json:"level" gorm:"not null;default:1"`
	Exp     int    `json:"exp" gorm:"not null;default:0"`
	Hunger  int    `json:"hunger" gorm:"not null;default:30"` // 0=full, 100=starving
	Mood    int    `json:"mood" gorm:"not null;default:70"`   // 0=sad, 100=happy
}

// PetInteraction records a single interaction with a pet.
type PetInteraction struct {
	BaseModel
	PetID uint   `json:"pet_id" gorm:"not null"`
	Type  string `json:"type" gorm:"size:20;not null"` // feed, play, chat, level_up
	Value int    `json:"value" gorm:"not null;default:0"`
}
