package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/crypto"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	"github.com/sreagent/sreagent/internal/pkg/hashring"
	"github.com/sreagent/sreagent/internal/pkg/metrics"
	"github.com/sreagent/sreagent/internal/repository"
)

// muteRuleRepoAdapter adapts a MuteRuleRepository to the MuteRuleChecker interface.
type muteRuleRepoAdapter struct {
	repo *repository.MuteRuleRepository
}

func (a *muteRuleRepoAdapter) FindEnabled(ctx context.Context) ([]model.MuteRule, error) {
	return a.repo.FindAllEnabled(ctx)
}

// stateLock pairs a per-fingerprint mutex with its AlertState,
// enabling fine-grained concurrent access instead of a single global lock.
type stateLock struct {
	mu    sync.Mutex
	state *AlertState
}

// AlertState tracks the state of an alert for a specific label combination.
type AlertState struct {
	Labels      map[string]string
	Status      string // "pending", "firing", "resolved"
	ActiveAt    time.Time
	FiredAt     time.Time
	ResolvedAt  time.Time
	Value       float64
	Annotations map[string]string
	// For recovery observation (留观时长)
	RecoveryHoldUntil time.Time
	// NoData tracking
	LastSeen time.Time
	// EventID of the firing event in the DB
	EventID uint
	// Revision is incremented on every state change for optimistic concurrency.
	Revision int64
}

// RuleEvaluator evaluates a single alert rule.
type RuleEvaluator struct {
	rule              *model.AlertRule
	datasource        *model.DataSource
	decryptedAuthConfig string // decrypted AuthConfig; never persisted back to DB
	states            sync.Map // map[string]*stateLock, key is fingerprint — per-fp locking
	db                *gorm.DB
	eventRepo         *repository.AlertEventRepository
	queryClient       *datasource.QueryClient
	stateStore        StateStore // optional; nil = in-memory only
	suppressor        *LevelSuppressor
	workerPool        AlertWorkerPoolSubmiter // optional bounded goroutine pool
	onAlert           func(ctx context.Context, event *model.AlertEvent)
	onLabelRecord     func(datasourceID uint, labels map[string]string) // passive label recording for all DS types
	labelRegistryRepo *repository.LabelRegistryRepository // optional; for variable filling
	ctx               context.Context // cancelled when evaluator stops
	stopCh            chan struct{}
	logger            *zap.Logger
	fallbackSem       chan struct{} // semaphore for onAlert when workerPool is nil (cap=16)

	// Reliability: consecutive query error tracking
	consecutiveErrors int
}

// Stop signals the evaluator to stop its Run loop.
func (re *RuleEvaluator) Stop() {
	select {
	case <-re.stopCh:
		// Already stopped
	default:
		close(re.stopCh)
	}
}

// PerDatasourceEvaluator is an isolated evaluation bucket for one datasource.
// Each datasource owns its own bucket so a DS outage doesn't affect others.
type PerDatasourceEvaluator struct {
	DatasourceID uint
	rules        sync.Map // map[uint]*RuleEvaluator, key is ruleID
	suppressor   *LevelSuppressor // shared; used to clean up entries on rule removal
	ctx          context.Context
	cancel       context.CancelFunc
	log          *zap.Logger
	addMu        sync.Mutex // protects AddRule to prevent orphan goroutines on concurrent calls
}

// NewPerDatasourceEvaluator creates a new per-datasource bucket.
func NewPerDatasourceEvaluator(parentCtx context.Context, dsID uint, log *zap.Logger) *PerDatasourceEvaluator {
	ctx, cancel := context.WithCancel(parentCtx)
	return &PerDatasourceEvaluator{
		DatasourceID: dsID,
		ctx:          ctx,
		cancel:       cancel,
		log:          log.With(zap.Uint("datasource_id", dsID)),
	}
}

// ruleVersion pairs a rule version with its evaluator for change detection.
type ruleVersion struct {
	version   int
	evaluator *RuleEvaluator
}

