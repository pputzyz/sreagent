package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sreagent/sreagent/internal/model"
)

// Test_matchesInhibition_matching_source_firing_returns_true verifies that when
// a firing source alert matches the rule's SourceMatch and the target alert
// matches TargetMatch, the inhibition logic returns true.
func Test_matchesInhibition_matching_source_firing_returns_true(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "DatabaseDown", "severity": "critical"},
		TargetMatch: model.JSONLabels{"team": "backend"},
		EqualLabels: "", // no equal-label constraint
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "backend", "instance": "web-1"},
	}

	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusFiring,
			Labels:    model.JSONLabels{"alertname": "DatabaseDown", "severity": "critical", "instance": "db-1"},
		},
	}

	assert.True(t, matchesInhibition(rule, target, firingEvents),
		"target should be inhibited when a matching source alert is firing")
}

// Test_matchesInhibition_no_matching_source_returns_false verifies that when
// no firing source alert matches the rule's SourceMatch, the target is NOT inhibited.
func Test_matchesInhibition_no_matching_source_returns_false(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "DatabaseDown"},
		TargetMatch: model.JSONLabels{"team": "backend"},
		EqualLabels: "",
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "backend"},
	}

	// Source alert does NOT match SourceMatch (different alertname).
	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusFiring,
			Labels:    model.JSONLabels{"alertname": "DiskFull", "severity": "warning"},
		},
	}

	assert.False(t, matchesInhibition(rule, target, firingEvents),
		"target should NOT be inhibited when no source matches")
}

// Test_matchesInhibition_target_does_not_match verifies that when the target
// alert does NOT match TargetMatch, it is not inhibited even if a source matches.
func Test_matchesInhibition_target_does_not_match(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "DatabaseDown"},
		TargetMatch: model.JSONLabels{"team": "backend"},
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "frontend"}, // does not match
	}

	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusFiring,
			Labels:    model.JSONLabels{"alertname": "DatabaseDown"},
		},
	}

	assert.False(t, matchesInhibition(rule, target, firingEvents),
		"target with non-matching labels should NOT be inhibited")
}

// Test_matchesInhibition_resolved_source_skipped verifies that a resolved
// source alert does NOT cause inhibition.
func Test_matchesInhibition_resolved_source_skipped(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "DatabaseDown"},
		TargetMatch: model.JSONLabels{"team": "backend"},
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "backend"},
	}

	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusResolved, // resolved, not firing
			Labels:    model.JSONLabels{"alertname": "DatabaseDown"},
		},
	}

	assert.False(t, matchesInhibition(rule, target, firingEvents),
		"resolved source alerts must NOT cause inhibition")
}

// Test_matchesInhibition_equal_labels_enforced verifies that EqualLabels
// constraint is respected: source and target must have the same value for
// each listed label.
func Test_matchesInhibition_equal_labels_enforced(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "DatabaseDown"},
		TargetMatch: model.JSONLabels{"team": "backend"},
		EqualLabels: "env", // source and target must share same "env" value
	}

	t.Run("equal_labels_match", func(t *testing.T) {
		target := &model.AlertEvent{
			BaseModel: model.BaseModel{ID: 100},
			Labels:    model.JSONLabels{"team": "backend", "env": "production"},
		}
		firingEvents := []model.AlertEvent{
			{
				BaseModel: model.BaseModel{ID: 1},
				Status:    model.EventStatusFiring,
				Labels:    model.JSONLabels{"alertname": "DatabaseDown", "env": "production"},
			},
		}
		assert.True(t, matchesInhibition(rule, target, firingEvents),
			"should inhibit when EqualLabels values match")
	})

	t.Run("equal_labels_mismatch", func(t *testing.T) {
		target := &model.AlertEvent{
			BaseModel: model.BaseModel{ID: 100},
			Labels:    model.JSONLabels{"team": "backend", "env": "staging"},
		}
		firingEvents := []model.AlertEvent{
			{
				BaseModel: model.BaseModel{ID: 1},
				Status:    model.EventStatusFiring,
				Labels:    model.JSONLabels{"alertname": "DatabaseDown", "env": "production"},
			},
		}
		assert.False(t, matchesInhibition(rule, target, firingEvents),
			"should NOT inhibit when EqualLabels values differ")
	})
}

// Test_inhibitionLabelsMatch verifies the label matching helper.
func Test_inhibitionLabelsMatch(t *testing.T) {
	tests := []struct {
		name     string
		matchers model.JSONLabels
		labels   model.JSONLabels
		expected bool
	}{
		{
			name:     "empty matchers always match",
			matchers: model.JSONLabels{},
			labels:   model.JSONLabels{"foo": "bar"},
			expected: true,
		},
		{
			name:     "all matchers present",
			matchers: model.JSONLabels{"a": "1", "b": "2"},
			labels:   model.JSONLabels{"a": "1", "b": "2", "c": "3"},
			expected: true,
		},
		{
			name:     "missing label",
			matchers: model.JSONLabels{"a": "1", "b": "2"},
			labels:   model.JSONLabels{"a": "1"},
			expected: false,
		},
		{
			name:     "value mismatch",
			matchers: model.JSONLabels{"a": "1"},
			labels:   model.JSONLabels{"a": "999"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, inhibitionLabelsMatch(tt.matchers, tt.labels))
		})
	}
}

// Test_parseEqualLabels verifies parsing of comma-separated label lists.
func Test_parseEqualLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"single label", "env", []string{"env"}},
		{"multiple labels", "env,region", []string{"env", "region"}},
		{"with spaces", " env , region ", []string{"env", "region"}},
		{"trailing comma", "env,", []string{"env"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseEqualLabels(tt.input))
		})
	}
}
