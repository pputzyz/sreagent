package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/pkg/fingerprint"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
)

// Run is the main loop for a single rule evaluator.
func (re *RuleEvaluator) Run() {
	// P0-5: Signal the parent WaitGroup when this goroutine exits so Stop()
	// can wait for in-flight evaluations to complete before tearing down
	// shared resources (Redis, DB connections, etc.).
	if re.runWG != nil {
		defer re.runWG.Done()
	}
	// Last-resort recover: anything outside the per-tick guards below.
	defer func() {
		if r := recover(); r != nil {
			re.logger.Error("rule evaluator Run() panic recovered",
				zap.Any("recover", r),
				zap.Uint("rule_id", re.rule.ID),
				zap.String("rule_name", re.rule.Name),
			)
		}
	}()

	// Parse evaluation interval from rule (default 60s)
	interval := time.Duration(re.rule.EvalInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	re.logger.Info("rule evaluator started",
		zap.Duration("interval", interval),
		zap.String("expression", re.rule.Expression),
	)

	// Restore persisted state from Redis (if available)
	re.loadPersistedState()

	// Run first evaluation immediately
	re.safeTick("evaluate", re.evaluate)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// GC resolved states every hour to prevent unbounded sync.Map growth.
	gcTicker := time.NewTicker(1 * time.Hour)
	defer gcTicker.Stop()

	for {
		select {
		case <-ticker.C:
			re.safeTick("evaluate", re.evaluate)
		case <-gcTicker.C:
			re.safeTick("gc_states", re.gcStates)
		case <-re.stopCh:
			re.logger.Info("rule evaluator stopped")
			return
		}
	}
}

// safeTick runs fn with a panic guard so a single bad evaluation cannot kill
// this evaluator's Run loop permanently. Without it, a panic in evaluate()
// would terminate Run() while the evaluator stays registered (same Version),
// so syncRules would never restart it — the rule silently stops evaluating.
func (re *RuleEvaluator) safeTick(op string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			re.logger.Error("rule evaluator tick panic recovered",
				zap.String("op", op),
				zap.Any("recover", r),
				zap.Uint("rule_id", re.rule.ID),
				zap.String("rule_name", re.rule.Name),
			)
		}
	}()
	fn()
}

