package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ValidationResult is the result of validating a PromQL expression.
type ValidationResult struct {
	Valid        bool     `json:"valid"`
	ResultType   string   `json:"result_type,omitempty"`
	SampleCount  int      `json:"sample_count,omitempty"`
	SampleLabels []string `json:"sample_labels,omitempty"`
	Error        string   `json:"error,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// DryRunResult combines a generated rule with its validation result.
type DryRunResult struct {
	Rule           *RuleGenerateResult `json:"rule"`
	Validation     *ValidationResult   `json:"validation,omitempty"`
	SeriesCount    int                 `json:"series_count"`
	SampleSeries   []map[string]string `json:"sample_series,omitempty"` // max 5
	WouldFire      bool                `json:"would_fire"`
	EvalDurationMs int64               `json:"eval_duration_ms"`
}

// DryRun generates a rule and validates its PromQL expression in one call.
// This lets the frontend preview the rule and test the expression before saving.
func (s *RuleGeneratorService) DryRun(ctx context.Context, req *RuleGenerateRequest) (*DryRunResult, error) {
	start := time.Now()

	result, err := s.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	dr := &DryRunResult{Rule: result}

	// Validate expression if datasource and expression are available
	if req.DatasourceID != nil && result.Expression != "" {
		vr, err := s.ValidateExpression(ctx, *req.DatasourceID, result.Expression)
		if err != nil {
			s.logger.Warn("dry-run validation failed", zap.Error(err))
		} else {
			dr.Validation = vr
			dr.SeriesCount = vr.SampleCount
			dr.WouldFire = vr.SampleCount > 0
			// Extract up to 5 sample label sets from the query response
			if vr.SampleCount > 0 {
				dr.SampleSeries = s.extractSampleSeries(ctx, *req.DatasourceID, result.Expression)
			}
		}
	}

	dr.EvalDurationMs = time.Since(start).Milliseconds()
	return dr, nil
}

// extractSampleSeries queries the datasource and returns up to 5 sample label sets.
func (s *RuleGeneratorService) extractSampleSeries(ctx context.Context, datasourceID uint, expression string) []map[string]string {
	resp, err := s.dsSvc.QueryDatasource(ctx, datasourceID, expression, time.Now())
	if err != nil {
		return nil
	}

	limit := 5
	if len(resp.Series) < limit {
		limit = len(resp.Series)
	}
	samples := make([]map[string]string, 0, limit)
	for i := 0; i < limit; i++ {
		samples = append(samples, resp.Series[i].Labels)
	}
	return samples
}

// ValidateExpression validates a PromQL expression against a datasource.
// First performs offline syntax parsing, then queries the datasource for runtime validation.
func (s *RuleGeneratorService) ValidateExpression(ctx context.Context, datasourceID uint, expression string) (*ValidationResult, error) {
	result := &ValidationResult{}

	// Step 1: Offline syntax validation (catches syntax errors without network dependency)
	if err := validatePromQLSyntax(expression); err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("syntax: %v", err)
		return result, nil
	}

	// Step 2: Query datasource for runtime validation (catches metric/label issues)
	resp, err := s.dsSvc.QueryDatasource(ctx, datasourceID, expression, time.Now())
	if err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("query: %v", err)
		return result, nil
	}

	result.Valid = true
	result.ResultType = resp.ResultType
	result.SampleCount = len(resp.Series)

	// Extract sample labels
	labelSet := make(map[string]bool)
	for _, series := range resp.Series {
		for k := range series.Labels {
			if !labelSet[k] {
				labelSet[k] = true
				result.SampleLabels = append(result.SampleLabels, k)
			}
		}
	}

	if result.SampleCount == 0 {
		result.Warnings = append(result.Warnings, "表达式返回 0 条时间序列，请检查指标名或标签条件是否正确")
	}

	return result, nil
}
