package api

import (
	"strings"
	"testing"
)

func TestGenerateSecurePassphraseLength(t *testing.T) {
	for _, n := range []int{1, 16, 32, 64, 128} {
		t.Run(itoaSimple(n), func(t *testing.T) {
			pw, err := generateSecurePassphrase(n)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if len(pw) != n {
				t.Errorf("len(pw) = %d, want %d", len(pw), n)
			}
		})
	}
}

func TestGenerateSecurePassphraseShellSafe(t *testing.T) {
	// Generate a bunch and confirm none contain shell-special bytes.
	// 32 chars * 1000 iterations = 32000 chars sampled. With the new
	// 64-char charset this is comfortably enough to flag a regression
	// that re-introduced $!@#%^&*()=+ etc.
	const iterations = 1000
	disallowed := []byte("!@#$%^&*()=+`\"'\\ \t\r\n")
	for i := 0; i < iterations; i++ {
		pw, err := generateSecurePassphrase(32)
		if err != nil {
			t.Fatalf("iter %d: err: %v", i, err)
		}
		for j := 0; j < len(pw); j++ {
			for _, bad := range disallowed {
				if pw[j] == bad {
					t.Fatalf("iter %d: pw contains disallowed byte 0x%02x (%q) at %d: %q", i, bad, string(bad), j, pw)
				}
			}
		}
	}
}

func TestValidatePassphraseAccepts(t *testing.T) {
	// Operator-supplied passphrases including shell metacharacters MUST
	// pass — the shell-quoting story is handled downstream via base64.
	// Only NUL and newline are rejected (debconf / preseed structure).
	good := []string{
		"",                                  // empty allowed; caller gates "required-when-encryption"
		"plain-passphrase",
		"with spaces and \ttab",             // tab is fine for debconf
		`with "double" and 'single' quotes`,
		`with $dollar and ` + "`backtick`",
		`with \backslash and !bang`,
		"unicode: 中文 emoji 🔐",
		strings.Repeat("a", 1024),           // long passphrase
	}
	for _, pw := range good {
		t.Run(pw, func(t *testing.T) {
			if err := validatePassphrase(pw); err != nil {
				t.Errorf("expected %q to be accepted, got %v", pw, err)
			}
		})
	}
}

func TestValidatePassphraseRejects(t *testing.T) {
	bad := []struct {
		name string
		pw   string
	}{
		{"nul-at-start", "\x00abc"},
		{"nul-in-middle", "ab\x00cd"},
		{"newline", "first-line\nsecond-line"},
		{"carriage-return", "first\rsecond"},
		{"crlf", "first\r\nsecond"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if err := validatePassphrase(tc.pw); err == nil {
				t.Errorf("expected %q to be rejected", tc.pw)
			}
		})
	}
}

// itoaSimple avoids importing strconv just for one test helper.
func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 4)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
