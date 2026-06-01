package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
)

// newTestDispatchService creates a minimal DispatchService for unit testing
// private methods that don't require DB access.
func newTestDispatchService() *DispatchService {
	return &DispatchService{logger: zap.NewNop()}
}

// ---------------------------------------------------------------------------
// isActiveNow tests
// ---------------------------------------------------------------------------

func Test_isActiveNow_empty_config_always_active(t *testing.T) {
	svc := newTestDispatchService()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow("", now), "empty config should always be active")
	assert.True(t, svc.isActiveNow("null", now), "null config should always be active")
}

func Test_isActiveNow_disabled_config_always_active(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":false,"timezone":"UTC","start_time":"09:00","end_time":"18:00"}`
	now := time.Date(2026, 5, 30, 3, 0, 0, 0, time.UTC) // 03:00 outside range

	assert.True(t, svc.isActiveNow(cfg, now),
		"disabled config should always be active regardless of time")
}

func Test_isActiveNow_normal_range_inside(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","start_time":"09:00","end_time":"18:00"}`
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow(cfg, now),
		"12:00 should be inside 09:00-18:00 normal range")
}

func Test_isActiveNow_normal_range_outside(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","start_time":"09:00","end_time":"18:00"}`
	now := time.Date(2026, 5, 30, 20, 0, 0, 0, time.UTC)

	assert.False(t, svc.isActiveNow(cfg, now),
		"20:00 should be outside 09:00-18:00 normal range")
}

func Test_isActiveNow_overnight_range_inside_late(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","start_time":"22:00","end_time":"06:00"}`
	now := time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow(cfg, now),
		"23:00 should be inside 22:00-06:00 overnight range")
}

func Test_isActiveNow_overnight_range_inside_early(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","start_time":"22:00","end_time":"06:00"}`
	now := time.Date(2026, 5, 30, 3, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow(cfg, now),
		"03:00 should be inside 22:00-06:00 overnight range")
}

func Test_isActiveNow_overnight_range_outside(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","start_time":"22:00","end_time":"06:00"}`
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.False(t, svc.isActiveNow(cfg, now),
		"12:00 should be outside 22:00-06:00 overnight range")
}

func Test_isActiveNow_day_filter_match(t *testing.T) {
	svc := newTestDispatchService()
	// 2026-06-01 is Monday (weekday 1)
	cfg := `{"enabled":true,"timezone":"UTC","days_of_week":[1,2,3,4,5]}`
	monday := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow(cfg, monday),
		"Monday should match days_of_week [1,2,3,4,5]")
}

func Test_isActiveNow_day_filter_no_match(t *testing.T) {
	svc := newTestDispatchService()
	// 2026-05-31 is Sunday (weekday 0)
	cfg := `{"enabled":true,"timezone":"UTC","days_of_week":[1,2,3,4,5]}`
	sunday := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	assert.False(t, svc.isActiveNow(cfg, sunday),
		"Sunday should NOT match days_of_week [1,2,3,4,5]")
}

func Test_isActiveNow_day_filter_empty_matches_all(t *testing.T) {
	svc := newTestDispatchService()
	cfg := `{"enabled":true,"timezone":"UTC","days_of_week":[]}`
	sunday := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow(cfg, sunday),
		"empty days_of_week should match all days")
}

func Test_isActiveNow_invalid_json_always_active(t *testing.T) {
	svc := newTestDispatchService()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	assert.True(t, svc.isActiveNow("not-json", now),
		"invalid JSON should default to always active")
}

// ---------------------------------------------------------------------------
// matchConditions tests
// ---------------------------------------------------------------------------

func Test_matchConditions_empty_matches_all(t *testing.T) {
	svc := newTestDispatchService()
	labels := model.JSONLabels{"env": "prod"}

	assert.True(t, svc.matchConditions("", labels, "critical"), "empty conditions should match all")
	assert.True(t, svc.matchConditions("[]", labels, "critical"), "[] conditions should match all")
	assert.True(t, svc.matchConditions("null", labels, "critical"), "null conditions should match all")
}

func Test_matchConditions_severity_eq(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[{"field":"severity","operator":"eq","value":"critical"}]`
	labels := model.JSONLabels{}

	assert.True(t, svc.matchConditions(cond, labels, "critical"), "severity eq critical should match")
	assert.False(t, svc.matchConditions(cond, labels, "warning"), "severity eq critical should not match warning")
}

func Test_matchConditions_label_contains(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[{"field":"labels.service","operator":"contains","value":"api"}]`
	labels := model.JSONLabels{"service": "api-gateway"}

	assert.True(t, svc.matchConditions(cond, labels, "critical"),
		"'api-gateway' contains 'api' should match")
}

