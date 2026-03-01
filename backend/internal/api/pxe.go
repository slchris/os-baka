package api

import (
	"fmt"
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
// SECURITY NOTE: PXE/preseed/kickstart scripts transmit sensitive data
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
// 3) OS-specific env (PXE_UBUNTU_MIRROR_URL, PXE_DEBIAN_MIRROR_URL, PXE_CENTOS_MIRROR_URL)
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

	case "centos", "rhel", "rocky":
		if v := trim(os.Getenv("PXE_CENTOS_MIRROR_URL")); v != "" {
			return v
		}
		if dbMirror != "" {
			return trim(dbMirror)
		}
		return "http://mirror.centos.org/centos"
	}

	// Fallback for unknown types
	if dbMirror != "" {
		return trim(dbMirror)
	}
	return "http://archive.ubuntu.com/ubuntu"
}

// normalizeMac converts MAC address to standard format (lowercase, colon-separated)
func normalizeMac(mac string) string {
	// Handle hyphen format from iPXE (aa-bb-cc-dd-ee-ff)
	mac = strings.ReplaceAll(mac, "-", ":")
	return strings.ToLower(mac)
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

	// If menu is forced or node not found, render a lightweight test menu to avoid blank screen during troubleshooting
	if menuForced || !nodeFound {
		var b strings.Builder
		b.WriteString("#!ipxe\n")
		b.WriteString("isset ${net0/ip} || dhcp\n")
		b.WriteString("menu OS-Baka PXE Test Menu\n")
		b.WriteString("item auto   Auto provision (backend rules)\n")
		b.WriteString("item local  Boot from local disk\n")
		b.WriteString("item shell  iPXE shell\n")
		b.WriteString("choose --timeout 15000 --default auto target && goto ${target} || goto shell\n")
		b.WriteString(fmt.Sprintf("\n:auto\nchain %s/api/v1/pxe/boot/${net0/mac}?menu=0 || goto shell\n", backendURL))
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

			script.WriteString(fmt.Sprintf(`set base-url %s
kernel ${base-url}/linux
initrd ${base-url}/initrd.gz
imgargs linux auto=true priority=critical url=%s/api/v1/pxe/preseed/${net0/mac:hexhyp} hostname=%s domain=os-baka.local interface=auto netcfg/dhcp_timeout=60%s
boot || shell
`, fullURL, backendURL, node.Hostname, extraArgs))

		case "debian":
			baseMirror := resolveMirror(node.OSType, node.MirrorURL, mirrorURL)
			release := node.OSVersion
			if release == "" {
				release = "bookworm" // default to Debian 12
			}
			fullURL := fmt.Sprintf("%s/dists/%s/main/installer-amd64/current/images/netboot/debian-installer/amd64", baseMirror, release)

			script.WriteString(fmt.Sprintf(`set base-url %s
kernel ${base-url}/linux
initrd ${base-url}/initrd.gz
imgargs linux auto=true priority=critical url=%s/pxe/preseed/${net0/mac:hexhyp} hostname=%s domain=os-baka.local interface=auto netcfg/dhcp_timeout=60%s
boot || shell
`, fullURL, backendURL, node.Hostname, extraArgs))

		case "centos", "rhel", "rocky":
			baseMirror := resolveMirror(node.OSType, node.MirrorURL, mirrorURL)
			release := node.OSVersion
			if release == "" {
				release = "9-stream" // default to CentOS Stream 9
			}
			fullURL := fmt.Sprintf("%s/%s/BaseOS/x86_64/os/images/pxeboot", baseMirror, release)

			script.WriteString(fmt.Sprintf(`set base-url %s
kernel ${base-url}/vmlinuz
initrd ${base-url}/initrd.img
imgargs vmlinuz inst.ks=%s/pxe/kickstart/${net0/mac:hexhyp} ip=dhcp%s
boot || shell
`, fullURL, backendURL, extraArgs))

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

	// Construct public backend URL
	serverIP := getPublicServerIP(c)
	backendURL := fmt.Sprintf("http://%s", serverIP)

	// Determine Mirror Config (node > env > DB > defaults)
	var config model.DHCPConfig
	_ = getDB().Where("is_active = ?", true).First(&config)
	baseMirror := resolveMirror(node.OSType, node.MirrorURL, config.MirrorURL)
	mirrorHost := ""
	mirrorDir := "/"
	if u, err := url.Parse(baseMirror); err == nil {
		mirrorHost = u.Host
		mirrorDir = u.Path
		if mirrorDir == "" {
			mirrorDir = "/"
		}
	}

	// Prepare partition configuration
	var partitionSection string

	passphrase := strings.ReplaceAll(node.EncryptionPassphrase, "\n", "")
	passphrase = strings.ReplaceAll(passphrase, "\r", "")
	if passphrase == "" {
		passphrase = "changeme"
	}

	if node.EncryptionEnabled {
		// LUKS encrypted partition with EXT4
		partitionSection = fmt.Sprintf(`### Partitioning (LUKS encrypted with EXT4)
d-i partman-auto/method string crypto
d-i partman-auto/disk string /dev/sda
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
d-i partman-auto-crypto/erase_disks boolean false`, passphrase, passphrase)
	} else {
		// Simple EXT4 without encryption
		partitionSection = `### Partitioning (EXT4 without encryption)
d-i partman-auto/method string regular
d-i partman-auto/disk string /dev/sda
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
d-i partman/confirm_nooverwrite boolean true`
	}

	// Use plain text password for root (Debian installer will hash it)
	rootPasswordPlain := "changeme"
	if node.RootPassword != "" {
		rootPasswordPlain = node.RootPassword
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

	// TPM2 auto-unlock configuration - moved to postinstall script
	// Just prepare the configuration file in late_command
	tpmSetupCommand := ""
	if node.EncryptionEnabled && node.TPMEnabled {
		// Determine PCR bindings (default to PCR 7 if not specified)
		pcrBindings := "7"
		if node.PCRBinding != "" {
			pcrBindings = node.PCRBinding
		}

		// Create config file for postinstall script to use
		tpmSetupCommand = fmt.Sprintf(`in-target sh -c 'echo "LUKS_PASSWORD=%s" > /etc/osbaka-tpm.conf'; \
    in-target sh -c 'echo "PCR_BINDINGS=%s" >> /etc/osbaka-tpm.conf'; \
    in-target sh -c 'echo "LUKS_DEVICE=/dev/sda3" >> /etc/osbaka-tpm.conf'; \
    in-target chmod 600 /etc/osbaka-tpm.conf; \
    `, passphrase, pcrBindings)
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
d-i apt-setup/security_path string %s-security

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
# Late command only does minimal setup, actual configuration happens in postinstall
d-i preseed/late_command string \
    %sin-target curl -s %s/pxe/postinstall/%s -o /root/postinstall.sh || true; \
    in-target chmod +x /root/postinstall.sh || true; \
    in-target sh -c "echo '[Unit]' > /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'Description=OS-Baka Post-Installation Script' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'After=network-online.target' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'Wants=network-online.target' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo '' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo '[Service]' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'Type=oneshot' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'ExecStart=/root/postinstall.sh' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'RemainAfterExit=no' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo '' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo '[Install]' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target sh -c "echo 'WantedBy=multi-user.target' >> /etc/systemd/system/osbaka-postinstall.service" || true; \
    in-target systemctl enable osbaka-postinstall.service || true; \
    sync
`, node.Hostname, node.Hostname, node.Hostname, mirrorHost, mirrorDir, mirrorHost, mirrorDir, rootPasswordPlain, rootPasswordPlain, timezone, partitionSection, packageSelection, tpmSetupCommand, backendURL, mac)

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, preseed)
}

// Kickstart godoc
// @Summary      Get Kickstart configuration for RHEL/CentOS
// @Description  Returns a kickstart file for automated installation
// @Tags         pxe
// @Produce      plain
// @Param        mac path string true "MAC Address"
// @Success      200 {string} string "Kickstart configuration"
// @Router       /pxe/kickstart/{mac} [get]
func (h *PXEHandler) Kickstart(c *gin.Context) {
	mac := normalizeMac(c.Param("mac"))

	var node model.Node
	if result := getDB().Where("LOWER(mac_address) = ?", mac).First(&node); result.Error != nil {
		c.String(http.StatusNotFound, "# Node not found")
		return
	}

	// Construct public backend URL
	serverIP := getPublicServerIP(c)
	backendURL := fmt.Sprintf("http://%s", serverIP)

	// Determine Mirror Config (env > DB > default)
	var config model.DHCPConfig
	_ = getDB().Where("is_active = ?", true).First(&config)
	baseMirror := resolveMirror(node.OSType, node.MirrorURL, config.MirrorURL)
	centosBase := fmt.Sprintf("%s/9-stream/BaseOS/x86_64/os/", strings.TrimRight(baseMirror, "/"))

	passphrase := strings.ReplaceAll(node.EncryptionPassphrase, "\n", "")
	passphrase = strings.ReplaceAll(passphrase, "\r", "")
	if passphrase == "" {
		passphrase = "changeme"
	}

	// Determine timezone (default to UTC if not specified)
	timezone := "UTC"
	if node.Timezone != "" {
		timezone = node.Timezone
	}

	autopartLine := "autopart --type=lvm"
	if node.EncryptionEnabled {
		autopartLine = fmt.Sprintf("autopart --type=lvm --encrypted --passphrase=%s", passphrase)
	}

	kickstart := fmt.Sprintf(`# Kickstart configuration for %s
# Generated by OS-Baka

# System language
lang en_US.UTF-8

# Keyboard layouts
keyboard us

# Network information
network --bootproto=dhcp --device=link --activate --hostname=%s.os-baka.local

# Root password
rootpw --plaintext changeme

# System timezone
timezone %s --utc

# Use network installation
url --url="%s"

# System bootloader configuration
bootloader --append="rhgb quiet" --location=mbr

# Partition clearing information
clearpart --all --initlabel

# Disk partitioning information
%s

# Accept EULA
eula --agreed

# Reboot after installation
reboot

%%packages
@^minimal-environment
openssh-server
curl
%%end

%%post
# Post-installation script
curl -s %s/pxe/postinstall/%s | bash
%%end
`, node.Hostname, node.Hostname, timezone, centosBase, autopartLine, backendURL, mac)

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, kickstart)
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

	// TPM2 auto-unlock setup
	tpmSetup := ""
	if node.EncryptionEnabled && node.TPMEnabled {
		tpmSetup = `
# Configure TPM2 auto-unlock
echo "OS-Baka: Configuring TPM2 auto-unlock..."

# Load configuration
if [ -f /etc/osbaka-tpm.conf ]; then
    source /etc/osbaka-tpm.conf
    
    # Install required packages
    apt-get update 2>/dev/null || true
    DEBIAN_FRONTEND=noninteractive apt-get install -y tpm2-tools cryptsetup cryptsetup-initramfs 2>/dev/null || true
    
    # Bind LUKS to TPM
    if [ -n "$LUKS_PASSWORD" ] && [ -n "$LUKS_DEVICE" ]; then
        echo "$LUKS_PASSWORD" | systemd-cryptenroll --tpm2-device=auto --tpm2-pcrs=${PCR_BINDINGS:-7} "$LUKS_DEVICE" 2>/dev/null || true
        
        # Update initramfs
        update-initramfs -u -k all 2>/dev/null || true
        
        echo "OS-Baka: TPM2 configuration complete"
    fi
    
    # Clean up sensitive config file
    rm -f /etc/osbaka-tpm.conf
else
    echo "OS-Baka: TPM config file not found, skipping"
fi
`
	}

	script := fmt.Sprintf(`#!/bin/bash
# Post-installation script for %s
# Generated by OS-Baka

set +e  # Don't exit on errors

echo "OS-Baka: Running post-installation..."

# Update node status to active (allow failure)
curl -s -X PUT "%s/api/v1/internal/nodes/%d/status" \
     -H "Content-Type: application/json" \
     -d '{"status": "active"}' 2>/dev/null || echo "Warning: Failed to update node status"

# Basic system configuration
hostnamectl set-hostname %s 2>/dev/null || echo "Warning: Failed to set hostname"

# Enable SSH (try both service names)
systemctl enable sshd 2>/dev/null || systemctl enable ssh 2>/dev/null || true
systemctl start sshd 2>/dev/null || systemctl start ssh 2>/dev/null || true
%s%s
# Remove this script and service after execution
rm -f /root/postinstall.sh
systemctl disable osbaka-postinstall.service 2>/dev/null || true
rm -f /etc/systemd/system/osbaka-postinstall.service

echo "OS-Baka: Post-installation complete!"

# Always exit successfully
exit 0
`, node.Hostname, backendURL, node.ID, node.Hostname, sshRootLoginSetup, tpmSetup)

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, script)
}
