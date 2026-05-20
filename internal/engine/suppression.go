package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
)

// severityOrder maps severity names to numeric priority (higher = more severe).
var severityOrder = map[string]int{
	"info":     1,
	"warning":  2,
	"critical": 3,
}

// severityRank returns the numeric rank for a severity string.
// Unknown severities are treated as "info" level (rank 1) and logged once.
func severityRank(sev string) int {
	if rank, ok := severityOrder[sev]; ok {
		return rank
	}
	log.Printf("[WARN] unknown severity %q, defaulting to info level", sev)
	return 1
}

// LevelSuppressor implements severity-level suppression.
// When multiple conditions in a rule trigger simultaneously,
// only the highest severity fires; lower severities are suppressed.
type LevelSuppressor struct {
	// Map of rule_id -> map of fingerprint -> highest severity firing
	activeSeverities map[uint]map[string]string
	// Map of rule_id -> map of fingerprint -> last update time (for GC)
	lastUpdates map[uint]map[string]time.Time
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *zap.Logger
}

// NewLevelSuppressor creates a new LevelSuppressor.
func NewLevelSuppressor() *LevelSuppressor {
	return &LevelSuppressor{
		activeSeverities: make(map[uint]map[string]string),
		lastUpdates:      make(map[uint]map[string]time.Time),
	}
}

// SetLogger sets the logger for the suppressor (called before Start).
func (s *LevelSuppressor) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// Start launches the background GC goroutine that removes stale entries every hour.
// Entries whose lastUpdate is older than 24 hours are deleted.
func (s *LevelSuppressor) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.gc()
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

// Stop terminates the background GC goroutine.
func (s *LevelSuppressor) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// gc removes entries whose lastUpdate is older than 24 hours.
// Must be called with no locks held; acquires write lock internally.
func (s *LevelSuppressor) gc() {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := time.Now().Add(-24 * time.Hour)
	removed := 0

	for ruleID, fpMap := range s.lastUpdates {
		for fp, lastUp := range fpMap {
			if lastUp.Before(threshold) {
				delete(fpMap, fp)
				// Also remove from activeSeverities
				if sevMap, ok := s.activeSeverities[ruleID]; ok {
					delete(sevMap, fp)
					if len(sevMap) == 0 {
						delete(s.activeSeverities, ruleID)
					}
				}
				removed++
			}
		}
		// Clean up empty maps
		if len(fpMap) == 0 {
			delete(s.lastUpdates, ruleID)
		}
	}

	if removed > 0 && s.logger != nil {
		s.logger.Debug("level suppressor GC completed",
			zap.Int("removed_entries", removed),
		)
	}
}

// ShouldSuppress returns true if this alert should be suppressed
// because a higher severity alert is already firing for the same fingerprint.
func (s *LevelSuppressor) ShouldSuppress(ruleID uint, fingerprint string, severity string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fpMap, ok := s.activeSeverities[ruleID]
	if !ok {
		return false
	}

	activeSev, ok := fpMap[fingerprint]
	if !ok {
		return false
	}

	activeOrder := severityRank(activeSev)
	newOrder := severityRank(severity)

	// Suppress if the currently active severity is higher than the new one
	return activeOrder > newOrder
}

// UpdateSeverity records that a specific severity is now active for a rule+fingerprint.
// Only updates if the new severity is higher than the current one.
func (s *LevelSuppressor) UpdateSeverity(ruleID uint, fingerprint string, severity string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fpMap, ok := s.activeSeverities[ruleID]
	if !ok {
		fpMap = make(map[string]string)
		s.activeSeverities[ruleID] = fpMap
	}

	existing, ok := fpMap[fingerprint]
	if !ok {
		fpMap[fingerprint] = severity
		s.touchLastUpdate(ruleID, fingerprint)
		return
	}

	existingOrder := severityRank(existing)
	newOrder := severityRank(severity)

	if newOrder > existingOrder {
		fpMap[fingerprint] = severity
		s.touchLastUpdate(ruleID, fingerprint)
	}
}

// RemoveRule removes all severity records for a rule (when rule is deleted/disabled).
func (s *LevelSuppressor) RemoveRule(ruleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeSeverities, ruleID)
	delete(s.lastUpdates, ruleID)
}

// RemoveSeverity removes a severity record (when alert resolves).
func (s *LevelSuppressor) RemoveSeverity(ruleID uint, fingerprint string, severity string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fpMap, ok := s.activeSeverities[ruleID]
	if !ok {
		return
	}

	activeSev, ok := fpMap[fingerprint]
	if !ok {
		return
	}

	// Only remove if the severity matches what's currently active
	if activeSev == severity {
		delete(fpMap, fingerprint)
		// Clean up lastUpdates
		if luMap, ok := s.lastUpdates[ruleID]; ok {
			delete(luMap, fingerprint)
			if len(luMap) == 0 {
				delete(s.lastUpdates, ruleID)
			}
		}
	}

	// Clean up empty maps
	if len(fpMap) == 0 {
		delete(s.activeSeverities, ruleID)
	}
}

// touchLastUpdate records the current time for a rule+fingerprint.
// Must be called with s.mu write lock held.
func (s *LevelSuppressor) touchLastUpdate(ruleID uint, fingerprint string) {
	luMap, ok := s.lastUpdates[ruleID]
	if !ok {
		luMap = make(map[string]time.Time)
		s.lastUpdates[ruleID] = luMap
	}
	luMap[fingerprint] = time.Now()
}
