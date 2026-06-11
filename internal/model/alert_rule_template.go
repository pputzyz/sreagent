package model

// AlertRuleTemplate is a reusable alert rule configuration.
type AlertRuleTemplate struct {
	BaseModel
	Category       string         `json:"category" gorm:"size:64;index;default:''"`
	Name           string         `json:"name" gorm:"size:256;not null;index"`
	Description    string         `json:"description" gorm:"type:text"`
	DatasourceType DataSourceType `json:"datasource_type" gorm:"size:32;index"`
	Expression     string         `json:"expression" gorm:"type:text;not null"`
	ForDuration    string         `json:"for_duration" gorm:"size:32;default:0s"`
	Severity       AlertSeverity  `json:"severity" gorm:"size:32;not null;index"`
	Labels         JSONLabels     `json:"labels" gorm:"type:json"`
	Annotations    JSONLabels     `json:"annotations" gorm:"type:json"`
	GroupName      string         `json:"group_name" gorm:"size:128"`
	EvalInterval   int            `json:"eval_interval" gorm:"default:60"`
	IsBuiltin      bool           `json:"is_builtin" gorm:"default:false"`
	UsageCount     int            `json:"usage_count" gorm:"default:0"`
	CreatedBy      uint           `json:"created_by" gorm:"index"`
	UpdatedBy      uint           `json:"updated_by"`
	// Alert rule fields that can be pre-configured
	NoDataEnabled  bool   `json:"nodata_enabled" gorm:"default:false"`
	NoDataDuration string `json:"nodata_duration" gorm:"size:32;default:5m"`
	AckSlaMinutes  int    `json:"ack_sla_minutes" gorm:"not null;default:0"`
}

func (AlertRuleTemplate) TableName() string {
	return "alert_rule_templates"
}
