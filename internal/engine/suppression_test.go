package engine

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// severityRank tests
// ---------------------------------------------------------------------------

func Test_severityRank_known_severities(t *testing.T) {
	tests := []struct {
		severity string
		expected int
	}{
		{"info", 1},
		{"warning", 2},
		{"critical", 3},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			assert.Equal(t, tt.expected, severityRank(tt.severity))
		})
	}
}

func Test_severityRank_legacy_p0_p4(t *testing.T) {
	tests := []struct {
		severity string
		expected int
	}{
		{"p0", 4},
		{"p1", 3},
		{"p2", 2},
		{"p3", 1},
		{"p4", 1},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			assert.Equal(t, tt.expected, severityRank(tt.severity),
				"legacy severity %q should map to rank %d", tt.severity, tt.expected)
		})
	}
}

func Test_severityRank_unknown_defaults_to_info(t *testing.T) {
	unknown := []string{"debug", "error", "", "CRITICAL"}
	for _, sev := range unknown {
		t.Run(sev, func(t *testing.T) {
			assert.Equal(t, 1, severityRank(sev), "unknown severity %q should default to 1", sev)
		})
	}
}

// ---------------------------------------------------------------------------
// NewLevelSuppressor
// ---------------------------------------------------------------------------

func Test_NewLevelSuppressor_initializes_empty(t *testing.T) {
	s := NewLevelSuppressor()
	assert.NotNil(t, s)
	assert.NotNil(t, s.activeSeverities)
	assert.Empty(t, s.activeSeverities)
}

// ---------------------------------------------------------------------------
// ShouldSuppress tests
// ---------------------------------------------------------------------------

func Test_ShouldSuppress_no_active_returns_false(t *testing.T) {
	s := NewLevelSuppressor()
	assert.False(t, s.ShouldSuppress(1, "fp1", "critical"),
		"should not suppress when no active severities exist")
}

func Test_ShouldSuppress_higher_active_suppresses_lower(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	// warning (2) < critical (3) → should suppress
	assert.True(t, s.ShouldSuppress(1, "fp1", "warning"),
		"warning should be suppressed when critical is active")

	// info (1) < critical (3) → should suppress
	assert.True(t, s.ShouldSuppress(1, "fp1", "info"),
		"info should be suppressed when critical is active")
}

func Test_ShouldSuppress_lower_active_does_not_suppress_higher(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "warning")

	// critical (3) > warning (2) → should NOT suppress
	assert.False(t, s.ShouldSuppress(1, "fp1", "critical"),
		"critical should NOT be suppressed when warning is active")
}

func Test_ShouldSuppress_equal_severity_does_not_suppress(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "warning")

	assert.False(t, s.ShouldSuppress(1, "fp1", "warning"),
		"equal severity should NOT suppress (same level)")
}

func Test_ShouldSuppress_different_rule_not_affected(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	// Different rule ID — should not be affected
	assert.False(t, s.ShouldSuppress(2, "fp1", "warning"),
		"suppress should not cross rule boundaries")
}

func Test_ShouldSuppress_different_fingerprint_not_affected(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	// Different fingerprint — should not be affected
	assert.False(t, s.ShouldSuppress(1, "fp2", "warning"),
		"suppress should not cross fingerprint boundaries")
}

// ---------------------------------------------------------------------------
// UpdateSeverity tests
// ---------------------------------------------------------------------------

func Test_UpdateSeverity_adds_new_entry(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "warning")

	s.mu.RLock()
	sev, ok := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.True(t, ok, "entry should be created")
	assert.Equal(t, "warning", sev)
}

func Test_UpdateSeverity_upgrades_to_higher(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "info")
	s.UpdateSeverity(1, "fp1", "critical")

	s.mu.RLock()
	sev := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.Equal(t, "critical", sev, "should upgrade to higher severity")
}

func Test_UpdateSeverity_does_not_downgrade(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(1, "fp1", "info")

	s.mu.RLock()
	sev := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.Equal(t, "critical", sev, "should NOT downgrade from critical to info")
}

func Test_UpdateSeverity_does_not_downgrade_warning_to_info(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "warning")
	s.UpdateSeverity(1, "fp1", "info")

	s.mu.RLock()
	sev := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.Equal(t, "warning", sev, "should NOT downgrade from warning to info")
}

func Test_UpdateSeverity_multiple_fingerprints(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(1, "fp2", "warning")
	s.UpdateSeverity(1, "fp3", "info")

	assert.Equal(t, "critical", s.activeSeverities[1]["fp1"])
	assert.Equal(t, "warning", s.activeSeverities[1]["fp2"])
	assert.Equal(t, "info", s.activeSeverities[1]["fp3"])
}

func Test_UpdateSeverity_multiple_rules(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(2, "fp1", "warning")

	assert.Equal(t, "critical", s.activeSeverities[1]["fp1"])
	assert.Equal(t, "warning", s.activeSeverities[2]["fp1"])
}

// ---------------------------------------------------------------------------
// RemoveRule tests
// ---------------------------------------------------------------------------

func Test_RemoveRule_cleans_all_fingerprints(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(1, "fp2", "warning")
	s.UpdateSeverity(1, "fp3", "info")

	s.RemoveRule(1)

	s.mu.RLock()
	_, exists := s.activeSeverities[1]
	s.mu.RUnlock()

	assert.False(t, exists, "rule entry should be completely removed")
}

