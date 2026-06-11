package model

import "time"

// LarkCardEntity tracks a CardKit card entity that can be sent to multiple chats
// and updated centrally. One alert event maps to one card entity; the entity is
// sent to N chats via lark_card_messages.
type LarkCardEntity struct {
	BaseModel
	EventID    *uint     `json:"event_id" gorm:"index"`
	CardID     string    `json:"card_id" gorm:"size:128;not null"`
	Sequence   int64     `json:"sequence" gorm:"not null;default:0"`
	CardStatus string    `json:"card_status" gorm:"size:32;not null;default:active"` // active | expired | superseded
	ExpiresAt  time.Time `json:"expires_at" gorm:"not null"`
}

// TableName overrides the table name to lark_card_entities.
func (LarkCardEntity) TableName() string { return "lark_card_entities" }

// LarkCardMessage records a single chat delivery of a card entity.
// One entity can be sent to multiple group chats; each delivery gets its own message_id.
type LarkCardMessage struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CardEntityID uint      `json:"card_entity_id" gorm:"not null;index"`
	ChatID       string    `json:"chat_id" gorm:"size:128;not null;index"`
	MessageID    string    `json:"message_id" gorm:"size:128;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName overrides the table name to lark_card_messages.
func (LarkCardMessage) TableName() string { return "lark_card_messages" }
