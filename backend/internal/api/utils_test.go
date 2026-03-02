package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ========= ParseIDParam Tests =========

func TestParseIDParam(t *testing.T) {
	tests := []struct {
		name     string
		paramVal string
		wantID   int
		wantOK   bool
		wantCode int
	}{
		{"valid ID", "1", 1, true, 0},
		{"valid large ID", "999", 999, true, 0},
		{"zero ID", "0", 0, false, http.StatusBadRequest},
		{"negative ID", "-1", 0, false, http.StatusBadRequest},
		{"non-numeric", "abc", 0, false, http.StatusBadRequest},
		{"empty string", "", 0, false, http.StatusBadRequest},
		{"float", "1.5", 0, false, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.paramVal}}

			gotID, gotOK := ParseIDParam(c, "id")

			if gotOK != tt.wantOK {
				t.Errorf("ParseIDParam() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotOK && gotID != tt.wantID {
				t.Errorf("ParseIDParam() id = %v, want %v", gotID, tt.wantID)
			}
			if !gotOK && w.Code != tt.wantCode {
				t.Errorf("ParseIDParam() status = %v, want %v", w.Code, tt.wantCode)
			}
		})
	}
}

// ========= ErrorResponse Tests =========

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorResponse(c, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("ErrorResponse() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("ErrorResponse() body is empty")
	}
	if !strings.Contains(body, "not found") {
		t.Error("ErrorResponse() body does not contain error message")
	}
}

func TestErrorResponseVariousCodes(t *testing.T) {
	tests := []struct {
		code    int
		message string
	}{
		{http.StatusBadRequest, "bad request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusInternalServerError, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ErrorResponse(c, tt.code, tt.message)

			if w.Code != tt.code {
				t.Errorf("ErrorResponse() status = %v, want %v", w.Code, tt.code)
			}
			if !strings.Contains(w.Body.String(), tt.message) {
				t.Errorf("ErrorResponse() body = %q, want to contain %q", w.Body.String(), tt.message)
			}
		})
	}
}

// ========= HashPassword / CheckPasswordHash Tests =========

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == "test-password" {
		t.Fatal("HashPassword() returned plaintext password")
	}
}

func TestHashPasswordDifferentOutputs(t *testing.T) {
	hash1, _ := HashPassword("test-password")
	hash2, _ := HashPassword("test-password")

	// bcrypt should produce different hashes for the same input (due to salt)
	if hash1 == hash2 {
		t.Error("HashPassword() should produce different hashes due to salt")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "my-secret-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash() returned false for correct password")
	}
}

func TestCheckPasswordHashWrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	if CheckPasswordHash("wrong-password", hash) {
		t.Error("CheckPasswordHash() returned true for wrong password")
	}
}

func TestCheckPasswordHashInvalidHash(t *testing.T) {
	if CheckPasswordHash("password", "not-a-valid-hash") {
		t.Error("CheckPasswordHash() returned true for invalid hash")
	}
}

// ========= GenerateToken Tests =========

func TestGenerateToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret-key-12345"

	token, err := GenerateToken(1, "testuser", false, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	// JWT should have 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("GenerateToken() token has %d parts, want 3", len(parts))
	}
}

func TestGenerateTokenDifferentUsers(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret-key-12345"

	token1, _ := GenerateToken(1, "user1", false, cfg)
	token2, _ := GenerateToken(2, "user2", false, cfg)

	if token1 == token2 {
		t.Error("GenerateToken() should produce different tokens for different users")
	}
}

// ========= AuthMiddleware Tests =========

