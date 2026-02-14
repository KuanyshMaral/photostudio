package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"os"
	"photostudio/internal/domain/auth"
	"photostudio/internal/pkg/response"
	"strings"
)

// MWorkUserAuth is a middleware that authenticates MWork internal requests
// and maps MWork User UUID to PhotoStudio internal user ID.
//
// Flow:
// 1. Validates Authorization: Bearer <MWORK_SYNC_TOKEN>
// 2. Extracts X-MWork-User-ID header (UUID)
// 3. Looks up PhotoStudio user by mwork_user_id
// 4. Sets user_id (int64) and role in Gin context
//
// Usage:
//   protected.Use(middleware.MWorkUserAuth(userRepo))
//   protected.POST("/bookings", bookingHandler.CreateBooking)
func MWorkUserAuth(userRepo *auth.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Validate internal token
		if !validateInternalToken(c) {
			response.CustomError(c, http.StatusUnauthorized, "AUTH_INVALID", "Invalid or missing internal token")
			c.Abort()
			return
		}

		// Step 2: Extract MWork User ID from header
		mworkUserID := c.GetHeader("X-MWork-User-ID")
		if mworkUserID == "" {
			response.CustomError(c, http.StatusBadRequest, "MWORK_USER_ID_MISSING", "X-MWork-User-ID header is required")
			c.Abort()
			return
		}

		// Step 3: Validate UUID format
		if _, err := uuid.Parse(mworkUserID); err != nil {
			response.CustomError(c, http.StatusBadRequest, "INVALID_USER_ID", "X-MWork-User-ID must be a valid UUID")
			c.Abort()
			return
		}

		// Step 4: Look up PhotoStudio user by mwork_user_id
		user, err := userRepo.GetByMworkUserID(c.Request.Context(), mworkUserID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// User not found - sync failed or never happened
				response.CustomError(c, http.StatusUnauthorized, "USER_NOT_SYNCED",
					"User not found in PhotoStudio. Please contact support if this persists.")
				c.Abort()
				return
			}

			// Database error
			response.CustomError(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to lookup user")
			c.Abort()
			return
		}

		// Step 5: Set PhotoStudio internal user ID and role in context
		c.Set("user_id", user.ID)     // int64 - PhotoStudio internal ID
		c.Set("role", string(user.Role)) // client, studio_owner, admin

		// Optional: Set MWork context for logging/auditing
		c.Set("mwork_user_id", mworkUserID)
		c.Set("mwork_role", user.MworkRole)

		c.Next()
	}
}

// validateInternalToken validates the Bearer token from Authorization header
func validateInternalToken(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}

	token := parts[1]
	expectedToken := os.Getenv("MWORK_SYNC_TOKEN")
	if expectedToken == "" {
		expectedToken = os.Getenv("PHOTO_STUDIO_INTERNAL_TOKEN")
	}

	return token != "" && token == expectedToken
}