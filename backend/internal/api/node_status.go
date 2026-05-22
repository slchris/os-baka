package api

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/os-baka/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Node lifecycle states.
//
// The canonical names are these constants — string comparisons against
// arbitrary literals elsewhere are a source of bugs (e.g. `"OFFLINE"` vs
// `"offline"`). New code should use these; old direct assignments are
// being migrated incrementally.
const (
	// NodeStatusDiscovered is set ONLY by PXE auto-discovery (see
	// pxe_discover.go). It means "a machine PXE-booted with this MAC but
	// no operator has approved it yet". Distinct from pending so the PXE
	// boot handler can refuse to install discovered nodes without changing
	// the legacy behavior of UI-registered pending nodes.
	NodeStatusDiscovered = "discovered"

	NodeStatusPending     = "pending"
	NodeStatusInstalling  = "installing"
	NodeStatusActive      = "active"
	NodeStatusMaintenance = "maintenance"
	NodeStatusError       = "error"
	NodeStatusOffline     = "offline"
)

// IsKnownNodeStatus reports whether s is one of the recognized statuses.
// Used to validate raw user input before persisting.
func IsKnownNodeStatus(s string) bool {
	switch s {
	case NodeStatusDiscovered, NodeStatusPending, NodeStatusInstalling,
		NodeStatusActive, NodeStatusMaintenance, NodeStatusError, NodeStatusOffline:
		return true
	}
	return false
}

// nodeStatusTransitions enumerates the legal source→target transitions.
// Self-transitions are not in the map but are allowed as no-ops — callers
// retrying a status update mid-flight should not get an error.
//
// Rationale:
//   - active ⇄ offline: heartbeat / stale-check oscillation
//   - active → maintenance / installing: operator drains a host or rebuilds
//   - installing → active (postinstall callback), error (timeout), pending (operator aborts)
//   - error / offline / maintenance → installing (rebuild from any terminal state)
//   - any → error: nothing is more wrong than already being wrong
var nodeStatusTransitions = map[string][]string{
	// Discovered → operator approves (promote to pending or directly to
	// install) or rejects (maintenance / error). Discovered is intentionally
	// NOT install-eligible from the PXE handler — see pxe.go BootScript.
	NodeStatusDiscovered:  {NodeStatusPending, NodeStatusInstalling, NodeStatusMaintenance, NodeStatusError},
	NodeStatusPending:     {NodeStatusInstalling, NodeStatusActive, NodeStatusError, NodeStatusMaintenance},
	NodeStatusInstalling:  {NodeStatusActive, NodeStatusError, NodeStatusPending},
	NodeStatusActive:      {NodeStatusOffline, NodeStatusMaintenance, NodeStatusInstalling, NodeStatusError},
	NodeStatusOffline:     {NodeStatusActive, NodeStatusInstalling, NodeStatusMaintenance, NodeStatusError},
	NodeStatusMaintenance: {NodeStatusActive, NodeStatusInstalling, NodeStatusOffline, NodeStatusError},
	NodeStatusError:       {NodeStatusInstalling, NodeStatusMaintenance, NodeStatusActive},
	// Empty status = brand-new row before any state has been assigned.
	// Discovered is in here because the PXE auto-discover path inserts
	// directly with status=discovered (no SetNodeStatus call in between).
	"": {NodeStatusDiscovered, NodeStatusPending, NodeStatusInstalling, NodeStatusActive, NodeStatusMaintenance, NodeStatusError},
}

// ErrIllegalTransition is returned by SetNodeStatus when from→to is not
// permitted by nodeStatusTransitions.
var ErrIllegalTransition = errors.New("illegal node status transition")

// canTransition reports whether moving from→to is allowed.
func canTransition(from, to string) bool {
	if from == to {
		return true // self is always fine; treated as no-op
	}
	return slices.Contains(nodeStatusTransitions[from], to)
}

// SetNodeStatus moves a node into target state with the necessary side
// effects (lifecycle timestamps, audit log) and returns an error if the
// transition is illegal or the database write fails.
//
// reason is recorded in the audit log details — short, present-tense:
// "postinstall callback", "install timeout 60m", "operator rebuild".
//
// c may be nil for system-initiated transitions (background checkers).
// In that case the audit log records actor="system" and IP="".
//
// IMPORTANT: this function is the single chokepoint for status mutation.
// Direct `node.Status = "..."` assignments scattered through the codebase
// bypass timestamp maintenance and audit logging. Prefer this even when
// you "know" the transition is fine.
func SetNodeStatus(c *gin.Context, db *gorm.DB, node *model.Node, target, reason string) error {
	if db == nil || node == nil {
		return errors.New("SetNodeStatus: nil db or node")
	}
	if !IsKnownNodeStatus(target) {
		return fmt.Errorf("%w: unknown target %q", ErrIllegalTransition, target)
	}

	from := node.Status
	if !canTransition(from, target) {
		return fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, fromOrEmpty(from), target)
	}

	if from == target {
		// No-op. Don't churn timestamps or audit logs — callers retrying
		// the same status should not produce noise.
		return nil
	}

	// Build the column updates as a map so we can null out
	// installing_started_at when leaving the provisioning states.
	// (gorm.Save with a nil *time.Time would not null an existing value.)
	updates := map[string]any{
		"status": target,
	}
	now := time.Now().UTC()
	switch target {
	case NodeStatusInstalling, NodeStatusPending:
		updates["installing_started_at"] = now
	default:
		updates["installing_started_at"] = nil
	}

	if err := db.Model(node).Updates(updates).Error; err != nil {
		return fmt.Errorf("persist status %s: %w", target, err)
	}
	// Keep the in-memory node consistent with the DB.
	node.Status = target
	switch target {
	case NodeStatusInstalling, NodeStatusPending:
		node.InstallingStartedAt = &now
	default:
		node.InstallingStartedAt = nil
	}

	// Audit log. Use a soft path so a nil context (system caller) still
	// records something meaningful.
	if c != nil {
		WriteAuditLog(c, "node.status_change", "node", strconv.FormatUint(uint64(node.ID), 10),
			fmt.Sprintf("%s -> %s: %s", fromOrEmpty(from), target, reason))
	} else {
		writeSystemAuditLog(db, "node.status_change", "node",
			strconv.FormatUint(uint64(node.ID), 10),
			fmt.Sprintf("%s -> %s: %s", fromOrEmpty(from), target, reason))
	}

	slog.Info("node status changed",
		"nodeID", node.ID,
		"from", fromOrEmpty(from),
		"to", target,
		"reason", reason)
	return nil
}

// writeSystemAuditLog records an audit entry without a gin.Context. Used
// by background checkers. Synchronous with a bounded timeout — same
// rationale as WriteAuditLog.
func writeSystemAuditLog(db *gorm.DB, action, resource, resourceID, details string) {
	log := model.AuditLog{
		Action:     action,
		Username:   "system",
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
	}
	writeAuditLogSync(db, &log)
}

func fromOrEmpty(s string) string {
	if s == "" {
		return "<new>"
	}
	return s
}
