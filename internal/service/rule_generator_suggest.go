package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// LabelSuggestionResult is AI-suggested labels for an expression.
type LabelSuggestionResult struct {
	DetectedMetrics    map[string]string     `json:"detected_metrics"`
	SuggestedLabels    map[string]LabelValue `json:"suggested_labels"`
	AvailableInstances []InstanceInfo        `json:"available_instances"`
}

// LabelValue holds a suggested label value with confidence.
type LabelValue struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
}

// InstanceInfo holds information about a metric instance.
type InstanceInfo struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

// SuggestLabels suggests labels for an expression using LLM + label registry data.
func (s *RuleGeneratorService) SuggestLabels(ctx context.Context, datasourceID uint, expression string) (*LabelSuggestionResult, error) {
	result := &LabelSuggestionResult{
		DetectedMetrics: make(map[string]string),
		SuggestedLabels: make(map[string]LabelValue),
	}

	// Extract metric names from expression
	metrics := extractMetricNames(expression)
	for _, m := range metrics {
		result.DetectedMetrics[m] = m
	}

	// Get label keys + values from registry
	dsIDs := []uint{datasourceID}
	keys, err := s.labelRegSvc.GetKeys(ctx, dsIDs)
	if err != nil || len(keys) == 0 {
		return result, nil
	}

	// Build label registry context for LLM (key → top 5 values)
	var sb strings.Builder
	sb.WriteString("Label Registry (key → common values):\n")
	limit := 80
	if len(keys) < limit {
		limit = len(keys)
	}
	for i := 0; i < limit; i++ {
		vals, vErr := s.labelRegSvc.GetValues(ctx, keys[i], dsIDs)
		if vErr == nil && len(vals) > 0 {
			vl := 5
			if len(vals) < vl {
				vl = len(vals)
			}
			sb.WriteString(fmt.Sprintf("  %s: %s\n", keys[i], strings.Join(vals[:vl], ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("  %s: (no values)\n", keys[i]))
		}
	}

	// Check AI availability — fall back to heuristic if disabled
	if err := s.checkAIEnabled(ctx); err != nil {
		return s.suggestLabelsHeuristic(result, keys, dsIDs), nil
	}

	// Call LLM to dynamically recommend labels
	systemPrompt := fmt.Sprintf(`你是 SRE 标签推荐助手。根据 PromQL 表达式和可用标签注册表数据，推荐最相关的标签及其值。

输出格式（严格 JSON）：
{
  "suggested_labels": {
    "label_key": {"value": "recommended_value", "confidence": 0.9, "reason": "推荐原因"}
  }
}

规则：
- 只推荐与表达式语义相关的标签（如 env, service, instance, cluster 等）
- value 必须从标签注册表的常用值中选取
- confidence 0.0-1.0，越相关越高
- 最多推荐 8 个标签
- 回复中只包含 JSON

%s`, sb.String())

	var llmResult struct {
		SuggestedLabels map[string]struct {
			Value      string  `json:"value"`
			Confidence float64 `json:"confidence"`
			Reason     string  `json:"reason"`
		} `json:"suggested_labels"`
	}
	if err := s.aiSvc.callLLMJSON(ctx, s.mustLoadConfig(ctx), systemPrompt, "PromQL: "+expression, &llmResult); err != nil {
		s.logger.Warn("LLM label suggestion failed, falling back to heuristic", zap.Error(err))
		return s.suggestLabelsHeuristic(result, keys, dsIDs), nil
	}

	for k, v := range llmResult.SuggestedLabels {
		result.SuggestedLabels[k] = LabelValue{
			Value:      v.Value,
			Confidence: v.Confidence,
			Source:     "llm",
		}
	}

	return result, nil
}

// suggestLabelsHeuristic provides a fallback when LLM is unavailable.
func (s *RuleGeneratorService) suggestLabelsHeuristic(result *LabelSuggestionResult, keys []string, dsIDs []uint) *LabelSuggestionResult {
	commonLabels := []string{"instance", "job", "env", "service", "namespace", "cluster", "pod", "container"}
	for _, key := range keys {
		for _, cl := range commonLabels {
			if key == cl {
				vals, err := s.labelRegSvc.GetValues(context.Background(), key, dsIDs)
				if err == nil && len(vals) > 0 {
					result.SuggestedLabels[key] = LabelValue{
						Value:      vals[0],
						Confidence: 0.8,
						Source:     "label_registry",
					}
				}
				break
			}
		}
	}
	return result
}
