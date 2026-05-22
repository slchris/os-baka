package api

import "regexp"

// diskPathPattern is a deliberately loose regex — it rejects obviously
// hostile input (shell metacharacters, NUL bytes, missing /dev prefix)
// but accepts the four shapes operators actually use:
//
//   /dev/sda
//   /dev/nvme0n1
//   /dev/vda
//   /dev/disk/by-id/scsi-SATA_Samsung_SSD_870_S6...
//
// Tighter validation isn't valuable: if the path is wrong, d-i fails the
// install loudly. Worth catching: a free-form string from a form that
// would inject shell args into the preseed.
var diskPathPattern = regexp.MustCompile(`^/dev/[a-zA-Z0-9/_.-]+$`)

// isValidDiskPath reports whether s is a plausible block-device path.
// Empty strings are NOT considered valid here — callers gate the check
// with their own "empty means use default" branch.
func isValidDiskPath(s string) bool {
	return diskPathPattern.MatchString(s)
}

// buildDiskSelection returns the preseed snippet that tells partman which
// device to install onto. Two modes:
//
//   1. Operator pinned a path via Node.TargetDisk (e.g. "/dev/nvme0n1"):
//      emit a static `partman-auto/disk` line. Use this when auto-detect
//      can't disambiguate (multiple same-size disks, controllers with
//      odd naming, data-disk-must-not-be-touched scenarios).
//
//   2. Field empty: emit a `partman/early_command` that runs in the d-i
//      initramfs BEFORE partman scans devices. The script picks the
//      largest non-removable non-read-only block device, preferring
//      nvme* > vd* (virtio) > sd*. Result is fed back via debconf-set.
//
// The early_command uses only tools that are available in the d-i
// initramfs: busybox sh, lsblk, awk, sort, head, debconf-set. No bash,
// no GNU extensions.
//
// On detection failure (no eligible disk found), the script leaves
// partman-auto/disk unset — partman will then prompt for a manual choice
// rather than guessing wrong and erasing the wrong drive.
func buildDiskSelection(targetDisk string) string {
	if targetDisk != "" {
		return "d-i partman-auto/disk string " + targetDisk
	}
	// Auto-detect path.
	//
	// lsblk -dn -o NAME,TYPE,SIZE,RM,RO produces lines like:
	//     nvme0n1 disk 512110190592 0 0
	//     sda     disk 1000204886016 0 0
	//     sdb     disk 15376000000   1 0   <- USB stick (RM=1), skip
	//
	// We filter to TYPE=disk, RM=0, RO=0, then sort by name prefix
	// preference and pick the largest within the top preference bucket.
	//
	// awk pipeline:
	//   - $2=="disk" && $4=="0" && $5=="0": real, non-removable, writable
	//   - Tag with priority 1/2/3 for nvme/vd/sd; 9 for anything else
	//   - sort numerically (priority asc, size desc) → first line wins
	//
	// The chosen path is /dev/$NAME. If no match, leave debconf alone.
	return `d-i partman/early_command string \
    chosen=$(lsblk -dn -o NAME,TYPE,SIZE,RM,RO 2>/dev/null | \
        awk '$2=="disk" && $4=="0" && $5=="0" { \
            pri=9; \
            if ($1 ~ /^nvme/) pri=1; \
            else if ($1 ~ /^vd/) pri=2; \
            else if ($1 ~ /^sd/) pri=3; \
            printf "%d %d /dev/%s\n", pri, $3, $1; \
        }' | \
        sort -k1,1n -k2,2nr | head -n1 | awk '{print $3}'); \
    if [ -n "$chosen" ]; then \
        echo "OS-Baka: auto-selected $chosen" >&2; \
        debconf-set partman-auto/disk "$chosen"; \
    else \
        echo "OS-Baka: no eligible disk; partman will prompt" >&2; \
    fi`
}
