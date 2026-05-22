package api

import (
	"testing"
)

func TestResolveSecurityMirrorPrecedence(t *testing.T) {
	// Cover the four-level precedence. Each subtest sets env / args
	// and asserts the expected winner. node > env > db > built-in.
	t.Run("node override wins", func(t *testing.T) {
		t.Setenv("PXE_DEBIAN_SECURITY_MIRROR_URL", "http://env.example/sec")
		got := resolveSecurityMirror("debian", "http://node.example/sec", "http://db.example/sec")
		if got != "http://node.example/sec" {
			t.Errorf("got %q, want node override", got)
		}
	})
	t.Run("env beats db and default", func(t *testing.T) {
		t.Setenv("PXE_DEBIAN_SECURITY_MIRROR_URL", "http://env.example/sec")
		got := resolveSecurityMirror("debian", "", "http://db.example/sec")
		if got != "http://env.example/sec" {
			t.Errorf("got %q, want env override", got)
		}
	})
	t.Run("db beats default", func(t *testing.T) {
		t.Setenv("PXE_DEBIAN_SECURITY_MIRROR_URL", "")
		got := resolveSecurityMirror("debian", "", "http://db.example/sec")
		if got != "http://db.example/sec" {
			t.Errorf("got %q, want db value", got)
		}
	})
	t.Run("debian built-in default", func(t *testing.T) {
		t.Setenv("PXE_DEBIAN_SECURITY_MIRROR_URL", "")
		got := resolveSecurityMirror("debian", "", "")
		if got != "http://security.debian.org/debian-security" {
			t.Errorf("got %q, want security.debian.org default", got)
		}
	})
	t.Run("ubuntu built-in default", func(t *testing.T) {
		t.Setenv("PXE_UBUNTU_SECURITY_MIRROR_URL", "")
		got := resolveSecurityMirror("ubuntu", "", "")
		if got != "http://security.ubuntu.com/ubuntu" {
			t.Errorf("got %q, want security.ubuntu.com default", got)
		}
	})
	t.Run("unknown os returns empty", func(t *testing.T) {
		got := resolveSecurityMirror("plan9", "", "")
		if got != "" {
			t.Errorf("got %q, want empty for unknown OS", got)
		}
	})
	t.Run("trailing slash trimmed", func(t *testing.T) {
		got := resolveSecurityMirror("debian", "http://node.example/sec/", "")
		if got != "http://node.example/sec" {
			t.Errorf("got %q, want trailing slash stripped", got)
		}
	})
}

func TestSplitMirrorURL(t *testing.T) {
	cases := []struct {
		in   string
		host string
		path string
	}{
		{"", "", "/"},
		{"http://deb.debian.org/debian", "deb.debian.org", "/debian"},
		{"http://security.debian.org/debian-security", "security.debian.org", "/debian-security"},
		{"http://mirror.intra:8080/debian", "mirror.intra:8080", "/debian"},
		{"https://secure.intra/debian", "secure.intra", "/debian"},
		{"http://host-only", "host-only", "/"}, // no path component
		{"not a url at all", "", "/"},          // url.Parse returns no Host
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			h, p := splitMirrorURL(tc.in)
			if h != tc.host || p != tc.path {
				t.Errorf("splitMirrorURL(%q) = (%q, %q), want (%q, %q)",
					tc.in, h, p, tc.host, tc.path)
			}
		})
	}
}

func TestIsValidMirrorURL(t *testing.T) {
	good := []string{
		"http://deb.debian.org/debian",
		"http://mirror.intra:8080/debian-security",
		"https://secure.example/path/to/mirror",
	}
	for _, u := range good {
		t.Run("good/"+u, func(t *testing.T) {
			if !isValidMirrorURL(u) {
				t.Errorf("expected %q to validate", u)
			}
		})
	}
	bad := []string{
		"",                          // empty rejected — caller's job to allow-empty
		"deb.debian.org/debian",     // no scheme
		"ftp://mirror.example",      // wrong scheme
		"file:///var/mirror",        // wrong scheme
		"http://",                   // no host
		"://no-scheme",              // garbage
	}
	for _, u := range bad {
		t.Run("bad/"+u, func(t *testing.T) {
			if isValidMirrorURL(u) {
				t.Errorf("expected %q to be rejected", u)
			}
		})
	}
}
