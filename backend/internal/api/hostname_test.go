package api

import (
	"strings"
	"testing"
)

func TestIsValidHostnameAccepts(t *testing.T) {
	good := []string{
		"a",                                   // single char
		"node01",
		"node-01",
		"NodeABC",                             // mixed case
		"x",                                   // bare letter
		"0",                                   // bare digit (RFC1123 allows; PVCs / Kubernetes do too)
		"rack1-row7-host42",
		strings.Repeat("a", 63),               // max length
	}
	for _, h := range good {
		t.Run("good/"+h, func(t *testing.T) {
			if !isValidHostname(h) {
				t.Errorf("expected %q to validate", h)
			}
		})
	}
}

func TestIsValidHostnameRejects(t *testing.T) {
	// Each of these is something an operator might type that would
	// break the preseed pipeline (or the installed OS later). All
	// must be rejected at the API edge.
	bad := []struct {
		name string
		h    string
	}{
		{"empty", ""},
		{"too-long-64", strings.Repeat("a", 64)},
		{"leading-hyphen", "-node"},
		{"trailing-hyphen", "node-"},
		{"contains-space", "node 1"},
		{"contains-tab", "node\t1"},
		{"contains-newline", "node\n1"},
		{"contains-double-quote", `node"1`},
		{"contains-single-quote", "node'1"},
		{"contains-dollar", "node$1"},
		{"contains-dot", "node.1"},                 // OS-Baka stores short name only
		{"contains-slash", "node/1"},
		{"contains-backslash", `node\1`},
		{"contains-backtick", "node`1"},
		{"contains-semicolon", "node;1"},
		{"unicode", "节点1"},                          // d-i + systemd choke on non-ASCII hostnames
		{"emoji", "node🚀"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if isValidHostname(tc.h) {
				t.Errorf("expected %q to be rejected", tc.h)
			}
		})
	}
}
