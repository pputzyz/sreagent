package muterule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sreagent/sreagent/internal/model"
)

// ---------------------------------------------------------------------------
// IsTimeWindowMuted tests
// ---------------------------------------------------------------------------

func Test_IsTimeWindowMuted_no_time_fields_always_muted(t *testing.T) {
	rule := &model.MuteRule{}
	assert.True(t, IsTimeWindowMuted(rule, time.Now()),
		"no time fields set should always return true (always muted)")
}

func Test_IsTimeWindowMuted_normal_range_inside(t *testing.T) {
	// Periodic window: 09:00-18:00
	rule := &model.MuteRule{
		PeriodicStart: "09:00",
		PeriodicEnd:   "18:00",
		Timezone:      "UTC",
	}
	// 12:00 UTC should be inside
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now),
		"12:00 should be inside 09:00-18:00 range")
}

func Test_IsTimeWindowMuted_normal_range_outside_before(t *testing.T) {
	rule := &model.MuteRule{
		PeriodicStart: "09:00",
		PeriodicEnd:   "18:00",
		Timezone:      "UTC",
	}
	// 08:00 UTC should be outside
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, now),
		"08:00 should be outside 09:00-18:00 range")
}

func Test_IsTimeWindowMuted_normal_range_outside_after(t *testing.T) {
	rule := &model.MuteRule{
		PeriodicStart: "09:00",
		PeriodicEnd:   "18:00",
		Timezone:      "UTC",
	}
	// 18:00 UTC should be outside (right-open)
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, now),
		"18:00 should be outside 09:00-18:00 range (right-open)")
}

func Test_IsTimeWindowMuted_overnight_range_inside_late(t *testing.T) {
	// Overnight window: 22:00-06:00
	rule := &model.MuteRule{
		PeriodicStart: "22:00",
		PeriodicEnd:   "06:00",
		Timezone:      "UTC",
	}
	// 23:00 UTC should be inside (after 22:00)
	now := time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now),
		"23:00 should be inside 22:00-06:00 overnight range")
}

func Test_IsTimeWindowMuted_overnight_range_inside_early(t *testing.T) {
	rule := &model.MuteRule{
		PeriodicStart: "22:00",
		PeriodicEnd:   "06:00",
		Timezone:      "UTC",
	}
	// 03:00 UTC should be inside (before 06:00)
	now := time.Date(2026, 5, 30, 3, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now),
		"03:00 should be inside 22:00-06:00 overnight range")
}

func Test_IsTimeWindowMuted_overnight_range_outside(t *testing.T) {
	rule := &model.MuteRule{
		PeriodicStart: "22:00",
		PeriodicEnd:   "06:00",
		Timezone:      "UTC",
	}
	// 12:00 UTC should be outside
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, now),
		"12:00 should be outside 22:00-06:00 overnight range")
}

func Test_IsTimeWindowMuted_day_of_week_match(t *testing.T) {
	// 2026-06-01 is a Monday (weekday 1)
	rule := &model.MuteRule{
		PeriodicStart: "00:00",
		PeriodicEnd:   "23:59",
		DaysOfWeek:    "1,2,3,4,5", // Mon-Fri
		Timezone:      "UTC",
	}
	monday := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, monday),
		"Monday should match DaysOfWeek 1,2,3,4,5")
}

func Test_IsTimeWindowMuted_day_of_week_no_match(t *testing.T) {
	// 2026-05-31 is a Sunday (weekday 0 → 7)
	rule := &model.MuteRule{
		PeriodicStart: "00:00",
		PeriodicEnd:   "23:59",
		DaysOfWeek:    "1,2,3,4,5", // Mon-Fri
		Timezone:      "UTC",
	}
	sunday := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, sunday),
		"Sunday should NOT match DaysOfWeek 1,2,3,4,5")
}

func Test_IsTimeWindowMuted_day_of_week_sunday_as_7(t *testing.T) {
	// 2026-05-31 is a Sunday (weekday 0 → 7 in ISO 8601)
	rule := &model.MuteRule{
		PeriodicStart: "00:00",
		PeriodicEnd:   "23:59",
		DaysOfWeek:    "7", // Sunday only
		Timezone:      "UTC",
	}
	sunday := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, sunday),
		"Sunday should match DaysOfWeek 7 (ISO 8601)")
}