func Test_matchConditions_label_in(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[{"field":"labels.env","operator":"in","value":"prod,staging"}]`

	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"env": "prod"}, "critical"))
	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"env": "staging"}, "critical"))
	assert.False(t, svc.matchConditions(cond, model.JSONLabels{"env": "dev"}, "critical"))
}

func Test_matchConditions_multiple_conditions_all_must_pass(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[
		{"field":"severity","operator":"eq","value":"critical"},
		{"field":"labels.env","operator":"eq","value":"prod"}
	]`

	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"env": "prod"}, "critical"))
	assert.False(t, svc.matchConditions(cond, model.JSONLabels{"env": "staging"}, "critical"),
		"should fail when one condition doesn't match")
}

func Test_matchConditions_invalid_json_matches_none(t *testing.T) {
	svc := newTestDispatchService()
	labels := model.JSONLabels{"env": "prod"}

	assert.False(t, svc.matchConditions("not-json", labels, "critical"),
		"invalid JSON should match none (fail-closed)")
}

// ---------------------------------------------------------------------------
// evalDispatchCondition tests
// ---------------------------------------------------------------------------

func Test_evalDispatchCondition_eq(t *testing.T) {
	assert.True(t, evalDispatchCondition("eq", "prod", "prod"))
	assert.False(t, evalDispatchCondition("eq", "prod", "staging"))
}

func Test_evalDispatchCondition_ne(t *testing.T) {
	assert.True(t, evalDispatchCondition("ne", "prod", "staging"))
	assert.False(t, evalDispatchCondition("ne", "prod", "prod"))
}

func Test_evalDispatchCondition_contains(t *testing.T) {
	assert.True(t, evalDispatchCondition("contains", "api-gateway", "api"))
	assert.False(t, evalDispatchCondition("contains", "web-frontend", "api"))
}

func Test_evalDispatchCondition_not_contains(t *testing.T) {
	assert.True(t, evalDispatchCondition("not_contains", "web-frontend", "api"))
	assert.False(t, evalDispatchCondition("not_contains", "api-gateway", "api"))
}

func Test_evalDispatchCondition_regex(t *testing.T) {
	assert.True(t, evalDispatchCondition("regex", "api-gateway", "^api-.*"))
	assert.False(t, evalDispatchCondition("regex", "web-frontend", "^api-.*"))
}

func Test_evalDispatchCondition_regex_invalid(t *testing.T) {
	assert.False(t, evalDispatchCondition("regex", "test", "[invalid"),
		"invalid regex should return false")
}

func Test_evalDispatchCondition_in(t *testing.T) {
	assert.True(t, evalDispatchCondition("in", "prod", "prod,staging,dev"))
	assert.True(t, evalDispatchCondition("in", "staging", "prod,staging,dev"))
	assert.False(t, evalDispatchCondition("in", "test", "prod,staging,dev"))
}

func Test_evalDispatchCondition_not_in(t *testing.T) {
	assert.True(t, evalDispatchCondition("not_in", "test", "prod,staging,dev"))
	assert.False(t, evalDispatchCondition("not_in", "prod", "prod,staging,dev"))
}

func Test_evalDispatchCondition_unknown_operator_matches_all(t *testing.T) {
	assert.True(t, evalDispatchCondition("unknown", "any", "any"),
		"unknown operator should match all (safe default)")
}

// ---------------------------------------------------------------------------
// expandTemplate tests
// ---------------------------------------------------------------------------

func Test_expandTemplate_basic_replacement(t *testing.T) {
	labels := model.JSONLabels{"env": "prod", "service": "api"}
	result := expandTemplate("{{env}}-{{service}}", labels)
	assert.Equal(t, "prod-api", result)
}

func Test_expandTemplate_labels_prefix(t *testing.T) {
	labels := model.JSONLabels{"env": "prod"}
	result := expandTemplate("{{labels.env}}", labels)
	assert.Equal(t, "prod", result)
}

func Test_expandTemplate_missing_key_keeps_placeholder(t *testing.T) {
	labels := model.JSONLabels{"env": "prod"}
	result := expandTemplate("{{env}}-{{missing}}", labels)
	assert.Equal(t, "prod-{{missing}}", result)
}

func Test_expandTemplate_no_placeholders(t *testing.T) {
	labels := model.JSONLabels{"env": "prod"}
	result := expandTemplate("no placeholders here", labels)
	assert.Equal(t, "no placeholders here", result)
}

