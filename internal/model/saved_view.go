package model

// SavedView stores a reusable explore query configuration for metrics or logs tabs.
type SavedView struct {
	BaseModel
	Name         string `json:"name" gorm:"size:200;not null"`
	Description  string `json:"description" gorm:"size:500;not null;default:''"`
	Tab          string `json:"tab" gorm:"size:20;not null"`                        // metrics, logs
	DatasourceID uint   `json:"datasource_id" gorm:"column:datasource_id;not null;default:0"`
	Expression   string `json:"expression" gorm:"type:text;not null"`
	QueryConfig  string `json:"query_config" gorm:"type:text"` // JSON blob for complex configs
	IsPublic     bool   `json:"is_public" gorm:"default:false"`
	CreatedBy    uint   `json:"created_by" gorm:"not null;default:0"`
	UpdatedBy    uint   `json:"updated_by" gorm:"not null;default:0"`
}

func (SavedView) TableName() string {
	return "saved_views"
}
