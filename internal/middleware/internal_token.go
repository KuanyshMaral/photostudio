package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// InternalTokenAuth protects internal endpoints using a static bearer token.
func InternalTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !syncEnabled() {
			logAuthFailure(c, http.StatusForbidden, "disabled")
			writeInternalError(c, http.StatusForbidden, "AUTH_INVALID", "MWork sync disabled", nil)
			c.Abort()
			return
		}

		if !ipAllowed(c) {
			logAuthFailure(c, http.StatusForbidden, "ip_not_allowed")
			writeInternalError(c, http.StatusForbidden, "AUTH_INVALID", "IP not allowed", nil)
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logAuthFailure(c, http.StatusUnauthorized, "missing_auth")
			writeInternalError(c, http.StatusUnauthorized, "AUTH_MISSING", "Authorization header is required", nil)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logAuthFailure(c, http.StatusUnauthorized, "invalid_auth_format")
			writeInternalError(c, http.StatusUnauthorized, "AUTH_INVALID", "Authorization header must be 'Bearer <token>'", nil)
			c.Abort()
			return
		}

		expected := os.Getenv("MWORK_SYNC_TOKEN")
		if expected == "" {
			expected = os.Getenv("PHOTO_STUDIO_INTERNAL_TOKEN")
		}
		if expected == "" {
			logAuthFailure(c, http.StatusInternalServerError, "token_not_configured")
			writeInternalError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal token is not configured", nil)
			c.Abort()
			return
		}

		if parts[1] != expected {
			logAuthFailure(c, http.StatusForbidden, "invalid_token")
			writeInternalError(c, http.StatusForbidden, "AUTH_INVALID", "Invalid internal token", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func writeInternalError(c *gin.Context, status int, code, message string, details map[string]any) {
	payload := gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
	if details != nil {
		payload["error"].(gin.H)["details"] = details
	}
	c.JSON(status, payload)
}

func syncEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("MWORK_SYNC_ENABLED")))
	if value == "" {
		return true
	}
	return value == "true" || value == "1"
}

func ipAllowed(c *gin.Context) bool {
	allowed := strings.TrimSpace(os.Getenv("MWORK_SYNC_ALLOWED_IPS"))
	if allowed == "" {
		return true
	}
	clientIP := c.ClientIP()
	for _, ip := range strings.Split(allowed, ",") {
		if strings.TrimSpace(ip) == clientIP {
			return true
		}
	}
	return false
}

func logAuthFailure(c *gin.Context, status int, reason string) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetHeader("X-Request-Id")
	}
	log.Printf("mwork_sync_auth status=%d request_id=%s reason=%s", status, requestID, reason)
}
