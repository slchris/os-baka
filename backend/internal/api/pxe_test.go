package api

import (
	"testing"
)

func TestNormalizeMac(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"colon format lowercase", "aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff"},
		{"colon format uppercase", "AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff"},
		{"hyphen format", "aa-bb-cc-dd-ee-ff", "aa:bb:cc:dd:ee:ff"},
		{"hyphen format uppercase", "AA-BB-CC-DD-EE-FF", "aa:bb:cc:dd:ee:ff"},
		{"mixed case", "Aa:Bb:Cc:Dd:Ee:Ff", "aa:bb:cc:dd:ee:ff"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeMac(tt.input)
			if got != tt.want {
				t.Errorf("normalizeMac(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveMirror(t *testing.T) {
	tests := []struct {
		name       string
		osType     string
		nodeMirror string
		dbMirror   string
		want       string
	}{
		// Priority 1: Node-specific mirror
		{
			name:       "node mirror takes priority",
			osType:     "ubuntu",
			nodeMirror: "http://my-mirror.local/ubuntu",
			dbMirror:   "http://db-mirror.local/ubuntu",
			want:       "http://my-mirror.local/ubuntu",
		},
		// Node mirror with trailing slash should be trimmed
		{
			name:       "node mirror trailing slash trimmed",
			osType:     "ubuntu",
			nodeMirror: "http://my-mirror.local/ubuntu/",
			dbMirror:   "",
			want:       "http://my-mirror.local/ubuntu",
		},
		// DB mirror fallback for ubuntu
		{
			name:       "ubuntu db mirror fallback",
			osType:     "ubuntu",
			nodeMirror: "",
			dbMirror:   "http://db-mirror.local/ubuntu",
			want:       "http://db-mirror.local/ubuntu",
		},
		// Default for ubuntu
		{
			name:       "ubuntu default",
			osType:     "ubuntu",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://archive.ubuntu.com/ubuntu",
		},
		// Default for linux (same as ubuntu)
		{
			name:       "linux default (same as ubuntu)",
			osType:     "linux",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://archive.ubuntu.com/ubuntu",
		},
		// Default for debian
		{
			name:       "debian default",
			osType:     "debian",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://deb.debian.org/debian",
		},
		// DB mirror fallback for debian
		{
			name:       "debian db mirror",
			osType:     "debian",
			nodeMirror: "",
			dbMirror:   "http://db-mirror.local/debian",
			want:       "http://db-mirror.local/debian",
		},
		// Default for centos
		{
			name:       "centos default",
			osType:     "centos",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://mirror.centos.org/centos",
		},
		// RHEL uses centos path
		{
			name:       "rhel default",
			osType:     "rhel",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://mirror.centos.org/centos",
		},
		// Rocky uses centos path
		{
			name:       "rocky default",
			osType:     "rocky",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://mirror.centos.org/centos",
		},
		// Unknown OS with DB mirror
		{
			name:       "unknown os with db mirror",
			osType:     "freebsd",
			nodeMirror: "",
			dbMirror:   "http://custom-mirror.local",
			want:       "http://custom-mirror.local",
		},
		// Unknown OS without any mirror
		{
			name:       "unknown os default fallback",
			osType:     "freebsd",
			nodeMirror: "",
			dbMirror:   "",
			want:       "http://archive.ubuntu.com/ubuntu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveMirror(tt.osType, tt.nodeMirror, tt.dbMirror)
			if got != tt.want {
				t.Errorf("resolveMirror(%q, %q, %q) = %q, want %q",
					tt.osType, tt.nodeMirror, tt.dbMirror, got, tt.want)
			}
		})
	}
}

func TestResolveMirrorEnvOverride(t *testing.T) {
	// Test env var override (Priority 2: PXE_MIRROR_URL)
	t.Setenv("PXE_MIRROR_URL", "http://env-mirror.local/ubuntu")

	got := resolveMirror("ubuntu", "", "")
	want := "http://env-mirror.local/ubuntu"
	if got != want {
		t.Errorf("resolveMirror() with PXE_MIRROR_URL = %q, want %q", got, want)
	}
}

func TestResolveMirrorOSSpecificEnv(t *testing.T) {
	// Test OS-specific env var (Priority 3)
	t.Setenv("PXE_UBUNTU_MIRROR_URL", "http://ubuntu-env.local/ubuntu")

	got := resolveMirror("ubuntu", "", "")
	want := "http://ubuntu-env.local/ubuntu"
	if got != want {
		t.Errorf("resolveMirror() with PXE_UBUNTU_MIRROR_URL = %q, want %q", got, want)
	}
}

func TestResolveMirrorDebianSpecificEnv(t *testing.T) {
	t.Setenv("PXE_DEBIAN_MIRROR_URL", "http://debian-env.local/debian")

	got := resolveMirror("debian", "", "")
	want := "http://debian-env.local/debian"
	if got != want {
		t.Errorf("resolveMirror() with PXE_DEBIAN_MIRROR_URL = %q, want %q", got, want)
	}
}

func TestResolveMirrorCentOSSpecificEnv(t *testing.T) {
	t.Setenv("PXE_CENTOS_MIRROR_URL", "http://centos-env.local/centos")

	got := resolveMirror("centos", "", "")
	want := "http://centos-env.local/centos"
	if got != want {
		t.Errorf("resolveMirror() with PXE_CENTOS_MIRROR_URL = %q, want %q", got, want)
	}
}

func TestResolveMirrorNodePriorityOverEnv(t *testing.T) {
	// Node mirror should take priority over env
	t.Setenv("PXE_MIRROR_URL", "http://env-mirror.local")

	got := resolveMirror("ubuntu", "http://node-mirror.local", "")
	want := "http://node-mirror.local"
	if got != want {
		t.Errorf("resolveMirror() node mirror should override env: got %q, want %q", got, want)
	}
}