// AddRule adds a rule to this bucket and starts its evaluator.
// If ruleID already exists with the same Version, it is a no-op (preserving
// in-flight state). If the Version changed, the old evaluator is stopped first.
// Mutex-protected to prevent orphan goroutines from concurrent AddRule calls
// for the same ruleID.
func (p *PerDatasourceEvaluator) AddRule(rule *model.AlertRule, ds *model.DataSource, deps evaluatorDeps) {
	p.addMu.Lock()
	defer p.addMu.Unlock()
	// Store suppressor reference for cleanup in Stop/RemoveRule (B4-7 fix)
	if p.suppressor == nil && deps.suppressor != nil {
		p.suppressor = deps.suppressor
	}
	if old, loaded := p.rules.Load(rule.ID); loaded {
		rv := old.(*ruleVersion)
		if rv.version == rule.Version {
			return // same version — no change, keep running evaluator
		}
		// Version changed — stop old evaluator before creating new one
		p.rules.Delete(rule.ID)
		rv.evaluator.Stop()
		p.log.Info("rule version changed, restarting evaluator",
			zap.Uint("rule_id", rule.ID),
			zap.Int("old_version", rv.version),
			zap.Int("new_version", rule.Version))
	}
	ev := newRuleEvaluatorFromDeps(rule, ds, deps)
	p.rules.Store(rule.ID, &ruleVersion{version: rule.Version, evaluator: ev})
	go ev.Run()
	p.log.Info("rule added to datasource bucket",
		zap.Uint("rule_id", rule.ID),
		zap.String("rule_name", rule.Name))
}

// RemoveRule stops a rule's evaluator in this bucket and cleans up suppressor entries.
func (p *PerDatasourceEvaluator) RemoveRule(ruleID uint) {
	if v, loaded := p.rules.LoadAndDelete(ruleID); loaded {
		v.(*ruleVersion).evaluator.Stop()
		// B4-7: Clean up suppressor entries to prevent memory leak
		if p.suppressor != nil {
			p.suppressor.RemoveRule(ruleID)
		}
		p.log.Info("rule removed from datasource bucket", zap.Uint("rule_id", ruleID))
	}
}

// Stop stops the entire bucket (all rule evaluators exit) and cleans up suppressor entries.
func (p *PerDatasourceEvaluator) Stop() {
	p.cancel()
	p.rules.Range(func(k, v any) bool {
		ruleID := k.(uint)
		v.(*ruleVersion).evaluator.Stop()
		// B4-7: Clean up suppressor entries to prevent memory leak
		if p.suppressor != nil {
			p.suppressor.RemoveRule(ruleID)
		}
		return true
	})
	p.log.Info("datasource bucket stopped", zap.Uint("datasource_id", p.DatasourceID))
}