func Test_IsTimeWindowMuted_one_time_window_inside(t *testing.T) {
	start := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 30, 14, 0, 0, 0, time.UTC)
	rule := &model.MuteRule{
		StartTime: &start,
		EndTime:   &end,
		Timezone:  "UTC",
	}
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now),
		"12:00 should be inside 10:00-14:00 one-time window")
}

func Test_IsTimeWindowMuted_one_time_window_outside(t *testing.T) {
	start := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 30, 14, 0, 0, 0, time.UTC)
	rule := &model.MuteRule{
		StartTime: &start,
		EndTime:   &end,
		Timezone:  "UTC",
	}
	now := time.Date(2026, 5, 30, 16, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, now),
		"16:00 should be outside 10:00-14:00 one-time window")
}

func Test_IsTimeWindowMuted_timezone_awareness(t *testing.T) {
	// Window: 09:00-18:00 in Asia/Shanghai (UTC+8)
	rule := &model.MuteRule{
		PeriodicStart: "09:00",
		PeriodicEnd:   "18:00",
		Timezone:      "Asia/Shanghai",
	}
	// 01:00 UTC = 09:00 Shanghai → inside
	now := time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now),
		"01:00 UTC (09:00 Shanghai) should be inside 09:00-18:00 Asia/Shanghai")

	// 08:00 UTC = 16:00 Shanghai → inside
	now2 := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	assert.True(t, IsTimeWindowMuted(rule, now2),
		"08:00 UTC (16:00 Shanghai) should be inside 09:00-18:00 Asia/Shanghai")

	// 23:00 UTC = 07:00 Shanghai (next day) → outside
	now3 := time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC)
	assert.False(t, IsTimeWindowMuted(rule, now3),
		"23:00 UTC (07:00 Shanghai) should be outside 09:00-18:00 Asia/Shanghai")
}

// ---------------------------------------------------------------------------
// IsMutedByRule tests
// ---------------------------------------------------------------------------

func Test_IsMutedByRule_label_match_all_required(t *testing.T) {
	rule := &model.MuteRule{
		MatchLabels: model.JSONLabels{"env": "prod", "service": "api"},
	}
	eventLabels := map[string]string{"env": "prod", "service": "api", "host": "web01"}

	assert.True(t, IsMutedByRule(rule, eventLabels, "critical", nil, time.Now()),
		"should match when event has ALL required labels")
}

func Test_IsMutedByRule_label_match_missing_label(t *testing.T) {
	rule := &model.MuteRule{
		MatchLabels: model.JSONLabels{"env": "prod", "service": "api"},
	}
	eventLabels := map[string]string{"env": "prod"} // missing "service"

	assert.False(t, IsMutedByRule(rule, eventLabels, "critical", nil, time.Now()),
		"should NOT match when event is missing a required label")
}

func Test_IsMutedByRule_label_match_wrong_value(t *testing.T) {
	rule := &model.MuteRule{
		MatchLabels: model.JSONLabels{"env": "prod"},
	}
	eventLabels := map[string]string{"env": "staging"}

	assert.False(t, IsMutedByRule(rule, eventLabels, "critical", nil, time.Now()),
		"should NOT match when label value differs")
}

func Test_IsMutedByRule_severity_filter_match(t *testing.T) {
	rule := &model.MuteRule{
		Severities: "critical,warning",
	}
	eventLabels := map[string]string{}

	assert.True(t, IsMutedByRule(rule, eventLabels, "critical", nil, time.Now()),
		"should match when event severity is in the filter")
}

func Test_IsMutedByRule_severity_filter_no_match(t *testing.T) {
	rule := &model.MuteRule{
		Severities: "critical,warning",
	}
	eventLabels := map[string]string{}

	assert.False(t, IsMutedByRule(rule, eventLabels, "info", nil, time.Now()),
		"should NOT match when event severity is not in the filter")
}

