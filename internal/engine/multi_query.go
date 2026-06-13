package engine

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/crypto"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
)

// varVarRe matches trigger expressions of the form $A op $B (var-to-var comparison).
// Compiled once at package level to avoid recompilation on every parseTriggerExp call.
var varVarRe = regexp.MustCompile(`^\$(\w+)\s*(>=|<=|!=|==|>|<)\s*\$(\w+)$`)

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
// When VarConfig is set, $var placeholders are substituted in each query's
// expression before execution (before_query strategy).
func (re *RuleEvaluator) executeMultiQuery(ctx context.Context) ([]datasource.QueryResult, error) {
	queries := re.rule.Queries
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries defined for multi-query rule")
	}

	// NOTE: TriggerExp emptiness is checked by the caller (evaluate()) before
	// calling evaluateTriggerExp, so we do not duplicate that guard here.

	if len(queries) > 2 {
		re.logger.Warn("multi-query: more than 2 queries defined; only the first 2 are used for joins (N-way join not yet implemented)",
			zap.Int("query_count", len(queries)),
			zap.Uint("rule_id", re.rule.ID),
		)
	}

	// Pre-resolve VarConfig values once (shared across all queries)
	var varParamNames []string
	var varValues map[string][]string
	if re.rule.VarConfig != nil && len(re.rule.VarConfig.Params) > 0 {
		var err error
		varValues, err = re.resolveAllVarValues(ctx, re.rule.VarConfig.Params)
		if err != nil {
			return nil, fmt.Errorf("multi-query var filling: resolve values failed: %w", err)
		}
		for _, p := range re.rule.VarConfig.Params {
			varParamNames = append(varParamNames, p.Name)
		}
		// Sort for deterministic order
		sort.Strings(varParamNames)
	}

	// Execute each query independently
	allResults := make([]queryResults, 0, len(queries))
	var lastQueryErr error
	for _, q := range queries {
		// Apply VarConfig substitution if configured
		queryExprs := []string{q.Expr}
		if len(varParamNames) > 0 && varValues != nil {
			queryExprs = re.expandVarInExpr(q.Expr, varParamNames, varValues)
		}

		var queryAllResults []datasource.QueryResult
		for _, expr := range queryExprs {
			queryWithExpr := q
			queryWithExpr.Expr = expr
			results, err := re.executeQueryByRef(ctx, queryWithExpr)
			if err != nil {
				re.logger.Warn("multi-query: query failed",
					zap.String("ref", q.Ref),
					zap.String("expr", expr),
					zap.Error(err),
				)
				lastQueryErr = err
				continue
			}
			queryAllResults = append(queryAllResults, results...)
		}
		allResults = append(allResults, queryResults{Ref: q.Ref, Results: queryAllResults})
	}

	// If ALL queries returned no results and at least one failed, surface the error
	// instead of returning empty results (which would silently suppress the failure).
	totalResults := 0
	for _, qr := range allResults {
		totalResults += len(qr.Results)
	}
	if totalResults == 0 && lastQueryErr != nil {
		return nil, fmt.Errorf("multi-query: all queries failed, last error: %w", lastQueryErr)
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
	var ac string
	if ds.ID == re.datasource.ID {
		ac = re.decryptedAuthConfig
	} else {
		ac = ds.AuthConfig
		if crypto.IsEncrypted(ac) {
			if decrypted, err := crypto.DecryptString(ac); err == nil {
				ac = decrypted
			} else {
				re.logger.Warn("multi-query: failed to decrypt cross-datasource auth config, queries may fail",
					zap.String("ref", q.Ref),
					zap.Uint("datasource_id", q.DatasourceID),
					zap.Error(err),
				)
			}
		}
	}

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

// expandVarInExpr generates all $var substitution variants of an expression.
// For params ["host","env"] with values {"host":["a","b"], "env":["prod","staging"]},
// it returns 4 expressions with all combinations substituted.
// If no variables are present in the expression, returns the original as a single-element slice.
func (re *RuleEvaluator) expandVarInExpr(expr string, paramNames []string, varValues map[string][]string) []string {
	// Check if any $var is actually present in the expression
	hasVar := false
	for _, name := range paramNames {
		if strings.Contains(expr, fmt.Sprintf("$%s", name)) {
			hasVar = true
			break
		}
	}
	if !hasVar {
		return []string{expr}
	}

	combinations, err := buildCombinations(paramNames, varValues)
	if err != nil || len(combinations) == 0 {
		return []string{expr}
	}

	result := make([]string, 0, len(combinations))
	for _, combo := range combinations {
		substituted := expr
		for i, name := range paramNames {
			substituted = strings.ReplaceAll(substituted, fmt.Sprintf("$%s", name), combo[i])
		}
		result = append(result, substituted)
	}
	return result
}

// joinQueryResults combines results from multiple queries based on the join type.
// The join operates on label key sets — matching label combinations are merged.
// Currently only binary joins (2 queries) are supported; queries beyond the
// second are silently ignored with a warning logged by the caller.
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
		return rightJoin(aResults.Results, bResults.Results, joinKeys), nil
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

// rightJoin returns all results from B, with matching results from A merged where
// available. A is always passed as the first argument to mergeResults so that
// label prefixes (A_/B_) and the synthetic "__B_value__" stay consistent with the
// $A/$B trigger-expression semantics regardless of join direction. (Previously this
// reused leftJoin with swapped args, which inverted the A/B operands.)
func rightJoin(aResults, bResults []datasource.QueryResult, joinKeys []string) []datasource.QueryResult {
	aIndex := indexResultsByKeys(aResults, joinKeys)
	var joined []datasource.QueryResult

	for _, b := range bResults {
		key := extractKeyFromLabels(b.Labels, joinKeys)
		if aMatches, ok := aIndex[key]; ok {
			for _, a := range aMatches {
				joined = append(joined, mergeResults(a, b, "A", "B"))
			}
		} else {
			// No match in A — include B alone. A var-to-var trigger ($A op $B)
			// cannot evaluate without an A value, so it will skip this row.
			joined = append(joined, b)
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
// B's last value is stored as a synthetic label "__B_value__" so that var-to-var
// trigger expressions (e.g. $A > $B) can retrieve it during evaluation.
//
// Label collision policy (A-wins): when A and B share an original label key,
// the un-prefixed slot keeps A's value. This is intentional — A is the "primary"
// query in a binary join, and its labels drive the fingerprint and routing.
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

	// Store B's value as a synthetic label for var-to-var trigger expressions
	if len(b.Values) > 0 {
		bVal := b.Values[len(b.Values)-1].Value
		merged.Labels[fmt.Sprintf("__%s_value__", bRef)] = fmt.Sprintf("%g", bVal)
	}

	return merged
}

// evaluateTriggerExp evaluates the trigger expression against joined results.
// The trigger expression can reference $A, $B, etc. to access query results.
// Supports two forms:
//   - $A op number  (e.g. "$A > 100") — compare each result value against a threshold
//   - $A op $B      (e.g. "$A > $B")  — compare values from two different query results
//
// Returns an error if the expression cannot be parsed (fail-closed: never return
// all results on parse failure, which would cause an alert storm).
func (re *RuleEvaluator) evaluateTriggerExp(results []datasource.QueryResult) ([]datasource.QueryResult, error) {
	triggerExp := re.rule.TriggerExp
	if triggerExp == "" {
		// No trigger expression — return all results (all are "firing")
		return results, nil
	}

	// Parse the expression — fail closed on parse error
	parts, err := parseTriggerExp(triggerExp)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger expression %q: %w", triggerExp, err)
	}

	// Var-to-var comparison: $A op $B
	if parts.isVarRef {
		return re.evaluateVarToVar(results, parts)
	}

	// Threshold comparison: $A op number
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

	return firing, nil
}

// evaluateVarToVar handles trigger expressions that compare two variable references,
// e.g. "$A > $B". In merged results, B's value is stored as a synthetic label
// "__B_value__" by mergeResults during the join phase.
func (re *RuleEvaluator) evaluateVarToVar(results []datasource.QueryResult, parts *triggerExpParts) ([]datasource.QueryResult, error) {
	valueLabelKey := fmt.Sprintf("__%s_value__", parts.rightRef)

	var firing []datasource.QueryResult
	for _, r := range results {
		if len(r.Values) == 0 {
			continue
		}
		aVal := r.Values[len(r.Values)-1].Value

		// B's value was stored as a synthetic label during mergeResults
		bValStr, hasB := r.Labels[valueLabelKey]
		if !hasB {
			continue
		}

		var bVal float64
		if _, err := fmt.Sscanf(bValStr, "%f", &bVal); err != nil {
			re.logger.Warn("var-to-var: cannot parse B value",
				zap.String("label_key", valueLabelKey),
				zap.String("value", bValStr),
				zap.Error(err),
			)
			continue
		}

		if evaluateCondition(aVal, parts.op, bVal) {
			firing = append(firing, r)
		}
	}

	return firing, nil
}

// triggerExpParts holds parsed components of a trigger expression.
type triggerExpParts struct {
	ref       string // A, B, C...
	op        string // >, <, >=, <=, ==, !=
	threshold float64

	// Var-to-var fields ($A op $B)
	isVarRef bool   // true when comparing two variable references
	rightRef string // B, C, etc. (the right-hand variable)
}

// parseTriggerExp parses a trigger expression in one of two forms:
//   - $A > 100   (var op number)
//   - $A > $B    (var op var)
//
// Returns an error if the expression is empty, has no valid structure,
// or the right-hand side is neither a number nor a variable reference.
func parseTriggerExp(exp string) (*triggerExpParts, error) {
	exp = strings.TrimSpace(exp)
	if exp == "" {
		return nil, fmt.Errorf("empty trigger expression")
	}

	// Try var-to-var first: $A op $B
	if m := varVarRe.FindStringSubmatch(exp); len(m) == 4 {
		return &triggerExpParts{
			ref:      m[1],
			op:       m[2],
			isVarRef: true,
			rightRef: m[3],
		}, nil
	}

	// Find the reference ($A, $B, etc.)
	refStart := strings.Index(exp, "$")
	if refStart < 0 {
		return nil, fmt.Errorf("missing $ reference in expression: %q", exp)
	}

	// Find the operator (try longest first to avoid ambiguity)
	operators := []string{">=", "<=", "!=", "==", ">", "<"}
	var op string
	opIdx := -1
	for _, o := range operators {
		idx := strings.Index(exp[refStart:], o)
		if idx >= 0 {
			absIdx := refStart + idx
			if opIdx < 0 || absIdx < opIdx {
				op = o
				opIdx = absIdx
			}
		}
	}

	if opIdx < 0 {
		return nil, fmt.Errorf("no comparison operator found in expression: %q", exp)
	}

	ref := strings.TrimSpace(exp[refStart+1 : opIdx])
	if ref == "" {
		return nil, fmt.Errorf("empty variable reference before operator in: %q", exp)
	}

	thresholdStr := strings.TrimSpace(exp[opIdx+len(op):])
	if thresholdStr == "" {
		return nil, fmt.Errorf("missing right-hand side after operator in: %q", exp)
	}

	var threshold float64
	if _, err := fmt.Sscanf(thresholdStr, "%f", &threshold); err != nil {
		return nil, fmt.Errorf("invalid threshold %q in expression: %w", thresholdStr, err)
	}

	return &triggerExpParts{
		ref:       ref,
		op:        op,
		threshold: threshold,
	}, nil
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
