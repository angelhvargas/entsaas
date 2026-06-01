package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize limits the size of incoming request bodies to prevent
// memory exhaustion attacks. This is critical because Gin's
// MaxMultipartMemory only applies to multipart forms, NOT to
// ShouldBindJSON — an attacker could POST a multi-GB JSON body
// without this middleware.
//
// Default limit: 1 MiB. Override with the maxBytes parameter.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
