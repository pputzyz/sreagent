package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// evalExpr tests — safe expression evaluator
// ---------------------------------------------------------------------------

func TestEvalExpr_SimpleEqual(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"equal_match", `.Severity == "critical"`, true},
		{"equal_no_match", `.Severity == "warning"`, false},
		{"label_equal_match", `.Labels.env == "prod"`, true},
		{"label_equal_no_match", `.Labels.env == "staging"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_SimpleNotEqual(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"neq_match", `.Severity != "warning"`, true},
		{"neq_no_match", `.Severity != "critical"`, false},
		{"label_neq_match", `.Labels.env != "staging"`, true},
		{"label_neq_no_match", `.Labels.env != "prod"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_And(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Status":   "firing",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"both_true", `.Severity == "critical" AND .Status == "firing"`, true},
		{"first_false", `.Severity == "warning" AND .Status == "firing"`, false},
		{"second_false", `.Severity == "critical" AND .Status == "resolved"`, false},
		{"both_false", `.Severity == "warning" AND .Status == "resolved"`, false},
		{"triple_and", `.Severity == "critical" AND .Status == "firing" AND .Labels.env == "prod"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_Or(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Status":   "firing",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"both_true", `.Severity == "critical" OR .Status == "firing"`, true},
		{"first_true", `.Severity == "critical" OR .Status == "resolved"`, true},
		{"second_true", `.Severity == "warning" OR .Status == "firing"`, true},
		{"both_false", `.Severity == "warning" OR .Status == "resolved"`, false},
		{"triple_or", `.Severity == "warning" OR .Status == "resolved" OR .Labels.env == "prod"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_Not(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Status":   "firing",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"not_true", `NOT .Severity == "warning"`, true},
		{"not_false", `NOT .Severity == "critical"`, false},
		{"double_not", `NOT NOT .Severity == "critical"`, true},
		{"not_with_and", `NOT .Severity == "warning" AND .Status == "firing"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_Nested(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Status":   "firing",
		"Labels":   map[string]string{"env": "prod", "service": "api"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{
			"parenthesized_or_and",
			`(.Severity == "critical" OR .Severity == "warning") AND .Status == "firing"`,
			true,
		},
		{
			"parenthesized_or_first_missing",
			`(.Severity == "info" OR .Severity == "warning") AND .Status == "firing"`,
			false,
		},
		{
			"nested_parens",
			`(.Severity == "critical" AND .Status == "firing") OR (.Labels.env == "staging")`,
			true,
		},
		{
			"complex_nested",
			`(.Severity == "warning" OR .Labels.env == "prod") AND NOT .Status == "resolved"`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvalExpr_MissingField(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
		"Labels":   map[string]string{"env": "prod"},
	}

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{
			"missing_top_level_field_eq_empty",
			`.Source == ""`,
			true,
		},
		{
			"missing_label_eq_value",
			`.Labels.missing == "value"`,
			false,
		},
		{
			"missing_label_ne_empty",
			`.Labels.missing != ""`,
			false,
		},
		{
			"missing_field_in_and",
			`.Severity == "critical" AND .Labels.missing == ""`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalExpr(tt.expr, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// parseExpr tests — syntax validation
// ---------------------------------------------------------------------------

func TestParseExpr_ValidExpressions(t *testing.T) {
	valid := []string{
		`.Severity == "critical"`,
		`.Labels.env != "prod"`,
		`.A == "1" AND .B == "2"`,
		`.A == "1" OR .B == "2"`,
		`NOT .A == "1"`,
		`(.A == "1" OR .A == "2") AND .B == "3"`,
		`true`,
		`false`,
	}

	for _, expr := range valid {
		t.Run(expr, func(t *testing.T) {
			_, err := parseExpr(expr)
			assert.NoError(t, err, "expression should parse without error: %s", expr)
		})
	}
}

func TestParseExpr_InvalidExpressions(t *testing.T) {
	invalid := []string{
		``,               // empty
		`.Severity`,      // missing operator
		`.Severity ==`,   // missing value
		`== "value"`,     // missing field
		`.A == .B`,       // non-string comparison value
		`unknown_ident`,  // unknown identifier
		`(.A == "1"`,     // unmatched paren
		`.A == "1" junk`, // trailing junk
	}

	for _, expr := range invalid {
		t.Run(expr, func(t *testing.T) {
			_, err := parseExpr(expr)
			assert.Error(t, err, "expression should fail to parse: %s", expr)
		})
	}
}

// ---------------------------------------------------------------------------
// newEventDrop constructor tests
// ---------------------------------------------------------------------------

func TestNewEventDrop_ValidCondition(t *testing.T) {
	config := map[string]interface{}{
		"condition": `.Severity == "critical"`,
	}
	proc, err := newEventDrop(config)
	assert.NoError(t, err)
	assert.NotNil(t, proc)
}

func TestNewEventDrop_MissingCondition(t *testing.T) {
	config := map[string]interface{}{}
	proc, err := newEventDrop(config)
	assert.Error(t, err)
	assert.Nil(t, proc)
	assert.Contains(t, err.Error(), "condition is required")
}

func TestNewEventDrop_InvalidCondition(t *testing.T) {
	config := map[string]interface{}{
		"condition": `invalid expression!!!`,
	}
	proc, err := newEventDrop(config)
	assert.Error(t, err)
	assert.Nil(t, proc)
	assert.Contains(t, err.Error(), "invalid condition")
}

// ---------------------------------------------------------------------------
// resolveField tests
// ---------------------------------------------------------------------------

func TestResolveField_NestedMapPath(t *testing.T) {
	data := map[string]interface{}{
		"Labels": map[string]string{"env": "prod"},
	}

	val, err := resolveField(".Labels.env", data)
	assert.NoError(t, err)
	assert.Equal(t, "prod", val)
}

func TestResolveField_TopLevelField(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
	}

	val, err := resolveField(".Severity", data)
	assert.NoError(t, err)
	assert.Equal(t, "critical", val)
}

func TestResolveField_MissingReturnsEmpty(t *testing.T) {
	data := map[string]interface{}{
		"Severity": "critical",
	}

	val, err := resolveField(".Missing", data)
	assert.NoError(t, err)
	assert.Equal(t, "", val)
}
