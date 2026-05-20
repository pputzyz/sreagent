package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validatePromQLSyntax_valid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple gauge", `up == 0`},
		{"with function", `rate(http_requests_total[5m])`},
		{"with labels", `cpu_usage{env="prod", job="web"}`},
		{"nested parens", `sum(rate(http_requests_total{job="api"}[5m])) by (status)`},
		{"empty braces", `up{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePromQLSyntax(tt.expr)
			assert.NoError(t, err)
		})
	}
}

func Test_validatePromQLSyntax_invalid(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr string
	}{
		{"empty", "", "表达式为空"},
		{"unclosed paren", `rate(http_requests_total[5m]`, "缺少 ')'"},
		{"extra closing", `rate(http_requests_total[5m]))`, "多余的 ')'"},
		{"mismatched brace", `cpu_usage{env="prod"`, "缺少 '}'"},
		{"mismatched bracket", `rate(http_requests_total[5m)`, "括号不匹配"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePromQLSyntax(tt.expr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func Test_tokenizeExpression(t *testing.T) {
	tokens := tokenizeExpression(`rate(http_requests_total{job="api"}[5m])`)
	assert.True(t, tokens["http_requests_total"])
	assert.True(t, tokens["rate"])
	assert.True(t, tokens["job"])
	assert.True(t, tokens["api"])
	// Single-char tokens should be excluded
	assert.False(t, tokens["m"])
}

func Test_jaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]bool
		b    map[string]bool
		want float64
	}{
		{
			name: "identical",
			a:    map[string]bool{"a": true, "b": true},
			b:    map[string]bool{"a": true, "b": true},
			want: 1.0,
		},
		{
			name: "disjoint",
			a:    map[string]bool{"a": true},
			b:    map[string]bool{"b": true},
			want: 0.0,
		},
		{
			name: "partial overlap",
			a:    map[string]bool{"a": true, "b": true, "c": true},
			b:    map[string]bool{"b": true, "c": true, "d": true},
			want: 0.5, // 2 intersect / 4 union
		},
		{
			name: "empty a",
			a:    map[string]bool{},
			b:    map[string]bool{"a": true},
			want: 0.0,
		},
		{
			name: "empty b",
			a:    map[string]bool{"a": true},
			b:    map[string]bool{},
			want: 0.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccardSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func Test_extractMetricNames(t *testing.T) {
	metrics := extractMetricNames(`rate(http_requests_total{job="api"}[5m])`)
	assert.Contains(t, metrics, "http_requests_total")
	assert.NotContains(t, metrics, "rate") // PromQL function keyword
}

func Test_extractKeywords(t *testing.T) {
	kw := extractKeywords("请帮我生成一个 CPU 使用率过高的告警规则")
	assert.Contains(t, kw, "cpu")
	assert.NotContains(t, kw, "请")
	assert.NotContains(t, kw, "帮我")
	assert.NotContains(t, kw, "生成")
	assert.NotContains(t, kw, "规则")
}

func Test_postProcessResult_severity_normalize(t *testing.T) {
	s := &RuleGeneratorService{}
	r := &RuleGenerateResult{
		Type:     "alert",
		Severity: "CRITICAL",
	}
	s.postProcessResult(r)
	assert.Equal(t, "warning", r.Severity)
	assert.Len(t, r.Warnings, 1)
}

func Test_postProcessResult_defaults(t *testing.T) {
	s := &RuleGeneratorService{}
	r := &RuleGenerateResult{
		Type: "alert",
	}
	s.postProcessResult(r)
	assert.Equal(t, "0s", r.ForDuration)
	assert.NotNil(t, r.Labels)
	assert.NotNil(t, r.Annotations)
}

func Test_postProcessResult_clamp_confidence(t *testing.T) {
	s := &RuleGeneratorService{}

	r1 := &RuleGenerateResult{Confidence: 1.5}
	s.postProcessResult(r1)
	assert.Equal(t, 1.0, r1.Confidence)

	r2 := &RuleGenerateResult{Confidence: -0.5}
	s.postProcessResult(r2)
	assert.Equal(t, 0.0, r2.Confidence)
}
