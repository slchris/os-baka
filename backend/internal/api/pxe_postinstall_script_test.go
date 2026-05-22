package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/os-baka/backend/internal/model"
)

func TestBuildPostinstallScriptHasRetryLoop(t *testing.T) {
	n := &model.Node{Hostname: "testhost"}
	n.ID = 7
	got := buildPostinstallScript(n, "http://10.0.0.10", "", "")

	if !strings.Contains(got, fmt.Sprintf("for attempt in $(seq 1 %d)", statusCallbackAttempts)) {
		t.Errorf("missing retry loop with %d attempts", statusCallbackAttempts)
	}
	if !strings.Contains(got, fmt.Sprintf("sleep %d", statusCallbackSleepSec)) {
		t.Errorf("missing sleep %d between attempts", statusCallbackSleepSec)
	}
	if !strings.Contains(got, "STATUS_UPDATED=true") {
		t.Error("missing success-flag set inside the loop")
	}
}

func TestBuildPostinstallScriptCleanupGatedOnSuccess(t *testing.T) {
	// The F bug was: cleanup ran unconditionally, so a single callback
	// failure deleted the script + disabled the unit, blocking retries.
	// Verify cleanup is *after* the "exit 0 if not updated" guard.
	n := &model.Node{Hostname: "testhost"}
	n.ID = 7
	got := buildPostinstallScript(n, "http://10.0.0.10", "", "")

	bail := strings.Index(got, `if [ "$STATUS_UPDATED" != "true" ]; then`)
	if bail == -1 {
		t.Fatal("missing the bail-out guard")
	}
	bailExit := strings.Index(got[bail:], "exit 0")
	if bailExit == -1 {
		t.Fatal("guard block has no exit 0 to actually bail")
	}
	bailEnd := bail + bailExit

	cleanupLines := []string{
		"rm -f /root/postinstall.sh",
		"systemctl disable osbaka-postinstall.service",
		"rm -f /etc/systemd/system/osbaka-postinstall.service",
	}
	for _, line := range cleanupLines {
		idx := strings.Index(got, line)
		if idx == -1 {
			t.Errorf("missing cleanup line: %q", line)
			continue
		}
		if idx < bailEnd {
			t.Errorf("cleanup line %q appears before bail-out — would run even on callback failure", line)
		}
	}
}

func TestBuildPostinstallScriptCurlSemantics(t *testing.T) {
	// curl must use -sfS so HTTP 4xx/5xx are treated as failures and
	// trigger a retry. Without -f, curl exit 0 + writes the error body
	// somewhere, and the retry loop thinks the callback succeeded.
	n := &model.Node{Hostname: "testhost"}
	n.ID = 7
	got := buildPostinstallScript(n, "http://10.0.0.10", "", "")

	if !strings.Contains(got, "curl -sfS -X PUT") {
		t.Errorf("expected curl -sfS -X PUT; got:\n%s", got)
	}
	if !strings.Contains(got, `"http://10.0.0.10/api/v1/internal/nodes/7/status"`) {
		t.Errorf("expected URL with backend + node ID; got snippet:\n%s",
			strings.SplitN(got, "\n", 20))
	}
	if !strings.Contains(got, `'{"status": "active"}'`) {
		t.Error("expected JSON body to mark active")
	}
}

func TestBuildPostinstallScriptHostnameInjected(t *testing.T) {
	n := &model.Node{Hostname: "rack01-node03"}
	n.ID = 7
	got := buildPostinstallScript(n, "http://10.0.0.10", "", "")
	if !strings.Contains(got, "hostnamectl set-hostname rack01-node03") {
		t.Error("expected hostnamectl set-hostname with node hostname")
	}
}

func TestBuildPostinstallScriptHostnameRunsAfterSuccessGuard(t *testing.T) {
	// hostnamectl must only run after the success guard — same reason
	// as cleanup. A node that never called back shouldn't half-configure
	// itself.
	n := &model.Node{Hostname: "testhost"}
	n.ID = 7
	got := buildPostinstallScript(n, "http://10.0.0.10", "", "")
	bail := strings.Index(got, `if [ "$STATUS_UPDATED" != "true" ]; then`)
	bailExit := strings.Index(got[bail:], "exit 0")
	bailEnd := bail + bailExit
	hostnameLine := strings.Index(got, "hostnamectl set-hostname")
	if hostnameLine == -1 {
		t.Fatal("hostnamectl missing")
	}
	if hostnameLine < bailEnd {
		t.Error("hostnamectl runs before success guard — would partially configure failed nodes")
	}
}

func TestBuildPostinstallScriptOptionalBlocksInterpolated(t *testing.T) {
	n := &model.Node{Hostname: "testhost"}
	n.ID = 7
	sshBlock := "# CUSTOM SSH ROOT BLOCK"
	tpmBlock := "# CUSTOM TPM BLOCK"
	got := buildPostinstallScript(n, "http://10.0.0.10", sshBlock, tpmBlock)

	if !strings.Contains(got, sshBlock) {
		t.Error("ssh root block missing")
	}
	if !strings.Contains(got, tpmBlock) {
		t.Error("tpm block missing")
	}
}
