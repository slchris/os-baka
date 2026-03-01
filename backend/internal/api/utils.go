package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/os-baka/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// ========= Utils =========

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID uint, username string, cfg *config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"name": username,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(cfg.Server.SecretKey))
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header required"})
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid authorization format, expected 'Bearer <token>'"})
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.Server.SecretKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token claims"})
			return
		}

		// Extract user ID (sub) — stored as float64 in MapClaims
		var userID uint
		switch sub := claims["sub"].(type) {
		case float64:
			userID = uint(sub)
		case json.Number:
			n, _ := sub.Int64()
			userID = uint(n)
		default:
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token: missing user ID"})
			return
		}

		username, _ := claims["name"].(string)

		// Store in context for downstream handlers
		c.Set("userID", userID)
		c.Set("username", username)

		c.Next()
	}
}

// GetAuthUserID extracts the authenticated user's ID from the gin context.
func GetAuthUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}

// GetAuthUsername extracts the authenticated user's username from the gin context.
func GetAuthUsername(c *gin.Context) string {
	val, _ := c.Get("username")
	s, _ := val.(string)
	return s
}

// ========= Error Response =========

func ErrorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{"error": message})
}

// ParseIDParam extracts and validates a numeric ID from a URL param.
// Returns the parsed ID and true, or sends a 400 error and returns 0, false.
func ParseIDParam(c *gin.Context, paramName string) (int, bool) {
	id, err := strconv.Atoi(c.Param(paramName))
	if err != nil || id <= 0 {
		ErrorResponse(c, http.StatusBadRequest, "Invalid "+paramName+" parameter")
		return 0, false
	}
	return id, true
}

// InternalAPIMiddleware restricts access to internal endpoints.
// Allows requests from private/loopback IPs or with a valid X-Internal-Token header.
func InternalAPIMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Option 1: Check for internal token header
		internalToken := c.GetHeader("X-Internal-Token")
		if internalToken != "" && internalToken == cfg.Server.SecretKey {
			c.Next()
			return
		}

		// Option 2: Check if request is from a private/loopback IP
		clientIP := c.ClientIP()
		if isPrivateIP(clientIP) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(403, gin.H{"error": "Access denied: internal API is restricted"})
	}
}

// isPrivateIP checks if an IP address is in a private or loopback range.
func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"127.", "10.", "192.168.", "::1", "fc00:", "fe80:",
	}
	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	// 172.16.0.0 - 172.31.255.255
	if strings.HasPrefix(ip, "172.") {
		parts := strings.SplitN(ip, ".", 3)
		if len(parts) >= 2 {
			var second int
			fmt.Sscanf(parts[1], "%d", &second)
			if second >= 16 && second <= 31 {
				return true
			}
		}
	}
	return false
}
