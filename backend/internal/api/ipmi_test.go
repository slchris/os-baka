package api

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/os-baka/backend/internal/model"
)

func TestBuildIpmitoolArgsBasicShape(t *testing.T) {
	n := &model.Node{
		IPMIAddress:  "10.0.0.50",
		IPMIUsername: "ADMIN",
	}
	args := buildIpmitoolArgs(n, "power", "status")

	// Must include the interface/host/user trio and the network-bounded
	// retry flags. Drift on these would make a flaky BMC able to hang the
	// caller indefinitely (regression test for the "no -N -R" original).
	required := []string{"-I", "lanplus", "-N", "5", "-R", "1", "-H", "10.0.0.50", "-U", "ADMIN", "power", "status"}
	for _, want := range required {
		if !slices.Contains(args, want) {
			t.Errorf("expected %q in argv, got %v", want, args)
		}
	}
}

func TestBuildIpmitoolArgsAddsPasswordWhenSet(t *testing.T) {
	// With no FIELD_ENCRYPTION_KEY set, EncryptField/DecryptField are
	// passthrough — so a literal password is fine for the test.
	n := &model.Node{
		IPMIAddress:  "10.0.0.50",
		IPMIUsername: "ADMIN",
		IPMIPassword: "s3cret",
	}
	args := buildIpmitoolArgs(n, "power", "status")
	pwIdx := slices.Index(args, "-P")
	if pwIdx == -1 {
		t.Fatalf("expected -P flag, got argv %v", args)
	}
	if args[pwIdx+1] != "s3cret" {
		t.Errorf("password not passed through correctly, got %q", args[pwIdx+1])
	}
}

func TestBuildIpmitoolArgsOmitsPasswordWhenEmpty(t *testing.T) {
	n := &model.Node{
		IPMIAddress:  "10.0.0.50",
		IPMIUsername: "ADMIN",
	}
	args := buildIpmitoolArgs(n, "power", "status")
	if slices.Contains(args, "-P") {
		t.Errorf("expected no -P flag for empty password, got %v", args)
	}
}

func TestBuildIpmitoolArgsAddsUntrustedFlag(t *testing.T) {
	n := &model.Node{
		IPMIAddress:        "10.0.0.50",
		IPMIUsername:       "ADMIN",
		IPMIAllowUntrusted: true,
	}
	args := buildIpmitoolArgs(n, "power", "cycle")
	cIdx := slices.Index(args, "-C")
	if cIdx == -1 || args[cIdx+1] != "0" {
		t.Errorf("expected -C 0 when AllowUntrusted, got %v", args)
	}
}

func TestAutoPowerCycleEnabledDefaultsTrue(t *testing.T) {
	t.Setenv("IPMI_AUTO_POWER_CYCLE", "")
	if !autoPowerCycleEnabled() {
		t.Error("default must be ON — disabling auto-cycle is opt-in")
	}
}

func TestAutoPowerCycleEnabledRecognizesOffStrings(t *testing.T) {
	cases := []string{"0", "false", "False", "FALSE", "no", "No"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("IPMI_AUTO_POWER_CYCLE", v)
			if autoPowerCycleEnabled() {
				t.Errorf("IPMI_AUTO_POWER_CYCLE=%q must disable feature", v)
			}
		})
	}
}

func TestAutoPowerCycleEnabledStaysOnForUnknownValues(t *testing.T) {
	// A typo or unrecognized value must NOT silently disable security-
	// affecting features (same convention as PXE_REQUIRE_TOKEN).
	t.Setenv("IPMI_AUTO_POWER_CYCLE", "maybe")
	if !autoPowerCycleEnabled() {
		t.Error("unknown env value must keep feature on")
	}
}

func TestTriggerPowerCycleSkipsWhenNoBMC(t *testing.T) {
	t.Setenv("IPMI_AUTO_POWER_CYCLE", "")
	// Make sure no goroutine fires — assert no ipmitool stub invocation.
	var called atomic.Int32
	stubRunIPMICommand(t, func(ctx context.Context, args []string) (string, error) {
		called.Add(1)
		return "", nil
	})

	n := &model.Node{} // no IPMIAddress
	got := TriggerPowerCycle(n)
	if got != powerCycleOutcomeSkippedNoBMC {
		t.Errorf("outcome = %q, want %q", got, powerCycleOutcomeSkippedNoBMC)
	}

	// Give any errant goroutine a chance to leak.
	time.Sleep(50 * time.Millisecond)
	if called.Load() != 0 {
		t.Errorf("ipmitool stub was called %d times despite no IPMIAddress", called.Load())
	}
}

func TestTriggerPowerCycleDisabledReturnsDisabled(t *testing.T) {
	t.Setenv("IPMI_AUTO_POWER_CYCLE", "false")
	var called atomic.Int32
	stubRunIPMICommand(t, func(ctx context.Context, args []string) (string, error) {
		called.Add(1)
		return "", nil
	})

	n := &model.Node{IPMIAddress: "10.0.0.50", IPMIUsername: "ADMIN"}
	if got := TriggerPowerCycle(n); got != powerCycleOutcomeDisabled {
		t.Errorf("outcome = %q, want %q", got, powerCycleOutcomeDisabled)
	}
	time.Sleep(50 * time.Millisecond)
	if called.Load() != 0 {
		t.Errorf("ipmitool stub called %d times despite IPMI_AUTO_POWER_CYCLE=false", called.Load())
	}
}

func TestTriggerPowerCycleScheduledInvokesIpmitool(t *testing.T) {
	t.Setenv("IPMI_AUTO_POWER_CYCLE", "")

	var (
		mu        sync.Mutex
		gotArgs   []string
		wg        sync.WaitGroup
	)
	wg.Add(1)
	stubRunIPMICommand(t, func(ctx context.Context, args []string) (string, error) {
		mu.Lock()
		gotArgs = append([]string(nil), args...)
		mu.Unlock()
		wg.Done()
		return "Chassis Power Control: Cycle", nil
	})

	n := &model.Node{
		IPMIAddress:  "10.0.0.50",
		IPMIUsername: "ADMIN",
	}
	if got := TriggerPowerCycle(n); got != powerCycleOutcomeScheduled {
		t.Fatalf("outcome = %q, want %q", got, powerCycleOutcomeScheduled)
	}

	// Wait for the background goroutine to call the stub.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine never invoked the stub")
	}

	mu.Lock()
	defer mu.Unlock()
	// Argv must end with "power cycle" — this is the contract the rebuild
	// flow depends on. Anything else (status, reset) would be wrong.
	if len(gotArgs) < 2 || gotArgs[len(gotArgs)-2] != "power" || gotArgs[len(gotArgs)-1] != "cycle" {
		t.Errorf("expected argv to end with [power cycle], got %v", gotArgs)
	}
	if !strings.Contains(strings.Join(gotArgs, " "), "-H 10.0.0.50") {
		t.Errorf("argv missing -H 10.0.0.50: %v", gotArgs)
	}
}

// stubRunIPMICommand temporarily replaces the package-level runIPMICommand
// with the given function for the duration of the test. Tests can't
// reasonably run real ipmitool in CI, and even if they could, hitting a
// real BMC is wildly out of scope.
func stubRunIPMICommand(t *testing.T, fn func(context.Context, []string) (string, error)) {
	t.Helper()
	prev := runIPMICommand
	runIPMICommand = fn
	t.Cleanup(func() {
		runIPMICommand = prev
	})
}
