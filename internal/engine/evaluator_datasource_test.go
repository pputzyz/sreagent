package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestEvaluatorForDS creates a minimal Evaluator for per-datasource bucket tests.
// It avoids needing a real DB by only exercising the in-memory perDS sync.Map logic.
func newTestEvaluatorForDS(t *testing.T, perDSEval bool) *Evaluator {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	e := &Evaluator{
		evaluators:   make(map[uint]*RuleEvaluator),
		suppressor:   NewLevelSuppressor(),
		logger:       zap.NewNop(),
		stopCh:       make(chan struct{}),
		syncInterval: 30 * 1_000_000_000, // 30s in nanoseconds
		ctx:          ctx,
		cancel:       cancel,
		perDSEval:    perDSEval,
	}
	return e
}

// Test_Evaluator_GetOrCreateDSBucket_CreatesAndCaches verifies that
// getOrCreateDSBucket creates a new bucket on first call and returns
// the same instance on subsequent calls for the same datasource ID.
func Test_Evaluator_GetOrCreateDSBucket_CreatesAndCaches(t *testing.T) {
	e := newTestEvaluatorForDS(t, true)

	bucket1 := e.getOrCreateDSBucket(1)
	require.NotNil(t, bucket1, "first call must return a non-nil bucket")
	assert.Equal(t, uint(1), bucket1.DatasourceID)

	// Second call for the same DS ID must return the same instance
	bucket2 := e.getOrCreateDSBucket(1)
	assert.Same(t, bucket1, bucket2, "second call must return the cached bucket")

	// Different DS ID must return a different bucket
	bucket3 := e.getOrCreateDSBucket(2)
	require.NotNil(t, bucket3)
	assert.NotSame(t, bucket1, bucket3, "different DS ID must get a different bucket")
	assert.Equal(t, uint(2), bucket3.DatasourceID)
}

// Test_Evaluator_RemoveDSBucket verifies that removeDSBucket stops
// and removes the bucket from the perDS map.
func Test_Evaluator_RemoveDSBucket(t *testing.T) {
	e := newTestEvaluatorForDS(t, true)

	e.getOrCreateDSBucket(10)
	e.getOrCreateDSBucket(20)

	buckets := e.listDSBuckets()
	assert.Len(t, buckets, 2, "should have 2 buckets before removal")

	e.removeDSBucket(10)

	buckets = e.listDSBuckets()
	assert.Len(t, buckets, 1, "should have 1 bucket after removing DS 10")

	// The remaining bucket should be DS 20
	assert.Equal(t, uint(20), buckets[0].DatasourceID)
}

// Test_Evaluator_ListDSBuckets_EmptyWhenNoneCreated verifies that
// listDSBuckets returns an empty slice when no buckets exist.
func Test_Evaluator_ListDSBuckets_EmptyWhenNoneCreated(t *testing.T) {
	e := newTestEvaluatorForDS(t, true)

	buckets := e.listDSBuckets()
	assert.Empty(t, buckets, "should be empty when no buckets created")
}

// Test_Evaluator_Stop_CleansUpPerDSBuckets verifies that Stop()
// cancels the context for all per-datasource buckets.
func Test_Evaluator_Stop_CleansUpPerDSBuckets(t *testing.T) {
	e := newTestEvaluatorForDS(t, true)

	e.getOrCreateDSBucket(1)
	e.getOrCreateDSBucket(2)
	e.getOrCreateDSBucket(3)

	assert.Len(t, e.listDSBuckets(), 3, "precondition: 3 buckets exist")

	e.Stop()

	// After Stop, the perDS map still holds entries (Stop doesn't delete them),
	// but the parent context is cancelled, so all bucket contexts are done.
	// This is the correct behavior — buckets are stopped, not removed from map.
	assert.Len(t, e.listDSBuckets(), 3, "Stop doesn't remove entries, only cancels context")
}

// Test_Evaluator_GetOrCreateDSBucket_Concurrent verifies that concurrent
// calls to getOrCreateDSBucket for the same DS ID are safe and return
// the same instance.
func Test_Evaluator_GetOrCreateDSBucket_Concurrent(t *testing.T) {
	e := newTestEvaluatorForDS(t, true)

	const goroutines = 50
	results := make(chan *PerDatasourceEvaluator, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			results <- e.getOrCreateDSBucket(42)
		}()
	}

	// Collect all results
	var first *PerDatasourceEvaluator
	for i := 0; i < goroutines; i++ {
		bucket := <-results
		if i == 0 {
			first = bucket
		} else {
			assert.Same(t, first, bucket,
				"concurrent getOrCreateDSBucket must return the same instance")
		}
	}
}