// evaluate performs one evaluation cycle.
// Uses per-fingerprint locking via sync.Map + stateLock instead of a single global mutex.
func (re *RuleEvaluator) evaluate() {
	ctx, cancel := context.WithTimeout(re.ctx, 30*time.Second)
	defer cancel()

	// Capture a single timestamp for the entire evaluation cycle to avoid
	// subtle inconsistencies from multiple time.Now() calls.
	now := time.Now()

	// 1. Execute query against datasource — dispatch by datasource type
	// Supports both single Expression and multi-query (Queries + TriggerExp)
	var results []datasource.QueryResult
	var err error

	if len(re.rule.Queries) > 0 {
		// Multi-query mode: evaluate each query, join results, apply trigger expression
		results, err = re.executeMultiQuery(ctx)
		if err == nil && re.rule.TriggerExp != "" {
			results, err = re.evaluateTriggerExp(results)
		}
	} else {
		// Legacy single-expression mode (backward compatible)
		results, err = re.executeQuery(ctx)
	}

	if err != nil {
		errCount := re.consecutiveErrors.Add(1)
		if errCount >= 5 {
			re.logger.Error("query execution failed repeatedly",
				zap.Error(err),
				zap.Int64("consecutive_errors", errCount),
			)
		} else {
			re.logger.Warn("query execution failed, will retry next cycle",
				zap.Error(err),
			)
		}
		metrics.IncAlertsEvaluated(strconv.FormatUint(uint64(re.rule.ID), 10), "error")
		// On error, we skip nodata detection to avoid false positives
		return
	}
	// Reset consecutive error count on success
	if prev := re.consecutiveErrors.Swap(0); prev > 0 {
		re.logger.Info("query execution recovered after errors",
			zap.Int64("previous_errors", prev),
		)
	}

	// Parse for_duration
	forDuration := parseDuration(re.rule.ForDuration)

	// Parse recovery_hold
	recoveryHold := parseDuration(re.rule.RecoveryHold)

	// Track which fingerprints were seen in this cycle
	seenFingerprints := make(map[string]bool, len(results))

	// 2. For each result series — lock only the affected fingerprint
	for _, result := range results {
		// Get value (use last value if multiple)
		if len(result.Values) == 0 {
			continue
		}
		value := result.Values[len(result.Values)-1].Value

		// Generate fingerprint from labels
		fp := generateFingerprint(result.Labels)
		seenFingerprints[fp] = true

		sl := re.lockState(fp)
		sl.mu.Lock()

		state := sl.state

		if state == nil {
			// New alert series detected — reuse cycle-level `now`
			// Deep copy annotations to avoid sharing the underlying map with the rule definition.
			annotations := make(map[string]string, len(re.rule.Annotations))
			for k, v := range re.rule.Annotations {
				annotations[k] = v
			}
			state = &AlertState{
				Labels:      result.Labels,
				Value:       value,
				Annotations: annotations,
				LastSeen:    now,
			}

			if forDuration <= 0 {
				// Immediately fire — check time-window mute first.
				// B4-3: ShouldSuppress is skipped because each rule has a single
				// severity, making cross-severity suppression a no-op.
				severity := string(re.rule.Severity)
				if muted, muteID := re.checkTimeWindowMute(state.Labels, severity); muted {
					re.logger.Debug("alert muted by time-window rule at engine level",
						zap.String("fingerprint", fp),
						zap.Uint("mute_rule_id", muteID),
					)
					state.Status = "pending"
					state.ActiveAt = now
					sl.state = state
					re.persistState(fp, state)
				} else {
					state.ActiveAt = now
					state.FiredAt = now

					if re.suppressor != nil {
						re.suppressor.UpdateSeverity(re.rule.ID, fp, severity)
					}
					// Set status to firing AFTER createAlertEvent succeeds to avoid
					// phantom firing states in GetFiringEvents when DB write fails.
					re.createAlertEvent(state, model.EventStatusFiring)
					if state.EventID == 0 {
						// DB write failed — state was reverted to "pending" inside createAlertEvent.
						sl.state = state
						re.persistState(fp, state)
						sl.mu.Unlock()
						continue
					}
					state.Status = "firing"
					state.Revision++
					sl.state = state
					metrics.IncAlertsEvaluated(strconv.FormatUint(uint64(re.rule.ID), 10), "firing")
					re.persistState(fp, state)
				}
			} else {
				// Enter pending state
				state.Status = "pending"
				state.ActiveAt = now
				state.Revision++
				sl.state = state
				re.persistState(fp, state)
			}
		} else {
			state.Value = value
			state.LastSeen = now
			// Reset recovery hold since alert is active again
			state.RecoveryHoldUntil = time.Time{}

			switch state.Status {
			case "pending":
				// Check if pending long enough
				if time.Since(state.ActiveAt) >= forDuration {
					severity := string(re.rule.Severity)
					// B4-3: ShouldSuppress is skipped (single-severity per rule).
					if muted, muteID := re.checkTimeWindowMute(state.Labels, severity); muted {
						re.logger.Debug("pending alert muted by time-window rule at engine level",
							zap.String("fingerprint", fp),
							zap.Uint("mute_rule_id", muteID),
						)
					} else {
						state.FiredAt = now
						if re.suppressor != nil {
							re.suppressor.UpdateSeverity(re.rule.ID, fp, severity)
						}
						re.createAlertEvent(state, model.EventStatusFiring)
						if state.EventID == 0 {
							// DB write failed — state was reverted to "pending" inside createAlertEvent.
							state.Status = "pending"
							state.FiredAt = time.Time{}
						} else {
							state.Status = "firing"
							state.Revision++
						}
						re.persistState(fp, state)
					}
				}

			case "firing":
				// Update value, increment fire count if event exists
				re.updateFiringEvent(state)
				re.persistState(fp, state) // refresh TTL

			case "resolved":
				// Alert came back, re-activate
				severity := string(re.rule.Severity)
				if forDuration <= 0 {
					// B4-3: ShouldSuppress is skipped (single-severity per rule).
					if muted, muteID := re.checkTimeWindowMute(state.Labels, severity); muted {
						re.logger.Debug("re-fired alert muted by time-window rule at engine level",
							zap.String("fingerprint", fp),
							zap.Uint("mute_rule_id", muteID),
						)
					} else {
						state.ActiveAt = now
						state.FiredAt = now
						if re.suppressor != nil {
							re.suppressor.UpdateSeverity(re.rule.ID, fp, severity)
						}
						re.createAlertEvent(state, model.EventStatusFiring)
						if state.EventID == 0 {
							// DB write failed — state was reverted to "pending" inside createAlertEvent.
							state.Status = "pending"
							state.FiredAt = time.Time{}
						} else {
							state.Status = "firing"
							state.Revision++
						}
						re.persistState(fp, state)
					}
				} else {
					state.Status = "pending"
					state.ActiveAt = now
					state.Revision++
					re.persistState(fp, state)
				}
			}
		}

		sl.mu.Unlock()
	}

	// 3. Check for resolved alerts — iterate with rangeStates, lock each fp individually
	noDataFP := fmt.Sprintf("nodata_%d", re.rule.ID)
	re.rangeStates(func(fp string, sl *stateLock) bool {
		if seenFingerprints[fp] {
			return true // skip, already processed above
		}
		// The synthetic NoData state is owned exclusively by step 4 below; it never
		// appears in seenFingerprints (which only holds real query-result fingerprints),
		// so without this guard step 3 would reset its ActiveAt every cycle (NoData
		// could never accumulate to fire) and resolve a firing NoData state prematurely.
		if fp == noDataFP {
			return true
		}

		sl.mu.Lock()
		state := sl.state
		if state == nil {
			sl.mu.Unlock()
			return true
		}

		switch state.Status {
		case "pending":
			// Pending alert disappeared, just remove it
			sl.state = nil
			re.deleteState(fp)
			re.deletePersistedState(fp)

		case "firing":
			if recoveryHold > 0 && state.RecoveryHoldUntil.IsZero() {
				// Start recovery observation period
				state.RecoveryHoldUntil = now.Add(recoveryHold)
				re.logger.Debug("alert entering recovery observation",
					zap.String("fingerprint", fp),
					zap.Duration("hold", recoveryHold),
				)
				re.persistState(fp, state)
			} else if recoveryHold > 0 && now.Before(state.RecoveryHoldUntil) {
				// Still in observation period, skip
			} else {
				// Resolve the alert
				state.Status = "resolved"
				state.ResolvedAt = now
				state.Revision++
				if re.suppressor != nil {
					re.suppressor.RemoveSeverity(re.rule.ID, fp, string(re.rule.Severity))
				}
				if err := re.resolveAlertEvent(state); err != nil {
					// resolveAlertEvent already reverted state to "firing" on failure.
					// Do NOT clear state — the next eval cycle will retry.
					sl.mu.Unlock()
					return true
				}
				metrics.IncAlertsEvaluated(strconv.FormatUint(uint64(re.rule.ID), 10), "resolved")
				sl.state = nil
				re.deleteState(fp)
				re.deletePersistedState(fp)
			}
		}

		sl.mu.Unlock()
		return true
	})

	// 4. NoData detection
	if re.rule.NoDataEnabled && len(results) == 0 {
		noDataDuration := parseDuration(re.rule.NoDataDuration)
		if noDataDuration <= 0 {
			noDataDuration = 5 * time.Minute
		}

		sl := re.lockState(noDataFP)
		sl.mu.Lock()

		if sl.state == nil {
			// First time seeing no data - start tracking
			newState := &AlertState{
				Labels: map[string]string{
					"alertname":  re.rule.Name,
					"severity":   string(re.rule.Severity),
					"__nodata__": "true",
				},
				Status:      "pending",
				ActiveAt:    now,
				LastSeen:    now,
				Annotations: map[string]string{"description": "No data received for rule: " + re.rule.Name},
			}
			sl.state = newState
			re.persistState(noDataFP, newState)
		} else if sl.state.Status == "pending" && time.Since(sl.state.ActiveAt) >= noDataDuration {
			sl.state.FiredAt = now
			re.createAlertEvent(sl.state, model.EventStatusFiring)
			if sl.state.EventID == 0 {
				// DB write failed — state was reverted to "pending" inside createAlertEvent.
				sl.state.Status = "pending"
				sl.state.FiredAt = time.Time{}
			} else {
				sl.state.Status = "firing"
				sl.state.Revision++
				metrics.IncAlertsEvaluated(strconv.FormatUint(uint64(re.rule.ID), 10), "nodata")
			}
			re.persistState(noDataFP, sl.state)
		}

		sl.mu.Unlock()
	} else {
		// Data received, clear nodata state if it exists
		sl := re.lockState(noDataFP)
		sl.mu.Lock()
		resolved := true
		if sl.state != nil {
			if sl.state.Status == "firing" {
				sl.state.Status = "resolved"
				sl.state.ResolvedAt = now
				sl.state.Revision++
				if err := re.resolveAlertEvent(sl.state); err != nil {
					// Resolution failed — keep state for retry on next cycle.
					resolved = false
				}
			}
			if resolved {
				sl.state = nil
				re.deleteState(noDataFP)
				re.deletePersistedState(noDataFP)
			}
		}
		sl.mu.Unlock()
	}
}

