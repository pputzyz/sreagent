package model

import (
	"fmt"

	"gorm.io/gorm"
)

// ESIndexPattern stores an Elasticsearch index pattern configuration
// associated with a datasource.
type ESIndexPattern struct {
	gorm.Model

	DatasourceID           uint   `json:"datasource_id" gorm:"not null;index:idx_ds_name,unique"`
	Name                   string `json:"name" gorm:"size:191;not null;index:idx_ds_name,unique"`
	TimeField              string `json:"time_field" gorm:"size:128;not null;default:'@timestamp'"`
	AllowHideSystemIndices bool   `json:"allow_hide_system_indices" gorm:"default:false"`
	FieldsFormat           string `json:"fields_format" gorm:"type:text"` // JSON
	CrossClusterEnabled    bool   `json:"cross_cluster_enabled" gorm:"default:false"`
	Note                   string `json:"note" gorm:"size:512"`
	CreatedBy              string `json:"created_by" gorm:"size:64"`
	UpdatedBy              string `json:"updated_by" gorm:"size:64"`
}

func (ESIndexPattern) TableName() string { return "es_index_patterns" }

// Verify validates required fields.
func (p *ESIndexPattern) Verify() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.DatasourceID == 0 {
		return fmt.Errorf("datasource_id is required")
	}
	return nil
}
