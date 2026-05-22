package api

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitForCount busy-waits up to timeout for getter to return at least want.
// Test scheduler uses time-based coalescing so we can't deterministically
// hook into "regen completed" without instrumenting the production code.
func waitForCount(t *testing.T, want uint64, timeout time.Duration, getter func() uint64) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if getter() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for count >= %d, got %d", want, getter())
}

func TestSchedulerCoalescesBurst(t *testing.T) {
	// Block the first regen until we've enqueued a burst, then release.
	// The burst MUST collapse into a single trailing run regardless of how
	// many calls landed during the first run.
	var runs atomic.Uint64
	release := make(chan struct{})
	firstRunStarted := make(chan struct{})
	var firstOnce sync.Once

	s := newDnsmasqScheduler(func() error {
		n := runs.Add(1)
		if n == 1 {
			firstOnce.Do(func() { close(firstRunStarted) })
			<-release
		}
		return nil
	}, 1*time.Millisecond) // tiny cooldown — we're testing coalesce, not cooldown
	s.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Stop(ctx)
	})

	// First schedule kicks off run 1.
	s.Schedule()
	select {
	case <-firstRunStarted:
	case <-time.After(time.Second):
		t.Fatal("first regen never started")
	}

	// Now fire 100 enqueues while run 1 is blocked. They must collapse to
	// exactly one trailing run (because the dirty bit is a single slot).
	for i := 0; i < 100; i++ {
		s.Schedule()
	}

	close(release)
	waitForCount(t, 2, time.Second, runs.Load)
	// Give a generous settle window — any further regen here is a bug.
	time.Sleep(50 * time.Millisecond)
	if got := runs.Load(); got != 2 {
		t.Fatalf("burst should collapse to 2 runs (leading + trailing), got %d", got)
	}
}

func TestSchedulerIdleAfterDrain(t *testing.T) {
	// After all dirty bits drain, the worker must NOT run regen on its own.
	var runs atomic.Uint64
	s := newDnsmasqScheduler(func() error {
		runs.Add(1)
		return nil
	}, 1*time.Millisecond)
	s.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Stop(ctx)
	})

	s.Schedule()
	waitForCount(t, 1, time.Second, runs.Load)

	// 200ms of nothing should produce no further runs.
	time.Sleep(200 * time.Millisecond)
	if got := runs.Load(); got != 1 {
		t.Fatalf("idle scheduler ran %d times, want 1", got)
	}
}

func TestSchedulerRecordsLastError(t *testing.T) {
	var runs atomic.Uint64
	wantErr := errors.New("disk full")
	s := newDnsmasqScheduler(func() error {
		n := runs.Add(1)
		if n == 1 {
			return wantErr
		}
		return nil
	}, 1*time.Millisecond)
	s.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Stop(ctx)
	})

	s.Schedule()
	waitForCount(t, 1, time.Second, runs.Load)

	_, lastErr, count := s.Stats()
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if lastErr != wantErr.Error() {
		t.Fatalf("lastError = %q, want %q", lastErr, wantErr.Error())
	}

	// A subsequent successful run must clear lastError.
	s.Schedule()
	waitForCount(t, 2, time.Second, runs.Load)
	_, lastErr2, _ := s.Stats()
	if lastErr2 != "" {
		t.Fatalf("after successful run, lastError = %q, want empty", lastErr2)
	}
}

func TestSchedulerStopWaitsForInflightRegen(t *testing.T) {
	// In-flight regen must complete before Stop returns.
	released := make(chan struct{})
	started := make(chan struct{})
	completed := atomic.Bool{}
	s := newDnsmasqScheduler(func() error {
		close(started)
		<-released
		completed.Store(true)
		return nil
	}, 1*time.Millisecond)
	s.Start()

	s.Schedule()
	<-started

	stopDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		stopDone <- s.Stop(ctx)
	}()

	// Stop is blocked on the worker, which is blocked on `released`.
	select {
	case <-stopDone:
		t.Fatal("Stop returned before in-flight regen finished")
	case <-time.After(50 * time.Millisecond):
		// expected
	}

	close(released)
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after regen completed")
	}

	if !completed.Load() {
		t.Fatal("regen did not complete before Stop returned")
	}
}

func TestSchedulerCooldownDelaysSecondRun(t *testing.T) {
	// Two enqueues spaced beyond the cooldown should still see the second
	// run delayed by AT LEAST the cooldown after the first completed.
	starts := make([]time.Time, 0, 2)
	var mu sync.Mutex
	s := newDnsmasqScheduler(func() error {
		mu.Lock()
		starts = append(starts, time.Now())
		mu.Unlock()
		return nil
	}, 100*time.Millisecond)
	s.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Stop(ctx)
	})

	s.Schedule()
	// Wait for run 1 to land
	for {
		mu.Lock()
		n := len(starts)
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Enqueue immediately — should be delayed by the cooldown
	s.Schedule()

	// Allow up to 500ms for run 2
	for i := 0; i < 100; i++ {
		mu.Lock()
		n := len(starts)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	if len(starts) < 2 {
		mu.Unlock()
		t.Fatalf("second run never happened")
	}
	gap := starts[1].Sub(starts[0])
	mu.Unlock()
	if gap < 100*time.Millisecond {
		t.Fatalf("second run started %v after first; cooldown should enforce ≥ 100ms", gap)
	}
}
