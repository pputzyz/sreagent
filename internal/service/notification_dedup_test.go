package service

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_notifDedup_TrySend_first_call_returns_true(t *testing.T) {
	d := newNotifDedup()
	assert.True(t, d.TrySend("key1"))
}

func Test_notifDedup_TrySend_duplicate_returns_false(t *testing.T) {
	d := newNotifDedup()
	assert.True(t, d.TrySend("key1"))
	assert.False(t, d.TrySend("key1"))
}

func Test_notifDedup_TrySend_different_keys_independent(t *testing.T) {
	d := newNotifDedup()
	assert.True(t, d.TrySend("key1"))
	assert.True(t, d.TrySend("key2"))
	assert.False(t, d.TrySend("key1"))
	assert.False(t, d.TrySend("key2"))
	assert.True(t, d.TrySend("key3"))
}

func Test_notifDedup_TrySend_v1_v2_keys_are_distinct(t *testing.T) {
	// v1 and v2 keys for the same event should be independent
	d := newNotifDedup()
	v1Key := fmt.Sprintf("v1:%d:%d:%s", 10, 5, "abc123")
	v2Key := fmt.Sprintf("v2:%d:%d:%s", 20, 3, "abc123")

	assert.True(t, d.TrySend(v1Key))
	assert.True(t, d.TrySend(v2Key))
	assert.False(t, d.TrySend(v1Key))
	assert.False(t, d.TrySend(v2Key))
}

func Test_notifDedup_TrySend_concurrent_access(t *testing.T) {
	d := newNotifDedup()
	var wg sync.WaitGroup
	results := make([]bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = d.TrySend("shared-key")
		}(i)
	}
	wg.Wait()

	// Exactly one goroutine should have gotten true
	trueCount := 0
	for _, r := range results {
		if r {
			trueCount++
		}
	}
	assert.Equal(t, 1, trueCount, "exactly one goroutine should win the dedup race")
}

func Test_notifDedup_TrySend_concurrent_different_keys(t *testing.T) {
	d := newNotifDedup()
	var wg sync.WaitGroup
	results := make([]bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = d.TrySend(fmt.Sprintf("key-%d", idx))
		}(i)
	}
	wg.Wait()

	// All goroutines should have gotten true (all different keys)
	for i, r := range results {
		assert.True(t, r, "goroutine %d should have succeeded with unique key", i)
	}
}
