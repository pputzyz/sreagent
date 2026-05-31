package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_RuleGenCache_Get_miss_returns_nil verifies that cache miss returns nil.
func Test_RuleGenCache_Get_miss_returns_nil(t *testing.T) {
	cache := NewRuleGenCache(10 * time.Minute)
	t.Cleanup(cache.Stop)
	result := cache.Get("nonexistent description", nil, "alert")
	assert.Nil(t, result, "cache miss should return nil")
}

// Test_RuleGenCache_Set_then_Get_hit_returns_result verifies basic cache
// set/get cycle works correctly.
func Test_RuleGenCache_Set_then_Get_hit_returns_result(t *testing.T) {
	cache := NewRuleGenCache(10 * time.Minute)
	t.Cleanup(cache.Stop)

	dsID := uint(42)
	original := &RuleGenerateResult{
		Name:       "HighCPU",
		Expression: "cpu_usage > 90",
		Severity:   "warning",
		Confidence: 0.85,
	}

	cache.Set("high cpu usage", &dsID, "alert", original)

	cached := cache.Get("high cpu usage", &dsID, "alert")
	require.NotNil(t, cached, "cache hit should return the stored result")
	assert.Equal(t, "HighCPU", cached.Name)
	assert.Equal(t, "cpu_usage > 90", cached.Expression)
	assert.Equal(t, 0.85, cached.Confidence)
}

// Test_RuleGenCache_expired_entry_returns_nil verifies that expired entries
// are treated as cache misses.
func Test_RuleGenCache_expired_entry_returns_nil(t *testing.T) {
	// Use a very short TTL
	cache := NewRuleGenCache(50 * time.Millisecond)
	t.Cleanup(cache.Stop)

	result := &RuleGenerateResult{
		Name:       "TestRule",
		Expression: "up == 0",
	}

	cache.Set("test", nil, "alert", result)

	// Should hit immediately
	cached := cache.Get("test", nil, "alert")
	assert.NotNil(t, cached, "should hit before expiry")

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)
	cached = cache.Get("test", nil, "alert")
	assert.Nil(t, cached, "expired entry should return nil")
}

// Test_RuleGenCache_different_keys_miss verifies that different descriptions
// produce different cache keys and don't collide.
func Test_RuleGenCache_different_keys_miss(t *testing.T) {
	cache := NewRuleGenCache(10 * time.Minute)
	t.Cleanup(cache.Stop)

	cache.Set("disk full", nil, "alert", &RuleGenerateResult{Name: "DiskFull"})
	cached := cache.Get("cpu high", nil, "alert")
	assert.Nil(t, cached, "different descriptions should not collide")
}

// Test_RuleGenCache_different_dsID_miss verifies that the same description
// with different datasource IDs produces different cache entries.
func Test_RuleGenCache_different_dsID_miss(t *testing.T) {
	cache := NewRuleGenCache(10 * time.Minute)
	t.Cleanup(cache.Stop)

	ds1 := uint(1)
	ds2 := uint(2)

	cache.Set("test rule", &ds1, "alert", &RuleGenerateResult{Name: "Rule1"})
	cached := cache.Get("test rule", &ds2, "alert")
	assert.Nil(t, cached, "different datasource IDs should not share cache")
}

// Test_DryRunResult_structure_fields verifies that DryRunResult struct
// has the expected fields and can be constructed properly.
func Test_DryRunResult_structure_fields(t *testing.T) {
	dr := DryRunResult{
		Rule: &RuleGenerateResult{
			Name:       "TestAlert",
			Expression: "up == 0",
			Severity:   "critical",
			Confidence: 0.9,
			Labels:     map[string]string{"severity": "critical"},
		},
		Validation: &ValidationResult{
			Valid:       true,
			SampleCount: 5,
		},
		SeriesCount:  5,
		WouldFire:    true,
		SampleSeries: []map[string]string{{"instance": "web-1"}},
	}

	assert.Equal(t, "TestAlert", dr.Rule.Name)
	assert.Equal(t, "up == 0", dr.Rule.Expression)
	assert.True(t, dr.Validation.Valid)
	assert.Equal(t, 5, dr.SeriesCount)
	assert.True(t, dr.WouldFire)
	assert.Len(t, dr.SampleSeries, 1)
}

// Test_RuleGenerateResult_inhibition_fields verifies that inhibition-type
// results can be constructed with the appropriate fields.
func Test_RuleGenerateResult_inhibition_fields(t *testing.T) {
	result := RuleGenerateResult{
		Type:         "inhibition",
		Name:         "InhibitWhenDatabaseDown",
		Description:  "Inhibit backend alerts when database is down",
		SourceLabels: []string{"alertname"},
		SourceValue:  "DatabaseDown",
		TargetLabels: []string{"team"},
		EqualLabels:  []string{"env"},
		Confidence:   0.75,
	}

	assert.Equal(t, "inhibition", result.Type)
	assert.Equal(t, "DatabaseDown", result.SourceValue)
	assert.Contains(t, result.SourceLabels, "alertname")
	assert.Contains(t, result.EqualLabels, "env")
}
