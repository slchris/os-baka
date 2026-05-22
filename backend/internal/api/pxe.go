package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
	"github.com/os-baka/backend/internal/sysutil"
)

// PXEHandler handles PXE boot script generation and provisioning endpoints.
//
// SECURITY NOTE: PXE/preseed scripts transmit sensitive data
// (root passwords, LUKS passphrases) over HTTP. In production environments:
//   - Use a dedicated, isolated provisioning VLAN
//   - Consider placing the backend behind an HTTPS reverse proxy
//   - Restrict PXE endpoint access to the provisioning network only
type PXEHandler struct{}

func NewPXEHandler() *PXEHandler {
	return &PXEHandler{}
}

// resolveMirror returns the effective mirror URL for a given OS type with the following priority:
// 1) Node-specific mirror URL (if provided)
// 2) Generic env PXE_MIRROR_URL
// 3) OS-specific env (PXE_UBUNTU_MIRROR_URL, PXE_DEBIAN_MIRROR_URL)
// 4) Active DHCPConfig.MirrorURL (from DB / web UI)
// 5) Built-in distro defaults
func resolveMirror(osType, nodeMirror, dbMirror string) string {
	trim := func(s string) string {
		return strings.TrimRight(strings.TrimSpace(s), "/")
	}

	// Priority 1: Node-specific mirror
	if v := trim(nodeMirror); v != "" {
		return v
	}

	// Priority 2: Generic env
	if v := trim(os.Getenv("PXE_MIRROR_URL")); v != "" {
		return v
	}

	switch strings.ToLower(osType) {
	case "ubuntu", "linux":
		if v := trim(os.Getenv("PXE_UBUNTU_MIRROR_URL")); v != "" {
			return v
		}
		if dbMirror != "" {
			return trim(dbMirror)
		}
		return "http://archive.ubuntu.com/ubuntu"

	case "debian":
		if v := trim(os.Getenv("PXE_DEBIAN_MIRROR_URL")); v != "" {
			return v
		}
		if dbMirror != "" {
			return trim(dbMirror)
		}
		return "http://deb.debian.org/debian"
	}

	// Fallback for unknown types
	if dbMirror != "" {
		return trim(dbMirror)
	}
	return "http://archive.ubuntu.com/ubuntu"
}

// resolveSecurityMirror returns the URL of the security archive for the
// given OS type. Same precedence as resolveMirror:
//
//   1) Node-specific (node.SecurityMirrorURL)
//   2) OS-specific env (PXE_DEBIAN_SECURITY_MIRROR_URL, PXE_UBUNTU_SECURITY_MIRROR_URL)
//   3) Active DHCPConfig.SecurityMirrorURL
//   4) Built-in distro default
//
// Why this exists separately from resolveMirror: on Debian and Ubuntu the
// security archive lives on a different host than the main archive (e.g.
// security.debian.org vs deb.debian.org). The old code just appended
// "-security" to the main mirror path, which produced 404s like
// http://deb.debian.org/debian-security.
//
// Intranet operators typically mirror both archives under the same host
// (`http://mirror.intra/debian` + `http://mirror.intra/debian-security`)
// and set this via DHCPConfig.SecurityMirrorURL.
//
// Returns the empty string if no mirror is determinable for an unknown
// OS — caller should skip emitting security_host/security_path in that
// case rather than guess.
func resolveSecurityMirror(osType, nodeMirror, dbMirror string) string {
	trim := func(s string) string {
		return strings.TrimRight(strings.TrimSpace(s), "/")
	}

	if v := trim(nodeMirror); v != "" {
		return v
	}

	switch strings.ToLower(osType) {
	case "debian":
		if v := trim(os.Getenv("PXE_DEBIAN_SECURITY_MIRROR_URL")); v != "" {
			return v
		}
		if dbMirror != "" {
			return trim(dbMirror)
		}
		return "http://security.debian.org/debian-security"

	case "ubuntu", "linux":
		if v := trim(os.Getenv("PXE_UBUNTU_SECURITY_MIRROR_URL")); v != "" {
			return v
		}
		if dbMirror != "" {
			return trim(dbMirror)
		}
		return "http://security.ubuntu.com/ubuntu"
	}

	// Unknown OS: caller should treat empty as "skip security mirror".
	return ""
}

