package engine

import (
	"testing"

	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/stretchr/testify/assert"
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
		name     string
		exp      string
		expected *triggerExpParts
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
			name:     "invalid",
			exp:      "invalid",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTriggerExp(tt.exp)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.ref, result.ref)
				assert.Equal(t, tt.expected.op, result.op)
				assert.Equal(t, tt.expected.threshold, result.threshold)
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
