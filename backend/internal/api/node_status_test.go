package api

import (
	"testing"
)

func TestIsKnownNodeStatus(t *testing.T) {
	known := []string{
		NodeStatusPending,
		NodeStatusInstalling,
		NodeStatusActive,
		NodeStatusMaintenance,
		NodeStatusError,
		NodeStatusOffline,
	}
	for _, s := range known {
		if !IsKnownNodeStatus(s) {
			t.Errorf("expected %q to be known", s)
		}
	}

	unknown := []string{"", "rebuilding", "ACTIVE", "online", "Offline", "deploying", " active"}
	for _, s := range unknown {
		if IsKnownNodeStatus(s) {
			t.Errorf("expected %q to be unknown", s)
		}
	}
}

func TestCanTransitionSelfIsAlwaysAllowed(t *testing.T) {
	for _, s := range []string{
		NodeStatusPending, NodeStatusInstalling, NodeStatusActive,
		NodeStatusMaintenance, NodeStatusError, NodeStatusOffline, "",
	} {
		if !canTransition(s, s) {
			t.Errorf("self-transition %q must be allowed (no-op)", s)
		}
	}
}

func TestCanTransitionLegalEdges(t *testing.T) {
	// A representative sampling of operationally important edges. If we
	// add or remove edges in nodeStatusTransitions, this test must move
	// in lockstep — failing here means somebody changed lifecycle policy
	// without updating the safety net.
	legal := []struct{ from, to string }{
		// Brand new -> any initial assignment
		{"", NodeStatusPending},
		{"", NodeStatusActive},
		{"", NodeStatusInstalling},

		// Provisioning happy path
		{NodeStatusPending, NodeStatusInstalling},
		{NodeStatusInstalling, NodeStatusActive},

		// Provisioning failure paths
		{NodeStatusInstalling, NodeStatusError},
		{NodeStatusInstalling, NodeStatusPending},

		// Heartbeat oscillation
		{NodeStatusActive, NodeStatusOffline},
		{NodeStatusOffline, NodeStatusActive},

		// Operator-driven
		{NodeStatusActive, NodeStatusMaintenance},
		{NodeStatusMaintenance, NodeStatusActive},
		{NodeStatusActive, NodeStatusInstalling}, // rebuild
		{NodeStatusOffline, NodeStatusInstalling},
		{NodeStatusError, NodeStatusInstalling},
		{NodeStatusError, NodeStatusMaintenance},
	}
	for _, tc := range legal {
		if !canTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be legal", fromOrEmpty(tc.from), tc.to)
		}
	}
}

func TestCanTransitionIllegalEdges(t *testing.T) {
	// Transitions that must be rejected. These are the policy decisions
	// that protect operators from surprising state jumps.
	illegal := []struct{ from, to string }{
		// You don't go back to pending from a running state
		{NodeStatusActive, NodeStatusPending},
		{NodeStatusOffline, NodeStatusPending},
		{NodeStatusMaintenance, NodeStatusPending},
		{NodeStatusError, NodeStatusPending},

		// No skipping installing into active without explicit postinstall
		// (we DO allow pending->active because operator may pre-mark a node)
		// Offline does not skip to error directly without explicit choice
		// — actually we DO allow offline->error so it's not in this list.

		// Maintenance doesn't auto-transition to active without operator
		// confirmation — but operator path is allowed, system path uses
		// CheckStaleNodes which only touches active. Skip this for now.
	}
	for _, tc := range illegal {
		if canTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be ILLEGAL", fromOrEmpty(tc.from), tc.to)
		}
	}
}

func TestCanTransitionUnknownTarget(t *testing.T) {
	// If a target string is not in the transitions map values for any
	// source, no source can reach it. Sanity check that typos don't
	// silently get accepted.
	for from := range nodeStatusTransitions {
		if canTransition(from, "definitely-not-a-status") {
			t.Errorf("unknown target accepted from %q", fromOrEmpty(from))
		}
	}
}