// generateFingerprint creates a unique fingerprint from label set.
// Delegates to fingerprint.Compute after filtering out internal labels (double-underscore prefix/suffix).
func generateFingerprint(labels map[string]string) string {
	filtered := make(map[string]string, len(labels))
	for k, v := range labels {
		if strings.HasPrefix(k, "__") && strings.HasSuffix(k, "__") {
			continue
		}
		filtered[k] = v
	}
	return fingerprint.Compute(filtered)
}

// parseDuration parses a duration string like "5m", "1h", "30s".
// Returns 0 on failure or empty string.
func parseDuration(s string) time.Duration {
	if s == "" || s == "0" || s == "0s" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		// Log warning for non-empty invalid input so operators can diagnose config issues.
		zap.L().Warn("invalid duration string, defaulting to 0",
			zap.String("input", s),
			zap.Error(err),
		)
		return 0
	}
	return d
}

// executeQuery dispatches the alert rule query to the appropriate backend
// based on the datasource type.
//   - Prometheus / VictoriaMetrics: PromQL instant query (/api/v1/query)
//   - Zabbix: JSON-RPC item.get by key pattern
//   - VictoriaLogs: LogsQL query (/select/logsql/query), returns match count
func (re *RuleEvaluator) executeQuery(ctx context.Context) ([]datasource.QueryResult, error) {
	// Variable filling: if VarConfig is set, substitute $var placeholders
	// in the expression with actual parameter values before querying.
	if re.rule.VarConfig != nil && len(re.rule.VarConfig.Params) > 0 {
		return re.executeQueryWithVarFilling(ctx)
	}

	ep := re.datasource.Endpoint
	at := re.datasource.AuthType
	ac := re.decryptedAuthConfig
	expr := re.rule.Expression

	switch re.datasource.Type {
	case "zabbix":
		return datasource.ZabbixInstantQuery(ctx, ep, at, ac, expr)
	case "victorialogs":
		lookback := time.Duration(re.rule.EvalInterval) * time.Second
		return datasource.VictoriaLogsInstantQuery(ctx, ep, at, ac, expr, lookback)
	default:
		// prometheus, victoriametrics and any future Prometheus-compatible sources
		return re.queryClient.InstantQuery(ctx, ep, at, ac, expr, time.Time{})
	}
}

