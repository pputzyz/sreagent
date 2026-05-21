package model

import "time"

// AIConversation stores an AI Agent conversation session.
type AIConversation struct {
	BaseModel
	UserID uint   `json:"user_id" gorm:"index;not null"`
	Title  string `json:"title" gorm:"size:255;default:''"`
	Status string `json:"status" gorm:"size:20;default:active"`
}

func (AIConversation) TableName() string { return "ai_conversations" }

// AIToolCall stores a single tool call within an AI conversation.
type AIToolCall struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	ConversationID uint   `json:"conversation_id" gorm:"index;not null"`
	StepIndex      int    `json:"step_index" gorm:"default:0"`
	ToolName       string `json:"tool_name" gorm:"size:100;not null"`
	Parameters     string `json:"parameters" gorm:"type:text"`
	Result         string `json:"result" gorm:"type:text"`
	Status         string `json:"status" gorm:"size:20;default:pending"`
	DurationMs     int64  `json:"duration_ms" gorm:"default:0"`
	Error          string `json:"error,omitempty" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AIToolCall) TableName() string { return "ai_tool_calls" }
