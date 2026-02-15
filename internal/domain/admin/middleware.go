package admin

import (
	"net/http"
	"photostudio/internal/pkg/jwt"
	"photostudio/internal/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminJWTAuth(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.CustomError(c, http.StatusUnauthorized, "AUTH_HEADER_MISSING", "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.CustomError(c, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Authorization header must be 'Bearer <token>'")
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(parts[1])
		if err != nil {
			response.CustomError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
			c.Abort()
			return
		}

		// Ensure it's an admin token
		if claims.AdminID == "" && claims.Role != "admin" && claims.Role != "super_admin" {
			response.CustomError(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		if claims.UserID > 0 {
			c.Set("user_id", claims.UserID)
		}
		c.Set("role", claims.Role)
		c.Next()
	}
}
