package engine

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
)

// Run is the main loop for a single rule evaluator.
func (re *RuleEvaluator) Run() {
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
	re.evaluate()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// GC resolved states every hour to prevent unbounded sync.Map growth.
	gcTicker := time.NewTicker(1 * time.Hour)
	defer gcTicker.Stop()

	for {
		select {
		case <-ticker.C:
			re.evaluate()
		case <-gcTicker.C:
			re.gcStates()
		case <-re.stopCh:
			re.logger.Info("rule evaluator stopped")
			return
		}
	}
}

// evaluate performs one evaluation cycle.
// Uses per-fingerprint locking via sync.Map + stateLock instead of a single global mutex.
func (re *RuleEvaluator) evaluate() {
	ctx, cancel := context.WithTimeout(re.ctx, 30*time.Second)
	defer cancel()

	// 1. Execute query against datasource — dispatch by datasource type
	results, err := re.executeQuery(ctx)
	if err != nil {
		re.consecutiveErrors++
		if re.consecutiveErrors >= 5 {
			re.logger.Error("query execution failed repeatedly",
				zap.Error(err),
				zap.Int("consecutive_errors", re.consecutiveErrors),
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
	if re.consecutiveErrors > 0 {
		re.logger.Info("query execution recovered after errors",
			zap.Int("previous_errors", re.consecutiveErrors),
		)
		re.consecutiveErrors = 0
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
			// New alert series detected
			now := time.Now()
			state = &AlertState{
				Labels:      result.Labels,
				Value:       value,
				Annotations: map[string]string(re.rule.Annotations),
				LastSeen:    now,
			}

			if forDuration <= 0 {
				// Immediately fire — check suppression first
				severity := string(re.rule.Severity)
				if re.suppressor != nil && re.suppressor.ShouldSuppress(re.rule.ID, fp, severity) {
					re.logger.Debug("alert suppressed by higher severity",
						zap.String("fingerprint", fp),
						zap.String("severity", severity),
					)
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
			state.LastSeen = time.Now()
			// Reset recovery hold since alert is active again
			state.RecoveryHoldUntil = time.Time{}

			switch state.Status {
			case "pending":
				// Check if pending long enough
				if time.Since(state.ActiveAt) >= forDuration {
					now := time.Now()
					severity := string(re.rule.Severity)
					if re.suppressor != nil && re.suppressor.ShouldSuppress(re.rule.ID, fp, severity) {
						re.logger.Debug("pending alert suppressed by higher severity",
							zap.String("fingerprint", fp),
							zap.String("severity", severity),
						)
					} else {
						state.Status = "firing"
						state.FiredAt = now
						state.Revision++
						if re.suppressor != nil {
							re.suppressor.UpdateSeverity(re.rule.ID, fp, severity)
						}
						re.createAlertEvent(state, model.EventStatusFiring)
						re.persistState(fp, state)
					}
				}

			case "firing":
				// Update value, increment fire count if event exists
				re.updateFiringEvent(state)
				re.persistState(fp, state) // refresh TTL

			case "resolved":
				// Alert came back, re-activate
				now := time.Now()
				severity := string(re.rule.Severity)
				if forDuration <= 0 {
					if re.suppressor != nil && re.suppressor.ShouldSuppress(re.rule.ID, fp, severity) {
						re.logger.Debug("re-fired alert suppressed by higher severity",
							zap.String("fingerprint", fp),
							zap.String("severity", severity),
						)
					} else {
						state.Status = "firing"
						state.ActiveAt = now
						state.FiredAt = now
						state.Revision++
						if re.suppressor != nil {
							re.suppressor.UpdateSeverity(re.rule.ID, fp, severity)
						}
						re.createAlertEvent(state, model.EventStatusFiring)
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
	now := time.Now()
	re.rangeStates(func(fp string, sl *stateLock) bool {
		if seenFingerprints[fp] {
			return true // skip, already processed above
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

		noDataFP := fmt.Sprintf("nodata_%d", re.rule.ID)
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
			sl.state.Status = "firing"
			sl.state.FiredAt = now
			sl.state.Revision++
			re.createAlertEvent(sl.state, model.EventStatusFiring)
			metrics.IncAlertsEvaluated(strconv.FormatUint(uint64(re.rule.ID), 10), "nodata")
			re.persistState(noDataFP, sl.state)
		}

		sl.mu.Unlock()
	} else {
		// Data received, clear nodata state if it exists
		noDataFP := fmt.Sprintf("nodata_%d", re.rule.ID)
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
func generateFingerprint(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		// Skip internal labels
		if strings.HasPrefix(k, "__") && strings.HasSuffix(k, "__") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte(',')
	}

	hash := md5.Sum([]byte(b.String()))
	return fmt.Sprintf("%x", hash)
}

// parseDuration parses a duration string like "5m", "1h", "30s".
// Returns 0 on failure or empty string.
func parseDuration(s string) time.Duration {
	if s == "" || s == "0" || s == "0s" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
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
	ep := re.datasource.Endpoint
	at := re.datasource.AuthType
	ac := re.datasource.AuthConfig
	expr := re.rule.Expression

	switch re.datasource.Type {
	case "zabbix":
		return datasource.ZabbixInstantQuery(ctx, ep, at, ac, expr)
	case "victorialogs":
		return datasource.VictoriaLogsInstantQuery(ctx, ep, at, ac, expr)
	default:
		// prometheus, victoriametrics and any future Prometheus-compatible sources
		return re.queryClient.InstantQuery(ctx, ep, at, ac, expr, time.Time{})
	}
}