func Test_expandTemplate_empty_template(t *testing.T) {
	labels := model.JSONLabels{"env": "prod"}
	result := expandTemplate("", labels)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// ApplyLabelEnhancements tests
// ---------------------------------------------------------------------------

func Test_ApplyLabelEnhancements_empty_rules_returns_original(t *testing.T) {
	svc := newTestDispatchService()
	labels := model.JSONLabels{"env": "prod"}

	assert.Equal(t, labels, svc.ApplyLabelEnhancements("", labels))
	assert.Equal(t, labels, svc.ApplyLabelEnhancements("[]", labels))
	assert.Equal(t, labels, svc.ApplyLabelEnhancements("null", labels))
}

func Test_ApplyLabelEnhancements_set_label(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"set","set_key":"team","set_value":"platform"}]`
	labels := model.JSONLabels{"env": "prod"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "platform", result["team"])
	assert.Equal(t, "prod", result["env"], "original labels should be preserved")
}

func Test_ApplyLabelEnhancements_set_overwrite_false(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"set","set_key":"team","set_value":"new-team","overwrite":false}]`
	labels := model.JSONLabels{"team": "old-team"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "old-team", result["team"],
		"should NOT overwrite existing value when overwrite=false")
}

func Test_ApplyLabelEnhancements_set_overwrite_true(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"set","set_key":"team","set_value":"new-team","overwrite":true}]`
	labels := model.JSONLabels{"team": "old-team"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "new-team", result["team"],
		"should overwrite existing value when overwrite=true")
}

func Test_ApplyLabelEnhancements_delete_label(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"delete","delete_key":"temp"}]`
	labels := model.JSONLabels{"env": "prod", "temp": "remove-me"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "prod", result["env"])
	_, exists := result["temp"]
	assert.False(t, exists, "temp label should be deleted")
}

func Test_ApplyLabelEnhancements_combine_template(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"combine","target_label":"full_name","template":"{{env}}-{{service}}"}]`
	labels := model.JSONLabels{"env": "prod", "service": "api"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "prod-api", result["full_name"])
}

func Test_ApplyLabelEnhancements_map_label(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"map","mapping_source_label":"env","target_label":"env_label","mapping_table":{"prod":"production","staging":"staging-env"}}]`
	labels := model.JSONLabels{"env": "prod"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	assert.Equal(t, "production", result["env_label"])
}

func Test_ApplyLabelEnhancements_map_no_match(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"map","mapping_source_label":"env","target_label":"env_label","mapping_table":{"prod":"production"}}]`
	labels := model.JSONLabels{"env": "dev"}

	result := svc.ApplyLabelEnhancements(rules, labels)
	_, exists := result["env_label"]
	assert.False(t, exists, "should not set target when mapping has no match")
}

func Test_ApplyLabelEnhancements_does_not_mutate_original(t *testing.T) {
	svc := newTestDispatchService()
	rules := `[{"type":"set","set_key":"new_key","set_value":"new_val"}]`
	labels := model.JSONLabels{"env": "prod"}

	_ = svc.ApplyLabelEnhancements(rules, labels)
	_, exists := labels["new_key"]
	assert.False(t, exists, "original labels should not be mutated")
}

// ---------------------------------------------------------------------------
// Additional matchConditions tests
// ---------------------------------------------------------------------------

func TestMatchConditions_LabelNotIn(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[{"field":"labels.env","operator":"not_in","value":"dev,test"}]`

	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"env": "prod"}, "critical"),
		"prod is not in dev,test so should match")
	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"env": "staging"}, "critical"),
		"staging is not in dev,test so should match")
	assert.False(t, svc.matchConditions(cond, model.JSONLabels{"env": "dev"}, "critical"),
		"dev is in dev,test so should NOT match")
	assert.False(t, svc.matchConditions(cond, model.JSONLabels{"env": "test"}, "critical"),
		"test is in dev,test so should NOT match")
}

func TestMatchConditions_RegexLabelMatch(t *testing.T) {
	svc := newTestDispatchService()
	cond := `[{"field":"labels.service","operator":"regex","value":"^api-.*"}]`

	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"service": "api-gateway"}, "critical"),
		"api-gateway matches ^api-.*")
	assert.True(t, svc.matchConditions(cond, model.JSONLabels{"service": "api-auth"}, "critical"),
		"api-auth matches ^api-.*")
	assert.False(t, svc.matchConditions(cond, model.JSONLabels{"service": "web-frontend"}, "critical"),
		"web-frontend does NOT match ^api-.*")
}

func TestExpandTemplate_LabelsDotPrefix_Multiple(t *testing.T) {
	labels := model.JSONLabels{
		"env":     "prod",
		"service": "api-gateway",
		"region":  "us-east-1",
	}
	result := expandTemplate("{{labels.env}}/{{labels.service}}@{{labels.region}}", labels)
	assert.Equal(t, "prod/api-gateway@us-east-1", result)
}
