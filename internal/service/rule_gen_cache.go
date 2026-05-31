package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ruleGenCacheEntry holds a cached AI generation result.
type ruleGenCacheEntry struct {
	result    *RuleGenerateResult
	expiresAt time.Time
}

// RuleGenCache is a simple in-memory TTL cache for AI rule generation results.
// It avoids repeated LLM calls for the same description within a short window.
type RuleGenCache struct {
	mu      sync.RWMutex
	entries map[string]*ruleGenCacheEntry
	ttl     time.Duration
	stop    chan struct{}
}

// NewRuleGenCache creates a cache with the given TTL.
func NewRuleGenCache(ttl time.Duration) *RuleGenCache {
	c := &RuleGenCache{
		entries: make(map[string]*ruleGenCacheEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	// Background cleanup every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.evict()
			case <-c.stop:
				return
			}
		}
	}()
	return c
}

// Stop terminates the background cleanup goroutine.
func (c *RuleGenCache) Stop() {
	close(c.stop)
}

// cacheKey builds a deterministic key from the generation request.
func cacheKey(description string, dsID *uint, ruleType string) string {
	h := sha256.New()
	h.Write([]byte(description))
	if dsID != nil {
		_, _ = fmt.Fprintf(h, ":%d", *dsID)
	}
	h.Write([]byte(":" + ruleType))
	return hex.EncodeToString(h.Sum(nil))
}

// Get returns a cached result if it exists and hasn't expired.
func (c *RuleGenCache) Get(description string, dsID *uint, ruleType string) *RuleGenerateResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := cacheKey(description, dsID, ruleType)
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.result
}

// Set stores a result in the cache.
func (c *RuleGenCache) Set(description string, dsID *uint, ruleType string, result *RuleGenerateResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := cacheKey(description, dsID, ruleType)
	c.entries[key] = &ruleGenCacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// evict removes expired entries.
func (c *RuleGenCache) evict() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
		}
	}
}
