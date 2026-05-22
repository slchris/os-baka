package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicWritesContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.conf")

	want := []byte("dhcp-host=aa:bb:cc:dd:ee:ff,192.168.10.50\n")
	if err := writeFileAtomic(target, want, 0644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("content mismatch:\n got:  %q\n want: %q", got, want)
	}
}

func TestWriteFileAtomicLeavesNoTempfileOnSuccess(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.conf")

	if err := writeFileAtomic(target, []byte("ok"), 0644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.conf" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected exactly [config.conf] in %s, got %v", dir, names)
	}
}

func TestWriteFileAtomicReplacesExisting(t *testing.T) {
	// Existing file content must be wholly replaced — not appended or
	// merged. This is the property the dnsmasq pipeline depends on.
	dir := t.TempDir()
	target := filepath.Join(dir, "config.conf")
	if err := os.WriteFile(target, []byte("old content stale"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	want := []byte("new")
	if err := writeFileAtomic(target, want, 0644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Errorf("content not replaced: got %q want %q", got, want)
	}
}

func TestWriteFileAtomicFailsCleanlyOnUnwritableDir(t *testing.T) {
	// Writing to a non-existent dir must fail and not leave anything.
	target := "/nonexistent-osbaka-test/path/config.conf"
	err := writeFileAtomic(target, []byte("x"), 0644)
	if err == nil {
		t.Fatal("expected error writing to non-existent dir")
	}
}

func TestGenerateDnsmasqConfigSkipsWhenDirMissing(t *testing.T) {
	// Dev mode: /etc/dnsmasq.d is not bind-mounted in. GenerateDnsmasqConfig
	// must return nil (not error) and must not attempt any file ops.
	prevDir := dnsmasqHostsDir
	prevWarned := dnsmasqMissingDirWarned.Load()
	t.Cleanup(func() {
		dnsmasqHostsDir = prevDir
		dnsmasqMissingDirWarned.Store(prevWarned)
	})

	dnsmasqHostsDir = "/definitely-not-a-real-path-osbaka-test"
	dnsmasqMissingDirWarned.Store(false)

	// Call twice — second call should be quiet (the atomic.Bool flips
	// after the first). Both must return nil.
	for i := 0; i < 2; i++ {
		if err := GenerateDnsmasqConfig(); err != nil {
			t.Fatalf("call %d returned error: %v (want nil for missing dir)", i, err)
		}
	}
	if !dnsmasqMissingDirWarned.Load() {
		t.Error("expected the missing-dir warn flag to be set after first call")
	}
}

// TestGenerateDnsmasqConfigTriggerOrderingDocumentsBehavior is a documenting
// test — it confirms the package-level constants point at the same parent
// directory so the reload trigger lands next to the config files. If
// someone moves the trigger to a different dir, atomicity guarantees break
// (rename across filesystems is not atomic) and so does the watcher's
// "see file → HUP" assumption.
func TestGenerateDnsmasqConfigTriggerLivesAlongsideConfigs(t *testing.T) {
	if filepath.Dir(dnsmasqMainConfig) != dnsmasqHostsDir {
		t.Errorf("dnsmasqMainConfig %q not under dnsmasqHostsDir %q", dnsmasqMainConfig, dnsmasqHostsDir)
	}
	if filepath.Dir(dnsmasqHostsConfig) != dnsmasqHostsDir {
		t.Errorf("dnsmasqHostsConfig %q not under dnsmasqHostsDir %q", dnsmasqHostsConfig, dnsmasqHostsDir)
	}
	if filepath.Dir(dnsmasqReloadTrigger) != dnsmasqHostsDir {
		t.Errorf("dnsmasqReloadTrigger %q not under dnsmasqHostsDir %q", dnsmasqReloadTrigger, dnsmasqHostsDir)
	}
}

