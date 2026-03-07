package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	tests := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		got := w.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}

	// CSP should be present
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header should be set")
	}

	// Permissions-Policy should be present
	pp := w.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("Permissions-Policy header should be set")
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		id, _ := c.Get("request_id")
		c.String(200, id.(string))
	})

	// Test auto-generated ID
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	responseID := w.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("X-Request-ID header should be set")
	}
	if len(responseID) != 16 { // 8 bytes hex = 16 chars
		t.Errorf("request ID length = %d, want 16", len(responseID))
	}

	// Test pass-through of existing ID
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Request-ID", "my-custom-id-123")
	r.ServeHTTP(w2, req2)

	gotID := w2.Header().Get("X-Request-ID")
	if gotID != "my-custom-id-123" {
		t.Errorf("X-Request-ID = %q, want %q", gotID, "my-custom-id-123")
	}

	// Body should also contain the passed-through ID
	if w2.Body.String() != "my-custom-id-123" {
		t.Errorf("body = %q, want %q", w2.Body.String(), "my-custom-id-123")
	}
}

func TestRateLimiter(t *testing.T) {
	// Test the token bucket directly to avoid Gin's ClientIP resolution issues in test mode
	rl := newRateLimiter(3, time.Second, 3) // 3 tokens per second, burst 3

	// First 3 requests should succeed (within burst)
	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4") {
			t.Errorf("request %d: should be allowed", i+1)
		}
	}

	// 4th request should be rate-limited (bucket exhausted)
	if rl.allow("1.2.3.4") {
		t.Error("request 4: should be rate-limited")
	}

	// Different key should not be rate-limited
	if !rl.allow("5.6.7.8") {
		t.Error("different key should be allowed")
	}
}

func TestFieldEncryption(t *testing.T) {
	// Without FIELD_ENCRYPTION_KEY, encryption should be a no-op
	plaintext := "my-secret-password"
	result := EncryptField(plaintext)

	// Without key, should return plaintext unchanged
	if result != plaintext {
		t.Errorf("EncryptField without key should return plaintext, got %q", result)
	}

	// Decrypt of non-encrypted value should return as-is
	decrypted := DecryptField(plaintext)
	if decrypted != plaintext {
		t.Errorf("DecryptField of plaintext should return plaintext, got %q", decrypted)
	}

	// Empty string should pass through
	if EncryptField("") != "" {
		t.Error("EncryptField of empty string should return empty string")
	}
}
