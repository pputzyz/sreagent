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

	t.Run("equal_labels_both_missing", func(t *testing.T) {
		// When both source and target lack the EqualLabel, they should NOT be
		// considered equal — the label must be present on both sides.
		target := &model.AlertEvent{
			BaseModel: model.BaseModel{ID: 100},
			Labels:    model.JSONLabels{"team": "backend"}, // no "env" label
		}
		firingEvents := []model.AlertEvent{
			{
				BaseModel: model.BaseModel{ID: 1},
				Status:    model.EventStatusFiring,
				Labels:    model.JSONLabels{"alertname": "DatabaseDown"}, // no "env" label
			},
		}
		assert.False(t, matchesInhibition(rule, target, firingEvents),
			"should NOT inhibit when EqualLabel is missing on both sides")
	})
}

// Test_matchesInhibition_labelmatch_operators verifies that InhibitionRule uses
// labelmatch.Match which supports regex operators (=~, !~, !=, exact).
func Test_matchesInhibition_labelmatch_operators(t *testing.T) {
	t.Run("regex_match_source", func(t *testing.T) {
		rule := &model.InhibitionRule{
			SourceMatch: model.JSONLabels{"severity": "=~warning|critical"},
			TargetMatch: model.JSONLabels{"team": "backend"},
		}
		target := &model.AlertEvent{
			BaseModel: model.BaseModel{ID: 100},
			Labels:    model.JSONLabels{"team": "backend"},
		}
		firingEvents := []model.AlertEvent{
			{
				BaseModel: model.BaseModel{ID: 1},
				Status:    model.EventStatusFiring,
				Labels:    model.JSONLabels{"severity": "critical"},
			},
		}
		assert.True(t, matchesInhibition(rule, target, firingEvents),
			"regex operator =~ should match source severity")
	})

	t.Run("not_equal_operator", func(t *testing.T) {
		rule := &model.InhibitionRule{
			SourceMatch: model.JSONLabels{"severity": "!=info"},
			TargetMatch: model.JSONLabels{"team": "backend"},
		}
		target := &model.AlertEvent{
			BaseModel: model.BaseModel{ID: 100},
			Labels:    model.JSONLabels{"team": "backend"},
		}
		firingEvents := []model.AlertEvent{
			{
				BaseModel: model.BaseModel{ID: 1},
				Status:    model.EventStatusFiring,
				Labels:    model.JSONLabels{"severity": "critical"},
			},
		}
		assert.True(t, matchesInhibition(rule, target, firingEvents),
			"!= operator should match when severity is not info")
	})
}

// Test_matchesInhibition_supports_regex_source_match verifies that SourceMatch
// with the =~ regex operator matches a firing event whose label value satisfies
// the regex pattern. Uses "~warning|critical" to match severity=critical.
func Test_matchesInhibition_supports_regex_source_match(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"severity": "=~warning|critical"},
		TargetMatch: model.JSONLabels{"team": "backend"},
		EqualLabels: "",
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
			Labels:    model.JSONLabels{"severity": "critical", "alertname": "HighCPU"},
		},
	}

	assert.True(t, matchesInhibition(rule, target, firingEvents),
		"SourceMatch regex =~warning|critical should match severity=critical")
}

// Test_matchesInhibition_regex_source_no_match verifies that a regex SourceMatch
// does NOT match when the firing event's label value does not satisfy the pattern.
func Test_matchesInhibition_regex_source_no_match(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"severity": "=~warning|critical"},
		TargetMatch: model.JSONLabels{"team": "backend"},
		EqualLabels: "",
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "backend"},
	}

	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusFiring,
			Labels:    model.JSONLabels{"severity": "info", "alertname": "LowTraffic"},
		},
	}

	assert.False(t, matchesInhibition(rule, target, firingEvents),
		"SourceMatch regex =~warning|critical should NOT match severity=info")
}

// Test_matchesInhibition_equal_labels_both_missing_does_not_suppress verifies
// that when EqualLabels specifies a label (e.g. "host") that is missing from
// both the source and target alerts, the inhibition does NOT fire. Both sides
// must have the label present with the same value for suppression to occur.
func Test_matchesInhibition_equal_labels_both_missing_does_not_suppress(t *testing.T) {
	rule := &model.InhibitionRule{
		SourceMatch: model.JSONLabels{"alertname": "NodeDown"},
		TargetMatch: model.JSONLabels{"team": "infra"},
		EqualLabels: "host", // both source and target must have "host" with same value
	}

	target := &model.AlertEvent{
		BaseModel: model.BaseModel{ID: 100},
		Status:    model.EventStatusFiring,
		Labels:    model.JSONLabels{"team": "infra", "env": "production"}, // no "host"
	}

	firingEvents := []model.AlertEvent{
		{
			BaseModel: model.BaseModel{ID: 1},
			Status:    model.EventStatusFiring,
			Labels:    model.JSONLabels{"alertname": "NodeDown", "env": "production"}, // no "host"
		},
	}

	assert.False(t, matchesInhibition(rule, target, firingEvents),
		"should NOT suppress when EqualLabel 'host' is missing on both source and target")
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
