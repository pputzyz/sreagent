package model

// PresetRule is a pre-defined alert rule template that can be applied to create
// actual AlertRules. Sourced from community best practices, vendor recommendations,
// or imported from Prometheus rule YAML files.
type PresetRule struct {
	BaseModel
	Name        string     `json:"name" gorm:"size:200;not null;index"`
	DisplayName string     `json:"display_name" gorm:"size:200"`
	Category    string     `json:"category" gorm:"size:50;index"`
	SubCategory string     `json:"sub_category" gorm:"size:50"`
	Component   string     `json:"component" gorm:"size:50"`
	Cluster     string     `json:"cluster" gorm:"size:100;index"`
	Expression  string     `json:"expression" gorm:"type:text;not null"`
	ForDuration string     `json:"for_duration" gorm:"size:32"`
	Severity    string     `json:"severity" gorm:"size:20;index"`
	AlertType   string     `json:"alert_type" gorm:"size:50"`
	Labels      JSONLabels `json:"labels" gorm:"type:json"`
	Annotations JSONLabels `json:"annotations" gorm:"type:json"`
	Source      string     `json:"source" gorm:"size:100"`
	IsBuiltin   bool       `json:"is_builtin" gorm:"default:true"`
	UsageCount  int        `json:"usage_count" gorm:"default:0"`
	Description string     `json:"description" gorm:"type:text"`
}

func (PresetRule) TableName() string { return "preset_rules" }
