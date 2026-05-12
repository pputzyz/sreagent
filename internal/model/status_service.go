package model

// StatusService represents a service shown on the public status page.
type StatusService struct {
	BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Status      string `json:"status" gorm:"size:32;not null;default:operational"`
	Description string `json:"description" gorm:"size:512;not null;default:''"`
	URL         string `json:"url" gorm:"size:512;not null;default:''"`
	Icon        string `json:"icon" gorm:"size:64;not null;default:''"`
	SortOrder   int    `json:"sort_order" gorm:"not null;default:0"`
}

func (StatusService) TableName() string {
	return "status_services"
}
