package api

import (
	"encoding/json"
	"testing"

	"github.com/os-baka/backend/internal/model"
)

// TestMakeNodeDeleteSnapshot pins the shape of the JSON payload embedded
// in audit details when a node is deleted. Operators and (someday) tools
// parse this string; changing the shape silently would break log archives.
//
// If you intentionally change the field set, update this test in the same
// commit so the break is visible in code review.
func TestMakeNodeDeleteSnapshotShape(t *testing.T) {
	groupID := uint(42)
	n := model.Node{
		Hostname:   "rack01-node03",
		IPAddress:  "10.0.5.3",
		MACAddress: "aa:bb:cc:dd:ee:03",
		AssetTag:   "ASSET-12345",
		Status:     "active",
		GroupID:    &groupID,
	}
	n.ID = 1234

	got := makeNodeDeleteSnapshot(&n)

	// Must be valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("snapshot is not valid JSON: %v\npayload: %s", err, got)
	}

	// Pin the expected key set. Extra keys are a contract change; missing
	// keys mean we just lost forensic information.
	wantKeys := map[string]bool{
		"id":          true,
		"hostname":    true,
		"mac_address": true,
		"ip_address":  true,
		"asset_tag":   true,
		"group_id":    true,
		"status":      true,
	}
	for k := range wantKeys {
		if _, ok := parsed[k]; !ok {
			t.Errorf("snapshot missing required key %q\npayload: %s", k, got)
		}
	}
	for k := range parsed {
		if !wantKeys[k] {
			t.Errorf("snapshot has unexpected key %q (contract change?)\npayload: %s", k, got)
		}
	}

	// Pin a couple of value mappings so we notice if json tag names drift.
	if parsed["hostname"] != "rack01-node03" {
		t.Errorf("hostname value lost: got %v", parsed["hostname"])
	}
	if parsed["mac_address"] != "aa:bb:cc:dd:ee:03" {
		t.Errorf("mac_address value lost: got %v", parsed["mac_address"])
	}
}

func TestMakeNodeDeleteSnapshotOmitsZeroOptionals(t *testing.T) {
	// asset_tag and group_id are tagged omitempty — a node without them
	// must not produce empty-string clutter in the audit log.
	n := model.Node{
		Hostname:   "minimal",
		MACAddress: "aa:bb:cc:dd:ee:99",
		IPAddress:  "10.0.0.99",
	}
	got := makeNodeDeleteSnapshot(&n)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if _, ok := parsed["asset_tag"]; ok {
		t.Errorf("empty asset_tag should be omitted, got: %s", got)
	}
	if _, ok := parsed["group_id"]; ok {
		t.Errorf("nil group_id should be omitted, got: %s", got)
	}
}
