package model

import (
	"encoding/json"
	"time"
)

// EventPipeline defines a reusable event processing pipeline.
// Each pipeline contains an ordered list of processor configs that are
// executed sequentially when an alert event matches the pipeline's filters.
type EventPipeline struct {
	BaseModel
	Name         string `json:"name" gorm:"size:256;not null"`
	Description  string `json:"description" gorm:"size:1024"`
	Disabled     bool   `json:"disabled" gorm:"not null;default:false"`
	FilterEnable bool   `json:"filter_enable" gorm:"not null;default:false"`
	// LabelFilters stored as JSON in DB
	LabelFiltersJSON string `json:"-" gorm:"column:label_filters;type:json"`
	// ProcessorConfigs stored as JSON in DB (replaces nodes from v1)
	ProcessorsJSON string `json:"-" gorm:"column:processor_configs;type:json;not null"`
	// Nodes kept for backward compat with v1 DAG structure (unused in linear mode)
	NodesJSON string `json:"-" gorm:"column:nodes;type:json"`
	CreatedBy uint   `json:"created_by" gorm:"default:0"`
	UpdatedBy uint   `json:"updated_by" gorm:"default:0"`

	// Frontend-facing fields (not mapped to DB)
	LabelFilters     []TagFilter       `json:"label_filters" gorm:"-"`
	ProcessorConfigs []ProcessorConfig `json:"processors" gorm:"-"`
}

// TableName returns the table name for EventPipeline.
func (EventPipeline) TableName() string {
	return "event_pipelines"
}

// TagFilter represents a label filter condition.
type TagFilter struct {
	Key   string      `json:"key"`
	Func  string      `json:"func"` // ==, =~, in, !=, !~, not in
	Value interface{} `json:"value"`
}

// ProcessorConfig represents a single processor step in the pipeline.
type ProcessorConfig struct {
	Typ    string                 `json:"typ"`
	Config map[string]interface{} `json:"config"`
}

// FE2DB serializes frontend fields to JSON for database storage.
func (p *EventPipeline) FE2DB() {
	if p.LabelFilters != nil {
		data, _ := json.Marshal(p.LabelFilters)
		p.LabelFiltersJSON = string(data)
	} else {
		p.LabelFiltersJSON = "[]"
	}
	if p.ProcessorConfigs != nil {
		data, _ := json.Marshal(p.ProcessorConfigs)
		p.ProcessorsJSON = string(data)
	} else {
		p.ProcessorsJSON = "[]"
	}
}

// DB2FE deserializes database JSON fields to frontend-facing structs.
func (p *EventPipeline) DB2FE() {
	if p.LabelFiltersJSON != "" {
		_ = json.Unmarshal([]byte(p.LabelFiltersJSON), &p.LabelFilters)
	}
	if p.ProcessorsJSON != "" {
		_ = json.Unmarshal([]byte(p.ProcessorsJSON), &p.ProcessorConfigs)
	}
	if p.LabelFilters == nil {
		p.LabelFilters = []TagFilter{}
	}
	if p.ProcessorConfigs == nil {
		p.ProcessorConfigs = []ProcessorConfig{}
	}
}

// Verify validates the pipeline configuration.
func (p *EventPipeline) Verify() error {
	if p.Name == "" {
		return apperrInvalidParam("name is required")
	}
	return nil
}

// EventPipelineExecution records a single execution of a pipeline.
type EventPipelineExecution struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	PipelineID   uint       `json:"pipeline_id" gorm:"index;not null"`
	PipelineName string     `json:"pipeline_name" gorm:"size:128"`
	EventID      uint       `json:"event_id" gorm:"index;default:0"`
	Mode         string     `json:"mode" gorm:"size:16;default:event"`
	Status       string     `json:"status" gorm:"size:20;not null;default:success"`
	NodeResults  string     `json:"node_results" gorm:"type:json"`
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	DurationMs   int64      `json:"duration_ms" gorm:"default:0"`
	TriggerBy    string     `json:"trigger_by" gorm:"size:64"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	FinishedAt   time.Time  `json:"finished_at"`
}

// TableName returns the table name for EventPipelineExecution.
func (EventPipelineExecution) TableName() string {
	return "event_pipeline_executions"
}

// NodeResult records the result of a single processor node execution.
type NodeResult struct {
	ProcessorType string `json:"processor_type"`
	Status        string `json:"status"` // success, failed, skipped
	Message       string `json:"message,omitempty"`
	DurationMs    int64  `json:"duration_ms"`
}

// Helper to create invalid param errors (avoids importing handler package in model).
func apperrInvalidParam(msg string) error {
	return &validationError{msg: msg}
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}
