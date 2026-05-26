package model

import "time"

// RecordingRuleExecution records the outcome of a single recording rule evaluation.
type RecordingRuleExecution struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleID       uint      `json:"rule_id" gorm:"column:rule_id;not null;index:idx_rule_executed,priority:1"`
	Status       string    `json:"status" gorm:"column:status;size:20;not null"` // success / error
	ErrorMessage string    `json:"error_message" gorm:"column:error_message;type:text"`
	DurationMs   int       `json:"duration_ms" gorm:"column:duration_ms;not null;default:0"`
	ExecutedAt   time.Time `json:"executed_at" gorm:"column:executed_at;not null;default:CURRENT_TIMESTAMP;index:idx_rule_executed,priority:2"`
}

func (RecordingRuleExecution) TableName() string { return "recording_rule_executions" }