// isValidMirrorURL reports whether s is a plausible mirror URL. Empty is
// NOT valid here — callers gate the check with an "empty means default"
// branch. Accepts http and https schemes with a non-empty host (port
// optional). Rejects garbage, file://, relative URLs, etc.
//
// Loose intentionally: anything d-i can later resolve is fine. We trust
// the installer to report a hard failure if the mirror is unreachable.
func isValidMirrorURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

// splitMirrorURL parses a mirror URL into the (host, path) pair preseed's
// apt-setup wants. `mirror/http/hostname` and `apt-setup/security_host`
// take a bare host (with optional :port); `mirror/http/directory` and
// `apt-setup/security_path` take a path beginning with /.
//
// Returns ("", "/") for empty input so callers can render the lines
// without nil-handling churn — the resulting preseed lines will be
// inert (empty host = installer falls back to its own default).
func splitMirrorURL(raw string) (host, path string) {
	if raw == "" {
		return "", "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "/"
	}
	path = u.Path
	if path == "" {
		path = "/"
	}
	return u.Host, path
}

// normalizeMac converts MAC address to standard format (lowercase, colon-separated)
func normalizeMac(mac string) string {
	// Handle hyphen format from iPXE (aa-bb-cc-dd-ee-ff)
	mac = strings.ReplaceAll(mac, "-", ":")
	return strings.ToLower(mac)
}

// resolveNodePassphrase returns the plaintext LUKS passphrase for a node by
// decrypting the AES-GCM ciphertext stored on the Node row. Returns
// ("", error) when the field is empty — callers MUST treat this as fatal and
// refuse to serve preseed; otherwise the installer would write garbage as
// the actual LUKS password and brick the box.
func resolveNodePassphrase(node *model.Node) (string, error) {
	raw := strings.TrimSpace(node.EncryptionPassphrase)
	if raw == "" {
		return "", fmt.Errorf("no passphrase stored for node %d", node.ID)
	}
	plain := DecryptField(raw)
	if plain == "" {
		return "", fmt.Errorf("decrypted passphrase is empty for node %d", node.ID)
	}
	return plain, nil
}

// getPublicServerIP returns the reachable IP/Hostname of the server
func getPublicServerIP(c *gin.Context) string {
	// 1. Environment Variable (Highest priority for Docker/NAT)
	if extIP := os.Getenv("EXTERNAL_IP"); extIP != "" {
		return extIP
	}

	// 2. Database Config
	var config model.DHCPConfig
	if result := getDB().Where("is_active = ?", true).First(&config); result.Error == nil {
		if config.BootServerIP != "" {
			return config.BootServerIP
		}
	}

	// 3. Request Host (best effort fallback)
	if host := c.Request.Host; host != "" {
		// If behind Nginx, this should be the external Host header
		return host
	}

	// 4. Dynamic detection fallback
	if localIP := sysutil.GetLocalIP(); localIP != "" {
		return localIP
	}

	return "127.0.0.1"
}

// InitScript godoc
// @Summary      Get initial iPXE chainloader script
// @Description  Returns a static iPXE script that chains to the dynamic boot script using iPXE variables
// @Tags         pxe
// @Produce      plain
// @Success      200 {string} string "iPXE init script"
// @Router       /pxe/init [get]
func (h *PXEHandler) InitScript(c *gin.Context) {
	// This endpoint is called by boot.ipxe after it detects backend server
	// We delegate to BootScript using the MAC from query/header/best-effort
	// Priority: ?mac= query param > X-Forwarded-For parsing > fall back to menu
	mac := c.Query("mac")
	if mac == "" {
		// Try to extract from User-Agent or other headers if available
		// For now, return a discovery menu that chains back with MAC
		serverHost := getPublicServerIP(c)
		script := fmt.Sprintf(`#!ipxe
echo Chaining to boot script with MAC ${net0/mac:hexhyp}
chain http://%s/pxe/boot/${net0/mac:hexhyp} || shell
`, serverHost)
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, script)
		return
	}

	// If MAC provided, redirect to BootScript handler
	c.Redirect(http.StatusFound, fmt.Sprintf("/api/v1/pxe/boot/%s", mac))
}

