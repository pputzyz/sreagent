package engine

import (
	"context"
	"time"
)

// StateEntry is the serializable form of AlertState for persistence.
type StateEntry struct {
	Fingerprint       string            `json:"fp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	Status            string            `json:"status"` // "pending", "firing", "resolved"
	ActiveAt          time.Time         `json:"active_at"`
	FiredAt           time.Time         `json:"fired_at,omitempty"`
	ResolvedAt        time.Time         `json:"resolved_at,omitempty"`
	Value             float64           `json:"value"`
	RecoveryHoldUntil time.Time         `json:"recovery_hold_until,omitempty"`
	LastSeen          time.Time         `json:"last_seen"`
	EventID           uint              `json:"event_id,omitempty"`
}

// StateStore is the interface for persisting alert engine state.
// Implementations are responsible for serialization and TTL management.
type StateStore interface {
	// SaveState persists a single alert state entry for a rule.
	// The ttl should be set to max(1 hour, 10× the rule evaluation interval).
	SaveState(ctx context.Context, ruleID uint, fp string, entry *StateEntry, ttl time.Duration) error

	// DeleteState removes a single alert state entry.
	DeleteState(ctx context.Context, ruleID uint, fp string) error

	// LoadStates loads all persisted alert states for a given rule.
	LoadStates(ctx context.Context, ruleID uint) (map[string]*StateEntry, error)

	// DeleteRuleStates removes all persisted states for a rule (when rule is stopped).
	DeleteRuleStates(ctx context.Context, ruleID uint) error
}

// toStateEntry converts an AlertState to a StateEntry for persistence.
func toStateEntry(fp string, s *AlertState) *StateEntry {
	return &StateEntry{
		Fingerprint:       fp,
		Labels:            copyStringMap(s.Labels),
		Annotations:       copyStringMap(s.Annotations),
		Status:            s.Status,
		ActiveAt:          s.ActiveAt,
		FiredAt:           s.FiredAt,
		ResolvedAt:        s.ResolvedAt,
		Value:             s.Value,
		RecoveryHoldUntil: s.RecoveryHoldUntil,
		LastSeen:          s.LastSeen,
		EventID:           s.EventID,
	}
}

// fromStateEntry converts a StateEntry back to an AlertState.
func fromStateEntry(e *StateEntry) *AlertState {
	return &AlertState{
		Labels:            copyStringMap(e.Labels),
		Annotations:       copyStringMap(e.Annotations),
		Status:            e.Status,
		ActiveAt:          e.ActiveAt,
		FiredAt:           e.FiredAt,
		ResolvedAt:        e.ResolvedAt,
		Value:             e.Value,
		RecoveryHoldUntil: e.RecoveryHoldUntil,
		LastSeen:          e.LastSeen,
		EventID:           e.EventID,
	}
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
