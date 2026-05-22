package api

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/os-baka/backend/internal/model"
)

func TestBuildTPMConfigBodyShape(t *testing.T) {
	got := buildTPMConfigBody("my-passphrase", "0,2,7")
	// Must be a tiny KEY=VALUE\n file the postinstall can `source`.
	// Deliberately no LUKS_DEVICE — postinstall discovers it.
	if !strings.Contains(got, "LUKS_PASSWORD=my-passphrase\n") {
		t.Errorf("missing LUKS_PASSWORD line, got:\n%s", got)
	}
	if !strings.Contains(got, "PCR_BINDINGS=0,2,7\n") {
		t.Errorf("missing PCR_BINDINGS line, got:\n%s", got)
	}
	if strings.Contains(got, "LUKS_DEVICE") {
		t.Errorf("LUKS_DEVICE must NOT be in tpm.conf — postinstall discovers it; got:\n%s", got)
	}
}

func TestBuildTPMPreseedLateCommandIsBase64Encoded(t *testing.T) {
	// The whole point of the base64 pipeline is that hostile passphrases
	// (with $, ', ", backtick) can't shell-inject. Verify the output
	// contains a printf | base64 -d pipeline, NOT a literal echo of the
	// passphrase.
	passphrase := `evil$pass'word"with` + "`backticks`"
	got := buildTPMPreseedLateCommand(passphrase, "7")

	if strings.Contains(got, passphrase) {
		t.Errorf("literal passphrase leaked into shell command:\n%s", got)
	}
	if !strings.Contains(got, "printf %s ") {
		t.Errorf("expected printf %%s pipeline, got:\n%s", got)
	}
	if !strings.Contains(got, "| base64 -d > /etc/osbaka-tpm.conf") {
		t.Errorf("expected base64 -d redirect, got:\n%s", got)
	}

	// The encoded payload must round-trip back to the expected config
	// body. Extract the base64 chunk by finding it between "printf %s "
	// and " | base64 -d".
	start := strings.Index(got, "printf %s ") + len("printf %s ")
	end := strings.Index(got, " | base64 -d")
	if start <= 0 || end <= start {
		t.Fatalf("couldn't locate base64 payload in:\n%s", got)
	}
	encoded := got[start:end]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	wantBody := buildTPMConfigBody(passphrase, "7")
	if string(decoded) != wantBody {
		t.Errorf("decoded != expected body\n got: %q\nwant: %q", decoded, wantBody)
	}
}

func TestBuildTPMPostinstallBlockDisabledWhenNoEncryption(t *testing.T) {
	cases := []struct {
		name string
		node *model.Node
	}{
		{"nil", nil},
		{"no encryption no tpm", &model.Node{}},
		{"encryption but no tpm", &model.Node{EncryptionEnabled: true}},
		{"tpm but no encryption", &model.Node{TPMEnabled: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildTPMPostinstallBlock(tc.node); got != "" {
				t.Errorf("expected empty block, got:\n%s", got)
			}
		})
	}
}

func TestBuildTPMPostinstallBlockInvariants(t *testing.T) {
	// Generated postinstall block must enforce the four invariants
	// documented in pxe_tpm.go. If anyone "simplifies" the script and
	// breaks these, this test should yell.
	n := &model.Node{EncryptionEnabled: true, TPMEnabled: true}
	got := buildTPMPostinstallBlock(n)

	// 1. Auto-discovery via blkid (no static /dev/sda3 anymore).
	if !strings.Contains(got, "blkid -t TYPE=crypto_LUKS -o device") {
		t.Error("invariant 1: must call blkid -t TYPE=crypto_LUKS")
	}
	if strings.Contains(got, "/dev/sda3") {
		t.Error("invariant 1: must NOT contain hardcoded /dev/sda3")
	}

	// 2. Output captured to /var/log/osbaka-tpm-setup.log.
	if !strings.Contains(got, "/var/log/osbaka-tpm-setup.log") {
		t.Error("invariant 2: must log to /var/log/osbaka-tpm-setup.log")
	}

	// 3. The block must explicitly `exit 0` on the failure paths so the
	//    surrounding postinstall.sh still updates status. (The block runs
	//    inside a subshell, so its exits don't break the outer script.)
	if !strings.Contains(got, "exit 0") {
		t.Error("invariant 3: must `exit 0` on failure paths so outer postinstall continues")
	}

	// 4. tpm.conf is removed regardless of outcome.
	if !strings.Contains(got, "rm -f /etc/osbaka-tpm.conf") {
		t.Error("invariant 4: must remove the plaintext config file at end")
	}

	// Bonus: the systemd-cryptenroll invocation must use printf rather
	// than echo, so a passphrase ending in -n doesn't get swallowed by
	// echo's flag parsing on some shells.
	if !strings.Contains(got, `printf '%s' "$LUKS_PASSWORD"`) {
		t.Errorf("bonus: cryptenroll input must use printf with %%s format, not echo")
	}
}