// BootScript godoc
// @Summary      Get iPXE boot script for a node
// @Description  Returns an iPXE script based on the node's MAC address
// @Tags         pxe
// @Produce      plain
// @Param        mac path string true "MAC Address"
// @Success      200 {string} string "iPXE script"
// @Router       /pxe/boot/{mac} [get]
func (h *PXEHandler) BootScript(c *gin.Context) {
	mac := normalizeMac(c.Param("mac"))

	// Construct public backend URL
	serverIP := getPublicServerIP(c)
	backendURL := fmt.Sprintf("http://%s", serverIP)

	// Optional test menu flag: ?menu=1 or PXE_MENU_ENABLED=1
	menuForced := c.Query("menu") == "1" || strings.EqualFold(os.Getenv("PXE_MENU_ENABLED"), "1")

	var node model.Node
	nodeFound := getDB().Where("LOWER(mac_address) = ?", mac).First(&node).Error == nil

	// Auto-discover: when an unknown MAC PXE-boots, create a `discovered`
	// row so operators see it in inventory. The row's status (discovered,
	// distinct from pending) means BootScript won't auto-install it — the
	// machine sees the test menu and waits for operator approval.
	discoveredJustNow := false
	if !nodeFound && !menuForced {
		if discovered, isNew := discoverNode(getDB(), mac, c.ClientIP()); discovered != nil {
			node = *discovered
			nodeFound = true
			discoveredJustNow = isNew
		}
	}

	// Discovered nodes never install via PXE — they need operator approval
	// first. Render the test menu the same way as truly-unknown MACs do.
	if menuForced || !nodeFound || node.Status == NodeStatusDiscovered {
		var b strings.Builder
		b.WriteString("#!ipxe\n")
		b.WriteString("isset ${net0/ip} || dhcp\n")
		switch {
		case discoveredJustNow:
			b.WriteString("echo OS-Baka: new node REGISTERED as discovered (node ID " + fmt.Sprintf("%d", node.ID) + ")\n")
			b.WriteString("echo OS-Baka: set OS config in the UI, then click Rebuild to install\n")
			b.WriteString("menu OS-Baka — newly discovered\n")
		case node.Status == NodeStatusDiscovered:
			b.WriteString("echo OS-Baka: node already discovered, awaiting operator approval\n")
			b.WriteString("menu OS-Baka — awaiting approval\n")
		default:
			b.WriteString("menu OS-Baka PXE Test Menu\n")
		}
		b.WriteString("item auto   Re-check backend (chain through again)\n")
		b.WriteString("item local  Boot from local disk\n")
		b.WriteString("item shell  iPXE shell\n")
		b.WriteString("choose --timeout 15000 --default local target && goto ${target} || goto shell\n")
		fmt.Fprintf(&b, "\n:auto\nchain %s/api/v1/pxe/boot/${net0/mac}?menu=0 || goto shell\n", backendURL)
		b.WriteString("\n:local\n")
		b.WriteString("sanboot --no-describe --drive 0x80 || goto shell\n")
		b.WriteString("\n:shell\n")
		b.WriteString("echo Dropping to iPXE shell...\n")
		b.WriteString("shell\n")

		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, b.String())
		return
	}

	// Get active config for kernel params and mirror
	var config model.DHCPConfig
	var extraArgs string
	var mirrorURL string

	if result := getDB().Where("is_active = ?", true).First(&config); result.Error == nil {
		extraArgs = " " + config.KernelParams
		mirrorURL = config.MirrorURL
	}

	// Determine boot script based on OS type and status
	// Use strings.Builder for safe construction preventing leading newlines
	var script strings.Builder
	script.WriteString("#!ipxe\n")

	// Mint a fresh provisioning token for this install attempt. It is
	// embedded in the preseed/postinstall URLs so downstream fetches can be
	// authorized. Failure to mint is fatal — without a token, Preseed will
	// refuse to serve secrets.
	var preseedQuery string
	if (node.Status == "installing" || node.Status == "pending") && pxeTokenRequired() {
		token, err := IssuePXEToken(getDB(), node.ID, c.ClientIP())
		if err != nil {
			slog.Error("Failed to issue PXE token", "nodeID", node.ID, "error", err)
			c.String(http.StatusInternalServerError, "# Failed to mint provisioning token")
			return
		}
		preseedQuery = "?t=" + token
	}

	switch node.Status {
	case "installing", "pending":
		// Generate installation script based on OS type
		switch node.OSType {
		case "ubuntu", "linux":
			baseMirror := resolveMirror(node.OSType, node.MirrorURL, mirrorURL)
			release := node.OSVersion
			if release == "" {
				release = "jammy" // default to Ubuntu 22.04
			}
			fullURL := fmt.Sprintf("%s/dists/%s/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64", baseMirror, release)

			fmt.Fprintf(&script, `set base-url %s
kernel ${base-url}/linux
initrd ${base-url}/initrd.gz
imgargs linux auto=true priority=critical url=%s/api/v1/pxe/preseed/${net0/mac:hexhyp}%s hostname=%s domain=os-baka.local interface=auto netcfg/dhcp_timeout=60%s
boot || shell
`, fullURL, backendURL, preseedQuery, node.Hostname, extraArgs)

		case "debian":
			baseMirror := resolveMirror(node.OSType, node.MirrorURL, mirrorURL)
			release := node.OSVersion
			if release == "" {
				release = "bookworm" // default to Debian 12
			}
			fullURL := fmt.Sprintf("%s/dists/%s/main/installer-amd64/current/images/netboot/debian-installer/amd64", baseMirror, release)

			fmt.Fprintf(&script, `set base-url %s
kernel ${base-url}/linux
initrd ${base-url}/initrd.gz
imgargs linux auto=true priority=critical url=%s/api/v1/pxe/preseed/${net0/mac:hexhyp}%s hostname=%s domain=os-baka.local interface=auto netcfg/dhcp_timeout=60%s
boot || shell
`, fullURL, backendURL, preseedQuery, node.Hostname, extraArgs)

		default:
			// Unsupported OS - drop to shell
			script.WriteString("shell\n")
		}

	case "active":
		// Boot from local disk
		script.WriteString("exit\n")

	case "maintenance":
		// Drop to shell for maintenance
		script.WriteString("shell\n")

	default:
		// Default: exit to local boot
		script.WriteString("exit\n")
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, script.String())
}

