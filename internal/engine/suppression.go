package engine

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/muterule"
)

// MuteRuleChecker abstracts the mute rule service so the engine can check
// time-window muting without importing the service layer directly.
type MuteRuleChecker interface {
	// FindEnabled returns all currently enabled mute rules.
	FindEnabled(ctx context.Context) ([]model.MuteRule, error)
}

// Mute-rule matching logic lives in internal/pkg/muterule (shared with service).

// severityOrder maps severity names to numeric priority (higher = more severe).
// Includes legacy p0-p4 values for backward compatibility with historical data.
var severityOrder = map[string]int{
	"info":     1,
	"warning":  2,
	"critical": 3,
	// Legacy severity levels
	"p0": 4, // equivalent to critical or higher
	"p1": 3,
	"p2": 2, // equivalent to warning
	"p3": 1,
	"p4": 1, // equivalent to info
}

// severityRank returns the numeric rank for a severity string.
// Unknown severities are treated as "info" level (rank 1).
func severityRank(sev string) int {
	if rank, ok := severityOrder[sev]; ok {
		return rank
	}
	return 1
}

// LevelSuppressor implements severity-level suppression.
// When multiple conditions in a rule trigger simultaneously,
// only the highest severity fires; lower severities are suppressed.
// It also provides engine-level time-window mute checking (Nightingale TimeSpanMuteStrategy equivalent).
type LevelSuppressor struct {
	// Map of rule_id -> map of fingerprint -> highest severity firing
	activeSeverities map[uint]map[string]string
	// Map of rule_id -> map of fingerprint -> last update time (for GC)
	lastUpdates map[uint]map[string]time.Time
	// Map of rule_id -> map of fingerprint -> last severity change time (for GC of long-firing alerts)
	lastChanges map[uint]map[string]time.Time
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *zap.Logger
	startOnce   sync.Once
	muteChecker MuteRuleChecker // optional; nil = skip engine-level mute checks
}

// NewLevelSuppressor creates a new LevelSuppressor.
func NewLevelSuppressor() *LevelSuppressor {
	return &LevelSuppressor{
		activeSeverities: make(map[uint]map[string]string),
		lastUpdates:      make(map[uint]map[string]time.Time),
		lastChanges:      make(map[uint]map[string]time.Time),
	}
}

// SetLogger sets the logger for the suppressor (called before Start).
func (s *LevelSuppressor) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// SetMuteChecker sets the mute rule checker for engine-level time-window muting.
func (s *LevelSuppressor) SetMuteChecker(checker MuteRuleChecker) {
	s.muteChecker = checker
}

// IsMutedByAnyRule checks whether an alert should be suppressed because it matches
// an enabled mute rule (label + severity + time window). This is the engine-level
// equivalent of Nightingale's TimeSpanMuteStrategy + EventMuteStrategy.
// Returns the mute rule ID if muted, or 0 if not muted.
func (s *LevelSuppressor) IsMutedByAnyRule(ctx context.Context, eventLabels map[string]string, eventSeverity string, ruleID *uint) (bool, uint) {
	if s.muteChecker == nil {
		return false, 0
	}

	rules, err := s.muteChecker.FindEnabled(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to load mute rules for engine check", zap.Error(err))
		}
		return false, 0
	}

	now := time.Now()
	for _, rule := range rules {
		if muterule.IsMutedByRule(&rule, eventLabels, eventSeverity, ruleID, now) {
			if s.logger != nil {
				s.logger.Info("alert muted at engine level",
					zap.Uint("mute_rule_id", rule.ID),
					zap.String("mute_rule_name", rule.Name),
				)
			}
			return true, rule.ID
		}
	}
	return false, 0
}

