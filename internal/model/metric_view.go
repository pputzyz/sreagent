package model

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// MetricView stores a saved metric exploration view with label filters,
// dynamic labels, and dimension labels.
type MetricView struct {
	gorm.Model

	Name      string `json:"name" gorm:"column:name;size:200;not null"`
	Configs   string `json:"configs" gorm:"column:configs;type:text;not null"`
	CreatedBy uint   `json:"created_by" gorm:"column:created_by;not null;default:0"`
	UpdatedBy uint   `json:"updated_by" gorm:"column:updated_by;not null;default:0"`

	// Frontend-facing (not stored directly in DB)
	ConfigsJSON *MetricViewConfig `json:"configs_json" gorm:"-"`
}

func (MetricView) TableName() string { return "metric_views" }

// MetricViewConfig holds the full configuration for a metric view.
type MetricViewConfig struct {
	Filters         []MetricViewFilter       `json:"filters"`
	DynamicLabels   []MetricViewDynamicLabel `json:"dynamicLabels"`
	DimensionLabels [][]string               `json:"dimensionLabels"`
	IgnorePrefix    string                   `json:"ignorePrefix"`
}

// MetricViewFilter is a single label filter condition.
type MetricViewFilter struct {
	Label string `json:"label"`
	Oper  string `json:"oper"` // =, !=, =~, !~
	Value string `json:"value"`
}

// MetricViewDynamicLabel is a dynamic label definition.
type MetricViewDynamicLabel struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FE2DB converts frontend-facing fields to DB-storable format.
func (m *MetricView) FE2DB() {
	if m.ConfigsJSON != nil {
		b, _ := json.Marshal(m.ConfigsJSON)
		m.Configs = string(b)
	}
}

// DB2FE converts DB fields back to frontend-facing format.
func (m *MetricView) DB2FE() {
	if m.Configs != "" {
		var cfg MetricViewConfig
		if err := json.Unmarshal([]byte(m.Configs), &cfg); err == nil {
			m.ConfigsJSON = &cfg
		}
	}
}

// Verify validates the metric view fields.
func (m *MetricView) Verify() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Configs == "" && m.ConfigsJSON == nil {
		return fmt.Errorf("configs is required")
	}
	// Validate filter operators
	if m.ConfigsJSON != nil {
		for _, f := range m.ConfigsJSON.Filters {
			switch f.Oper {
			case "=", "!=", "=~", "!~":
				// valid
			default:
				return fmt.Errorf("invalid filter operator: %s (must be =, !=, =~, or !~)", f.Oper)
			}
		}
	}
	return nil
}
