package api

import (
	"testing"
)

func TestAutoDiscoverEnabledDefaultsTrue(t *testing.T) {
	t.Setenv("PXE_AUTO_DISCOVER", "")
	if !autoDiscoverEnabled() {
		t.Error("default must be ON — discovery is the whole point of the platform")
	}
}

func TestAutoDiscoverEnabledRecognizesOffStrings(t *testing.T) {
	cases := []string{"0", "false", "False", "FALSE", "no", "No"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("PXE_AUTO_DISCOVER", v)
			if autoDiscoverEnabled() {
				t.Errorf("PXE_AUTO_DISCOVER=%q must disable", v)
			}
		})
	}
}

func TestAutoDiscoverEnabledStaysOnForUnknownValues(t *testing.T) {
	// Same typo-safe convention as IPMI_AUTO_POWER_CYCLE / PXE_REQUIRE_TOKEN.
	t.Setenv("PXE_AUTO_DISCOVER", "perhaps")
	if !autoDiscoverEnabled() {
		t.Error("unknown env value must NOT silently disable")
	}
}

func TestMacSuffix(t *testing.T) {
	cases := []struct {
		mac, want string
	}{
		{"aa:bb:cc:dd:ee:ff", "ddeeff"},
		{"AA:BB:CC:DD:EE:FF", "DDEEFF"}, // case preserved; caller normalises
		{"aa-bb-cc-dd-ee-ff", "ddeeff"},
		{"aabbccddeeff", "ddeeff"},
		{"aa:bb", "aabb"}, // shorter than 6 — return as-is, no panic
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.mac, func(t *testing.T) {
			if got := macSuffix(tc.mac); got != tc.want {
				t.Errorf("macSuffix(%q) = %q, want %q", tc.mac, got, tc.want)
			}
		})
	}
}

func TestNodeStatusDiscoveredIsKnown(t *testing.T) {
	if !IsKnownNodeStatus(NodeStatusDiscovered) {
		t.Error("discovered must be a recognized status")
	}
}

func TestDiscoveredTransitions(t *testing.T) {
	// discovered → operator approvals.
	allowed := []string{
		NodeStatusPending,
		NodeStatusInstalling,
		NodeStatusMaintenance,
		NodeStatusError,
	}
	for _, target := range allowed {
		if !canTransition(NodeStatusDiscovered, target) {
			t.Errorf("expected discovered -> %s to be legal", target)
		}
	}

	// discovered MUST NOT transition straight to active / offline — the
	// node hasn't actually been provisioned yet, those would be lies.
	forbidden := []string{NodeStatusActive, NodeStatusOffline}
	for _, target := range forbidden {
		if canTransition(NodeStatusDiscovered, target) {
			t.Errorf("expected discovered -> %s to be ILLEGAL", target)
		}
	}
}

func TestEmptyStatusCanReachDiscovered(t *testing.T) {
	// discoverNode inserts with Status=discovered via gorm Create (no
	// SetNodeStatus call). The transition map's "" source must allow it
	// or future audit / SetNodeStatus operations targeting a row that
	// somehow has empty status couldn't promote to discovered.
	if !canTransition("", NodeStatusDiscovered) {
		t.Error("empty status must be able to reach discovered")
	}
}
