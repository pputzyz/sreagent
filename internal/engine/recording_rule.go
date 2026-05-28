package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/repository"
)

// RecordingRuleEngine executes recording rules on a cron schedule.
// Each enabled rule runs independently via its own cron entry.
// The engine reads PromQL from the rule, queries the datasource,
// and records the execution outcome (success/failure/duration).
//
// Phase 1: validates queries and logs results.
// Phase 2 (future): writes results back as new time series via remote-write.
type RecordingRuleEngine struct {
	ruleRepo  *repository.RecordingRuleRepository
	dsRepo    *repository.DataSourceRepository
	execDB    *gorm.DB // for direct insert of execution records
	queryCli  *datasource.QueryClient
	leader    LeaderElection // optional; nil = always run
	logger    *zap.Logger

	cron    *cron.Cron
	entries  map[uint]cron.EntryID // ruleID → cron entry
	patterns map[uint]string       // ruleID → cron pattern (for change detection)
	mu      sync.Mutex
	stopCh  chan struct{}
	stopped bool
}

// NewRecordingRuleEngine creates a new recording rule execution engine.
func NewRecordingRuleEngine(
	ruleRepo *repository.RecordingRuleRepository,
	dsRepo *repository.DataSourceRepository,
	db *gorm.DB,
	queryCli *datasource.QueryClient,
	logger *zap.Logger,
) *RecordingRuleEngine {
	return &RecordingRuleEngine{
		ruleRepo: ruleRepo,
		dsRepo:   dsRepo,
		execDB:   db,
		queryCli: queryCli,
		logger:   logger,
		cron:      cron.New(),
		entries:   make(map[uint]cron.EntryID),
		patterns:  make(map[uint]string),
		stopCh:    make(chan struct{}),
	}
}

// SetLeaderElection sets an optional distributed leader election mechanism.
// When set, only the leader instance will run recording rules.
func (e *RecordingRuleEngine) SetLeaderElection(le LeaderElection) {
	e.leader = le
}

// Start loads enabled recording rules from the database and schedules them.
func (e *RecordingRuleEngine) Start(ctx context.Context) {
	e.logger.Info("recording rule engine starting")

	if err := e.syncRules(ctx); err != nil {
		e.logger.Error("recording rule engine: initial sync failed", zap.Error(err))
	}

	e.cron.Start()

	// Periodic sync loop to pick up new/changed/deleted rules.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("recording rule engine sync loop panic recovered", zap.Any("recover", r))
			}
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if e.leader != nil && !e.leader.IsLeader() {
					continue
				}
				if err := e.syncRules(context.Background()); err != nil {
					e.logger.Error("recording rule engine: sync failed", zap.Error(err))
				}
			case <-e.stopCh:
				return
			}
		}
	}()

	e.logger.Info("recording rule engine started")
}

// Stop gracefully stops the recording rule engine.
func (e *RecordingRuleEngine) Stop() {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	e.mu.Unlock()

	close(e.stopCh)
	ctx := e.cron.Stop()
	<-ctx.Done()
	e.logger.Info("recording rule engine stopped")
}

// syncRules loads enabled rules from DB and reconciles cron entries.
func (e *RecordingRuleEngine) syncRules(ctx context.Context) error {
	rules, err := e.ruleRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list enabled recording rules: %w", err)
	}

	activeIDs := make(map[uint]bool, len(rules))

	for i := range rules {
		rule := &rules[i]
		activeIDs[rule.ID] = true

		e.mu.Lock()
		existingEntryID, exists := e.entries[rule.ID]
		oldPattern := e.patterns[rule.ID]
		e.mu.Unlock()

		if exists {
			// Detect cron pattern change — remove old entry and re-add
			newPattern := rule.CronPattern
			if newPattern == "" {
				newPattern = "@every 60s"
			}
			if oldPattern != newPattern {
				e.cron.Remove(existingEntryID)
				e.mu.Lock()
				delete(e.entries, rule.ID)
				delete(e.patterns, rule.ID)
				e.mu.Unlock()

				if err := e.addRule(rule); err != nil {
					e.logger.Error("recording rule engine: failed to reschedule rule after pattern change",
						zap.Uint("rule_id", rule.ID),
						zap.String("name", rule.Name),
						zap.Error(err),
					)
				} else {
					e.logger.Info("recording rule engine: rescheduled rule with new pattern",
						zap.Uint("rule_id", rule.ID),
						zap.String("old_pattern", oldPattern),
						zap.String("new_pattern", newPattern),
					)
				}
			}
			continue
		}

		if err := e.addRule(rule); err != nil {
			e.logger.Error("recording rule engine: failed to schedule rule",
				zap.Uint("rule_id", rule.ID),
				zap.String("name", rule.Name),
				zap.Error(err),
			)
		}
	}

	// Remove entries for rules that no longer exist or are disabled.
	e.mu.Lock()
	for ruleID, entryID := range e.entries {
		if !activeIDs[ruleID] {
			e.cron.Remove(entryID)
			delete(e.entries, ruleID)
			delete(e.patterns, ruleID)
			e.logger.Info("recording rule engine: removed rule", zap.Uint("rule_id", ruleID))
		}
	}
	e.mu.Unlock()

	e.logger.Debug("recording rule engine sync completed",
		zap.Int("active_rules", len(rules)),
		zap.Int("scheduled_entries", len(e.entries)),
	)

	return nil
}