// Start launches the background GC goroutine that removes stale entries every hour.
// Entries whose lastUpdate is older than 24 hours are deleted. Safe to call multiple times.
func (s *LevelSuppressor) Start() {
	s.startOnce.Do(func() {
		s.ctx, s.cancel = context.WithCancel(context.Background())

		go func() {
			s.gc() // initial cleanup on startup
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
	})
}

// Stop terminates the background GC goroutine.
func (s *LevelSuppressor) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// gc removes stale entries. Two conditions trigger removal:
//  1. lastUpdate older than 24h — entry has not been touched recently (resolved alerts).
//  2. lastChange older than 7 days — severity hasn't changed in a week (long-firing alerts
//     that are likely stale or forgotten). This prevents memory leaks for alerts that keep
//     calling UpdateSeverity every eval cycle without ever resolving.
//
// Must be called with no locks held; acquires write lock internally.
func (s *LevelSuppressor) gc() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	updateThreshold := now.Add(-24 * time.Hour)
	changeThreshold := now.Add(-7 * 24 * time.Hour)
	removed := 0

	for ruleID, fpMap := range s.lastUpdates {
		for fp, lastUp := range fpMap {
			shouldGC := false

			// Condition 1: not touched in 24h (resolved or abandoned)
			if lastUp.Before(updateThreshold) {
				shouldGC = true
			}

			// Condition 2: severity unchanged for 7 days (long-firing, likely stale)
			if !shouldGC {
				if lcMap, ok := s.lastChanges[ruleID]; ok {
					if lastChange, exists := lcMap[fp]; exists && lastChange.Before(changeThreshold) {
						shouldGC = true
					}
				}
			}

			if shouldGC {
				delete(fpMap, fp)
				// Also remove from activeSeverities
				if sevMap, ok := s.activeSeverities[ruleID]; ok {
					delete(sevMap, fp)
					if len(sevMap) == 0 {
						delete(s.activeSeverities, ruleID)
					}
				}
				// Also remove from lastChanges
				if lcMap, ok := s.lastChanges[ruleID]; ok {
					delete(lcMap, fp)
					if len(lcMap) == 0 {
						delete(s.lastChanges, ruleID)
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
		s.touchLastChange(ruleID, fingerprint)
		return
	}

	existingOrder := severityRank(existing)
	newOrder := severityRank(severity)

	if newOrder > existingOrder {
		fpMap[fingerprint] = severity
		s.touchLastUpdate(ruleID, fingerprint)
		s.touchLastChange(ruleID, fingerprint)
	}
}

// RemoveRule removes all severity records for a rule (when rule is deleted/disabled).
func (s *LevelSuppressor) RemoveRule(ruleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeSeverities, ruleID)
	delete(s.lastUpdates, ruleID)
	delete(s.lastChanges, ruleID)
}

// RemoveSeverity removes a severity record (when alert resolves).
// Only removes if the current active severity matches the given one.
func (s *LevelSuppressor) RemoveSeverity(ruleID uint, fingerprint string, severity string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fpMap, ok := s.activeSeverities[ruleID]
	if !ok {
		return
	}

	activeSev, ok := fpMap[fingerprint]
	if !ok || activeSev != severity {
		return
	}

	delete(fpMap, fingerprint)
	// Clean up lastUpdates
	if luMap, ok := s.lastUpdates[ruleID]; ok {
		delete(luMap, fingerprint)
		if len(luMap) == 0 {
			delete(s.lastUpdates, ruleID)
		}
	}
	// Clean up lastChanges
	if lcMap, ok := s.lastChanges[ruleID]; ok {
		delete(lcMap, fingerprint)
		if len(lcMap) == 0 {
			delete(s.lastChanges, ruleID)
		}
	}
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

// touchLastChange records the current time as the last severity change for a rule+fingerprint.
// Must be called with s.mu write lock held.
func (s *LevelSuppressor) touchLastChange(ruleID uint, fingerprint string) {
	lcMap, ok := s.lastChanges[ruleID]
	if !ok {
		lcMap = make(map[string]time.Time)
		s.lastChanges[ruleID] = lcMap
	}
	lcMap[fingerprint] = time.Now()
}
