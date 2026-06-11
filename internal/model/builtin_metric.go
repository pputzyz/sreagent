package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BuiltinMetric represents a centralized metric catalog entry with metadata.
type BuiltinMetric struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Collector      string `json:"collector" gorm:"column:collector;size:191;not null;default:''"`
	Typ            string `json:"typ" gorm:"column:typ;size:191;not null;default:''"`
	Name           string `json:"name" gorm:"column:name;size:191;not null;default:''"`
	Unit           string `json:"unit" gorm:"column:unit;size:191;not null;default:''"`
	Note           string `json:"note" gorm:"column:note;size:4096;not null;default:''"`
	Lang           string `json:"lang" gorm:"column:lang;size:32;not null;default:'zh'"`
	Expression     string `json:"expression" gorm:"column:expression;size:4096;not null;default:''"`
	ExpressionType string `json:"expression_type" gorm:"column:expression_type;size:32;not null;default:'metric_name'"`
	MetricType     string `json:"metric_type" gorm:"column:metric_type;size:64;not null;default:''"`
	ExtraFields    string `json:"extra_fields" gorm:"column:extra_fields;type:text"`
	Translation    string `json:"translation" gorm:"column:translation;type:text"`
	CreatedBy      string `json:"created_by" gorm:"column:created_by;size:64;not null;default:''"`
	UpdatedBy      string `json:"updated_by" gorm:"column:updated_by;size:64;not null;default:''"`

	// Frontend-facing fields
	ExtraFieldsJSON map[string]string  `json:"extra_fields_json" gorm:"-"`
	TranslationJSON []TranslationEntry `json:"translation_json" gorm:"-"`
}

func (BuiltinMetric) TableName() string { return "builtin_metrics" }

// TranslationEntry holds a name+note translation for a specific language.
type TranslationEntry struct {
	Lang string `json:"lang"`
	Name string `json:"name"`
	Note string `json:"note"`
}

// MetricFilter represents a saved filter configuration for metrics.
type MetricFilter struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name       string `json:"name" gorm:"column:name;size:191;not null;default:''"`
	Configs    string `json:"configs" gorm:"column:configs;size:4096;not null;default:'[]'"`
	GroupsPerm string `json:"groups_perm" gorm:"column:groups_perm;type:text"`
	CreatedBy  string `json:"created_by" gorm:"column:created_by;size:64;not null;default:''"`
	UpdatedBy  string `json:"updated_by" gorm:"column:updated_by;size:64;not null;default:''"`

	// Frontend-facing
	ConfigsJSON    []FilterConfig `json:"configs_json" gorm:"-"`
	GroupsPermJSON []GroupPerm    `json:"groups_perm_json" gorm:"-"`
}

func (MetricFilter) TableName() string { return "metric_filters" }

// FilterConfig holds a single label filter condition.
type FilterConfig struct {
	Label    string `json:"label"`
	Operator string `json:"operator"` // =, !=, =~, !~
	Value    string `json:"value"`
}

// GroupPerm holds team-based permission for a filter.
type GroupPerm struct {
	GID   int64 `json:"gid"`
	Write bool  `json:"write"`
}

// FE2DB converts frontend fields to DB format.
func (m *BuiltinMetric) FE2DB() {
	if m.ExtraFieldsJSON != nil {
		b, _ := json.Marshal(m.ExtraFieldsJSON)
		m.ExtraFields = string(b)
	}
	if m.TranslationJSON != nil {
		b, _ := json.Marshal(m.TranslationJSON)
		m.Translation = string(b)
	}
}

// DB2FE converts DB fields to frontend format.
func (m *BuiltinMetric) DB2FE() {
	if m.ExtraFields != "" {
		_ = json.Unmarshal([]byte(m.ExtraFields), &m.ExtraFieldsJSON)
	}
	if m.Translation != "" {
		_ = json.Unmarshal([]byte(m.Translation), &m.TranslationJSON)
	}
}

// FE2DB converts frontend fields to DB format.
func (f *MetricFilter) FE2DB() {
	if f.ConfigsJSON != nil {
		b, _ := json.Marshal(f.ConfigsJSON)
		f.Configs = string(b)
	}
	if f.GroupsPermJSON != nil {
		b, _ := json.Marshal(f.GroupsPermJSON)
		f.GroupsPerm = string(b)
	}
}

// DB2FE converts DB fields to frontend format.
func (f *MetricFilter) DB2FE() {
	if f.Configs != "" {
		_ = json.Unmarshal([]byte(f.Configs), &f.ConfigsJSON)
	}
	if f.GroupsPerm != "" {
		_ = json.Unmarshal([]byte(f.GroupsPerm), &f.GroupsPermJSON)
	}
}

// Verify validates the builtin metric.
func (m *BuiltinMetric) Verify() error {
	if m.Expression == "" {
		return fmt.Errorf("expression is required")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Lang == "" {
		m.Lang = "zh"
	}
	if m.ExpressionType == "" {
		m.ExpressionType = "metric_name"
	}
	if m.ExpressionType != "metric_name" && m.ExpressionType != "promql" {
		return fmt.Errorf("expression_type must be metric_name or promql")
	}
	return nil
}
