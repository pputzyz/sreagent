package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
)

// JoinType defines the type of join operation for multi-query results.
type JoinType string

const (
	JoinTypeInner JoinType = "inner_join"
	JoinTypeLeft  JoinType = "left_join"
	JoinTypeRight JoinType = "right_join"
	JoinTypeNone  JoinType = "none"
)

// queryResults holds the results for a single query reference (A, B, C...).
type queryResults struct {
	Ref     string
	Results []datasource.QueryResult
}

// executeMultiQuery evaluates all queries in a multi-query rule and returns
// the joined results based on the configured join type and keys.
func (re *RuleEvaluator) executeMultiQuery(ctx context.Context) ([]datasource.QueryResult, error) {
	queries := re.rule.Queries
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries defined for multi-query rule")
	}

	// Execute each query independently
	allResults := make([]queryResults, 0, len(queries))
	for _, q := range queries {
		results, err := re.executeQueryByRef(ctx, q)
		if err != nil {
			re.logger.Warn("multi-query: query failed",
				zap.String("ref", q.Ref),
				zap.Error(err),
			)
			// Continue with other queries — a single query failure shouldn't stop the rule
			allResults = append(allResults, queryResults{Ref: q.Ref, Results: nil})
			continue
		}
		allResults = append(allResults, queryResults{Ref: q.Ref, Results: results})
	}

	// Apply join logic
	joinType := JoinType(re.rule.JoinType)
	if joinType == "" {
		joinType = JoinTypeNone
	}

	joinedResults, err := joinQueryResults(allResults, joinType, re.rule.JoinKeys)
	if err != nil {
		return nil, fmt.Errorf("join failed: %w", err)
	}

	return joinedResults, nil
}

// executeQueryByRef executes a single query from the multi-query definition.
func (re *RuleEvaluator) executeQueryByRef(ctx context.Context, q model.RuleQuery) ([]datasource.QueryResult, error) {
	// Resolve datasource — use query-specific DS if set, otherwise fall back to rule's DS
	ds := re.datasource
	if q.DatasourceID > 0 && q.DatasourceID != re.datasource.ID {
		var queryDS model.DataSource
		if err := re.db.WithContext(ctx).First(&queryDS, q.DatasourceID).Error; err != nil {
			return nil, fmt.Errorf("multi-query %s: failed to resolve datasource %d: %w", q.Ref, q.DatasourceID, err)
		}
		ds = &queryDS
		re.logger.Debug("multi-query: resolved query-specific datasource",
			zap.String("ref", q.Ref),
			zap.Uint("query_ds_id", q.DatasourceID),
			zap.String("ds_type", string(ds.Type)),
		)
	}

	ep := ds.Endpoint
	at := ds.AuthType
	ac := ds.AuthConfig

	switch ds.Type {
	case "zabbix":
		return datasource.ZabbixInstantQuery(ctx, ep, at, ac, q.Expr)
	case "victorialogs":
		lookback := time.Duration(re.rule.EvalInterval) * time.Second
		return datasource.VictoriaLogsInstantQuery(ctx, ep, at, ac, q.Expr, lookback)
	default:
		return re.queryClient.InstantQuery(ctx, ep, at, ac, q.Expr, time.Time{})
	}
}

// joinQueryResults combines results from multiple queries based on the join type.
// The join operates on label key sets — matching label combinations are merged.
func joinQueryResults(allResults []queryResults, joinType JoinType, joinKeys []string) ([]datasource.QueryResult, error) {
	if len(allResults) == 0 {
		return nil, nil
	}

	// For "none" join or single query, return all results concatenated
	if joinType == JoinTypeNone || len(allResults) == 1 {
		var combined []datasource.QueryResult
		for _, qr := range allResults {
			combined = append(combined, qr.Results...)
		}
		return combined, nil
	}

	// For joins, we need at least 2 queries
	if len(allResults) < 2 {
		var combined []datasource.QueryResult
		for _, qr := range allResults {
			combined = append(combined, qr.Results...)
		}
		return combined, nil
	}

	// Use the first query as "A" and the second as "B" for binary join
	// (Nightingale also primarily supports binary joins)
	aResults := allResults[0]
	bResults := allResults[1]

	switch joinType {
	case JoinTypeInner:
		return innerJoin(aResults.Results, bResults.Results, joinKeys), nil
	case JoinTypeLeft:
		return leftJoin(aResults.Results, bResults.Results, joinKeys), nil
	case JoinTypeRight:
		return leftJoin(bResults.Results, aResults.Results, joinKeys), nil
	default:
		return nil, fmt.Errorf("unsupported join type: %s", joinType)
	}
}

// innerJoin returns results where label key sets match in both A and B.
func innerJoin(aResults, bResults []datasource.QueryResult, joinKeys []string) []datasource.QueryResult {
	bIndex := indexResultsByKeys(bResults, joinKeys)
	var joined []datasource.QueryResult

	for _, a := range aResults {
		key := extractKeyFromLabels(a.Labels, joinKeys)
		if bMatches, ok := bIndex[key]; ok {
			for _, b := range bMatches {
				merged := mergeResults(a, b, "A", "B")
				joined = append(joined, merged)
			}
		}
	}

	return joined
}

