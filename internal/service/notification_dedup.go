package service

import (
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"
)

// notifDedup is a lightweight in-memory dedup cache that prevents the same
// notification (identified by a string key) from being sent more than once
// within a short TTL window.  This prevents duplicate notifications when the
// same alert event matches both a NotifyRule and a SubscribeRule.
type notifDedup struct {
	mu     sync.Mutex
	sent   map[string]time.Time
	ttl    time.Duration
	logger *zap.Logger
	stopCh chan struct{}
}

func newNotifDedup(logger *zap.Logger) *notifDedup {
	d := &notifDedup{
		sent:   make(map[string]time.Time),
		ttl:    4 * time.Hour, // match Redis dedup TTL to prevent inconsistency on Redis outage
		logger: logger,
		stopCh: make(chan struct{}),
	}
	go d.cleanup()
	return d
}

// Stop signals the background cleanup goroutine to exit.
func (d *notifDedup) Stop() {
	select {
	case <-d.stopCh:
		// Already stopped
	default:
		close(d.stopCh)
	}
}

// cleanup periodically removes expired entries to bound memory usage.
func (d *notifDedup) cleanup() {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("notifDedup cleanup panic recovered", zap.Any("recover", r), zap.String("stack", string(debug.Stack())))
		}
	}()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.mu.Lock()
			now := time.Now()
			for k, t := range d.sent {
				if now.Sub(t) > d.ttl {
					delete(d.sent, k)
				}
			}
			d.mu.Unlock()
		case <-d.stopCh:
			d.logger.Info("notifDedup cleanup goroutine stopped")
			return
		}
	}
}

// routeDedup prevents duplicate notifications from being dispatched within
// a short time window (e.g. when both notify rules and subscriptions match).
var routeDedup = newNotifDedup(zap.L())

// StopRouteDedup stops the background cleanup goroutine of the package-level
// routeDedup instance.  Called during application shutdown.
func StopRouteDedup() {
	routeDedup.Stop()
}

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
