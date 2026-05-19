package engine

import (
	"log"
	"sync"
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
	mu               sync.RWMutex
}

// NewLevelSuppressor creates a new LevelSuppressor.
func NewLevelSuppressor() *LevelSuppressor {
	return &LevelSuppressor{
		activeSeverities: make(map[uint]map[string]string),
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
		return
	}

	existingOrder := severityRank(existing)
	newOrder := severityRank(severity)

	if newOrder > existingOrder {
		fpMap[fingerprint] = severity
	}
}

// RemoveRule removes all severity records for a rule (when rule is deleted/disabled).
func (s *LevelSuppressor) RemoveRule(ruleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeSeverities, ruleID)
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
	}

	// Clean up empty maps
	if len(fpMap) == 0 {
		delete(s.activeSeverities, ruleID)
	}
}
