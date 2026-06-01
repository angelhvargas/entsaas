package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders injects security response headers on every request.
// Covers OWASP recommended headers: CSP, HSTS, X-Frame-Options, etc.
func SecurityHeaders() gin.HandlerFunc {
	cspReportURI := os.Getenv("CSP_REPORT_URI")

	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		// SEC-12: X-XSS-Protection intentionally omitted — it is deprecated
		// and can introduce XSS in legacy IE. CSP provides proper XSS mitigation.
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// Permissions policy
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS (only in production — don't break local dev with HTTP)
		if os.Getenv("ENTSAAS_DEV") != "1" {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		// Content Security Policy
		csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'"
		if cspReportURI != "" {
			csp += "; report-uri " + cspReportURI
		}
		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}
