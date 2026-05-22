package api

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// dnsmasqRegenFunc is the function the scheduler invokes. Indirected so
// tests can swap in a stub.
type dnsmasqRegenFunc func() error

// dnsmasqScheduler coalesces calls to regenerate the dnsmasq configuration
// so that bursts of mutations (e.g. CSV import of 500 nodes, bulk delete,
// rapid UI edits) collapse to at most one in-flight regeneration plus one
// trailing run that picks up everything that arrived during the first.
//
// Pattern: leading-edge + trailing-edge single-flight.
//
//  1. ScheduleRegen() sets a dirty bit (non-blocking).
//  2. The worker goroutine sees the bit, clears it, runs regen.
//  3. If anyone called ScheduleRegen() while regen was running, the bit
//     is set again — the worker loops and runs once more.
//  4. A min-interval cooldown between starts prevents thrashing under
//     pathological call patterns (e.g. UI drag-and-drop).
//
// Invariant: N concurrent ScheduleRegen() calls produce at most 2 regen
// invocations after the first run completes — the in-flight one and a
// single trailing one. No matter how many further calls arrive during
// that trailing run, the next loop iteration collapses them again.
type dnsmasqScheduler struct {
	regen       dnsmasqRegenFunc
	minInterval time.Duration
	dirty       chan struct{}
	stop        chan struct{}
	done        chan struct{}

	mu        sync.RWMutex
	lastRun   time.Time
	lastError string // empty when last run succeeded
	runCount  uint64
}

// dnsmasqMinInterval is the minimum delay between two regen starts. Picked
// small enough that interactive flows feel synchronous, large enough that
// hot loops (CSV import) collapse meaningfully.
const dnsmasqMinInterval = 200 * time.Millisecond

// newDnsmasqScheduler constructs a scheduler. The worker does NOT start
// until Start is called.
func newDnsmasqScheduler(regen dnsmasqRegenFunc, minInterval time.Duration) *dnsmasqScheduler {
	if minInterval <= 0 {
		minInterval = dnsmasqMinInterval
	}
	return &dnsmasqScheduler{
		regen:       regen,
		minInterval: minInterval,
		// Capacity 1 turns the channel into a dirty bit: a second pending
		// signal is meaningless because the worker will pick up whatever
		// state exists when it runs.
		dirty: make(chan struct{}, 1),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Start launches the worker goroutine.
func (s *dnsmasqScheduler) Start() {
	go s.loop()
}

// Schedule signals the worker to regenerate. Non-blocking. Safe to call
// from anywhere — even hot loops.
func (s *dnsmasqScheduler) Schedule() {
	select {
	case s.dirty <- struct{}{}:
	default:
		// Bit already set — the worker will see whatever DB state exists
		// when it gets there.
	}
}

// Stop signals the worker to exit and waits up to ctx for it to do so.
// In-flight regen completes; queued dirty bit is dropped.
func (s *dnsmasqScheduler) Stop(ctx context.Context) error {
	close(s.stop)
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns a snapshot of the scheduler's state.
func (s *dnsmasqScheduler) Stats() (lastRun time.Time, lastError string, runCount uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRun, s.lastError, s.runCount
}

func (s *dnsmasqScheduler) loop() {
	defer close(s.done)
	// Cooldown timer governs the MINIMUM time between two starts. We
	// rearm it after each run.
	var cooldown <-chan time.Time
	for {
		select {
		case <-s.stop:
			return
		case <-s.dirty:
			// If we're still inside the cooldown window, wait it out
			// before running. New dirty bits set during the wait remain
			// set, so we'll naturally pick them up.
			if cooldown != nil {
				select {
				case <-cooldown:
				case <-s.stop:
					return
				}
			}
			s.runOnce()
			cooldown = time.After(s.minInterval)
		}
	}
}

func (s *dnsmasqScheduler) runOnce() {
	start := time.Now()
	err := s.regen()
	s.mu.Lock()
	s.lastRun = start
	s.runCount++
	if err != nil {
		s.lastError = err.Error()
		slog.Error("dnsmasq: scheduled regen failed",
			"error", err,
			"runCount", s.runCount)
	} else {
		s.lastError = ""
	}
	s.mu.Unlock()
}

// ── Package-level singleton ───────────────────────────────────────────────

var (
	dnsmasqOnce sync.Once
	dnsmasqInst *dnsmasqScheduler
)

// StartDnsmasqScheduler initializes the global dnsmasq scheduler exactly
// once. Subsequent calls are no-ops. Call from main during startup, after
// the DB is ready.
func StartDnsmasqScheduler() {
	dnsmasqOnce.Do(func() {
		dnsmasqInst = newDnsmasqScheduler(GenerateDnsmasqConfig, dnsmasqMinInterval)
		dnsmasqInst.Start()
	})
}

// ScheduleDnsmasqRegen is the call-site API. Non-blocking. Use for all
// mutations that change the desired dnsmasq state EXCEPT operator-initiated
// explicit restart actions (which should call GenerateDnsmasqConfig
// synchronously to surface errors).
func ScheduleDnsmasqRegen() {
	if dnsmasqInst == nil {
		// Scheduler not started (test or startup ordering bug). Fall back
		// to synchronous to preserve correctness; log so it's visible.
		slog.Warn("dnsmasq: scheduler not initialized, running regen inline")
		if err := GenerateDnsmasqConfig(); err != nil {
			slog.Error("dnsmasq: inline regen failed", "error", err)
		}
		return
	}
	dnsmasqInst.Schedule()
}

// DnsmasqStats returns scheduler state for dashboard/status endpoints.
// Returns zero values when the scheduler is not initialized.
func DnsmasqStats() (lastRun time.Time, lastError string, runCount uint64) {
	if dnsmasqInst == nil {
		return time.Time{}, "", 0
	}
	return dnsmasqInst.Stats()
}

// StopDnsmasqScheduler stops the scheduler. Used by graceful shutdown.
func StopDnsmasqScheduler(ctx context.Context) error {
	if dnsmasqInst == nil {
		return nil
	}
	return dnsmasqInst.Stop(ctx)
}