func TestAuthMiddlewareNoHeader(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler := AuthMiddleware(cfg)
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware() without header: status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "InvalidFormat")

	handler := AuthMiddleware(cfg)
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware() with invalid format: status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token-string")

	handler := AuthMiddleware(cfg)
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware() with invalid token: status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "test-secret-key-12345"

	// Generate a valid token
	token, err := GenerateToken(42, "admin", true, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	var capturedUserID uint
	var capturedUsername string

	r.GET("/test", AuthMiddleware(cfg), func(c *gin.Context) {
		id, _ := GetAuthUserID(c)
		capturedUserID = id
		capturedUsername = GetAuthUsername(c)
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Errorf("AuthMiddleware() with valid token: status = %v, want %v", w.Code, http.StatusOK)
	}
	if capturedUserID != 42 {
		t.Errorf("AuthMiddleware() userID = %v, want 42", capturedUserID)
	}
	if capturedUsername != "admin" {
		t.Errorf("AuthMiddleware() username = %q, want %q", capturedUsername, "admin")
	}
}

func TestAuthMiddlewareWrongSecret(t *testing.T) {
	cfg1 := &config.Config{}
	cfg1.Server.SecretKey = "secret-1"
	cfg2 := &config.Config{}
	cfg2.Server.SecretKey = "secret-2"

	// Token signed with secret-1
	token, _ := GenerateToken(1, "user", false, cfg1)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", AuthMiddleware(cfg2), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware() with wrong secret: status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

// ========= GetAuthUserID / GetAuthUsername Tests =========

func TestGetAuthUserIDNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id, ok := GetAuthUserID(c)
	if ok {
		t.Error("GetAuthUserID() should return false when not set")
	}
	if id != 0 {
		t.Errorf("GetAuthUserID() id = %v, want 0", id)
	}
}

func TestGetAuthUserIDSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(42))

	id, ok := GetAuthUserID(c)
	if !ok {
		t.Error("GetAuthUserID() should return true when set")
	}
	if id != 42 {
		t.Errorf("GetAuthUserID() id = %v, want 42", id)
	}
}

func TestGetAuthUserIDWrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "not-a-uint") // wrong type

	_, ok := GetAuthUserID(c)
	if ok {
		t.Error("GetAuthUserID() should return false for wrong type")
	}
}

func TestGetAuthUsernameNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	name := GetAuthUsername(c)
	if name != "" {
		t.Errorf("GetAuthUsername() = %q, want empty string", name)
	}
}

func TestGetAuthUsernameSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("username", "admin")

	name := GetAuthUsername(c)
	if name != "admin" {
		t.Errorf("GetAuthUsername() = %q, want %q", name, "admin")
	}
}

// ========= InternalAPIMiddleware Tests =========

func TestInternalAPIMiddlewareWithValidToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "internal-secret"

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/internal/test", InternalAPIMiddleware(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	req.Header.Set("X-Internal-Token", "internal-secret")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("InternalAPIMiddleware() with valid token: status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestInternalAPIMiddlewareWithInvalidToken(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "internal-secret"

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/internal/test", InternalAPIMiddleware(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	req.Header.Set("X-Internal-Token", "wrong-secret")
	// Remote address must be a public IP to be denied
	req.RemoteAddr = "203.0.113.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("InternalAPIMiddleware() with invalid token: status = %v, want %v", w.Code, http.StatusForbidden)
	}
}

func TestInternalAPIMiddlewareFromPrivateIP(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.SecretKey = "internal-secret"

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/internal/test", InternalAPIMiddleware(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	// No token, but from loopback
	req.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("InternalAPIMiddleware() from loopback: status = %v, want %v", w.Code, http.StatusOK)
	}
}

// ========= isPrivateIP Tests =========

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// Loopback
		{"127.0.0.1", true},
		{"127.0.0.100", true},
		// 10.x.x.x
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		// 192.168.x.x
		{"192.168.1.1", true},
		{"192.168.0.100", true},
		// 172.16-31.x.x
		{"172.16.0.1", true},
		{"172.20.10.1", true},
		{"172.31.255.255", true},
		// Non-private 172.x
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		// IPv6 loopback
		{"::1", true},
		// IPv6 private
		{"fc00::1", true},
		{"fe80::1", true},
		// Public IPs
		{"8.8.8.8", false},
		{"203.0.113.1", false},
		{"1.1.1.1", false},
		// Edge cases
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := isPrivateIP(tt.ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
