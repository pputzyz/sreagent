package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewAlertWorkerPool_DefaultConcurrency(t *testing.T) {
	p := NewAlertWorkerPool(0, zap.NewNop())
	if cap(p.sem) != 64 {
		t.Errorf("default concurrency = %d, want 64", cap(p.sem))
	}
}

func TestNewAlertWorkerPool_NegativeConcurrency(t *testing.T) {
	p := NewAlertWorkerPool(-5, zap.NewNop())
	if cap(p.sem) != 64 {
		t.Errorf("negative concurrency = %d, want 64", cap(p.sem))
	}
}

func TestNewAlertWorkerPool_CustomConcurrency(t *testing.T) {
	p := NewAlertWorkerPool(16, zap.NewNop())
	if cap(p.sem) != 16 {
		t.Errorf("custom concurrency = %d, want 16", cap(p.sem))
	}
}

func TestAlertWorkerPool_Submit_ExecutesTask(t *testing.T) {
	p := NewAlertWorkerPool(4, zap.NewNop())
	var done atomic.Bool

	ok := p.Submit(context.Background(), func(ctx context.Context) {
		done.Store(true)
	})
	if !ok {
		t.Fatal("Submit returned false, want true")
	}
	p.Wait()

	if !done.Load() {
		t.Error("task was not executed")
	}
}

func TestAlertWorkerPool_Submit_FullPool(t *testing.T) {
	p := NewAlertWorkerPool(1, zap.NewNop())
	blocker := make(chan struct{})
	started := make(chan struct{})

	// Fill the pool
	ok1 := p.Submit(context.Background(), func(ctx context.Context) {
		close(started)
		<-blocker
	})
	if !ok1 {
		t.Fatal("first Submit returned false")
	}
	<-started

	// Pool is full — next submit should fail
	ok2 := p.Submit(context.Background(), func(ctx context.Context) {
		t.Error("this task should not run")
	})
	if ok2 {
		t.Error("Submit on full pool returned true, want false")
	}

	close(blocker)
	p.Wait()
}

func TestAlertWorkerPool_Submit_PanicRecovery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	p := NewAlertWorkerPool(4, logger)

	var done atomic.Bool
	// This task panics — pool should survive
	p.Submit(context.Background(), func(ctx context.Context) {
		panic("boom")
	})
	// This task should still run
	p.Submit(context.Background(), func(ctx context.Context) {
		done.Store(true)
	})
	p.Wait()

	if !done.Load() {
		t.Error("task after panic was not executed")
	}
}

func TestAlertWorkerPool_Submit_AddsDeadline(t *testing.T) {
	p := NewAlertWorkerPool(4, zap.NewNop())
	var hasDeadline atomic.Bool

	p.Submit(context.Background(), func(ctx context.Context) {
		_, ok := ctx.Deadline()
		hasDeadline.Store(ok)
	})
	p.Wait()

	if !hasDeadline.Load() {
		t.Error("expected context to have deadline")
	}
}

func TestAlertWorkerPool_Submit_PreservesExistingDeadline(t *testing.T) {
	p := NewAlertWorkerPool(4, zap.NewNop())
	// Set a very long deadline — the pool should NOT override it
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	var deadlineIsHour atomic.Bool
	p.Submit(ctx, func(ctx context.Context) {
		dl, ok := ctx.Deadline()
		if ok && time.Until(dl) > 30*time.Minute {
			deadlineIsHour.Store(true)
		}
	})
	p.Wait()

	if !deadlineIsHour.Load() {
		t.Error("expected original deadline to be preserved")
	}
}

func TestAlertWorkerPool_Wait_Empty(t *testing.T) {
	p := NewAlertWorkerPool(4, zap.NewNop())
	// Wait on empty pool should return immediately
	done := make(chan struct{})
	go func() {
		p.Wait()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Error("Wait on empty pool did not return promptly")
	}
}

func TestAlertWorkerPool_ConcurrentSubmit(t *testing.T) {
	p := NewAlertWorkerPool(8, zap.NewNop())
	var count atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Submit(context.Background(), func(ctx context.Context) {
				count.Add(1)
			})
		}()
	}
	wg.Wait()
	p.Wait()

	if count.Load() == 0 {
		t.Error("no tasks executed")
	}
}
