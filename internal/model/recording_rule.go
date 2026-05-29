package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Prometheus metric name regex (subset: ASCII letters, digits, underscores, colons).
var metricNameRE = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

// RecordingRule represents a pre-computed PromQL expression that writes results
// as a new time series at a fixed interval.
type RecordingRule struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	GroupID       uint   `json:"group_id" gorm:"column:group_id;not null;default:0"`
	Name          string `json:"name" gorm:"column:name;size:255;not null;default:''"`
	PromQL        string `json:"prom_ql" gorm:"column:prom_ql;type:text;not null"`
	DatasourceIDs string `json:"-" gorm:"column:datasource_ids;size:1024;not null;default:'[]'"`
	CronPattern   string `json:"cron_pattern" gorm:"column:cron_pattern;size:64;not null;default:'@every 60s'"`
	Disabled      int    `json:"disabled" gorm:"column:disabled;not null;default:0"`
	WriteBack     int    `json:"write_back" gorm:"column:write_back;not null;default:1"` // 1=write results back to datasource, 0=disabled
	AppendTags    string `json:"-" gorm:"column:append_tags;type:text"`
	Note          string `json:"note" gorm:"column:note;size:1024;not null;default:''"`
	QueryConfigs  string `json:"-" gorm:"column:query_configs;type:text"`
	CreatedBy     string `json:"created_by" gorm:"column:created_by;size:64;not null;default:''"`
	UpdatedBy     string `json:"updated_by" gorm:"column:updated_by;size:64;not null;default:''"`

	// Frontend-facing fields (not stored directly in DB)
	DatasourceIDsJSON []int64       `json:"datasource_ids" gorm:"-"`
	AppendTagsJSON    []string      `json:"append_tags" gorm:"-"`
	QueryConfigsJSON  []QueryConfig `json:"query_configs" gorm:"-"`
}

func (RecordingRule) TableName() string { return "recording_rules" }

// QueryConfig holds structured query configuration for advanced recording rules.
type QueryConfig struct {
	Queries           []Query `json:"queries"`
	NewMetric         string  `json:"new_metric"`
	Exp               string  `json:"exp"`
	WriteDatasourceID int64   `json:"write_datasource_id"`
	Delay             int     `json:"delay"`
	WritebackEnabled  bool    `json:"writeback_enabled"`
}

// Query holds a single sub-query within a QueryConfig.
type Query struct {
	DatasourceIDs []int64 `json:"datasource_ids"`
	Cate          string  `json:"cate"`
	Config        any     `json:"config"`
}

// FE2DB converts frontend-facing fields to DB-storable format.
func (r *RecordingRule) FE2DB() {
	if len(r.AppendTagsJSON) > 0 {
		r.AppendTags = strings.Join(r.AppendTagsJSON, " ")
	} else {
		r.AppendTags = ""
	}
	if len(r.DatasourceIDsJSON) > 0 {
		b, _ := json.Marshal(r.DatasourceIDsJSON)
		r.DatasourceIDs = string(b)
	} else {
		r.DatasourceIDs = "[]"
	}
	if len(r.QueryConfigsJSON) > 0 {
		b, _ := json.Marshal(r.QueryConfigsJSON)
		r.QueryConfigs = string(b)
	}
}

// DB2FE converts DB fields back to frontend-facing format.
func (r *RecordingRule) DB2FE() {
	// AppendTags
	r.AppendTagsJSON = nil
	if r.AppendTags != "" {
		for _, tag := range strings.Split(r.AppendTags, " ") {
			if tag = strings.TrimSpace(tag); tag != "" {
				r.AppendTagsJSON = append(r.AppendTagsJSON, tag)
			}
		}
	}

	// DatasourceIDs
	r.DatasourceIDsJSON = nil
	if r.DatasourceIDs != "" && r.DatasourceIDs != "[]" {
		_ = json.Unmarshal([]byte(r.DatasourceIDs), &r.DatasourceIDsJSON)
	}

	// QueryConfigs
	r.QueryConfigsJSON = nil
	if r.QueryConfigs != "" {
		_ = json.Unmarshal([]byte(r.QueryConfigs), &r.QueryConfigsJSON)
	}

	// Backward compat: if CronPattern is empty, derive from a default
	if r.CronPattern == "" {
		r.CronPattern = "@every 60s"
	}
}

// Verify validates the recording rule fields.
func (r *RecordingRule) Verify() error {
	if r.PromQL == "" {
		return fmt.Errorf("prom_ql is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !metricNameRE.MatchString(r.Name) {
		return fmt.Errorf("invalid metric name: %s (must match %s)", r.Name, metricNameRE.String())
	}
	if r.CronPattern == "" {
		r.CronPattern = "@every 60s"
	}
	// Validate append tags
	if r.AppendTags != "" {
		for _, tag := range strings.Split(r.AppendTags, " ") {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			parts := strings.SplitN(tag, "=", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid append tag: %s (must be key=value)", tag)
			}
		}
	}
	// Validate QueryConfigs size
	if len(r.QueryConfigs) > 65535 {
		return fmt.Errorf("query_configs exceeds maximum size (65535 bytes)")
	}
	return nil
}
