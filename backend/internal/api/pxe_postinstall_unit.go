package api

import (
	"encoding/base64"
	"fmt"
)

// postinstallUnitPath is where the systemd unit lands in the installed
// target. Service name `osbaka-postinstall.service` matches the preseed
// late_command's systemctl enable invocation.
const postinstallUnitPath = "/etc/systemd/system/osbaka-postinstall.service"

// buildPostinstallSystemdUnit returns the contents of the systemd unit
// that runs /root/postinstall.sh on first boot. Kept as a single string
// (not 14 separate echo commands written via in-target sh) so:
//
//  1. We can test the unit body in isolation.
//  2. The file lands atomically in the target — partial writes can't
//     produce a syntactically-invalid unit that silently fails to
//     enable, leaving the node stuck in `installing` forever.
//  3. Editing/extending the unit (e.g. adding Restart= or After= deps)
//     is one Go edit rather than 14 echo lines that all need `|| true`
//     suppression to not abort the install.
//
// The unit's `Type=oneshot` + `RemainAfterExit=no` means systemd treats
// the script as a transient task; once it exits the unit is "inactive"
// and won't be re-run on the next boot. postinstall.sh self-disables
// before exiting, so re-run is moot anyway.
func buildPostinstallSystemdUnit() string {
	return `[Unit]
Description=OS-Baka Post-Installation Script
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/root/postinstall.sh
RemainAfterExit=no

[Install]
WantedBy=multi-user.target
`
}

// buildPostinstallLateCommand returns the preseed late_command fragment
// that fetches the postinstall script and installs the systemd unit.
//
// Three changes from the previous implementation:
//
//  1. No `|| true` on critical steps. The old code suppressed every
//     error, so a curl failure (network blip, backend restart, wrong
//     EXTERNAL_IP) produced an empty postinstall.sh that was happily
//     chmod-ed and "enabled" — but at first boot did nothing, so the
//     node never called back its `active` status and silently rotted
//     into `error` after the install-timeout watcher fired. Better to
//     fail the install loudly here than ship a half-dead node.
//
//  2. The systemd unit body is written atomically via
//     `printf %s <b64> | base64 -d > unit`. Same pattern task C used
//     for /etc/osbaka-tpm.conf — base64 alphabet is shell-safe in any
//     quoting context, eliminating an entire class of unit-file
//     corruption bugs.
//
//  3. systemctl enable failure now propagates (no `|| true`). Enabling
//     a unit that doesn't exist or has bad syntax is a real failure
//     and should fail the install.
//
// Args: backendURL is the http://<EXTERNAL_IP> string; mac is the
// node MAC (lowercase, colon-separated); postinstallQuery is the
// "?t=<token>" string (or empty if PXE_REQUIRE_TOKEN is off).
func buildPostinstallLateCommand(backendURL, mac, postinstallQuery string) string {
	unitB64 := base64.StdEncoding.EncodeToString([]byte(buildPostinstallSystemdUnit()))
	// Note the doubled `%%s` for the inner printf — `fmt.Sprintf` will
	// reduce that to a single `%s` that the shell's printf then sees.
	return fmt.Sprintf(
		`in-target curl -sfS %s/api/v1/pxe/postinstall/%s%s -o /root/postinstall.sh; \
    in-target chmod +x /root/postinstall.sh; \
    in-target sh -c 'printf %%s %s | base64 -d > %s'; \
    in-target chmod 644 %s; \
    in-target systemctl enable osbaka-postinstall.service; \
    sync`,
		backendURL, mac, postinstallQuery,
		unitB64, postinstallUnitPath,
		postinstallUnitPath,
	)
}