// RuleCount returns the number of rules in this bucket.
func (p *PerDatasourceEvaluator) RuleCount() int {
	count := 0
	p.rules.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// evaluatorDeps bundles dependencies for creating a RuleEvaluator.
type evaluatorDeps struct {
	db               *gorm.DB
	eventRepo        *repository.AlertEventRepository
	queryClient      *datasource.QueryClient
	stateStore       StateStore
	suppressor       *LevelSuppressor
	workerPool       AlertWorkerPoolSubmiter
	onAlert          func(ctx context.Context, event *model.AlertEvent)
	onLabelRecord    func(datasourceID uint, labels map[string]string)
	labelRegistryRepo *repository.LabelRegistryRepository
	ctx              context.Context
	logger           *zap.Logger
}

// newRuleEvaluatorFromDeps creates a RuleEvaluator from bundled deps.
func newRuleEvaluatorFromDeps(rule *model.AlertRule, ds *model.DataSource, deps evaluatorDeps) *RuleEvaluator {
	// Decrypt AuthConfig once at construction so every query uses plaintext
	// without mutating the model (which would corrupt the DB on save).
	ac := ds.AuthConfig
	if crypto.IsEncrypted(ac) {
		if plain, err := crypto.DecryptString(ac); err != nil {
			deps.logger.Error("failed to decrypt datasource auth_config, queries may fail",
				zap.Uint("datasource_id", ds.ID), zap.Error(err))
		} else {
			ac = plain
		}
	}

	return &RuleEvaluator{
		rule:                rule,
		datasource:          ds,
		decryptedAuthConfig: ac,
		db:                  deps.db,
		eventRepo:           deps.eventRepo,
		queryClient:         deps.queryClient,
		stateStore:          deps.stateStore,
		suppressor:          deps.suppressor,
		workerPool:          deps.workerPool,
		onAlert:             deps.onAlert,
		onLabelRecord:       deps.onLabelRecord,
		labelRegistryRepo:   deps.labelRegistryRepo,
		ctx:                 deps.ctx,
		stopCh:              make(chan struct{}),
		logger:              deps.logger.With(zap.Uint("rule_id", rule.ID), zap.String("rule_name", rule.Name)),
		fallbackSem:         make(chan struct{}, 16),
	}
}

// EngineStatus represents the status of the evaluation engine.
type EngineStatus struct {
	Running       bool   `json:"running"`
	TotalRules    int    `json:"total_rules"`
	ActiveAlerts  int    `json:"active_alerts"`
	Uptime        string `json:"uptime"`
	IsLeader      bool   `json:"is_leader"`
	HashRingMode  bool   `json:"hash_ring_mode"`
	InstanceID    string `json:"instance_id,omitempty"`
}

// Evaluator manages all rule evaluators.
type Evaluator struct {
	db           *gorm.DB
	dsRepo       *repository.DataSourceRepository
	ruleRepo     *repository.AlertRuleRepository
	eventRepo    *repository.AlertEventRepository
	timelineRepo *repository.AlertTimelineRepository
	queryClient  *datasource.QueryClient
	stateStore   StateStore              // optional; nil = in-memory only
	workerPool   AlertWorkerPoolSubmiter // optional bounded goroutine pool
	evaluators        map[uint]*RuleEvaluator // key: rule ID
	onAlert           func(ctx context.Context, event *model.AlertEvent)
	onLabelRecord     func(datasourceID uint, labels map[string]string) // passive label recording for all DS types
	labelRegistryRepo *repository.LabelRegistryRepository // optional; for variable filling
	suppressor        *LevelSuppressor
	mu           sync.RWMutex
	logger       *zap.Logger
	ctx          context.Context    // cancelled on Stop()
	cancel       context.CancelFunc // cancels ctx
	stopCh       chan struct{}
	startedAt    time.Time
	syncInterval time.Duration
	perDS        sync.Map           // map[uint]*PerDatasourceEvaluator, key is datasourceID
	perDSEval    bool               // feature flag: per-datasource bucket evaluation
	leader     LeaderElection // optional; nil = single-instance mode (no election)
	startOnce  sync.Once
	stopOnce   sync.Once
	wg         sync.WaitGroup

	// Hash ring mode: distribute rules across multiple instances.
	// When enabled, leader election is bypassed — all instances run
	// independently and each evaluates only its assigned rules.
	hashRing     *hashring.Ring // consistent hash ring for rule distribution
	instanceID   string         // this instance's identifier in the ring (e.g. hostname:pid)

	// Firing events TTL cache — reduces lock contention for high-frequency callers.
	firingCache    []*AlertState
	firingCacheAt  time.Time
	firingCacheMu  sync.RWMutex
	firingCacheTTL time.Duration // default 5s

	// forceSync is set by OnDatasourceUpdated to trigger a full sync on the
	// next tick, ensuring rules pick up changed datasource endpoints.
	forceSync    atomic.Bool
	forceSyncDSs sync.Map // map[uint]bool — specific datasource IDs to restart (empty = all)
}

// AlertWorkerPoolSubmiter is the subset of AlertWorkerPool used by the evaluator.
type AlertWorkerPoolSubmiter interface {
	Submit(ctx context.Context, fn func(context.Context)) bool
	Wait()
}

// NewEvaluator creates a new alert evaluation engine.
func NewEvaluator(
	db *gorm.DB,
	dsRepo *repository.DataSourceRepository,
	ruleRepo *repository.AlertRuleRepository,
	eventRepo *repository.AlertEventRepository,
	timelineRepo *repository.AlertTimelineRepository,
	queryClient *datasource.QueryClient,
	logger *zap.Logger,
) *Evaluator {
	return &Evaluator{
		db:           db,
		dsRepo:       dsRepo,
		ruleRepo:     ruleRepo,
		eventRepo:    eventRepo,
		timelineRepo: timelineRepo,
		queryClient:  queryClient,
		evaluators:   make(map[uint]*RuleEvaluator),
		suppressor:   NewLevelSuppressor(),
		logger:       logger,
		stopCh:       make(chan struct{}),
		syncInterval: 9 * time.Second, // match Nightingale's 9s sync frequency for faster rule propagation
	}
}

// SetSyncInterval configures how often rules are synced from DB.
func (e *Evaluator) SetSyncInterval(d time.Duration) {
	if d > 0 {
		e.syncInterval = d
	}
}

// SetPerDatasourceEval enables/disables per-datasource bucket evaluation.
func (e *Evaluator) SetPerDatasourceEval(enabled bool) {
	e.perDSEval = enabled
}

// SetLeaderElection sets an optional distributed leader election mechanism.
// When set, only the leader instance will run rule evaluators.
func (e *Evaluator) SetLeaderElection(le LeaderElection) {
	e.leader = le
}

// SetHashRing enables consistent-hash-ring mode for distributing rules
// across multiple engine instances. When set, leader election is bypassed
// and each instance evaluates only the rules it owns based on hash(ruleID).
func (e *Evaluator) SetHashRing(ring *hashring.Ring, instanceID string) {
	e.hashRing = ring
	e.instanceID = instanceID
}

// UpdateHashRing replaces the current hash ring (e.g. after membership change).
// Thread-safe: can be called from a background goroutine.
func (e *Evaluator) UpdateHashRing(ring *hashring.Ring) {
	e.hashRing = ring
}

// buildEvaluatorDeps bundles current Evaluator fields into evaluatorDeps for PerDatasourceEvaluator.
func (e *Evaluator) buildEvaluatorDeps() evaluatorDeps {
	return evaluatorDeps{
		db:                e.db,
		eventRepo:         e.eventRepo,
		queryClient:       e.queryClient,
		stateStore:        e.stateStore,
		suppressor:        e.suppressor,
		workerPool:        e.workerPool,
		onAlert:           e.onAlert,
		onLabelRecord:     e.onLabelRecord,
		labelRegistryRepo: e.labelRegistryRepo,
		ctx:               e.ctx,
		logger:            e.logger,
	}
}

// getOrCreateDSBucket gets or creates a per-datasource evaluation bucket.
func (e *Evaluator) getOrCreateDSBucket(dsID uint) *PerDatasourceEvaluator {
	if v, ok := e.perDS.Load(dsID); ok {
		if pde, ok := v.(*PerDatasourceEvaluator); ok {
			return pde
		}
	}
	bucket := NewPerDatasourceEvaluator(e.ctx, dsID, e.logger)
	actual, loaded := e.perDS.LoadOrStore(dsID, bucket)
	if loaded {
		bucket.cancel()
		if pde, ok := actual.(*PerDatasourceEvaluator); ok {
			return pde
		}
	}
	return bucket
}

// removeDSBucket removes a per-datasource evaluation bucket.
func (e *Evaluator) removeDSBucket(dsID uint) {
	if v, loaded := e.perDS.LoadAndDelete(dsID); loaded {
		if pde, ok := v.(*PerDatasourceEvaluator); ok {
			pde.Stop()
		}
	}
}

// listDSBuckets returns all per-datasource buckets (for testing/monitoring).
func (e *Evaluator) listDSBuckets() []*PerDatasourceEvaluator {
	var buckets []*PerDatasourceEvaluator
	e.perDS.Range(func(_, v any) bool {
		if pde, ok := v.(*PerDatasourceEvaluator); ok {
			buckets = append(buckets, pde)
		}
		return true
	})
	return buckets
}

// SetOnAlert sets the callback function called when a new alert event is created.
func (e *Evaluator) SetOnAlert(fn func(ctx context.Context, event *model.AlertEvent)) {
	e.onAlert = fn
}

// SetLabelRecorder sets the callback for passive label recording from alert events.
// This enables label registry population for non-Prometheus datasources (Zabbix, VictoriaLogs, etc.)
// that cannot be scraped via /api/v1/label/*/values.
func (e *Evaluator) SetLabelRecorder(fn func(datasourceID uint, labels map[string]string)) {
	e.onLabelRecord = fn
}

// SetStateStore sets the optional state persistence store.
// If nil, the evaluator operates in memory-only mode.
func (e *Evaluator) SetStateStore(ss StateStore) {
	e.stateStore = ss
}

// SetWorkerPool sets the bounded goroutine pool for onAlert callbacks.
func (e *Evaluator) SetWorkerPool(p AlertWorkerPoolSubmiter) {
	e.workerPool = p
}

// SetMuteRuleRepository sets the mute rule repository for engine-level time-window muting.
func (e *Evaluator) SetMuteRuleRepository(repo *repository.MuteRuleRepository) {
	e.suppressor.SetMuteChecker(&muteRuleRepoAdapter{repo: repo})
}

// SetLabelRegistryRepository sets the label registry repository for variable filling support.
func (e *Evaluator) SetLabelRegistryRepository(repo *repository.LabelRegistryRepository) {
	e.labelRegistryRepo = repo
}

// OnDatasourceUpdated implements service.DatasourceChangeCallback.
// When a datasource endpoint or auth config changes, this forces the evaluator
// to re-sync all rules on the next tick, ensuring stale cached datasource
// objects (with old endpoints) are replaced.
func (e *Evaluator) OnDatasourceUpdated(dsID uint) {
	e.forceSync.Store(true)
	e.forceSyncDSs.Store(dsID, true)
	e.logger.Info("datasource updated, forcing rule re-sync on next tick",
		zap.Uint("datasource_id", dsID),
	)
}

// Start begins the evaluation loop:
// 1. Load all enabled rules from DB
// 2. Start a goroutine for each rule
// 3. Periodically sync rules from DB (detect new/deleted/changed rules)
func (e *Evaluator) Start() {
	e.startOnce.Do(func() {
		e.ctx, e.cancel = context.WithCancel(context.Background())
		e.startedAt = time.Now()
		e.logger.Info("starting alert evaluator")

		// Start level suppressor GC
		e.suppressor.SetLogger(e.logger)
		e.suppressor.Start()

		if e.hashRing != nil {
			// Hash ring mode: all instances run independently,
			// each evaluates only the rules assigned to it.
			e.logger.Info("hash ring mode enabled",
				zap.String("instance_id", e.instanceID),
				zap.Int("ring_size", e.hashRing.Size()),
			)
			e.syncRules()
		} else if e.leader != nil {
			// Leader election mode: try to acquire and start renewal
			acquired := e.leader.TryAcquire(e.ctx)
			e.leader.Start(e.ctx)
			if !acquired {
				e.logger.Info("evaluator waiting for leadership (standby mode)")
			} else {
				e.syncRules()
			}
		} else {
			// Single-instance mode — no election needed
			e.syncRules()
		}

		// Periodic sync loop
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					e.logger.Error("evaluator sync loop panic recovered", zap.Any("recover", r))
				}
			}()
			ticker := time.NewTicker(e.syncInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if e.hashRing != nil {
						// Hash ring mode: always sync (ring membership may change)
						e.syncRules()
					} else if e.leader != nil && !e.leader.IsLeader() {
						// Not leader — stop any running evaluators and skip sync
						e.stopAllEvaluators()
						continue
					} else {
						e.syncRules()
					}
				case <-e.stopCh:
					return
				}
			}
		}()
	})
}

