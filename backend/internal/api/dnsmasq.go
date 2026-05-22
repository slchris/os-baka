package api

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/os-baka/backend/internal/model"
)

// dnsmasq config paths. Declared as var (not const) so tests can redirect
// to a tempdir without bringing the entire /etc/dnsmasq.d hierarchy along.
// Don't mutate at runtime outside tests.
var (
	dnsmasqHostsDir      = "/etc/dnsmasq.d"
	dnsmasqMainConfig    = "/etc/dnsmasq.d/00-main.conf"
	dnsmasqHostsConfig   = "/etc/dnsmasq.d/01-hosts.conf"
	dnsmasqReloadTrigger = "/etc/dnsmasq.d/.reload"
)

// dnsmasqMissingDirWarned is flipped after the first "directory doesn't
// exist" warning so dev mode (where /etc/dnsmasq.d is not bind-mounted)
// doesn't log on every scheduled regen.
var dnsmasqMissingDirWarned atomic.Bool

// GenerateDnsmasqConfig regenerates the full dnsmasq configuration on disk.
// Returns a non-nil error if either the main or hosts file write failed.
// Callers must NOT swallow the error: surface it to the user when running
// in response to an explicit user action, or feed it into the dnsmasq
// scheduler's LastError so operators can see something is wrong.
//
// Trigger semantics: the .reload sentinel is written exactly once, AFTER
// both config files have been atomically renamed into place. This avoids
// a race where the pxe-services watcher (`start.sh`) reads .reload and
// HUPs dnsmasq while the hosts file is still mid-write. Without the
// sentinel-last ordering, dnsmasq could load a half-written hosts config.
func GenerateDnsmasqConfig() error {
	// Dev / local builds typically don't have /etc/dnsmasq.d bind-mounted
	// in. Treat the missing-directory case as a benign no-op so the
	// dashboard's "DNSMasq Config Sync" tile doesn't flag permanent red.
	if _, err := os.Stat(dnsmasqHostsDir); os.IsNotExist(err) {
		if !dnsmasqMissingDirWarned.Swap(true) {
			slog.Info("dnsmasq: config dir not present, skipping regen (dev mode?)",
				"dir", dnsmasqHostsDir)
		}
		return nil
	}

	var errs []error
	if err := generateMainConfig(); err != nil {
		slog.Warn("dnsmasq: main config generation failed", "error", err)
		errs = append(errs, fmt.Errorf("main config: %w", err))
	}
	if err := generateHostsConfig(); err != nil {
		slog.Warn("dnsmasq: hosts config generation failed", "error", err)
		errs = append(errs, fmt.Errorf("hosts config: %w", err))
	}

	// Only signal reload if at least one file write succeeded. If both
	// failed, the on-disk state is whatever it was before — pointless to
	// HUP dnsmasq for no change. The watcher polls every 2s so failure
	// to signal here is recoverable on the next successful regen.
	if len(errs) < 2 {
		// 0644: the pxe-services container's start.sh polls this file
		// (a tiny sentinel containing the word "reload") to detect
		// config changes and HUP dnsmasq. Cross-container reads require
		// world-readable mode; tighter perms would silently break the
		// reload pipeline.
		if err := os.WriteFile(dnsmasqReloadTrigger, []byte("reload"), 0644); err != nil { // #nosec G306 — see above
			slog.Warn("dnsmasq: reload trigger write failed", "error", err)
		}
	}
	return errors.Join(errs...)
}

