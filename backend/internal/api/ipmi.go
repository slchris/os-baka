package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// Outcome strings returned by TriggerPowerCycle. They surface in the HTTP
// response of /nodes/:id/rebuild so the operator knows what to expect.
const (
	powerCycleOutcomeScheduled = "scheduled" // queued; check audit log for result
	powerCycleOutcomeSkippedNoBMC = "skipped:no_bmc"
	powerCycleOutcomeDisabled  = "disabled" // IPMI_AUTO_POWER_CYCLE=false
)

// powerCycleSem bounds how many concurrent ipmitool processes the backend
// will spawn. Picked conservatively: 8 BMCs at once is well within fabric
// limits and won't pin a CI runner, but rebuilding a rack of 100 still
// completes in ~12 batches × ~12s = ~2.5 min. Tune via IPMI_POWER_CYCLE_CONCURRENCY.
var powerCycleSem chan struct{}

const defaultPowerCycleConcurrency = 8

// powerCycleTimeout caps a single ipmitool invocation, including any
// retries the tool does internally. The tool's -N 5 -R 1 gives a 10s
// worst-case for the network layer; this adds slack for fork/exec.
const powerCycleTimeout = 30 * time.Second

func init() {
	n := defaultPowerCycleConcurrency
	if v := os.Getenv("IPMI_POWER_CYCLE_CONCURRENCY"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	powerCycleSem = make(chan struct{}, n)
}

// autoPowerCycleEnabled reports whether the auto-cycle-on-rebuild feature
// is turned on. Default: true. Set IPMI_AUTO_POWER_CYCLE=false to disable
// (useful for environments without BMCs, or during incident response when
// you want to control reboots manually).
func autoPowerCycleEnabled() bool {
	v := os.Getenv("IPMI_AUTO_POWER_CYCLE")
	switch v {
	case "0", "false", "False", "FALSE", "no", "No":
		return false
	}
	return true // default ON, including unknown values (don't silently break)
}

// TriggerPowerCycle fires (in a background goroutine) an IPMI power-cycle
// against the node, with audit logging on outcome. Returns one of the
// powerCycleOutcome* constants describing what the caller should report
// to the HTTP client. The actual cycle completes asynchronously — caller
// must not assume the node has rebooted by the time this returns.
//
// The goroutine uses context.Background() (not the request context) since
// the HTTP request returns before the cycle finishes. The semaphore caps
// global concurrency.
func TriggerPowerCycle(node *model.Node) string {
	if !autoPowerCycleEnabled() {
		return powerCycleOutcomeDisabled
	}
	if node == nil || node.IPMIAddress == "" {
		// Skip is recorded so operators looking at audit logs can see
		// "we wanted to cycle but couldn't" — not just silence.
		writeSystemAuditLog(getDB(), "ipmi.power_cycle_skipped",
			"node", fmtNodeID(node),
			"no IPMI address configured")
		return powerCycleOutcomeSkippedNoBMC
	}

	// Snapshot enough of the node to do the call in the goroutine without
	// re-reading from DB. The model's password field is already AES-GCM
	// ciphertext; buildIpmitoolArgs decrypts it inside the goroutine.
	snapshot := *node

	go runPowerCycle(&snapshot)
	return powerCycleOutcomeScheduled
}

// TriggerPowerCycleBatch is the bulk-rebuild counterpart. Returns a tally
// of outcomes for the HTTP response. The goroutines are bounded by the
// shared powerCycleSem semaphore.
func TriggerPowerCycleBatch(nodes []model.Node) map[string]int {
	tally := map[string]int{}
	if !autoPowerCycleEnabled() {
		tally[powerCycleOutcomeDisabled] = len(nodes)
		return tally
	}
	for i := range nodes {
		n := &nodes[i]
		if n.IPMIAddress == "" {
			writeSystemAuditLog(getDB(), "ipmi.power_cycle_skipped",
				"node", fmtNodeID(n),
				"no IPMI address configured (bulk rebuild)")
			tally[powerCycleOutcomeSkippedNoBMC]++
			continue
		}
		snapshot := *n
		go runPowerCycle(&snapshot)
		tally[powerCycleOutcomeScheduled]++
	}
	return tally
}

// runPowerCycle is the goroutine body. Acquires a semaphore slot, runs
// ipmitool, records the outcome. Never returns an error to the caller —
// the audit log is the contract.
func runPowerCycle(node *model.Node) {
	// Block until a concurrency slot is free. Large bulk rebuilds queue
	// here rather than fork-bombing the BMC fabric.
	powerCycleSem <- struct{}{}
	defer func() { <-powerCycleSem }()

	ctx, cancel := context.WithTimeout(context.Background(), powerCycleTimeout)
	defer cancel()

	args := buildIpmitoolArgs(node, "power", "cycle")
	output, err := runIPMICommand(ctx, args)
	resourceID := fmtNodeID(node)
	if err != nil {
		slog.Warn("ipmi: power cycle failed",
			"nodeID", node.ID,
			"hostname", node.Hostname,
			"error", err,
			"output", output)
		writeSystemAuditLog(getDB(), "ipmi.power_cycle_error",
			"node", resourceID,
			fmt.Sprintf("ipmitool: %v; output=%s", err, truncate(output, 200)))
		return
	}

	slog.Info("ipmi: power cycle ok",
		"nodeID", node.ID,
		"hostname", node.Hostname,
		"output", output)
	writeSystemAuditLog(getDB(), "ipmi.power_cycle_ok",
		"node", resourceID,
		truncate(output, 200))
}

// fmtNodeID returns the ID as a string for audit log fields. Handles nil
// gracefully so a misconfigured caller doesn't panic the goroutine.
func fmtNodeID(node *model.Node) string {
	if node == nil {
		return ""
	}
	return fmt.Sprintf("%d", node.ID)
}

// truncate caps an audit-log details string to maxLen characters. ipmitool
// can spit out multi-line output that bloats the audit row; the meaningful
// signal is in the first line or two.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

// IPMIHandler handles IPMI/BMC power management endpoints.
type IPMIHandler struct{}

func NewIPMIHandler() *IPMIHandler {
	return &IPMIHandler{}
}

type PowerActionRequest struct {
	Action string `json:"action" binding:"required"` // on, off, reset, cycle, status
}

// runIPMICommand is the actual exec.CommandContext invocation. Indirected
// through a package variable so tests can stub IPMI calls without needing
// ipmitool on PATH.
//
// Safety note for gosec G204: args is exclusively produced by
// buildIpmitoolArgs from operator-supplied Node fields. ipmitool does not
// pass args through a shell — exec.CommandContext forks ipmitool directly
// with the argv vector, so there's no shell-metacharacter expansion risk.
// The remaining attack surface is "an operator with admin role types a
// hostile IPMI flag into the BMC config UI" which is exactly the threat
// model that admin role exists for.
var runIPMICommand = func(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "ipmitool", args...) // #nosec G204 — see above
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// buildIpmitoolArgs assembles the ipmitool arguments for a given node and
// power subcommand. It includes a short network timeout (-N 5) and one
// retry (-R 1) so a wedged BMC can't hang the calling request indefinitely.
// The subcommand argv (e.g. {"power","status"}) is appended verbatim.
//
// Callers must invoke this from a context with its own timeout — the
// -N/-R flags only bound the network layer, not the overall process.
func buildIpmitoolArgs(node *model.Node, subcommand ...string) []string {
	args := []string{
		"-I", "lanplus",
		"-N", "5", // network timeout per attempt (seconds)
		"-R", "1", // one retry on transient failure
		"-H", node.IPMIAddress,
		"-U", node.IPMIUsername,
	}
	if pw := DecryptField(node.IPMIPassword); pw != "" {
		args = append(args, "-P", pw)
	}
	if node.IPMIAllowUntrusted {
		args = append(args, "-C", "0") // disable cipher-suite enforcement
	}
	args = append(args, subcommand...)
	return args
}

// PowerAction godoc
// @Summary      Execute IPMI power action
// @Description  Send a power command (on/off/reset/cycle/status) to a node via IPMI
// @Tags         ipmi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Param        action body PowerActionRequest true "Power action"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/power [post]
func (h *IPMIHandler) PowerAction(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var node model.Node
	if err := getDB().First(&node, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	if node.IPMIAddress == "" {
		ErrorResponse(c, http.StatusBadRequest, "IPMI not configured for this node")
		return
	}

	var req PowerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	validActions := map[string]string{
		"on":     "on",
		"off":    "off",
		"reset":  "reset",
		"cycle":  "cycle",
		"status": "status",
	}

	ipmiAction, ok := validActions[req.Action]
	if !ok {
		ErrorResponse(c, http.StatusBadRequest, "Invalid action. Must be one of: on, off, reset, cycle, status")
		return
	}

	args := buildIpmitoolArgs(&node, "power", ipmiAction)
	if req.Action == "status" {
		args = buildIpmitoolArgs(&node, "power", "status")
	}

	// Bound the overall call. ipmitool's -N/-R only cap the network layer;
	// without this the process can still hang on DNS or stuck FDs.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	outputStr, err := runIPMICommand(ctx, args)
	if err != nil {
		WriteAuditLog(c, "ipmi.power_error", "node", fmt.Sprintf("%d", id),
			fmt.Sprintf("Action: %s, Error: %v, Output: %s", req.Action, err, outputStr))
		ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("IPMI command failed: %s", outputStr))
		return
	}

	WriteAuditLog(c, "ipmi.power_"+req.Action, "node", fmt.Sprintf("%d", id),
		fmt.Sprintf("Action: %s, Result: %s", req.Action, outputStr))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"action":  req.Action,
		"output":  outputStr,
		"node_id": id,
	})
}

// TestIPMI godoc
// @Summary      Test IPMI connectivity
// @Description  Test if the node's IPMI/BMC is reachable and credentials are valid
// @Tags         ipmi
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/ipmi/test [get]
func (h *IPMIHandler) TestIPMI(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var node model.Node
	if err := getDB().First(&node, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	if node.IPMIAddress == "" {
		ErrorResponse(c, http.StatusBadRequest, "IPMI not configured for this node")
		return
	}

	args := buildIpmitoolArgs(&node, "power", "status")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	start := time.Now()
	output, err := runIPMICommand(ctx, args)
	elapsed := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"reachable":  false,
			"error":      output,
			"latency_ms": elapsed.Milliseconds(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reachable":    true,
		"power_status": output,
		"latency_ms":   elapsed.Milliseconds(),
	})
}