// Stop gracefully stops all evaluators.
func (e *Evaluator) Stop() {
	e.logger.Info("stopping alert evaluator")

	e.stopOnce.Do(func() {
		close(e.stopCh)

		// Release leadership first — other instances can take over immediately
		if e.leader != nil {
			e.leader.Stop()
		}

		// Stop level suppressor GC
		e.suppressor.Stop()

		// Collect evaluator references under lock, then stop them after releasing the lock.
		e.mu.Lock()
		evals := make([]*RuleEvaluator, 0, len(e.evaluators))
		for _, re := range e.evaluators {
			evals = append(evals, re)
		}
		e.evaluators = make(map[uint]*RuleEvaluator) // clear map
		e.mu.Unlock()

		for _, re := range evals {
			re.Stop()
		}

		// Cancel the evaluator context AFTER stopping evaluators so that
		// in-flight operations can finish gracefully before context is cancelled.
		if e.cancel != nil {
			e.cancel()
		}

		// Clean up per-datasource buckets
		e.perDS.Range(func(k, v any) bool {
			if pde, ok := v.(*PerDatasourceEvaluator); ok {
				pde.Stop()
			}
			return true
		})

		// Wait for background goroutines (e.g. sync loop) to exit
		e.wg.Wait()
	})

	e.logger.Info("alert evaluator stopped")
}

