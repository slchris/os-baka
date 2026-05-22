package api

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestHashPXETokenIsDeterministic(t *testing.T) {
	a := hashPXEToken("abc")
	b := hashPXEToken("abc")
	if a != b {
		t.Errorf("hash should be deterministic: %q vs %q", a, b)
	}
	if a == hashPXEToken("abd") {
		t.Error("different inputs must produce different hashes")
	}
	if len(a) != 64 {
		t.Errorf("expected hex SHA-256 (64 chars), got %d", len(a))
	}
}

func TestInstallTimeoutFromEnvDefault(t *testing.T) {
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "")
	if got := installTimeoutFromEnv(); got != defaultInstallTimeoutMinutes {
		t.Errorf("default install timeout = %d, want %d", got, defaultInstallTimeoutMinutes)
	}
}

func TestInstallTimeoutFromEnvOverride(t *testing.T) {
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "45")
	if got := installTimeoutFromEnv(); got != 45 {
		t.Errorf("got %d, want 45", got)
	}
}

func TestInstallTimeoutFromEnvInvalidFallsBack(t *testing.T) {
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "garbage")
	if got := installTimeoutFromEnv(); got != defaultInstallTimeoutMinutes {
		t.Errorf("invalid value should fall back to default; got %d", got)
	}
}

func TestInstallTimeoutFromEnvRejectsZeroAndNegative(t *testing.T) {
	for _, v := range []string{"0", "-1", "-100"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("INSTALL_TIMEOUT_MINUTES", v)
			if got := installTimeoutFromEnv(); got != defaultInstallTimeoutMinutes {
				t.Errorf("value %q should be rejected (non-positive), got %d", v, got)
			}
		})
	}
}

func TestPXETokenTTLDerivedFromInstallTimeout(t *testing.T) {
	// Neither env set: token TTL defaults to 2x the install timeout
	// default (120 → 240 minutes). Verify the derivation rather than
	// the magic number so the test stays right when the default changes.
	t.Setenv("PXE_TOKEN_TTL_MINUTES", "")
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "")
	want := time.Duration(defaultInstallTimeoutMinutes*pxeTokenTTLMultiplier) * time.Minute
	if got := pxeTokenTTL(); got != want {
		t.Errorf("default TTL = %v, want %v (= %dx install timeout %d min)",
			got, want, pxeTokenTTLMultiplier, defaultInstallTimeoutMinutes)
	}
}

func TestPXETokenTTLDerivedFollowsInstallTimeoutEnv(t *testing.T) {
	// Operator only sets INSTALL_TIMEOUT_MINUTES; token TTL must follow.
	// This is the "one knob to rule them all" property we documented.
	t.Setenv("PXE_TOKEN_TTL_MINUTES", "")
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "30")
	if got, want := pxeTokenTTL(), 60*time.Minute; got != want {
		t.Errorf("token TTL = %v, want %v (= 2x INSTALL_TIMEOUT_MINUTES)", got, want)
	}
}

func TestPXETokenTTLEnvOverride(t *testing.T) {
	// Explicit PXE_TOKEN_TTL_MINUTES wins over the derived default,
	// even when INSTALL_TIMEOUT_MINUTES is also set.
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "120")
	t.Setenv("PXE_TOKEN_TTL_MINUTES", "30")
	if got, want := pxeTokenTTL(), 30*time.Minute; got != want {
		t.Errorf("TTL = %v, want %v (explicit env wins)", got, want)
	}
}

func TestPXETokenTTLInvalidFallsBackToDerivedDefault(t *testing.T) {
	t.Setenv("INSTALL_TIMEOUT_MINUTES", "")
	t.Setenv("PXE_TOKEN_TTL_MINUTES", "garbage")
	want := time.Duration(defaultInstallTimeoutMinutes*pxeTokenTTLMultiplier) * time.Minute
	if got := pxeTokenTTL(); got != want {
		t.Errorf("invalid TTL should fall back to derived default, got %v want %v", got, want)
	}
}

func TestPXETokenRequiredDefaultTrue(t *testing.T) {
	// Make sure no env leak from other tests
	_ = os.Unsetenv("PXE_REQUIRE_TOKEN")
	if !pxeTokenRequired() {
		t.Error("token gate must default to ON — opt-out should require explicit env")
	}
}

func TestPXETokenRequiredOptOut(t *testing.T) {
	cases := []string{"0", "false", "False", "no"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("PXE_REQUIRE_TOKEN", v)
			if pxeTokenRequired() {
				t.Errorf("PXE_REQUIRE_TOKEN=%q should disable the gate", v)
			}
		})
	}
}

func TestPXETokenRequiredStaysOnForUnknownValues(t *testing.T) {
	// Anything that isn't a recognized "off" string keeps the gate on —
	// a typo in the env var must not silently disable security.
	t.Setenv("PXE_REQUIRE_TOKEN", "maybe")
	if !pxeTokenRequired() {
		t.Error("unknown values for PXE_REQUIRE_TOKEN must NOT disable the gate")
	}
}

func TestFieldEncryptionRoundTripWithKey(t *testing.T) {
	// 32 bytes of '0' as the AES key
	t.Setenv("FIELD_ENCRYPTION_KEY", strings.Repeat("00", 32))

	// Re-set the package-level cipher so the new key takes effect even if a
	// prior test in the same process initialized it once.
	resetFieldCipherForTest()

	plain := "root-password-with-symbols!@#$"
	enc := EncryptField(plain)

	if enc == plain {
		t.Fatal("EncryptField returned plaintext — key not loaded?")
	}
	if !strings.HasPrefix(enc, fieldEncryptionPrefix) {
		t.Errorf("ciphertext should start with %q, got %q", fieldEncryptionPrefix, enc)
	}

	if got := DecryptField(enc); got != plain {
		t.Errorf("round-trip failed: got %q want %q", got, plain)
	}

	// Re-encrypting an already-encrypted value should be a no-op.
	if again := EncryptField(enc); again != enc {
		t.Errorf("double-encrypting changed value: %q -> %q", enc, again)
	}
}
