package engine

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_Evaluator_ConcurrentLoadOrStore_Consistency verifies that LoadOrStore
// returns the same stateLock for the same fingerprint under contention.
func Test_Evaluator_ConcurrentLoadOrStore_Consistency(t *testing.T) {
	re := &RuleEvaluator{}
	fp := "contested-fp"

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	pointers := make([]*stateLock, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			sl := re.lockState(fp)
			pointers[id] = sl
		}(g)
	}

	wg.Wait()

	// All goroutines must have received the same *stateLock pointer
	for i := 1; i < goroutines; i++ {
		assert.Same(t, pointers[0], pointers[i],
			"LoadOrStore must return the same pointer for the same key")
	}
}

// Test_Evaluator_ConcurrentUpdate_NoLost verifies that concurrent updates
// to the same fingerprint via lockState do not lose writes (run with -race).
func Test_Evaluator_ConcurrentUpdate_NoLost(t *testing.T) {
	re := &RuleEvaluator{}
	fp := "test-fingerprint"

	const goroutines = 100
	const increments = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < increments; i++ {
				sl := re.lockState(fp)
				sl.mu.Lock()
				if sl.state == nil {
					sl.state = &AlertState{
						Labels: map[string]string{"alertname": "Test"},
						Status: "firing",
					}
				}
				sl.state.Value++
				sl.mu.Unlock()
			}
		}()
	}

	wg.Wait()

	sl := re.lockState(fp)
	sl.mu.Lock()
	defer sl.mu.Unlock()

	assert.NotNil(t, sl.state, "state must not be nil after concurrent writes")
	assert.Equal(t, float64(goroutines*increments), sl.state.Value,
		"no updates should be lost under concurrent access")
}

// Test_Evaluator_ConcurrentDifferentFingerprints verifies that concurrent
// updates to different fingerprints do not interfere with each other.
func Test_Evaluator_ConcurrentDifferentFingerprints(t *testing.T) {
	re := &RuleEvaluator{}

	const fingerprints = 50
	const goroutinesPerFP = 10
	const increments = 20

	var wg sync.WaitGroup
	wg.Add(fingerprints * goroutinesPerFP)

	for fpIdx := 0; fpIdx < fingerprints; fpIdx++ {
		fp := fmt.Sprintf("fp-%d", fpIdx)
		for g := 0; g < goroutinesPerFP; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < increments; i++ {
					sl := re.lockState(fp)
					sl.mu.Lock()
					if sl.state == nil {
						sl.state = &AlertState{
							Labels: map[string]string{"fingerprint": fp},
							Status: "firing",
						}
					}
					sl.state.Value++
					sl.mu.Unlock()
				}
			}()
		}
	}

	wg.Wait()

	// Verify each fingerprint has the correct total
	for fpIdx := 0; fpIdx < fingerprints; fpIdx++ {
		fp := fmt.Sprintf("fp-%d", fpIdx)
		sl := re.lockState(fp)
		sl.mu.Lock()
		assert.NotNil(t, sl.state, "state for %s must not be nil", fp)
		assert.Equal(t, float64(goroutinesPerFP*increments), sl.state.Value,
			"no updates should be lost for fingerprint %s", fp)
		sl.mu.Unlock()
	}
}

// Test_Evaluator_ConcurrentRangeAndStore verifies that Range and Store
// can be called concurrently without panics or races.
func Test_Evaluator_ConcurrentRangeAndStore(t *testing.T) {
	re := &RuleEvaluator{}

	const writers = 20
	const readers = 20
	const ops = 100

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	// Writers: store new stateLock entries
	for w := 0; w < writers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				fp := fmt.Sprintf("w%d-fp%d", id, i)
				sl := re.lockState(fp)
				sl.mu.Lock()
				sl.state = &AlertState{
					Labels: map[string]string{"writer": fmt.Sprintf("%d", id)},
					Status: "pending",
					Value:  float64(i),
				}
				sl.mu.Unlock()
			}
		}(w)
	}

	// Readers: Range over all entries
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				re.states.Range(func(k, v any) bool {
					sl := v.(*stateLock)
					sl.mu.Lock()
					_ = sl.state // just read
					sl.mu.Unlock()
					return true
				})
			}
		}()
	}

	wg.Wait()

	// Verify total count
	var count atomic.Int32
	re.states.Range(func(k, v any) bool {
		count.Add(1)
		return true
	})
	assert.Equal(t, int32(writers*ops), count.Load(),
		"total stored entries should equal writers * ops")
}
