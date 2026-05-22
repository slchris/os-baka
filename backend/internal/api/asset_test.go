package api

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIsSafeAssetFilenameAccepts(t *testing.T) {
	good := []string{
		"vmlinuz",
		"initrd.img",
		"ubuntu-22.04-server-amd64.iso",
		"a",
		strings.Repeat("a", 200),
	}
	for _, n := range good {
		t.Run("good/"+n, func(t *testing.T) {
			if !isSafeAssetFilename(n) {
				t.Errorf("expected %q to validate", n)
			}
		})
	}
}

func TestIsSafeAssetFilenameRejects(t *testing.T) {
	bad := []struct {
		name string
		val  string
	}{
		{"empty", ""},
		{"dot", "."},
		{"dotdot", ".."},
		{"hidden", ".bashrc"},
		{"slash", "etc/passwd"},
		{"backslash", `etc\passwd`},
		{"absolute", "/etc/passwd"},
		{"traversal-prefix", "../etc/passwd"},
		{"space", "ubuntu installer.iso"},
		{"quote", `name"with quote`},
		{"shell-meta-dollar", "name$1"},
		{"unicode", "vmlinuz_中文"},
		{"too-long-201", strings.Repeat("a", 201)},
		{"semicolon", "vmlinuz;rm"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if isSafeAssetFilename(tc.val) {
				t.Errorf("expected %q to be rejected", tc.val)
			}
		})
	}
}

func TestResolveSafeAssetPathStaysUnderBaseDir(t *testing.T) {
	base := t.TempDir()
	got, err := resolveSafeAssetPath(base, "vmlinuz")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	want := filepath.Join(base, "vmlinuz")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveSafeAssetPathRejectsTraversal(t *testing.T) {
	// Even if a caller forgets to validate, the secondary "stays under
	// baseDir" check catches escape attempts. These are the exact inputs
	// that survived the old `filepath.Base` alone.
	base := t.TempDir()
	bad := []string{
		"..",         // would resolve to parent of baseDir
		".",          // resolves to baseDir itself, not a file
		"/etc/passwd",
		"foo/../bar", // looks innocent but contains separators
	}
	for _, name := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := resolveSafeAssetPath(base, name); err == nil {
				t.Errorf("expected error for %q", name)
			}
		})
	}
}
