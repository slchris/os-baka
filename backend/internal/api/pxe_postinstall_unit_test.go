package api

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildPostinstallSystemdUnitShape(t *testing.T) {
	got := buildPostinstallSystemdUnit()

	// All three required sections.
	for _, section := range []string{"[Unit]", "[Service]", "[Install]"} {
		if !strings.Contains(got, section) {
			t.Errorf("missing section %s", section)
		}
	}

	// Key directives — if any of these drift, the postinstall service
	// won't run on first boot and nodes will silently stay in installing
	// until they time out into error.
	required := []string{
		"After=network-online.target",
		"Wants=network-online.target",
		"Type=oneshot",
		"ExecStart=/root/postinstall.sh",
		"WantedBy=multi-user.target",
	}
	for _, line := range required {
		if !strings.Contains(got, line) {
			t.Errorf("missing required directive %q\nunit:\n%s", line, got)
		}
	}
}

func TestBuildPostinstallLateCommandUsesBase64(t *testing.T) {
	got := buildPostinstallLateCommand("http://10.0.0.10", "aa:bb:cc:dd:ee:ff", "?t=abc")

	// The echo-chain pattern from the old code is forbidden — its
	// per-line failures + suppression produced corrupt unit files.
	if strings.Contains(got, "echo '[Unit]'") || strings.Contains(got, "echo '[Service]'") {
		t.Errorf("must NOT use echo-chain for unit body; got:\n%s", got)
	}

	// Must use the base64 atomic-write pipeline.
	if !strings.Contains(got, "printf %s ") {
		t.Errorf("expected printf %%s base64 pipeline; got:\n%s", got)
	}
	if !strings.Contains(got, "| base64 -d > /etc/systemd/system/osbaka-postinstall.service") {
		t.Errorf("expected base64 -d redirect to unit path; got:\n%s", got)
	}

	// Critical steps must NOT suppress errors. The whole reason we're
	// rewriting this is that the old `|| true` produced zombie nodes.
	for _, step := range []string{
		"curl -sfS http://10.0.0.10/api/v1/pxe/postinstall/aa:bb:cc:dd:ee:ff?t=abc",
		"chmod +x /root/postinstall.sh",
		"systemctl enable osbaka-postinstall.service",
	} {
		if !strings.Contains(got, step) {
			t.Errorf("missing critical step %q; got:\n%s", step, got)
		}
		// Check that this specific step doesn't end with `|| true` by
		// looking for the step followed somewhere by " || true" before
		// the next semicolon. Cheap heuristic — anything between the
		// step and "; \\" must not include " || true".
		stepIdx := strings.Index(got, step)
		if stepIdx == -1 {
			continue
		}
		nextSemi := strings.Index(got[stepIdx:], ";")
		if nextSemi == -1 {
			nextSemi = len(got) - stepIdx
		}
		fragment := got[stepIdx : stepIdx+nextSemi]
		if strings.Contains(fragment, "|| true") {
			t.Errorf("critical step %q still uses `|| true` suppression: %q", step, fragment)
		}
	}

	// curl uses -f (fail on HTTP error) and -S (show errors on stderr).
	// Without -f, curl writes the 4xx/5xx body to /root/postinstall.sh
	// and exits 0 — node ends up "successfully" installed with an HTML
	// error page as its postinstall script. Don't regress.
	if !strings.Contains(got, "curl -sfS ") {
		t.Errorf("curl must use -sfS (silent, fail-on-error, show-errors); got:\n%s", got)
	}
}

func TestBuildPostinstallLateCommandBase64RoundTrip(t *testing.T) {
	got := buildPostinstallLateCommand("http://10.0.0.10", "aa:bb:cc:dd:ee:ff", "")

	// Extract the base64 payload between "printf %s " and " | base64 -d"
	// and confirm it decodes to the exact unit body. If somebody edits
	// the unit body without re-running this test, the round-trip catches
	// the inconsistency.
	const marker = "printf %s "
	start := strings.Index(got, marker)
	if start == -1 {
		t.Fatalf("no printf marker found in: %s", got)
	}
	start += len(marker)
	end := strings.Index(got[start:], " | base64 -d")
	if end == -1 {
		t.Fatal("no | base64 -d found")
	}
	encoded := got[start : start+end]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	if string(decoded) != buildPostinstallSystemdUnit() {
		t.Errorf("decoded payload != unit body\n got: %q\nwant: %q", decoded, buildPostinstallSystemdUnit())
	}
}

func TestBuildPostinstallLateCommandEmptyToken(t *testing.T) {
	// When PXE_REQUIRE_TOKEN=false the query string is empty. URL must
	// still be valid (no dangling `?` etc).
	got := buildPostinstallLateCommand("http://10.0.0.10", "aa:bb:cc:dd:ee:ff", "")
	want := "curl -sfS http://10.0.0.10/api/v1/pxe/postinstall/aa:bb:cc:dd:ee:ff -o /root/postinstall.sh"
	if !strings.Contains(got, want) {
		t.Errorf("expected URL without trailing ?; got:\n%s", got)
	}
}