// Preseed godoc
// @Summary      Get Preseed configuration for Debian/Ubuntu
// @Description  Returns a preseed file for automated installation
// @Tags         pxe
// @Produce      plain
// @Param        mac path string true "MAC Address"
// @Success      200 {string} string "Preseed configuration"
// @Router       /pxe/preseed/{mac} [get]
func (h *PXEHandler) Preseed(c *gin.Context) {
	mac := normalizeMac(c.Param("mac"))

	var node model.Node
	if result := getDB().Where("LOWER(mac_address) = ?", mac).First(&node); result.Error != nil {
		c.String(http.StatusNotFound, "# Node not found")
		return
	}

	// Security: only serve preseed to nodes that are actively provisioning.
	// This prevents information disclosure for nodes that are already active.
	if node.Status != "installing" && node.Status != "pending" {
		slog.Warn("Preseed requested for non-provisioning node",
			"mac", mac, "status", node.Status, "remote_addr", c.ClientIP())
		c.String(http.StatusForbidden, "# Node is not in provisioning state")
		return
	}

	// One-time provisioning token gate. Do not consume here — installers
	// (anaconda, debian-installer) may legitimately re-fetch preseed during
	// the install. PostInstall is the consumer.
	token := c.Query("t")
	if pxeTokenRequired() {
		// requireIP=false: anaconda often egresses on a different NIC than
		// the iPXE phase. Status + node-binding + TTL are the load-bearing
		// checks; IP binding is enforced only at PostInstall.
		if _, err := ValidatePXEToken(getDB(), token, node.ID, c.ClientIP(), false); err != nil {
			slog.Warn("Preseed token validation failed",
				"mac", mac, "nodeID", node.ID, "remote_addr", c.ClientIP(), "error", err)
			c.String(http.StatusForbidden, "# Invalid or expired provisioning token")
			return
		}
	}
	postinstallQuery := ""
	if token != "" {
		postinstallQuery = "?t=" + token
	}

	// Construct public backend URL
	serverIP := getPublicServerIP(c)
	backendURL := fmt.Sprintf("http://%s", serverIP)

	// Determine Mirror Config (node > env > DB > defaults). Two distinct
	// archives on Debian/Ubuntu: the main package archive and a separate
	// security archive on a different host. The previous code shared the
	// same host between the two and just appended "-security" to the
	// path — that produced http://deb.debian.org/debian-security which
	// 404s, leaving installed nodes without security updates.
	var config model.DHCPConfig
	_ = getDB().Where("is_active = ?", true).First(&config)

	baseMirror := resolveMirror(node.OSType, node.MirrorURL, config.MirrorURL)
	mirrorHost, mirrorDir := splitMirrorURL(baseMirror)

	securityMirror := resolveSecurityMirror(node.OSType, node.SecurityMirrorURL, config.SecurityMirrorURL)
	securityHost, securityDir := splitMirrorURL(securityMirror)

	// Prepare partition configuration
	var partitionSection string

	// Disk selection: either the operator pinned a path (node.TargetDisk)
	// or we let the installer auto-detect at runtime via early_command.
	// Either way the result lands in partman-auto/disk before partman runs.
	diskSection := buildDiskSelection(node.TargetDisk)

	// Resolve the LUKS passphrase by decrypting the stored field. Refuse to
	// serve preseed if encryption is enabled but no passphrase is recoverable
	// — serving garbage would brick the disk and lock the node out
	// permanently.
	var passphrase string
	if node.EncryptionEnabled {
		p, err := resolveNodePassphrase(&node)
		if err != nil {
			slog.Error("Refusing preseed: no recoverable LUKS passphrase",
				"nodeID", node.ID, "mac", mac, "error", err)
			c.String(http.StatusInternalServerError, "# No recoverable LUKS passphrase; aborting to avoid bricking the disk")
			return
		}
		passphrase = strings.ReplaceAll(p, "\n", "")
		passphrase = strings.ReplaceAll(passphrase, "\r", "")
	}

	if node.EncryptionEnabled {
		// LUKS encrypted partition with EXT4
		partitionSection = fmt.Sprintf(`### Partitioning (LUKS encrypted with EXT4)
%s
d-i partman-auto/method string crypto
d-i partman-auto-lvm/guided_size string max
d-i partman-auto-lvm/new_vg_name string vg0

# Suppress prompts
d-i partman-lvm/device_remove_lvm boolean true
d-i partman-md/device_remove_md boolean true
d-i partman-lvm/confirm boolean true
d-i partman-lvm/confirm_nooverwrite boolean true

# LUKS passphrase
d-i partman-crypto/passphrase password %s
d-i partman-crypto/passphrase-again password %s
d-i partman-crypto/weak_passphrase boolean true
d-i partman-crypto/confirm boolean true

# Custom recipe: boot + LUKS(root)
d-i partman-auto/expert_recipe string                         \
      boot-luks ::                                            \
              512 512 512 ext4                                \
                      $primary{ } $bootable{ }                \
                      method{ format } format{ }              \
                      use_filesystem{ } filesystem{ ext4 }    \
                      mountpoint{ /boot }                     \
              .                                               \
              1000 10000 -1 ext4                              \
                      $lvmok{ }                               \
                      lv_name{ root }                         \
                      method{ format } format{ }              \
                      use_filesystem{ } filesystem{ ext4 }    \
                      mountpoint{ / }                         \
              .                                               \
              2048 4096 4096 linux-swap                       \
                      $lvmok{ }                               \
                      lv_name{ swap }                         \
                      method{ swap } format{ }                \
              .

d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true
d-i partman-auto-crypto/erase_disks boolean false`, diskSection, passphrase, passphrase)
	} else {
		// Simple EXT4 without encryption
		partitionSection = fmt.Sprintf(`### Partitioning (EXT4 without encryption)
%s
d-i partman-auto/method string regular
d-i partman-lvm/device_remove_lvm boolean true
d-i partman-md/device_remove_md boolean true
d-i partman-lvm/confirm boolean true
d-i partman-lvm/confirm_nooverwrite boolean true

# Custom recipe: boot + root
d-i partman-auto/expert_recipe string                         \
      boot-root ::                                            \
              512 512 512 ext4                                \
                      $primary{ } $bootable{ }                \
                      method{ format } format{ }              \
                      use_filesystem{ } filesystem{ ext4 }    \
                      mountpoint{ /boot }                     \
              .                                               \
              2048 4096 4096 linux-swap                       \
                      method{ swap } format{ }                \
              .                                               \
              1000 10000 -1 ext4                              \
                      $primary{ }                             \
                      method{ format } format{ }              \
                      use_filesystem{ } filesystem{ ext4 }    \
                      mountpoint{ / }                         \
              .

d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true`, diskSection)
	}

	// Use plain text password for root (Debian installer will hash it).
	// node.RootPassword is AES-GCM ciphertext (enc:base64...) — decrypt before emitting.
	rootPasswordPlain := "changeme"
	if node.RootPassword != "" {
		if decrypted := DecryptField(node.RootPassword); decrypted != "" {
			rootPasswordPlain = decrypted
		}
	}

	// Determine timezone (default to UTC if not specified)
	timezone := "UTC"
	if node.Timezone != "" {
		timezone = node.Timezone
	}

	// Package selection - proper syntax for SSH
	packageSelection := "tasksel tasksel/first multiselect standard"
	if node.SSHEnabled {
		packageSelection = "tasksel tasksel/first multiselect standard, ssh-server"
	}

	// TPM2 auto-unlock configuration handed off to the postinstall script
	// via a small config file written in late_command.
	//
	// The config content is base64-encoded in Go and written via
	// `printf %s ... | base64 -d` because the LUKS passphrase is operator-
	// supplied (or randomly generated from a now-safe charset) and may in
	// theory contain shell metacharacters that would otherwise be expanded
	// by the layered single/double-quoted echo. Base64 alphabet is purely
	// [A-Za-z0-9+/=] — safe for any quoting context the d-i late_command
	// might be evaluated in.
	tpmSetupCommand := ""
	if node.EncryptionEnabled && node.TPMEnabled {
		pcrBindings := "7"
		if node.PCRBinding != "" {
			pcrBindings = node.PCRBinding
		}
		tpmSetupCommand = buildTPMPreseedLateCommand(passphrase, pcrBindings)
	}

	preseed := fmt.Sprintf(`# Preseed configuration for %s
# Generated by OS-Baka

### Localization
d-i debian-installer/locale string en_US.UTF-8
d-i keyboard-configuration/xkb-keymap select us

### Network configuration
d-i netcfg/choose_interface select auto
d-i netcfg/get_hostname string %s
d-i netcfg/get_domain string os-baka.local
d-i netcfg/hostname string %s

### Mirror settings
d-i mirror/country string manual
d-i mirror/http/hostname string %s
d-i mirror/http/directory string %s
d-i mirror/http/proxy string

### APT setup
d-i apt-setup/non-free boolean true
d-i apt-setup/contrib boolean true
d-i apt-setup/services-select multiselect security, updates
d-i apt-setup/security_host string %s
d-i apt-setup/security_path string %s

### Account setup
d-i passwd/root-login boolean true
d-i passwd/root-password password %s
d-i passwd/root-password-again password %s
d-i passwd/user-fullname string OS-Baka Admin
d-i passwd/username string osbaka
d-i passwd/user-password password changeme
d-i passwd/user-password-again password changeme

### Clock and time zone setup
d-i clock-setup/utc boolean true
d-i time/zone string %s

%s

### Package selection
%s

### Boot loader installation
d-i grub-installer/only_debian boolean true
d-i grub-installer/with_other_os boolean true
d-i grub-installer/bootdev string default

### Finish installation and reboot
d-i finish-install/reboot_in_progress note
d-i debian-installer/exit/halt boolean false
d-i debian-installer/exit/poweroff boolean false
d-i cdrom-detect/eject boolean false

### Post-installation commands
# Late command only does minimal setup; actual configuration happens in
# postinstall (downloaded from backend, run via systemd on first boot).
# The unit body is written atomically via base64 to avoid the prior
# echo-chain that silently produced half-written units; critical steps
# no longer suppress errors (curl, base64 decode, systemctl enable) so
# install failures surface as preseed errors instead of zombie nodes.
d-i preseed/late_command string \
    %s%s
`, node.Hostname, node.Hostname, node.Hostname, mirrorHost, mirrorDir, securityHost, securityDir, rootPasswordPlain, rootPasswordPlain, timezone, partitionSection, packageSelection, tpmSetupCommand, buildPostinstallLateCommand(backendURL, mac, postinstallQuery))

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, preseed)
}

