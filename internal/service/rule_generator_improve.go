package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/promql/parser"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// ImproveRuleRequest is the input for rule improvement.
type ImproveRuleRequest struct {
	Rule         RuleGenerateResult `json:"rule" binding:"required"`
	Feedback     string             `json:"feedback" binding:"required"`
	DatasourceID *uint              `json:"datasource_id"`
}

// ConflictCheck holds the result of conflict detection for a rule.
type ConflictCheck struct {
	HasConflict  bool     `json:"has_conflict"`
	SimilarRules []string `json:"similar_rules,omitempty"` // names of similar existing rules
	SyntaxValid  bool     `json:"syntax_valid"`
	SyntaxError  string   `json:"syntax_error,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// CheckConflicts detects potential conflicts for an alert rule expression.
func (s *RuleGeneratorService) CheckConflicts(ctx context.Context, expression string, datasourceID *uint) *ConflictCheck {
	check := &ConflictCheck{SyntaxValid: true}

	// 1. Basic PromQL syntax validation
	if err := validatePromQLSyntax(expression); err != nil {
		check.SyntaxValid = false
		check.SyntaxError = err.Error()
		check.HasConflict = true
		check.Warnings = append(check.Warnings, "表达式语法异常: "+err.Error())
	}

	// 2. Jaccard similarity — find existing rules with overlapping expressions
	rules, _, err := s.ruleSvc.List(ctx, "", "", "", "", "", nil, 1, 200)
	if err == nil {
		newTokens := tokenizeExpression(expression)
		for _, r := range rules {
			if r.Expression == "" {
				continue
			}
			existingTokens := tokenizeExpression(r.Expression)
			jaccard := jaccardSimilarity(newTokens, existingTokens)
			if jaccard >= 0.7 {
				check.SimilarRules = append(check.SimilarRules, r.Name)
				check.HasConflict = true
			}
		}
		if len(check.SimilarRules) > 0 {
			check.Warnings = append(check.Warnings, fmt.Sprintf("发现 %d 条相似规则，建议检查是否重复", len(check.SimilarRules)))
		}
	}

	return check
}

// ImproveRule takes an existing AI-generated rule and user feedback, returns an improved version.
// Pre-LLM: validates input expression. Post-LLM: validates output expression and detects conflicts.
func (s *RuleGeneratorService) ImproveRule(ctx context.Context, req *ImproveRuleRequest) (*RuleGenerateResult, error) {
	if err := s.checkAIEnabled(ctx); err != nil {
		return nil, err
	}

	// Pre-LLM: validate input expression syntax
	if req.Rule.Type == "alert" && req.Rule.Expression != "" {
		if err := validatePromQLSyntax(req.Rule.Expression); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("输入表达式语法错误: %v", err))
		}
	}

	ruleJSON, _ := json.Marshal(req.Rule)

	systemPrompt := `你是 SRE 告警规则优化助手。用户会给你一条已有的告警规则和改进反馈，请根据反馈优化规则。

输出格式与输入相同（严格 JSON），保持原有字段不变，只修改反馈中提到的问题。
- 如果反馈提到表达式问题，修正 expression
- 如果反馈提到严重等级，调整 severity
- 如果反馈提到标签，修改 labels
- 如果反馈提到持续时间，调整 for_duration
- 在 warnings 中说明做了哪些修改`

	userPrompt := fmt.Sprintf("当前规则:\n%s\n\n改进反馈: %s", string(ruleJSON), req.Feedback)

	var result RuleGenerateResult
	if err := s.aiSvc.callLLMJSON(ctx, s.mustLoadConfig(ctx), systemPrompt, userPrompt, &result); err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, fmt.Sprintf("AI rule improvement failed: %v", err))
	}

	result.Type = req.Rule.Type
	s.postProcessResult(&result)

	// Post-LLM: validate output expression syntax + conflict detection
	if result.Type == "alert" && result.Expression != "" {
		if err := validatePromQLSyntax(result.Expression); err != nil {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("AI 生成的表达式语法错误: %v", err))
		}

		conflict := s.CheckConflicts(ctx, result.Expression, req.DatasourceID)
		for _, w := range conflict.Warnings {
			result.Warnings = append(result.Warnings, "冲突检测: "+w)
		}
	}

	return &result, nil
}

// validatePromQLSyntax performs real PromQL syntax validation using the Prometheus parser.
func validatePromQLSyntax(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return fmt.Errorf("expression is empty")
	}
	_, err := parser.ParseExpr(expr)
	if err != nil {
		return fmt.Errorf("PromQL syntax error: %w", err)
	}
	return nil
}

// tokenizeExpression splits a PromQL expression into unique tokens for similarity comparison.
// PromQL function keywords are excluded so Jaccard similarity reflects actual metric overlap.
func tokenizeExpression(expr string) map[string]bool {
	tokens := make(map[string]bool)
	for _, tok := range metricRegexp.FindAllString(expr, -1) {
		tok = strings.ToLower(tok)
		if len(tok) > 1 && !promqlKeywords[tok] {
			tokens[tok] = true
		}
	}
	return tokens
}

// promqlKeywords is the set of PromQL function/keyword names to exclude from tokenization.
var promqlKeywords = map[string]bool{
	"sum": true, "avg": true, "min": true, "max": true, "count": true,
	"rate": true, "irate": true, "increase": true, "delta": true,
	"by": true, "without": true, "on": true, "ignoring": true,
	"group_left": true, "group_right": true, "bool": true,
	"topk": true, "bottomk": true, "sort": true, "sort_desc": true,
	"abs": true, "ceil": true, "floor": true, "round": true,
	"clamp_min": true, "clamp_max": true, "label_replace": true,
	"label_join": true, "absent": true, "absent_over_time": true,
	"vector": true, "scalar": true, "time": true, "timestamp": true,
	"histogram_quantile": true,
}

// jaccardSimilarity computes the Jaccard similarity coefficient between two token sets.
func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