// addRule registers a single recording rule with the cron scheduler.
func (e *RecordingRuleEngine) addRule(rule *model.RecordingRule) error {
	pattern := rule.CronPattern
	if pattern == "" {
		pattern = "@every 60s"
	}

	// Capture for closure.
	ruleCopy := *rule

	entryID, err := e.cron.AddFunc(pattern, func() {
		if e.leader != nil && !e.leader.IsLeader() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		e.RunOnce(ctx, &ruleCopy)
	})
	if err != nil {
		return fmt.Errorf("invalid cron pattern %q: %w", pattern, err)
	}

	e.mu.Lock()
	e.entries[rule.ID] = entryID
	e.patterns[rule.ID] = pattern
	e.mu.Unlock()

	e.logger.Info("recording rule engine: scheduled rule",
		zap.Uint("rule_id", rule.ID),
		zap.String("name", rule.Name),
		zap.String("cron", pattern),
	)

	return nil
}

// RunOnce executes a single recording rule against all its configured datasources.
func (e *RecordingRuleEngine) RunOnce(ctx context.Context, rule *model.RecordingRule) {
	start := time.Now()

	rule.DB2FE()

	dsIDs := rule.DatasourceIDsJSON
	if len(dsIDs) == 0 {
		e.logger.Warn("recording rule has no datasources, skipping",
			zap.Uint("rule_id", rule.ID),
			zap.String("name", rule.Name),
		)
		return
	}

	var lastErr error
	var totalSeries int

	for _, dsID := range dsIDs {
		ds, err := e.dsRepo.GetByID(ctx, uint(dsID))
		if err != nil {
			e.logger.Error("recording rule: failed to get datasource",
				zap.Uint("rule_id", rule.ID),
				zap.Int64("datasource_id", dsID),
				zap.Error(err),
			)
			lastErr = err
			continue
		}

		if !ds.IsEnabled {
			e.logger.Debug("recording rule: datasource disabled, skipping",
				zap.Uint("rule_id", rule.ID),
				zap.Int64("datasource_id", dsID),
			)
			continue
		}

		results, err := e.queryCli.InstantQuery(ctx, ds.Endpoint, ds.AuthType, ds.AuthConfig, rule.PromQL, time.Time{})
		if err != nil {
			e.logger.Error("recording rule: query failed",
				zap.Uint("rule_id", rule.ID),
				zap.String("name", rule.Name),
				zap.String("datasource", ds.Name),
				zap.Error(err),
			)
			lastErr = err
			continue
		}

		totalSeries += len(results)

		e.logger.Debug("recording rule: query succeeded",
			zap.Uint("rule_id", rule.ID),
			zap.String("name", rule.Name),
			zap.String("datasource", ds.Name),
			zap.Int("series", len(results)),
		)

		// Phase 1 limitation: query is validated and execution recorded, but results
		// are NOT written back to the datasource as new time series.
		// Users expecting derived metrics (e.g. instance:cpu_usage:5m_avg) to appear
		// in Prometheus will see "no data" — this is a known limitation until Phase 2
		// implements remote-write support.
		e.logger.Warn("recording rule: Phase 1 — query validated but results NOT written back to datasource",
			zap.Uint("rule_id", rule.ID),
			zap.String("name", rule.Name),
			zap.String("metric", rule.Name),
			zap.Int("series_count", len(results)),
		)
	}

	duration := time.Since(start)
	durationMs := int(duration.Milliseconds())

	// Record execution status.
	execution := &model.RecordingRuleExecution{
		RuleID:     rule.ID,
		DurationMs: durationMs,
		ExecutedAt: time.Now(),
	}

	if lastErr != nil {
		execution.Status = "error"
		execution.ErrorMessage = lastErr.Error()
		metrics.IncRecordingRuleExecution("error")
		e.logger.Warn("recording rule execution finished with errors",
			zap.Uint("rule_id", rule.ID),
			zap.String("name", rule.Name),
			zap.Duration("duration", duration),
			zap.Error(lastErr),
		)
	} else {
		execution.Status = "success"
		metrics.IncRecordingRuleExecution("success")
		e.logger.Info("recording rule execution succeeded",
			zap.Uint("rule_id", rule.ID),
			zap.String("name", rule.Name),
			zap.Int("total_series", totalSeries),
			zap.Duration("duration", duration),
		)
	}

	if err := e.execDB.WithContext(ctx).Create(execution).Error; err != nil {
		e.logger.Error("recording rule: failed to save execution record",
			zap.Uint("rule_id", rule.ID),
			zap.Error(err),
		)
	}
}

