package middleware

import (
	"net/http"
	"strings"

	"entsaas/internal/store"

	"github.com/gin-gonic/gin"
)

// SessionAuth extracts and validates the JWT from the Authorization header.
// On success, it injects user claims into the Gin context.
func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing authorization header"},
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization format"},
			})
			return
		}

		claims, err := store.VerifyJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
			})
			return
		}

		// Inject claims into context for downstream handlers.
		c.Set("user_id", claims.UserID)
		c.Set("org_id", claims.OrgID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole returns middleware that ensures the authenticated user has
// one of the specified roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			return
		}
		if !roleSet[role.(string)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "Insufficient permissions"},
			})
			return
		}
		c.Next()
	}
}