func Test_RemoveRule_does_not_affect_other_rules(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(2, "fp1", "warning")

	s.RemoveRule(1)

	_, exists := s.activeSeverities[1]
	assert.False(t, exists, "rule 1 should be removed")

	sev, exists := s.activeSeverities[2]["fp1"]
	assert.True(t, exists, "rule 2 should still exist")
	assert.Equal(t, "warning", sev)
}

func Test_RemoveRule_nonexistent_rule_no_panic(t *testing.T) {
	s := NewLevelSuppressor()
	assert.NotPanics(t, func() {
		s.RemoveRule(999)
	}, "removing non-existent rule should not panic")
}

// ---------------------------------------------------------------------------
// RemoveSeverity tests
// ---------------------------------------------------------------------------

func Test_RemoveSeverity_removes_matching(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	s.RemoveSeverity(1, "fp1", "critical")

	s.mu.RLock()
	_, exists := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.False(t, exists, "matching severity should be removed")
}

func Test_RemoveSeverity_does_not_remove_mismatched(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	// Try to remove "warning" but the active one is "critical"
	s.RemoveSeverity(1, "fp1", "warning")

	s.mu.RLock()
	sev, exists := s.activeSeverities[1]["fp1"]
	s.mu.RUnlock()

	assert.True(t, exists, "entry should still exist")
	assert.Equal(t, "critical", sev, "severity should not be changed")
}

func Test_RemoveSeverity_cleans_empty_rule_map(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	s.RemoveSeverity(1, "fp1", "critical")

	s.mu.RLock()
	_, exists := s.activeSeverities[1]
	s.mu.RUnlock()

	assert.False(t, exists, "empty rule map should be cleaned up")
}

func Test_RemoveSeverity_keeps_other_fingerprints(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")
	s.UpdateSeverity(1, "fp2", "warning")

	s.RemoveSeverity(1, "fp1", "critical")

	s.mu.RLock()
	_, fp1Exists := s.activeSeverities[1]["fp1"]
	fp2Sev, fp2Exists := s.activeSeverities[1]["fp2"]
	s.mu.RUnlock()

	assert.False(t, fp1Exists, "fp1 should be removed")
	assert.True(t, fp2Exists, "fp2 should still exist")
	assert.Equal(t, "warning", fp2Sev)
}

func Test_RemoveSeverity_nonexistent_rule_no_panic(t *testing.T) {
	s := NewLevelSuppressor()
	assert.NotPanics(t, func() {
		s.RemoveSeverity(999, "fp1", "critical")
	})
}

func Test_RemoveSeverity_nonexistent_fingerprint_no_panic(t *testing.T) {
	s := NewLevelSuppressor()
	s.UpdateSeverity(1, "fp1", "critical")

	assert.NotPanics(t, func() {
		s.RemoveSeverity(1, "fp_nonexistent", "critical")
	})
}

// ---------------------------------------------------------------------------
// Concurrent access safety
// ---------------------------------------------------------------------------

func Test_LevelSuppressor_concurrent_access(t *testing.T) {
	s := NewLevelSuppressor()

	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ruleID := uint(id%5 + 1)
			fp := "fp_common"
			sev := []string{"info", "warning", "critical"}[id%3]
			s.UpdateSeverity(ruleID, fp, sev)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ruleID := uint(id%5 + 1)
			s.ShouldSuppress(ruleID, "fp_common", "info")
		}(i)
	}

	// Concurrent removes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ruleID := uint(id%5 + 1)
			s.RemoveSeverity(ruleID, "fp_common", "info")
		}(i)
	}

	wg.Wait()
	// No race condition = test passes (run with -race flag)
}

// ---------------------------------------------------------------------------
// End-to-end suppression scenario
// ---------------------------------------------------------------------------

func Test_suppression_scenario_critical_suppresses_warning_and_info(t *testing.T) {
	s := NewLevelSuppressor()
	ruleID := uint(1)
	fp := "test-fingerprint"

	// Step 1: info fires first
	s.UpdateSeverity(ruleID, fp, "info")
	assert.False(t, s.ShouldSuppress(ruleID, fp, "info"), "info should not suppress itself")

	// Step 2: warning fires — should NOT be suppressed (higher than info)
	assert.False(t, s.ShouldSuppress(ruleID, fp, "warning"), "warning should not be suppressed by info")
	s.UpdateSeverity(ruleID, fp, "warning") // upgrades to warning

	// Step 3: critical fires — should NOT be suppressed (higher than warning)
	assert.False(t, s.ShouldSuppress(ruleID, fp, "critical"), "critical should not be suppressed by warning")
	s.UpdateSeverity(ruleID, fp, "critical") // upgrades to critical

	// Step 4: another warning attempt — SHOULD be suppressed now
	assert.True(t, s.ShouldSuppress(ruleID, fp, "warning"), "warning should be suppressed by critical")

	// Step 5: another info attempt — SHOULD be suppressed
	assert.True(t, s.ShouldSuppress(ruleID, fp, "info"), "info should be suppressed by critical")

	// Step 6: resolve critical
	s.RemoveSeverity(ruleID, fp, "critical")

	// Step 7: warning should now be allowed again
	assert.False(t, s.ShouldSuppress(ruleID, fp, "warning"), "warning should not be suppressed after critical resolves")
}
