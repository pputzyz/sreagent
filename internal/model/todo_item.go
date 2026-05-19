package model

import "time"

// TodoStatus defines the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending   TodoStatus = "pending"
	TodoStatusCompleted TodoStatus = "completed"
	TodoStatusDismissed TodoStatus = "dismissed"
)

// TodoPriority defines the priority of a todo item.
type TodoPriority string

const (
	TodoPriorityHigh   TodoPriority = "high"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityLow    TodoPriority = "low"
)

// TodoItem represents a user's todo/task item.
type TodoItem struct {
	BaseModel
	UserID      uint         `json:"user_id" gorm:"index;not null"`
	Title       string       `json:"title" gorm:"size:256;not null"`
	Description string       `json:"description" gorm:"size:1024"`
	Type        string       `json:"type" gorm:"size:32;not null;default:manual"` // manual, alert, incident
	Status      TodoStatus   `json:"status" gorm:"size:32;not null;index;default:pending"`
	Priority    TodoPriority `json:"priority" gorm:"size:32;not null;default:medium"`
	Link        string       `json:"link" gorm:"size:512"`
	DueAt       *time.Time   `json:"due_at"`
	CompletedAt *time.Time   `json:"completed_at"`
}

func (TodoItem) TableName() string {
	return "todo_items"
}