// Restart stops the evaluator and restarts it.
// Unlike Start(), which is a no-op after the first call due to sync.Once,
// Restart() resets the internal state so the evaluator can be started again.
// This is useful when the evaluator needs to be re-initialized after a
// configuration change or recovery from a fatal error.
func (e *Evaluator) Restart() {
	e.logger.Info("restarting alert evaluator")
	e.Stop()
	// Wait for all background goroutines to exit before resetting
	e.wg.Wait()
	// Reset sync.Once and channel so Start() can run again
	e.startOnce = sync.Once{}
	e.stopOnce = sync.Once{}
	e.stopCh = make(chan struct{})
	e.Start()
}

// shouldEvaluateRule returns true if the current instance should evaluate
// the given rule. In hash ring mode, this checks ring ownership; otherwise
// all rules are evaluated (single-leader or single-instance mode).
func (e *Evaluator) shouldEvaluateRule(ruleID uint) bool {
	if e.hashRing == nil {
		return true
	}
	key := hashring.RuleRingKey(ruleID)
	return e.hashRing.IsHit(key, e.instanceID)
}

// syncRules loads rules from DB and starts/stops evaluators as needed.
func (e *Evaluator) syncRules() {
	ctx := context.Background()

	// When a datasource endpoint changes, stop only the evaluators for the
	// affected datasource(s) so they are re-created with fresh objects.
	if e.forceSync.CompareAndSwap(true, false) {
		var affectedDSs []uint
		e.forceSyncDSs.Range(func(k, _ any) bool {
			affectedDSs = append(affectedDSs, k.(uint))
			e.forceSyncDSs.Delete(k)
			return true
		})
		if len(affectedDSs) > 0 {
			metrics.IncForceSync()
			e.logger.Info("forced sync triggered by datasource change — restarting affected evaluators",
				zap.Uints("datasource_ids", affectedDSs))
			e.stopEvaluatorsByDatasource(affectedDSs)
		}
	}

	var rules []model.AlertRule
	if err := e.db.WithContext(ctx).
		Preload("DataSource").
		Where("status = ? AND (rule_type IS NULL OR rule_type <> ?)", model.RuleStatusActive, model.RuleTypeHeartbeat).
		Find(&rules).Error; err != nil {
		e.logger.Error("failed to load alert rules for sync", zap.Error(err))
		return
	}

	activeRuleIDs := make(map[uint]bool, len(rules))

	if e.perDSEval {
		// Per-datasource bucket mode: AddRule is idempotent (replaces old evaluator)
		for i := range rules {
			rule := &rules[i]
			activeRuleIDs[rule.ID] = true
			if !e.shouldEvaluateRule(rule.ID) {
				continue // hash ring: not assigned to this instance
			}
			e.startRuleEvaluators(ctx, rule)
		}
		// Clean up removed/disabled rules from all buckets
		e.perDS.Range(func(_, v any) bool {
			bucket, ok := v.(*PerDatasourceEvaluator)
			if !ok {
				return true
			}
			bucket.rules.Range(func(k, _ any) bool {
				ruleID, ok := k.(uint)
				if !ok {
					return true
				}
				if !activeRuleIDs[ruleID] || !e.shouldEvaluateRule(ruleID) {
					bucket.RemoveRule(ruleID)
				}
				return true
			})
			return true
		})
	} else {
		// Legacy mode: individual evaluators in e.evaluators map
		for i := range rules {
			rule := &rules[i]
			activeRuleIDs[rule.ID] = true

			if !e.shouldEvaluateRule(rule.ID) {
				// Hash ring: not assigned to this instance — stop if previously running
				e.mu.RLock()
				_, exists := e.evaluators[rule.ID]
				e.mu.RUnlock()
				if exists {
					e.stopRuleEvaluator(rule.ID)
				}
				continue
			}

			e.mu.RLock()
			existing, exists := e.evaluators[rule.ID]
			e.mu.RUnlock()

			if exists {
				// NOTE: Change detection relies solely on the Version field. The Version is
				// incremented on every Update call (including batch status changes), so any
				// modification made through the application layer is detected. However, direct
				// database edits that bypass the application (e.g. manual SQL UPDATE) will not
				// be picked up until the next full restart. This is an acceptable trade-off:
				// the application guarantees version increment on all write paths.
				oldVersion := existing.rule.Version
				if oldVersion != rule.Version {
					e.logger.Info("rule updated, restarting evaluator",
						zap.Uint("rule_id", rule.ID),
						zap.String("name", rule.Name),
					)
					e.stopRuleEvaluator(rule.ID)
					e.startRuleEvaluators(ctx, rule)
				}
			} else {
				e.startRuleEvaluators(ctx, rule)
			}
		}

		// Stop evaluators for rules that are no longer enabled
		e.mu.RLock()
		toStop := make([]uint, 0)
		for ruleID := range e.evaluators {
			if !activeRuleIDs[ruleID] {
				toStop = append(toStop, ruleID)
			}
		}
		evaluatorCount := len(e.evaluators)
		e.mu.RUnlock()

		for _, ruleID := range toStop {
			e.logger.Info("stopping evaluator for removed/disabled rule", zap.Uint("rule_id", ruleID))
			e.stopRuleEvaluator(ruleID)
		}

		e.invalidateFiringCache()

		e.logger.Debug("rule sync completed",
			zap.Int("active_rules", len(rules)),
			zap.Int("evaluators", evaluatorCount),
		)
		return
	}

	e.invalidateFiringCache()

	e.logger.Debug("rule sync completed",
		zap.Int("active_rules", len(rules)),
	)
}