// executeQueryWithVarFilling implements variable filling for alert rules.
// When VarConfig is set, it substitutes $var placeholders in the expression
// with actual parameter values, then executes each variant and combines results.
//
// Strategy "before_query": Replace $var in expression, execute each variant.
//   - Required when expression contains aggregation functions (sum, avg, etc.)
//   - Example: avg(mem_used_percent{host="$host"}) > $val
//     becomes: avg(mem_used_percent{host="web01"}) > 90, then > 90 for web02, etc.
//
// Strategy "after_query": Execute query first, then filter by matching labels.
//   - More efficient for simple expressions without aggregation
//   - Example: mem_used_percent{host="$host"} > $val
//     queries mem_used_percent{} > 90, then filters results where host matches
func (re *RuleEvaluator) executeQueryWithVarFilling(ctx context.Context) ([]datasource.QueryResult, error) {
	vc := re.rule.VarConfig
	strategy := vc.Strategy
	if strategy == "" {
		strategy = "before_query"
	}

	// Resolve all variable values
	varValues, err := re.resolveAllVarValues(ctx, vc.Params)
	if err != nil {
		return nil, fmt.Errorf("var filling: resolve values failed: %w", err)
	}

	// Check that all params have at least one value
	for _, p := range vc.Params {
		if len(varValues[p.Name]) == 0 {
			re.logger.Warn("variable has no values, skipping var filling",
				zap.String("var", p.Name),
				zap.String("type", p.Type))
			return nil, nil
		}
	}

	switch strategy {
	case "after_query":
		return re.executeVarFillingAfterQuery(ctx, vc, varValues)
	default: // "before_query"
		return re.executeVarFillingBeforeQuery(ctx, vc, varValues)
	}
}

