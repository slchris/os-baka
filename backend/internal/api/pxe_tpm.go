package api

import (
	"encoding/base64"
	"fmt"

	"github.com/os-baka/backend/internal/model"
)

// buildTPMConfigBody returns the plaintext content of /etc/osbaka-tpm.conf
// (consumed by the postinstall script). It deliberately does NOT include
// LUKS_DEVICE — postinstall discovers it via blkid. If a future operator
// override is needed, this is the spot to add it.
//
// Format is a tiny `KEY=VALUE\n` file sourced by /bin/sh; the values are
// safe to pass through because they don't go through shell parsing here
// (see buildTPMPreseedLateCommand for the safe-write pipeline).
func buildTPMConfigBody(passphrase, pcrBindings string) string {
	return "LUKS_PASSWORD=" + passphrase + "\n" +
		"PCR_BINDINGS=" + pcrBindings + "\n"
}

// buildTPMPreseedLateCommand returns the preseed late_command fragment
// that writes /etc/osbaka-tpm.conf inside the target chroot.
//
// The config body is base64-encoded in Go and decoded on the target via
// `printf %s <b64> | base64 -d > /etc/osbaka-tpm.conf`. Reason: the LUKS
// passphrase is either operator-supplied or randomly generated, and may
// in theory contain shell metacharacters that would otherwise be expanded
// by the layered single/double-quoted echo the previous code used. Base64
// alphabet [A-Za-z0-9+/=] is safe in any quoting context.
//
// Returned string is intended to be interpolated into the larger
// late_command pipeline — it includes the trailing `; \` continuation.
func buildTPMPreseedLateCommand(passphrase, pcrBindings string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(buildTPMConfigBody(passphrase, pcrBindings)))
	return fmt.Sprintf(
		`in-target sh -c 'printf %%s %s | base64 -d > /etc/osbaka-tpm.conf'; \
    in-target chmod 600 /etc/osbaka-tpm.conf; \
    `, encoded)
}

// buildTPMPostinstallBlock returns the shell snippet that the postinstall
// script injects to bind LUKS to TPM2 on first boot. Returns "" when the
// node either doesn't have encryption enabled or doesn't have TPM enabled.
//
// Behaviour invariants the snippet enforces (tested in pxe_tpm_test.go):
//
//   - The LUKS underlying block device is auto-discovered via
//     `blkid -t TYPE=crypto_LUKS -o device`. The previous code hardcoded
//     /dev/sda3 which was wrong for both the partman recipe (which produces
//     sda2) AND for any node where the dynamic disk-selection picked
//     something other than /dev/sd*.
//
//   - All output is captured to /var/log/osbaka-tpm-setup.log. Silently
//     suppressing failures (the prior `|| true` pattern) lost critical
//     diagnostics — operators couldn't tell why a TPM bind had failed.
//
//   - The block always exits 0. A TPM bind failure means "boot prompts
//     for the passphrase", not "machine bricked"; the surrounding
//     postinstall script still flips status=active so the node leaves
//     the `installing` state.
//
//   - tpm.conf is removed regardless of outcome. Leaving the plaintext
//     passphrase on disk after the bind is the worse failure mode; if the
//     bind failed the operator still has the passphrase via Key Vault.
func buildTPMPostinstallBlock(node *model.Node) string {
	if node == nil || !node.EncryptionEnabled || !node.TPMEnabled {
		return ""
	}
	return `
# Configure TPM2 auto-unlock
TPM_LOG=/var/log/osbaka-tpm-setup.log
echo "OS-Baka: Configuring TPM2 auto-unlock (log: $TPM_LOG)..."

(
    set +e
    echo "=== osbaka TPM setup $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="

    if [ ! -f /etc/osbaka-tpm.conf ]; then
        echo "TPM config file not found; nothing to do"
        exit 0
    fi
    # shellcheck disable=SC1091
    . /etc/osbaka-tpm.conf

    if [ -z "$LUKS_PASSWORD" ]; then
        echo "ERROR: LUKS_PASSWORD missing from /etc/osbaka-tpm.conf"
        exit 0
    fi

    # Install dependencies. apt-get may fail in air-gapped intranets that
    # haven't mirrored these packages — log loudly so operator notices.
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        tpm2-tools cryptsetup cryptsetup-initramfs

    # Auto-discover the LUKS underlying block device unless explicitly
    # overridden via LUKS_DEVICE in tpm.conf. blkid returns the raw
    # block device (e.g. /dev/sda2, /dev/nvme0n1p2) — that's exactly
    # what systemd-cryptenroll needs.
    if [ -z "$LUKS_DEVICE" ]; then
        LUKS_DEVICE=$(blkid -t TYPE=crypto_LUKS -o device | head -n1)
    fi
    if [ -z "$LUKS_DEVICE" ]; then
        echo "ERROR: no LUKS device found via blkid; cannot enroll TPM"
        exit 0
    fi

    # Warn if multiple LUKS volumes exist — we only enroll the first.
    n=$(blkid -t TYPE=crypto_LUKS -o device | wc -l)
    if [ "$n" -gt 1 ]; then
        echo "WARN: $n LUKS volumes detected; enrolling only $LUKS_DEVICE"
        blkid -t TYPE=crypto_LUKS -o device
    fi

    echo "Enrolling TPM2 against $LUKS_DEVICE (PCRs: ${PCR_BINDINGS:-7})"
    if printf '%s' "$LUKS_PASSWORD" | systemd-cryptenroll \
        --tpm2-device=auto --tpm2-pcrs="${PCR_BINDINGS:-7}" "$LUKS_DEVICE"; then
        echo "cryptenroll OK"
    else
        rc=$?
        echo "ERROR: cryptenroll failed with exit $rc"
        exit 0
    fi

    echo "Updating initramfs to include TPM unlock"
    if update-initramfs -u -k all; then
        echo "initramfs update OK"
    else
        echo "ERROR: update-initramfs failed; TPM keyslot enrolled but boot may still prompt"
    fi

    echo "OS-Baka: TPM2 configuration complete"
) >> "$TPM_LOG" 2>&1

# Remove the sensitive config regardless of outcome — leaving it on
# disk after a successful TPM bind is worse (plaintext passphrase
# lying around). If the enroll failed the operator still has it via
# the Key Vault page.
rm -f /etc/osbaka-tpm.conf
`
}