// startRuleEvaluators dispatches rule evaluation:
// - If perDSEval is true, uses per-datasource bucket isolation.
// - If perDSEval is false (legacy), fans out to individual evaluators.
func (e *Evaluator) startRuleEvaluators(ctx context.Context, rule *model.AlertRule) {
	dsList := e.resolveDatasources(ctx, rule)
	if len(dsList) == 0 {
		return
	}

	if e.perDSEval {
		// Per-datasource bucket mode: each DS gets an isolated bucket
		deps := e.buildEvaluatorDeps()
		for _, ds := range dsList {
			bucket := e.getOrCreateDSBucket(ds.ID)
			bucket.AddRule(rule, ds, deps)
		}
	} else {
		// Legacy mode: individual evaluators
		for _, ds := range dsList {
			e.startRuleEvaluator(rule, ds)
		}
	}
}

// resolveDatasources returns the list of datasources a rule should evaluate against.
func (e *Evaluator) resolveDatasources(ctx context.Context, rule *model.AlertRule) []*model.DataSource {
	if rule.DataSourceID != nil {
		ds := rule.DataSource
		if ds == nil {
			e.logger.Warn("rule has datasource_id but DataSource is nil after preload — skipping",
				zap.Uint("rule_id", rule.ID))
			return nil
		}
		return []*model.DataSource{ds}
	}

	if rule.DatasourceType == "" {
		e.logger.Warn("rule has no datasource_id and no datasource_type — skipping",
			zap.Uint("rule_id", rule.ID))
		return nil
	}

	dsList, err := e.dsRepo.ListEnabledByType(ctx, rule.DatasourceType)
	if err != nil {
		e.logger.Error("failed to list datasources by type for rule",
			zap.Uint("rule_id", rule.ID),
			zap.String("type", string(rule.DatasourceType)),
			zap.Error(err),
		)
		return nil
	}
	if len(dsList) == 0 {
		e.logger.Warn("no enabled datasources found for rule type",
			zap.Uint("rule_id", rule.ID),
			zap.String("type", string(rule.DatasourceType)),
		)
		return nil
	}

	result := make([]*model.DataSource, len(dsList))
	for i := range dsList {
		result[i] = &dsList[i]
	}
	return result
}