// resolveAllVarValues resolves the value list for each variable parameter.
func (re *RuleEvaluator) resolveAllVarValues(ctx context.Context, params []model.VarParam) (map[string][]string, error) {
	result := make(map[string][]string, len(params))
	for _, p := range params {
		vals, err := re.resolveVarValues(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("var %q (type=%s): %w", p.Name, p.Type, err)
		}
		result[p.Name] = vals
	}
	return result, nil
}

// resolveVarValues resolves the value list for a single variable parameter.
//   - type "enum": use explicit Values list
//   - type "host": query the label registry for known host/instance values
//   - type "device": query the label registry for known device values
func (re *RuleEvaluator) resolveVarValues(ctx context.Context, p model.VarParam) ([]string, error) {
	// Explicit values always take priority
	if len(p.Values) > 0 {
		return p.Values, nil
	}

	switch p.Type {
	case "enum":
		// enum without explicit values — nothing to resolve
		return nil, nil

	case "host", "device":
		// Query label registry for known values of this label key
		if re.labelRegistryRepo == nil {
			re.logger.Warn("label registry not available for host/device variable resolution",
				zap.String("var", p.Name))
			return nil, nil
		}
		// Try common label key names: the var name itself, "instance", "host", "ident"
		keys := []string{p.Name}
		if p.Name != "instance" {
			keys = append(keys, "instance")
		}
		if p.Name != "host" && p.Name != "instance" {
			keys = append(keys, "host")
		}

		var dsIDs []uint
		if re.datasource != nil {
			dsIDs = []uint{re.datasource.ID}
		}

		for _, key := range keys {
			vals, err := re.labelRegistryRepo.GetValues(ctx, key, dsIDs)
			if err != nil {
				re.logger.Warn("label registry query failed",
					zap.String("key", key), zap.Error(err))
				continue
			}
			if len(vals) > 0 {
				return vals, nil
			}
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown variable type: %s", p.Type)
	}
}

// executeVarFillingBeforeQuery substitutes $var in the expression and queries each variant.
// Used when the expression contains aggregation functions that would lose labels.
func (re *RuleEvaluator) executeVarFillingBeforeQuery(ctx context.Context, vc *model.VarConfig, varValues map[string][]string) ([]datasource.QueryResult, error) {
	ep := re.datasource.Endpoint
	at := re.datasource.AuthType
	ac := re.decryptedAuthConfig

	// Build all parameter combinations
	paramNames := make([]string, 0, len(vc.Params))
	for _, p := range vc.Params {
		paramNames = append(paramNames, p.Name)
	}
	// Sort for deterministic order
	sort.Strings(paramNames)

	combinations, err := buildCombinations(paramNames, varValues)
	if err != nil {
		return nil, err
	}
	if len(combinations) == 0 {
		return nil, nil
	}

	// Limit concurrency to avoid overwhelming the datasource
	const maxConcurrency = 50
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex
	var allResults []datasource.QueryResult
	var firstErr error

	var wg sync.WaitGroup
	const maxAllResults = 100000
	for _, combo := range combinations {
		// Safety cap: stop spawning queries if result set is already huge.
		mu.Lock()
		overLimit := len(allResults) >= maxAllResults
		mu.Unlock()
		if overLimit {
			break
		}

		expr := re.rule.Expression
		for i, name := range paramNames {
			expr = strings.ReplaceAll(expr, fmt.Sprintf("$%s", name), combo[i])
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(queryExpr string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			var results []datasource.QueryResult
			var err error
			switch re.datasource.Type {
			case "zabbix":
				results, err = datasource.ZabbixInstantQuery(ctx, ep, at, ac, queryExpr)
			case "victorialogs":
				lookback := time.Duration(re.rule.EvalInterval) * time.Second
				results, err = datasource.VictoriaLogsInstantQuery(ctx, ep, at, ac, queryExpr, lookback)
			default:
				results, err = re.queryClient.InstantQuery(ctx, ep, at, ac, queryExpr, time.Time{})
			}

			if err != nil {
				re.logger.Debug("var filling query failed",
					zap.String("expr", queryExpr), zap.Error(err))
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(expr)
	}
	wg.Wait()

	// If ALL queries failed, return the error; partial success is OK
	if len(allResults) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return allResults, nil
}

// executeVarFillingAfterQuery executes the query first, then filters results by matching labels.
// Used for simple expressions without aggregation (more efficient).
func (re *RuleEvaluator) executeVarFillingAfterQuery(ctx context.Context, vc *model.VarConfig, varValues map[string][]string) ([]datasource.QueryResult, error) {
	ep := re.datasource.Endpoint
	at := re.datasource.AuthType
	ac := re.decryptedAuthConfig

	// Remove $var label selectors from the expression for the broad query
	broadExpr := removeVarLabelSelectors(re.rule.Expression, vc.Params)

	var allResults []datasource.QueryResult
	var err error
	switch re.datasource.Type {
	case "zabbix":
		allResults, err = datasource.ZabbixInstantQuery(ctx, ep, at, ac, broadExpr)
	case "victorialogs":
		lookback := time.Duration(re.rule.EvalInterval) * time.Second
		allResults, err = datasource.VictoriaLogsInstantQuery(ctx, ep, at, ac, broadExpr, lookback)
	default:
		allResults, err = re.queryClient.InstantQuery(ctx, ep, at, ac, broadExpr, time.Time{})
	}
	if err != nil {
		return nil, err
	}

	// Build allowed value sets for each variable (used for label matching)
	allowedSets := make(map[string]map[string]bool, len(vc.Params))
	for _, p := range vc.Params {
		vals := varValues[p.Name]
		if len(vals) == 0 {
			continue
		}
		m := make(map[string]bool, len(vals))
		for _, v := range vals {
			m[v] = true
		}
		allowedSets[p.Name] = m
	}

	// Extract variable-to-label mapping from the expression
	varToLabel := extractVarLabelMapping(re.rule.Expression)

	// Filter results: keep only those whose labels match all variable constraints
	filtered := make([]datasource.QueryResult, 0, len(allResults))
	for _, r := range allResults {
		match := true
		for varName, allowed := range allowedSets {
			labelKey := varToLabel[varName]
			if labelKey == "" {
				labelKey = varName // fallback: label key == variable name
			}
			labelVal, ok := r.Labels[labelKey]
			if !ok || !allowed[labelVal] {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

// buildCombinations generates all permutations of parameter values.
// For params ["host","env"] with values {"host":["a","b"], "env":["prod","staging"]},
// returns [["a","prod"],["a","staging"],["b","prod"],["b","staging"]].
//
// A hard limit of maxCombinations (1,000) prevents cartesian product explosion
// that could overwhelm the TSDB with queries.
func buildCombinations(paramNames []string, varValues map[string][]string) ([][]string, error) {
	if len(paramNames) == 0 {
		return nil, nil
	}

	// Pre-check total combinations to prevent explosion.
	const maxCombinations = 1000
	total := 1
	for _, p := range paramNames {
		total *= len(varValues[p])
		if total > maxCombinations {
			return nil, fmt.Errorf("variable filling: %d combinations exceeds limit %d (params: %v)", total, maxCombinations, paramNames)
		}
	}

	var result [][]string
	combo := make([]string, len(paramNames))
	var build func(depth int)
	build = func(depth int) {
		if depth == len(paramNames) {
			c := make([]string, len(combo))
			copy(c, combo)
			result = append(result, c)
			return
		}
		for _, v := range varValues[paramNames[depth]] {
			combo[depth] = v
			build(depth + 1)
		}
	}
	build(0)
	return result, nil
}

// removeVarLabelSelectors removes label selectors containing $var from the expression.
// e.g. mem_used_percent{host="$host",env="prod"} > $val
// becomes mem_used_percent{env="prod"} > $val
func removeVarLabelSelectors(expr string, params []model.VarParam) string {
	// Build a set of variable names for quick lookup
	varNames := make(map[string]bool, len(params))
	for _, p := range params {
		varNames[p.Name] = true
	}

	// Find the label selector block(s) in curly braces
	var result strings.Builder
	i := 0
	for i < len(expr) {
		if expr[i] == '{' {
			// Find the closing brace
			end := strings.IndexByte(expr[i:], '}')
			if end < 0 {
				result.WriteByte(expr[i])
				i++
				continue
			}
			inner := expr[i+1 : i+end]
			// Split by comma, keep only non-variable selectors
			parts := strings.Split(inner, ",")
			var kept []string
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				isVar := false
				for varName := range varNames {
					if strings.Contains(trimmed, fmt.Sprintf("$%s", varName)) {
						isVar = true
						break
					}
				}
				if !isVar {
					kept = append(kept, part)
				}
			}
			if len(kept) > 0 {
				result.WriteByte('{')
				result.WriteString(strings.Join(kept, ","))
				result.WriteByte('}')
			}
			i += end + 1
		} else {
			result.WriteByte(expr[i])
			i++
		}
	}
	return result.String()
}

// extractVarLabelMapping extracts variable-to-label mappings from PromQL expressions.
// e.g. mem_used_percent{host="$my_host"} -> {"my_host": "host"}
func extractVarLabelMapping(expr string) map[string]string {
	mapping := make(map[string]string)
	// Find label selectors in curly braces
	for {
		start := strings.IndexByte(expr, '{')
		if start < 0 {
			break
		}
		end := strings.IndexByte(expr[start:], '}')
		if end < 0 {
			break
		}
		inner := expr[start+1 : start+end]
		pairs := strings.Split(inner, ",")
		for _, pair := range pairs {
			// Handle =, !=, =~, !~ operators
			var kv []string
			if strings.Contains(pair, "!=") {
				kv = strings.SplitN(pair, "!=", 2)
			} else if strings.Contains(pair, "=~") {
				kv = strings.SplitN(pair, "=~", 2)
			} else if strings.Contains(pair, "!~") {
				kv = strings.SplitN(pair, "!~", 2)
			} else {
				kv = strings.SplitN(pair, "=", 2)
			}
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
			if strings.HasPrefix(value, "$") {
				varName := value[1:]
				mapping[varName] = key
			}
		}
		expr = expr[start+end+1:]
	}
	return mapping
}
