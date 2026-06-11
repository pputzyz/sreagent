package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"
)

const (
	convContextPrefix = "sreagent:lark:conv:"
	convContextTTL    = 30 * time.Minute
	convMaxTurns      = 20 // max conversation turns to retain
)

// ConversationTurn represents a single turn in a Lark bot conversation.
type ConversationTurn struct {
	Role      string    `json:"role"`       // "user" or "assistant"
	Content   string    `json:"content"`    // message text
	Timestamp time.Time `json:"timestamp"`  // when the turn occurred
}

// ConversationContext holds the conversation history for a Lark chat + user pair.
type ConversationContext struct {
	ChatID  string            `json:"chat_id"`
	UserID  string            `json:"user_id"`
	Turns   []ConversationTurn `json:"turns"`
	Updated time.Time         `json:"updated"`
}

// ConversationStore manages conversation context in Redis for Lark bot multi-turn sessions.
// Accepts a raw go-redis client to avoid import cycles with the redis wrapper package.
type ConversationStore struct {
	rdb *goredis.Client
}

// NewConversationStore creates a new ConversationStore backed by Redis.
func NewConversationStore(rdb *goredis.Client) *ConversationStore {
	return &ConversationStore{rdb: rdb}
}

// convKey returns the Redis key for a chat+user conversation context.
func convKey(chatID, userID string) string {
	return convContextPrefix + chatID + ":" + userID
}

// Get retrieves the conversation context for a given chat and user.
// Returns nil if no conversation exists (not an error).
func (s *ConversationStore) Get(ctx context.Context, chatID, userID string) (*ConversationContext, error) {
	key := convKey(chatID, userID)
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get conversation context: %w", err)
	}

	var conv ConversationContext
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("unmarshal conversation context: %w", err)
	}
	return &conv, nil
}

// Append adds a new turn to the conversation context and persists it to Redis.
// Old turns beyond convMaxTurns are dropped (FIFO).
func (s *ConversationStore) Append(ctx context.Context, chatID, userID string, turn ConversationTurn) error {
	conv, err := s.Get(ctx, chatID, userID)
	if err != nil {
		return err
	}

	if conv == nil {
		conv = &ConversationContext{
			ChatID: chatID,
			UserID: userID,
		}
	}

	turn.Timestamp = time.Now()
	conv.Turns = append(conv.Turns, turn)
	conv.Updated = turn.Timestamp

	// Trim old turns if exceeding the limit.
	if len(conv.Turns) > convMaxTurns {
		conv.Turns = conv.Turns[len(conv.Turns)-convMaxTurns:]
	}

	data, err := json.Marshal(conv)
	if err != nil {
		return fmt.Errorf("marshal conversation context: %w", err)
	}

	key := convKey(chatID, userID)
	if err := s.rdb.Set(ctx, key, data, convContextTTL).Err(); err != nil {
		return fmt.Errorf("set conversation context: %w", err)
	}
	return nil
}

// Clear removes the conversation context for a given chat and user.
func (s *ConversationStore) Clear(ctx context.Context, chatID, userID string) error {
	key := convKey(chatID, userID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("clear conversation context: %w", err)
	}
	return nil
}