// startRuleEvaluator creates and starts a goroutine for a single rule against a specific datasource.
func (e *Evaluator) startRuleEvaluator(rule *model.AlertRule, ds *model.DataSource) {
	// Decrypt AuthConfig once at construction so every query uses plaintext
	// without mutating the model (which would corrupt the DB on save).
	ac := ds.AuthConfig
	if crypto.IsEncrypted(ac) {
		if plain, err := crypto.DecryptString(ac); err != nil {
			e.logger.Error("failed to decrypt datasource auth_config, queries may fail",
				zap.Uint("datasource_id", ds.ID), zap.Error(err))
		} else {
			ac = plain
		}
	}

	re := &RuleEvaluator{
		rule:                rule,
		datasource:          ds,
		decryptedAuthConfig: ac,
		// states is zero-value sync.Map, ready to use
		db:                  e.db,
		eventRepo:           e.eventRepo,
		queryClient:         e.queryClient,
		stateStore:          e.stateStore,
		suppressor:          e.suppressor,
		workerPool:          e.workerPool,
		onAlert:             e.onAlert,
		onLabelRecord:       e.onLabelRecord,
		labelRegistryRepo:   e.labelRegistryRepo,
		ctx:                 e.ctx,
		stopCh:              make(chan struct{}),
		logger:              e.logger.With(zap.Uint("rule_id", rule.ID), zap.String("rule_name", rule.Name)),
		fallbackSem:         make(chan struct{}, 16),
	}

	e.mu.Lock()
	if old, exists := e.evaluators[rule.ID]; exists {
		old.Stop()
	}
	e.evaluators[rule.ID] = re
	e.mu.Unlock()

	go re.Run()

	e.logger.Info("started evaluator for rule",
		zap.Uint("rule_id", rule.ID),
		zap.String("name", rule.Name),
		zap.String("datasource", ds.Name),
	)
}

