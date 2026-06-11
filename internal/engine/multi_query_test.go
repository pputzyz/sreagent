package engine

import (
	"testing"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInnerJoin(t *testing.T) {
	aResults := []datasource.QueryResult{
		{
			Labels: map[string]string{"host": "server1", "env": "prod"},
			Values: []datasource.DataPoint{{Value: 100}},
		},
		{
			Labels: map[string]string{"host": "server2", "env": "prod"},
			Values: []datasource.DataPoint{{Value: 200}},
		},
	}

	bResults := []datasource.QueryResult{
		{
			Labels: map[string]string{"host": "server1", "env": "prod"},
			Values: []datasource.DataPoint{{Value: 50}},
		},
		{
			Labels: map[string]string{"host": "server3", "env": "dev"},
			Values: []datasource.DataPoint{{Value: 75}},
		},
	}

	joinKeys := []string{"host", "env"}
	joined := innerJoin(aResults, bResults, joinKeys)

	assert.Len(t, joined, 1)
	assert.Equal(t, float64(100), joined[0].Values[0].Value)
}

func TestLeftJoin(t *testing.T) {
	aResults := []datasource.QueryResult{
		{
			Labels: map[string]string{"host": "server1"},
			Values: []datasource.DataPoint{{Value: 100}},
		},
		{
			Labels: map[string]string{"host": "server2"},
			Values: []datasource.DataPoint{{Value: 200}},
		},
	}

	bResults := []datasource.QueryResult{
		{
			Labels: map[string]string{"host": "server1"},
			Values: []datasource.DataPoint{{Value: 50}},
		},
	}

	joinKeys := []string{"host"}
	joined := leftJoin(aResults, bResults, joinKeys)

	assert.Len(t, joined, 2)
}

func TestJoinQueryResults_None(t *testing.T) {
	allResults := []queryResults{
		{
			Ref: "A",
			Results: []datasource.QueryResult{
				{Labels: map[string]string{"host": "server1"}, Values: []datasource.DataPoint{{Value: 100}}},
			},
		},
		{
			Ref: "B",
			Results: []datasource.QueryResult{
				{Labels: map[string]string{"host": "server2"}, Values: []datasource.DataPoint{{Value: 200}}},
			},
		},
	}

	joined, err := joinQueryResults(allResults, JoinTypeNone, nil)
	assert.NoError(t, err)
	assert.Len(t, joined, 2)
}

func TestParseTriggerExp(t *testing.T) {
	tests := []struct {
		name      string
		exp       string
		expectErr bool
		expected  *triggerExpParts
	}{
		{
			name:     "greater than",
			exp:      "$A > 100",
			expected: &triggerExpParts{ref: "A", op: ">", threshold: 100},
		},
		{
			name:     "less than",
			exp:      "$B < 50",
			expected: &triggerExpParts{ref: "B", op: "<", threshold: 50},
		},
		{
			name:     "greater or equal",
			exp:      "$A >= 80.5",
			expected: &triggerExpParts{ref: "A", op: ">=", threshold: 80.5},
		},
		{
			name:     "var-to-var greater than",
			exp:      "$A > $B",
			expected: &triggerExpParts{ref: "A", op: ">", isVarRef: true, rightRef: "B"},
		},
		{
			name:     "var-to-var less or equal",
			exp:      "$A <= $C",
			expected: &triggerExpParts{ref: "A", op: "<=", isVarRef: true, rightRef: "C"},
		},
		{
			name:      "invalid expression",
			exp:       "invalid",
			expectErr: true,
		},
		{
			name:      "empty expression",
			exp:       "",
			expectErr: true,
		},
		{
			name:      "no operator",
			exp:       "$A 100",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTriggerExp(tt.exp)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.ref, result.ref)
				assert.Equal(t, tt.expected.op, result.op)
				assert.Equal(t, tt.expected.threshold, result.threshold)
				assert.Equal(t, tt.expected.isVarRef, result.isVarRef)
				assert.Equal(t, tt.expected.rightRef, result.rightRef)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		op        string
		threshold float64
		expected  bool
	}{
		{"greater than true", 150, ">", 100, true},
		{"greater than false", 50, ">", 100, false},
		{"less than true", 50, "<", 100, true},
		{"less than false", 150, "<", 100, false},
		{"equal", 100, "==", 100, true},
		{"not equal", 100, "!=", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateCondition(tt.value, tt.op, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractKeyFromLabels(t *testing.T) {
	labels := map[string]string{
		"host": "server1",
		"env":  "prod",
		"app":  "web",
	}

	// With specific keys
	key := extractKeyFromLabels(labels, []string{"host", "env"})
	assert.Contains(t, key, "host=server1")
	assert.Contains(t, key, "env=prod")

	// With all labels
	key = extractKeyFromLabels(labels, nil)
	assert.Contains(t, key, "host=server1")
	assert.Contains(t, key, "env=prod")
	assert.Contains(t, key, "app=web")
}

func TestMergeResults_stores_B_value_label(t *testing.T) {
	a := datasource.QueryResult{
		Labels: map[string]string{"host": "server1"},
		Values: []datasource.DataPoint{{Value: 100}},
	}
	b := datasource.QueryResult{
		Labels: map[string]string{"host": "server1"},
		Values: []datasource.DataPoint{{Value: 50}},
	}

	merged := mergeResults(a, b, "A", "B")

	// B's value should be stored as a synthetic label
	bVal, ok := merged.Labels["__B_value__"]
	assert.True(t, ok, "merged result should contain __B_value__ label")
	assert.Equal(t, "50", bVal)

	// A's value should still be primary
	assert.Equal(t, float64(100), merged.Values[0].Value)

	// Prefixed labels should exist
	assert.Equal(t, "server1", merged.Labels["A_host"])
	assert.Equal(t, "server1", merged.Labels["B_host"])
}

// Test_evaluateTriggerExp_fail_closed verifies that an invalid trigger expression
// returns nil + error (fail-closed), never returning all results. This prevents
// an alert storm caused by a malformed expression letting every series through.
func Test_evaluateTriggerExp_fail_closed(t *testing.T) {
	// Build a minimal RuleEvaluator with an invalid TriggerExp.
	// evaluateTriggerExp only reads re.rule.TriggerExp — no DB/datasource needed.
	re := &RuleEvaluator{
		rule: &model.AlertRule{
			TriggerExp: "totally invalid expression",
		},
		logger: zap.NewNop(),
	}

	results, err := re.evaluateTriggerExp([]datasource.QueryResult{
		{Labels: map[string]string{"host": "web-1"}, Values: []datasource.DataPoint{{Value: 42}}},
	})

	assert.Error(t, err, "invalid trigger expression must return error")
	assert.Nil(t, results, "fail-closed: nil results on parse failure, not all results")
	assert.Contains(t, err.Error(), "invalid trigger expression", "error should mention invalid expression")
}

// Test_parseTriggerExp_var_to_var_standalone is a focused regression test for the
// $A op $B pattern, ensuring isVarRef and rightRef are set correctly.
func Test_parseTriggerExp_var_to_var_standalone(t *testing.T) {
	tests := []struct {
		name     string
		exp      string
		ref      string
		op       string
		rightRef string
	}{
		{"$A > $B", "$A > $B", "A", ">", "B"},
		{"$A < $B", "$A < $B", "A", "<", "B"},
		{"$A >= $B", "$A >= $B", "A", ">=", "B"},
		{"$A <= $B", "$A <= $B", "A", "<=", "B"},
		{"$A == $B", "$A == $B", "A", "==", "B"},
		{"$A != $B", "$A != $B", "A", "!=", "B"},
		{"$X > $Y", "$X > $Y", "X", ">", "Y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTriggerExp(tt.exp)
			require.NoError(t, err, "parseTriggerExp should succeed for %q", tt.exp)
			require.NotNil(t, result)
			assert.Equal(t, tt.ref, result.ref)
			assert.Equal(t, tt.op, result.op)
			assert.True(t, result.isVarRef, "isVarRef must be true for var-to-var")
			assert.Equal(t, tt.rightRef, result.rightRef)
			assert.Equal(t, float64(0), result.threshold, "threshold should be zero for var-to-var")
		})
	}
}

func TestExpandVarInExpr(t *testing.T) {
	re := &RuleEvaluator{}

	tests := []struct {
		name       string
		expr       string
		paramNames []string
		varValues  map[string][]string
		expected   []string
	}{
		{
			name:       "no vars in expression",
			expr:       "cpu_usage > 90",
			paramNames: []string{"host"},
			varValues:  map[string][]string{"host": {"a", "b"}},
			expected:   []string{"cpu_usage > 90"},
		},
		{
			name:       "single var substitution",
			expr:       `cpu_usage{host="$host"} > 90`,
			paramNames: []string{"host"},
			varValues:  map[string][]string{"host": {"web01", "web02"}},
			expected:   []string{`cpu_usage{host="web01"} > 90`, `cpu_usage{host="web02"} > 90`},
		},
		{
			name:       "two vars cartesian product",
			expr:       `cpu{host="$host",env="$env"} > $val`,
			paramNames: []string{"env", "host", "val"},
			varValues:  map[string][]string{"host": {"a"}, "env": {"p"}, "val": {"90"}},
			expected:   []string{`cpu{host="a",env="p"} > 90`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := re.expandVarInExpr(tt.expr, tt.paramNames, tt.varValues)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
