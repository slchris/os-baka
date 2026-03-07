package api

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware generates a unique request ID for each request and
// adds it to the response header and Gin context. This enables request
// tracing across logs.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use incoming header if present (from load balancer / gateway)
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// RequestLoggerMiddleware logs each request with method, path, status,
// latency, and the request ID for correlation.
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		requestID, _ := c.Get("request_id")

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"ip", clientIP,
			"latency_ms", latency.Milliseconds(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}
		if requestID != nil {
			attrs = append(attrs, "request_id", requestID)
		}
		if status >= 500 {
			slog.Error("Request completed", attrs...)
		} else if status >= 400 {
			slog.Warn("Request completed", attrs...)
		} else {
			slog.Info("Request completed", attrs...)
		}
	}
}

// generateRequestID creates a short random hex ID (16 chars = 8 bytes).
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