// stopRuleEvaluator stops and removes an evaluator.
func (e *Evaluator) stopRuleEvaluator(ruleID uint) {
	e.mu.Lock()
	re, exists := e.evaluators[ruleID]
	if exists {
		delete(e.evaluators, ruleID)
	}
	e.mu.Unlock()

	if exists && re != nil {
		re.Stop()
	}

	// Clean up suppressor entries for this rule to prevent memory leak.
	if e.suppressor != nil {
		e.suppressor.RemoveRule(ruleID)
	}
	e.invalidateFiringCache()
}

// stopEvaluatorsByDatasource stops only the evaluators whose rules reference
// the given datasource IDs.  Used for incremental forceSync.
// Handles both legacy evaluators map and per-datasource buckets.
func (e *Evaluator) stopEvaluatorsByDatasource(dsIDs []uint) {
	dsSet := make(map[uint]bool, len(dsIDs))
	for _, id := range dsIDs {
		dsSet[id] = true
	}

	e.mu.Lock()
	toStop := make([]*RuleEvaluator, 0)
	for ruleID, re := range e.evaluators {
		if re.datasource != nil && dsSet[re.datasource.ID] {
			delete(e.evaluators, ruleID)
			toStop = append(toStop, re)
		}
	}
	e.mu.Unlock()

	for _, re := range toStop {
		re.Stop()
		if e.suppressor != nil {
			e.suppressor.RemoveRule(re.rule.ID)
		}
	}

	// Also stop and remove per-datasource buckets for affected datasource IDs
	// so they are re-created with fresh datasource objects on the next sync.
	if e.perDSEval {
		for _, dsID := range dsIDs {
			if v, loaded := e.perDS.LoadAndDelete(dsID); loaded {
				if pde, ok := v.(*PerDatasourceEvaluator); ok {
					pde.Stop()
					e.logger.Info("stopped per-datasource bucket for affected datasource",
						zap.Uint("datasource_id", dsID))
				}
			}
		}
	}

	if len(toStop) > 0 {
		e.logger.Info("stopped evaluators for affected datasources",
			zap.Int("count", len(toStop)),
			zap.Uints("datasource_ids", dsIDs),
		)
		e.invalidateFiringCache()
	}
}

// stopAllEvaluators stops all running evaluators and clears the maps.
// Used when leadership is lost to avoid duplicate evaluations.
func (e *Evaluator) stopAllEvaluators() {
	e.mu.Lock()
	evaluators := make(map[uint]*RuleEvaluator, len(e.evaluators))
	for id, re := range e.evaluators {
		evaluators[id] = re
	}
	e.evaluators = make(map[uint]*RuleEvaluator)
	e.mu.Unlock()

	for _, re := range evaluators {
		re.Stop()
	}

	// Also clear per-datasource evaluators
	if e.perDSEval {
		e.perDS.Range(func(key, value interface{}) bool {
			if bucket, ok := value.(*PerDatasourceEvaluator); ok {
				bucket.Stop()
			}
			e.perDS.Delete(key)
			return true
		})
	}

	if len(evaluators) > 0 {
		e.logger.Info("stopped all evaluators (leadership lost)", zap.Int("count", len(evaluators)))
	}
	e.invalidateFiringCache()
}