func Test_IsMutedByRule_severity_filter_empty_matches_all(t *testing.T) {
	rule := &model.MuteRule{
		Severities: "", // empty = all severities
	}
	eventLabels := map[string]string{}

	assert.True(t, IsMutedByRule(rule, eventLabels, "info", nil, time.Now()),
		"empty severities should match all")
}

func Test_IsMutedByRule_rule_id_filter_match(t *testing.T) {
	ruleID := uint(42)
	rule := &model.MuteRule{
		RuleIDs: "42,43,44",
	}
	eventLabels := map[string]string{}

	assert.True(t, IsMutedByRule(rule, eventLabels, "critical", &ruleID, time.Now()),
		"should match when rule ID is in the filter")
}

func Test_IsMutedByRule_rule_id_filter_no_match(t *testing.T) {
	ruleID := uint(99)
	rule := &model.MuteRule{
		RuleIDs: "42,43,44",
	}
	eventLabels := map[string]string{}

	assert.False(t, IsMutedByRule(rule, eventLabels, "critical", &ruleID, time.Now()),
		"should NOT match when rule ID is not in the filter")
}

func Test_IsMutedByRule_rule_id_filter_nil_matches_all(t *testing.T) {
	rule := &model.MuteRule{
		RuleIDs: "42,43,44",
	}
	eventLabels := map[string]string{}

	assert.True(t, IsMutedByRule(rule, eventLabels, "critical", nil, time.Now()),
		"nil ruleID should skip rule ID filter")
}

func Test_IsMutedByRule_time_window_outside_no_match(t *testing.T) {
	rule := &model.MuteRule{
		PeriodicStart: "02:00",
		PeriodicEnd:   "06:00",
		Timezone:      "UTC",
	}
	eventLabels := map[string]string{}
	// 12:00 UTC is outside 02:00-06:00
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.False(t, IsMutedByRule(rule, eventLabels, "critical", nil, now),
		"should NOT match when current time is outside the window")
}

func Test_IsMutedByRule_combined_all_criteria_pass(t *testing.T) {
	ruleID := uint(42)
	rule := &model.MuteRule{
		MatchLabels:   model.JSONLabels{"env": "prod"},
		Severities:    "critical,warning",
		RuleIDs:       "42",
		PeriodicStart: "00:00",
		PeriodicEnd:   "23:59",
		Timezone:      "UTC",
	}
	eventLabels := map[string]string{"env": "prod", "service": "api"}
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.True(t, IsMutedByRule(rule, eventLabels, "critical", &ruleID, now),
		"should match when ALL criteria pass: labels + severity + rule ID + time")
}

func Test_IsMutedByRule_combined_one_criterion_fails(t *testing.T) {
	ruleID := uint(42)
	rule := &model.MuteRule{
		MatchLabels:   model.JSONLabels{"env": "prod"},
		Severities:    "critical,warning",
		RuleIDs:       "42",
		PeriodicStart: "00:00",
		PeriodicEnd:   "23:59",
		Timezone:      "UTC",
	}
	// Missing "env" label
	eventLabels := map[string]string{"service": "api"}
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.False(t, IsMutedByRule(rule, eventLabels, "critical", &ruleID, now),
		"should NOT match when labels don't match even if other criteria pass")
}

// ---------------------------------------------------------------------------
// LoadMuteTimezone tests
// ---------------------------------------------------------------------------

func Test_LoadMuteTimezone_empty_defaults_to_shanghai(t *testing.T) {
	loc := LoadMuteTimezone("")
	assert.Equal(t, "Asia/Shanghai", loc.String())
}

func Test_LoadMuteTimezone_valid_timezone(t *testing.T) {
	loc := LoadMuteTimezone("America/New_York")
	assert.Equal(t, "America/New_York", loc.String())
}

func Test_LoadMuteTimezone_invalid_falls_back_to_shanghai(t *testing.T) {
	loc := LoadMuteTimezone("Invalid/Timezone")
	assert.Equal(t, "Asia/Shanghai", loc.String())
}
