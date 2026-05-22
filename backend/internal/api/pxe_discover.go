package api

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/os-baka/backend/internal/model"
	"gorm.io/gorm"
)

// autoDiscoverEnabled reports whether unknown MACs hitting /pxe/boot/:mac
// should be registered as `pending` nodes. Defaults ON — the whole point
// of running OS-Baka is to manage nodes; auto-discovery removes the
// "operator copies MAC off the rack screen" step.
//
// Disable via PXE_AUTO_DISCOVER=false. Useful when the PXE network has
// stray devices (test labs, shared L2 segments) that shouldn't pollute
// the inventory.
func autoDiscoverEnabled() bool {
	v := os.Getenv("PXE_AUTO_DISCOVER")
	switch v {
	case "0", "false", "False", "FALSE", "no", "No":
		return false
	}
	return true
}

// macSuffix returns the last 6 hex chars of the MAC, suitable for an
// ephemeral hostname like "discovered-aabbcc". Trims separators.
func macSuffix(mac string) string {
	trim := strings.NewReplacer(":", "", "-", "", ".", "")
	cleaned := trim.Replace(mac)
	if len(cleaned) < 6 {
		return cleaned
	}
	return cleaned[len(cleaned)-6:]
}

// discoverNode is called by the PXE boot handler when a MAC arrives that
// has no existing Node row. Best-effort registration with three outcomes:
//
//   1. Brand-new row inserted (the common case) → returns (node, true).
//   2. Race: another request inserted between our First() check and our
//      Create() — return the existing row, no error (false).
//   3. Auto-discover disabled OR DB error → returns (nil, false). Caller
//      proceeds with the unknown-node menu path so PXE still gets a script.
//
// The created row uses Status=discovered (NOT pending) so the BootScript
// switch does NOT route it into the installation path. An operator has to
// promote discovered → pending|installing in the UI before any OS gets
// written to disk. This is the safety property that makes auto-discovery
// acceptable on a shared L2 segment with stray PXE-capable devices.
func discoverNode(db *gorm.DB, mac, clientIP string) (*model.Node, bool) {
	if !autoDiscoverEnabled() {
		return nil, false
	}
	if db == nil || mac == "" {
		return nil, false
	}

	// MAC is already normalized to lowercase by the caller (normalizeMac).
	// We pass it through unchanged so the unique index lookup matches.
	now := time.Now().UTC()
	node := model.Node{
		MACAddress: mac,
		IPAddress:  clientIP,
		Hostname:   "discovered-" + macSuffix(mac),
		Status:     NodeStatusDiscovered,
		// installing_started_at intentionally NOT set — discovered means
		// "registered, not yet approved for provisioning". The install-
		// timeout watcher only fires on actual installing nodes.
	}

	err := db.Create(&node).Error
	if err == nil {
		slog.Info("pxe: discovered new node",
			"mac", mac,
			"ip", clientIP,
			"nodeID", node.ID,
			"hostname", node.Hostname)
		writeSystemAuditLog(db, "node.discovered", "node",
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("auto-discover via PXE boot: mac=%s client_ip=%s discovered_at=%s",
				mac, clientIP, now.Format(time.RFC3339)))
		return &node, true
	}

	// On insert failure — most likely the partial unique index on
	// (mac_address) WHERE deleted_at IS NULL. Could be a concurrent
	// discover request that won the race. Re-fetch and treat it as if
	// the row was already there (which it is).
	var existing model.Node
	if findErr := db.Where("LOWER(mac_address) = ?", mac).First(&existing).Error; findErr == nil {
		// The race-loser path. Return the existing row, don't audit-log
		// again (the winner already did).
		return &existing, false
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		// Something else went wrong (DB outage, etc). Log and bail —
		// the caller will fall back to the unknown-node menu path.
		slog.Warn("pxe: discover failed and existing-row fetch also failed",
			"mac", mac, "createErr", err, "fetchErr", findErr)
	} else {
		// Insert failed but existing row not found — unusual. Probably
		// a non-conflict error (bad data, schema drift). Log loudly.
		slog.Warn("pxe: discover insert failed and no existing row",
			"mac", mac, "error", err)
	}
	return nil, false
}