// leftJoin returns all results from A, with matching results from B merged where available.
func leftJoin(aResults, bResults []datasource.QueryResult, joinKeys []string) []datasource.QueryResult {
	bIndex := indexResultsByKeys(bResults, joinKeys)
	var joined []datasource.QueryResult

	for _, a := range aResults {
		key := extractKeyFromLabels(a.Labels, joinKeys)
		if bMatches, ok := bIndex[key]; ok {
			for _, b := range bMatches {
				merged := mergeResults(a, b, "A", "B")
				joined = append(joined, merged)
			}
		} else {
			// No match in B — include A with null B values
			joined = append(joined, a)
		}
	}

	return joined
}

// indexResultsByKeys creates a map from join key values to results.
func indexResultsByKeys(results []datasource.QueryResult, joinKeys []string) map[string][]datasource.QueryResult {
	index := make(map[string][]datasource.QueryResult)
	for _, r := range results {
		key := extractKeyFromLabels(r.Labels, joinKeys)
		index[key] = append(index[key], r)
	}
	return index
}

// extractKeyFromLabels builds a composite key from the specified label keys.
func extractKeyFromLabels(labels map[string]string, joinKeys []string) string {
	if len(joinKeys) == 0 {
		// If no join keys specified, use all labels (sorted)
		return labelsToSortedKey(labels)
	}

	parts := make([]string, 0, len(joinKeys))
	for _, k := range joinKeys {
		v := labels[k]
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// labelsToSortedKey creates a deterministic key from all labels.
func labelsToSortedKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}

// mergeResults combines two query results, prefixing labels with query references.
func mergeResults(a, b datasource.QueryResult, aRef, bRef string) datasource.QueryResult {
	merged := datasource.QueryResult{
		Labels: make(map[string]string),
		Values: a.Values, // Use A's values as primary
	}

	// Merge labels from A with prefix
	for k, v := range a.Labels {
		merged.Labels[aRef+"_"+k] = v
	}

	// Merge labels from B with prefix
	for k, v := range b.Labels {
		merged.Labels[bRef+"_"+k] = v
	}

	// Keep original labels for compatibility
	for k, v := range a.Labels {
		if _, exists := merged.Labels[k]; !exists {
			merged.Labels[k] = v
		}
	}

	return merged
}

// evaluateTriggerExp evaluates the trigger expression against joined results.
// The trigger expression can reference $A, $B, etc. to access query results.
// For now, this implements a simple threshold-based evaluation.
func (re *RuleEvaluator) evaluateTriggerExp(results []datasource.QueryResult) []datasource.QueryResult {
	triggerExp := re.rule.TriggerExp
	if triggerExp == "" {
		// No trigger expression — return all results (all are "firing")
		return results
	}

	// Simple threshold evaluation for v1
	// Format: "$A > 100", "$A < 50", "$A >= 80", etc.
	// Parse the expression
	parts := parseTriggerExp(triggerExp)
	if parts == nil {
		re.logger.Warn("invalid trigger expression, returning all results",
			zap.String("trigger_exp", triggerExp),
		)
		return results
	}

	var firing []datasource.QueryResult
	for _, r := range results {
		if len(r.Values) == 0 {
			continue
		}
		value := r.Values[len(r.Values)-1].Value
		if evaluateCondition(value, parts.op, parts.threshold) {
			firing = append(firing, r)
		}
	}

	return firing
}

// triggerExpParts holds parsed components of a trigger expression.
type triggerExpParts struct {
	ref       string  // A, B, C...
	op        string  // >, <, >=, <=, ==, !=
	threshold float64
}

// parseTriggerExp parses a simple trigger expression like "$A > 100".
func parseTriggerExp(exp string) *triggerExpParts {
	exp = strings.TrimSpace(exp)

	// Find the reference ($A, $B, etc.)
	refStart := strings.Index(exp, "$")
	if refStart < 0 {
		return nil
	}

	// Find the operator
	operators := []string{">=", "<=", "!=", "==", ">", "<"}
	var op string
	opIdx := -1
	for _, o := range operators {
		idx := strings.Index(exp, o)
		if idx >= 0 && (opIdx < 0 || idx < opIdx) {
			op = o
			opIdx = idx
		}
	}

	if opIdx < 0 {
		return nil
	}

	ref := strings.TrimSpace(exp[refStart+1 : opIdx])
	thresholdStr := strings.TrimSpace(exp[opIdx+len(op):])

	var threshold float64
	if _, err := fmt.Sscanf(thresholdStr, "%f", &threshold); err != nil {
		return nil
	}

	return &triggerExpParts{
		ref:       ref,
		op:        op,
		threshold: threshold,
	}
}

// evaluateCondition checks if a value satisfies the condition.
func evaluateCondition(value float64, op string, threshold float64) bool {
	switch op {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}