// writeFileAtomic writes data to a fresh tempfile in the same directory as
// path, fsyncs it, then renames over the target. The pxe-services watcher
// polls every 2s and HUPs dnsmasq when it sees .reload — without an atomic
// swap, dnsmasq could be HUP'd into reading a half-written file.
//
// We use the same directory for the tempfile so rename stays atomic
// (cross-device rename would degrade to copy+unlink and lose atomicity).
func writeFileAtomic(path string, data []byte, perm os.FileMode) (retErr error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if retErr != nil {
			// Best-effort cleanup. If this fails we'll just leave a stray
			// .tmp file — better than partial config in place.
			if err := os.Remove(tmpName); err != nil && !os.IsNotExist(err) {
				slog.Warn("dnsmasq: tempfile cleanup failed", "path", tmpName, "error", err)
			}
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// generateMainConfig generates the main dnsmasq configuration from DB
func generateMainConfig() error {
	var config model.DHCPConfig
	result := getDB().Where("is_active = ?", true).First(&config)
	if result.Error != nil {
		// No active config, use defaults
		return nil
	}

	var content strings.Builder
	content.WriteString("# Auto-generated by OS-Baka\n")
	content.WriteString("# Do not edit manually - changes will be overwritten\n\n")

	// Force bind to all interfaces for host network usage in Docker
	// Since we are running with network_mode: host, we want to listen on all appropriate interfaces
	// or rely on dnsmasq's default behavior, but the user requested "bind 0.0.0.0".
	// Dnsmasq doesn't use 0.0.0.0 directly in config, but we can omit 'interface=' or use specific flags.
	// However, if we want to respect the user's "unconditional bind", we usually remove interface restrictions.

	// If the user selects an interface in UI, we might still want to prioritize it, OR
	// typically if running host mode container, we just bind wild.
	// The user asked "unconditional bind 0.0.0.0", which usually means:
	// except-interface=lo
	// bind-dynamic
	// OR just listing interfaces.

	// Let's implement what is requested: remove the specific interface binding restriction
	// and allow it to listen on the selected interface explicitly if provided, but also ensure it works across host.

	// Actually, the user says "I want to bind 0.0.0.0 ... so other machines can access".
	// This implies removing `bind-interfaces` which restricts it to only interfaces up at start,
	// and ensuring we don't limit to loopback.

	if config.Interface != "" {
		// If interface acts as a filter
		fmt.Fprintf(&content, "interface=%s\n", config.Interface)
	}
	// content.WriteString("bind-interfaces\n\n") -- Removing this as it's too restrictive for some docker setups

	// DHCP Range
	fmt.Fprintf(&content, "dhcp-range=%s,%s,%s,%s\n",
		config.RangeStart, config.RangeEnd, config.SubnetMask, config.LeaseTime)

	// Gateway
	if config.Gateway != "" {
		fmt.Fprintf(&content, "dhcp-option=option:router,%s\n", config.Gateway)
	}

	// DNS Server
	if config.DNSServer != "" {
		fmt.Fprintf(&content, "dhcp-option=option:dns-server,%s\n", config.DNSServer)
	}

	// Domain
	if config.Domain != "" {
		fmt.Fprintf(&content, "domain=%s\n", config.Domain)
	}

	content.WriteString("\n")

	// PXE Boot Configuration
	if config.EnablePXE {
		content.WriteString("# PXE Boot Configuration\n")
		content.WriteString("enable-tftp\n")
		content.WriteString("tftp-root=/tftpboot\n\n")

		// BIOS clients
		content.WriteString("# BIOS PXE Boot\n")
		content.WriteString("dhcp-match=set:bios,option:client-arch,0\n")
		bootFile := config.BootFile
		if bootFile == "" {
			bootFile = "undionly.kpxe"
		}
		if config.NextServer != "" {
			fmt.Fprintf(&content, "dhcp-boot=tag:bios,%s,,%s\n\n", bootFile, config.NextServer)
		} else {
			fmt.Fprintf(&content, "dhcp-boot=tag:bios,%s\n\n", bootFile)
		}

		// UEFI clients
		content.WriteString("# UEFI PXE Boot\n")
		content.WriteString("dhcp-match=set:efi64,option:client-arch,7\n")
		content.WriteString("dhcp-match=set:efi64,option:client-arch,9\n")
		content.WriteString("dhcp-boot=tag:efi64,ipxe.efi\n\n")

		// iPXE chainloading
		content.WriteString("# iPXE Chainloading\n")
		content.WriteString("dhcp-match=set:ipxe,175\n")

		// Determine server IP for HTTP boot
		// This should be the external IP address of the server running Nginx/Backend
		serverIP := config.BootServerIP
		if serverIP == "" {
			serverIP = config.NextServer
		}
		// Use static external IP from environment if available (robust Docker default)
		extIP := os.Getenv("EXTERNAL_IP")
		if serverIP == "" {
			serverIP = extIP
		}
		if serverIP == "" {
			serverIP = config.TFTPServer
		}
		if serverIP == "" {
			serverIP = config.Gateway
		}
		if serverIP == "" {
			// Absolute fallback
			serverIP = "192.168.10.1"
		}

		// Use port 80 (Nginx) which proxies to backend:8000
		// We point to a static init script which handles the variable expansion safely within iPXE
		fmt.Fprintf(&content, "dhcp-boot=tag:ipxe,http://%s/api/v1/pxe/init\n\n", serverIP)
	}

	content.WriteString("# Logging\n")
	content.WriteString("log-dhcp\n")

	// Atomic write: tempfile + rename. The reload trigger is the caller's
	// responsibility (see GenerateDnsmasqConfig) — writing it from here
	// would race the hosts-config write that follows.
	if err := writeFileAtomic(dnsmasqMainConfig, []byte(content.String()), 0644); err != nil {
		return err
	}

	return nil
}

// generateHostsConfig generates DHCP reservations from both nodes and explicit reservations
func generateHostsConfig() error {
	var content strings.Builder
	content.WriteString("# Auto-generated DHCP Host Reservations by OS-Baka\n")
	content.WriteString("# Do not edit manually - changes will be overwritten\n\n")

	// Get active DHCP reservations
	var reservations []model.DHCPReservation
	getDB().Where("is_active = ?", true).Find(&reservations)

	if len(reservations) > 0 {
		content.WriteString("# Explicit DHCP Reservations\n")
		for _, r := range reservations {
			if r.MACAddress != "" && r.IPAddress != "" {
				// Format: dhcp-host=<mac>,<ip>,<hostname>
				if r.Hostname != "" {
					fmt.Fprintf(&content, "dhcp-host=%s,%s,%s\n",
						r.MACAddress, r.IPAddress, r.Hostname)
				} else {
					fmt.Fprintf(&content, "dhcp-host=%s,%s\n",
						r.MACAddress, r.IPAddress)
				}
			}
		}
		content.WriteString("\n")
	}

	// Get nodes that don't have explicit reservations
	var nodes []model.Node
	getDB().Find(&nodes)

	// Create a map of existing reservations
	reservedMACs := make(map[string]bool)
	for _, r := range reservations {
		reservedMACs[strings.ToLower(r.MACAddress)] = true
	}

	// Add nodes that aren't already reserved
	nodeReservations := 0
	for _, node := range nodes {
		if node.MACAddress == "" || node.IPAddress == "" {
			continue
		}
		if reservedMACs[strings.ToLower(node.MACAddress)] {
			continue
		}

		if nodeReservations == 0 {
			content.WriteString("# Node-based Reservations\n")
		}

		if node.Hostname != "" {
			fmt.Fprintf(&content, "dhcp-host=%s,%s,%s\n",
				node.MACAddress, node.IPAddress, node.Hostname)
		} else {
			fmt.Fprintf(&content, "dhcp-host=%s,%s\n",
				node.MACAddress, node.IPAddress)
		}
		nodeReservations++
	}

	if err := writeFileAtomic(dnsmasqHostsConfig, []byte(content.String()), 0644); err != nil {
		return err
	}

	return nil
}

