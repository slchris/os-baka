package api

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adds standard security headers to all responses.
// These headers protect against clickjacking, MIME-type sniffing, XSS, and
// help enforce HTTPS and content security policies.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent the page from being embedded in iframes (clickjacking protection)
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME-type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable browser XSS filter
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information leakage
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features/APIs
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Content Security Policy — allow self and common inline patterns
		// Relaxed enough for the SPA frontend proxied through the backend
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"connect-src 'self' ws: wss:; "+
				"font-src 'self' data:; "+
				"frame-ancestors 'none'")

		c.Next()
	}
}