// PostInstall godoc
// @Summary      Get post-installation script
// @Description  Returns a script to run after OS installation
// @Tags         pxe
// @Produce      plain
// @Param        mac path string true "MAC Address"
// @Success      200 {string} string "Post-install script"
// @Router       /pxe/postinstall/{mac} [get]
func (h *PXEHandler) PostInstall(c *gin.Context) {
	mac := normalizeMac(c.Param("mac"))

	var node model.Node
	if result := getDB().Where("LOWER(mac_address) = ?", mac).First(&node); result.Error != nil {
		c.String(http.StatusNotFound, "# Node not found")
		return
	}

	// Token gate. The installer fetched preseed (no consume) and now fetches
	// postinstall.sh — this is the single-use endpoint. Bind to client IP:
	// the request comes from the node itself during late_command on the
	// install network, so IP should match the boot script's issuer.
	if pxeTokenRequired() {
		token := c.Query("t")
		if _, err := ValidatePXEToken(getDB(), token, node.ID, c.ClientIP(), true); err != nil {
			slog.Warn("PostInstall token validation failed",
				"mac", mac, "nodeID", node.ID, "remote_addr", c.ClientIP(), "error", err)
			c.String(http.StatusForbidden, "# Invalid or expired provisioning token")
			return
		}
		if err := ConsumePXEToken(getDB(), token); err != nil {
			// Race or replay — refuse rather than serve a script twice.
			slog.Warn("PostInstall token consume failed",
				"mac", mac, "nodeID", node.ID, "error", err)
			c.String(http.StatusForbidden, "# Provisioning token already used")
			return
		}
	}

	// Construct public backend URL
	serverIP := getPublicServerIP(c)
	backendURL := fmt.Sprintf("http://%s", serverIP)

	// SSH root login configuration
	sshRootLoginSetup := ""
	if node.SSHEnabled && node.SSHRootLogin {
		sshRootLoginSetup = `
# Configure SSH root login
echo "OS-Baka: Configuring SSH root login..."
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config 2>/dev/null || true
systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || true
`
	}

	tpmSetup := buildTPMPostinstallBlock(&node)

	script := buildPostinstallScript(&node, backendURL, sshRootLoginSetup, tpmSetup)

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, script)
}
