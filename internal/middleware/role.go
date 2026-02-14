package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"photostudio/internal/pkg/response"
)

// RequireRole ensures that the authenticated user has the specified role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Role not found in token")
			c.Abort()
			return
		}

		if role.(string) != requiredRole {
			response.CustomError(c, http.StatusForbidden, "FORBIDDEN", "Access denied: insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}