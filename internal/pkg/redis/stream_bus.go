package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// StreamMessage represents a message read from a Redis stream.
type StreamMessage struct {
	ID     string
	Fields map[string]interface{}
}

// SSE event types published to the stream.
const (
	SSEEventInit    = "init"    // stream existence marker (not forwarded to client)
	SSEEventTask    = "task"    // full AgentTask snapshot
	SSEEventToken   = "token"   // LLM streaming token
	SSEEventStep    = "step"    // agent step progress
	SSEEventTool    = "tool"    // tool call result
	SSEEventDone    = "done"    // task completed
	SSEEventError   = "error"   // task failed
	SSEEventFinish  = "finish"  // stream termination marker (not forwarded to client)
)

// streamBus constants (aligned with Nightingale).
const (
	// streamMaxLen limits stream length to prevent memory bloat.
	// XADD MAXLEN ~ is approximate — Redis trims at list node boundaries.
	streamBusMaxLen = 5000

	// streamBusTTL controls how long a stream lives after the last write.
	streamBusTTL = 24 * time.Hour
)

// streamBusXReadBlockTimeout is the idle timeout for XREAD BLOCK.
// Redis wakes blocked readers immediately on XADD, so this is just a safety net.
// Made var so tests can override (miniredis doesn't interrupt XREAD on conn close).
var streamBusXReadBlockTimeout = 30 * time.Second

// streamBusMaxConsecutiveErrors is the number of consecutive XREAD errors
// before the Subscribe goroutine gives up. Prevents infinite retry loops
// when Redis is permanently down.
const streamBusMaxConsecutiveErrors = 10

// StreamBus provides pub/sub via Redis Streams for multi-instance SSE.
// Any instance can Publish (write) and Subscribe (read); the stream is
// the single source of truth shared across all instances.
type StreamBus struct {
	client *Client
	logger *zap.Logger
}

// NewStreamBus creates a new StreamBus backed by the given Redis client.
func NewStreamBus(client *Client, logger *zap.Logger) *StreamBus {
	return &StreamBus{
		client: client,
		logger: logger.Named("stream_bus"),
	}
}

// streamKey returns the Redis stream key for a given task.
// Format: sreagent:sse:agent:{taskID}
func streamKey(taskID string) string {
	return fmt.Sprintf("sreagent:sse:agent:%s", taskID)
}

// Init creates the stream key with an init marker so that Subscribe can
// distinguish "legitimate stream, owner hasn't written yet" from "nonexistent".
func (b *StreamBus) Init(ctx context.Context, taskID string) error {
	key := streamKey(taskID)
	pipe := b.client.rdb.Pipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{
		Stream: key,
		MaxLen: streamBusMaxLen,
		Approx: true,
		Values: map[string]interface{}{"event": SSEEventInit, "data": ""},
	})
	pipe.Expire(ctx, key, streamBusTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("StreamBus.Init: %w", err)
	}
	return nil
}

// Publish adds an event to the stream. Each event has an "event" type field
// and a "data" field containing JSON-serialized payload.
func (b *StreamBus) Publish(ctx context.Context, taskID string, event string, data interface{}) error {
	key := streamKey(taskID)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("StreamBus.Publish marshal: %w", err)
	}

	pipe := b.client.rdb.Pipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{
		Stream: key,
		MaxLen: streamBusMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"event": event,
			"data":  string(jsonData),
		},
	})
	pipe.Expire(ctx, key, streamBusTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("StreamBus.Publish: %w", err)
	}
	return nil
}

// Finish writes a finish marker to the stream. All blocked XREAD consumers
// are woken up, read the marker, and exit.
func (b *StreamBus) Finish(ctx context.Context, taskID string) error {
	key := streamKey(taskID)
	pipe := b.client.rdb.Pipeline()
	pipe.XAdd(ctx, &goredis.XAddArgs{
		Stream: key,
		Values: map[string]interface{}{"event": SSEEventFinish, "data": ""},
	})
	pipe.Expire(ctx, key, streamBusTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("StreamBus.Finish: %w", err)
	}
	return nil
}

// Subscribe returns a channel that yields StreamMessage values (as interface{}
// to satisfy the service.AgentStreamBus interface without import cycles).
// It reads from the given lastID (use "0" to start from the beginning).
// The channel is closed when:
//   - A finish marker is read
//   - The context is cancelled (e.g. SSE client disconnects)
//   - A persistent Redis error occurs
//
// Callers MUST cancel the context when done to avoid goroutine leaks.
// Callers type-assert: msg := (<-ch).(StreamMessage)
func (b *StreamBus) Subscribe(ctx context.Context, taskID string, lastID string) <-chan interface{} {
	out := make(chan interface{}, 256)
	key := streamKey(taskID)

	go func() {
		defer close(out)

		cursor := lastID
		if cursor == "" {
			cursor = "0"
		}

		consecutiveErrors := 0

		for {
			if ctx.Err() != nil {
				return
			}

			res, err := b.client.rdb.XRead(ctx, &goredis.XReadArgs{
				Streams: []string{key, cursor},
				Block:   streamBusXReadBlockTimeout,
				Count:   100,
			}).Result()

			if errors.Is(err, goredis.Nil) {
				// BLOCK timeout with no new data — loop again.
				consecutiveErrors = 0
				continue
			}
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				consecutiveErrors++
				if consecutiveErrors >= streamBusMaxConsecutiveErrors {
					b.logger.Error("XREAD failed too many times, giving up",
						zap.String("key", key),
						zap.Int("consecutive_errors", consecutiveErrors),
						zap.Error(err),
					)
					return
				}
				b.logger.Warn("XREAD error, retrying after 1s",
					zap.String("key", key),
					zap.Int("consecutive_errors", consecutiveErrors),
					zap.Error(err),
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
				continue
			}

			consecutiveErrors = 0

			for _, stream := range res {
				for _, entry := range stream.Messages {
					cursor = entry.ID
					event, _ := entry.Values["event"].(string)

					if event == SSEEventFinish {
						return
					}
					if event == SSEEventInit {
						// Init marker — don't forward to consumers.
						continue
					}

					msg := StreamMessage{
						ID:     entry.ID,
						Fields: entry.Values,
					}

					select {
					case out <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out
}

// DeleteStream removes the stream key from Redis (cleanup after task completes).
func (b *StreamBus) DeleteStream(ctx context.Context, taskID string) error {
	key := streamKey(taskID)
	if err := b.client.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("StreamBus.DeleteStream: %w", err)
	}
	return nil
}
