package service

import (
	"sync"
	"time"
)

// notifDedup is a lightweight in-memory dedup cache that prevents the same
// notification (identified by a string key) from being sent more than once
// within a short TTL window.  This prevents duplicate notifications when the
// same alert event matches both a NotifyRule and a SubscribeRule.
type notifDedup struct {
	mu   sync.Mutex
	sent map[string]time.Time
	ttl  time.Duration
}

func newNotifDedup() *notifDedup {
	d := &notifDedup{
		sent: make(map[string]time.Time),
		ttl:  5 * time.Minute,
	}
	go d.cleanup()
	return d
}

// cleanup periodically removes expired entries to bound memory usage.
func (d *notifDedup) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for k, t := range d.sent {
			if now.Sub(t) > d.ttl {
				delete(d.sent, k)
			}
		}
		d.mu.Unlock()
	}
}

// routeDedup prevents duplicate notifications from being dispatched within
// a short time window (e.g. when both notify rules and subscriptions match).
var routeDedup = newNotifDedup()

// TrySend returns true if this notification key hasn't been sent recently,
// and records it.  Returns false if the key was already seen within the TTL.
func (d *notifDedup) TrySend(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.sent[key]; exists {
		return false
	}
	d.sent[key] = time.Now()
	return true
}
