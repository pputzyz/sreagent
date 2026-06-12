package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

func newLarkCardTestRepo(t *testing.T) *LarkCardRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// sqlite :memory: is per-connection — without this, concurrent goroutines
	// get fresh connections pointing at EMPTY databases. A single shared
	// connection still lets the statements of two goroutines interleave,
	// which is exactly the race the transactional fix must survive.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.LarkCardEntity{}, &model.LarkCardMessage{}))
	t.Cleanup(func() {
		if sqlDB, _ := db.DB(); sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return NewLarkCardRepository(db)
}

// Test_IncrementSequence_Concurrent_NoDuplicates is the regression test for
// the read-after-update race: a bare `UPDATE seq=seq+1` followed by an
// independent SELECT lets two concurrent callers read the SAME final value —
// and CardKit rejects duplicate sequences (300317). The transactional
// implementation must hand every caller a unique, monotonically increasing
// number.
func Test_IncrementSequence_Concurrent_NoDuplicates(t *testing.T) {
	repo := newLarkCardTestRepo(t)

	eventID := uint(1)
	entity := &model.LarkCardEntity{
		EventID:    &eventID,
		CardID:     "card_test_1",
		Sequence:   1,
		CardStatus: "active",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.CreateEntity(context.Background(), entity))

	const workers = 50
	results := make(chan int64, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seq, err := repo.IncrementSequence(context.Background(), entity.ID)
			if err != nil {
				results <- -1 // sqlite may briefly contend; recorded as failure
				return
			}
			results <- seq
		}()
	}
	wg.Wait()
	close(results)

	seen := make(map[int64]bool)
	failures := 0
	for seq := range results {
		if seq < 0 {
			failures++
			continue
		}
		assert.False(t, seen[seq], "sequence %d handed out twice — CardKit would reject the duplicate with 300317", seq)
		seen[seq] = true
	}
	assert.Equal(t, 0, failures, "all increments should succeed")
	assert.Len(t, seen, workers, "every caller must receive a unique sequence")
}

// Test_JumpSequence_AdvancesPastRemote verifies the 300317 re-sync path jumps
// the counter forward by the requested step.
func Test_JumpSequence_AdvancesPastRemote(t *testing.T) {
	repo := newLarkCardTestRepo(t)

	eventID := uint(2)
	entity := &model.LarkCardEntity{
		EventID:    &eventID,
		CardID:     "card_test_2",
		Sequence:   5,
		CardStatus: "active",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.CreateEntity(context.Background(), entity))

	seq, err := repo.JumpSequence(context.Background(), entity.ID, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(105), seq)
}

// Test_GetEntityByEventID_OnlyActive ensures superseded/expired entities are
// not returned (re-issue depends on this).
func Test_GetEntityByEventID_OnlyActive(t *testing.T) {
	repo := newLarkCardTestRepo(t)
	ctx := context.Background()

	eventID := uint(3)
	old := &model.LarkCardEntity{
		EventID: &eventID, CardID: "old", Sequence: 1,
		CardStatus: "active", ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.CreateEntity(ctx, old))
	require.NoError(t, repo.UpdateStatus(ctx, old.ID, "superseded"))

	fresh := &model.LarkCardEntity{
		EventID: &eventID, CardID: "fresh", Sequence: 1,
		CardStatus: "active", ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.CreateEntity(ctx, fresh))

	got, err := repo.GetEntityByEventID(ctx, eventID)
	require.NoError(t, err)
	assert.Equal(t, "fresh", got.CardID, "only the active entity may be returned")
}
