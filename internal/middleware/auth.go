package middleware

import (
	"net/http"
	"photostudio/internal/pkg/jwt"

	"photostudio/internal/pkg/response"
	"photostudio/internal/repository"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// OwnershipChecker provides middleware to verify resource ownership
type OwnershipChecker struct {
	studioRepo *repository.StudioRepository
	roomRepo   *repository.RoomRepository
}

// NewOwnershipChecker creates a new ownership checker
func NewOwnershipChecker(
	studioRepo *repository.StudioRepository,
	roomRepo *repository.RoomRepository,
) *OwnershipChecker {
	return &OwnershipChecker{
		studioRepo: studioRepo,
		roomRepo:   roomRepo,
	}
}

// CheckStudioOwnership verifies the user owns the studio
// Expects studio ID in URL param "id"
func (oc *OwnershipChecker) CheckStudioOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			return
		}

		studioIDStr := c.Param("id")
		studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   gin.H{"code": "INVALID_ID", "message": "Invalid studio ID"},
			})
			return
		}

		studio, err := oc.studioRepo.GetByID(c.Request.Context(), studioID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "NOT_FOUND", "message": "Studio not found"},
			})
			return
		}

		if studio.OwnerID != userID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "You don't own this studio"},
			})
			return
		}

		c.Next()
	}
}

// CheckRoomOwnership verifies the user owns the studio that owns the room
// Expects room ID in URL param "id"
func (oc *OwnershipChecker) CheckRoomOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			return
		}

		roomIDStr := c.Param("id")
		roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   gin.H{"code": "INVALID_ID", "message": "Invalid room ID"},
			})
			return
		}

		room, err := oc.roomRepo.GetByID(c.Request.Context(), roomID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "NOT_FOUND", "message": "Room not found"},
			})
			return
		}

		studio, err := oc.studioRepo.GetByID(c.Request.Context(), room.StudioID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "NOT_FOUND", "message": "Studio not found"},
			})
			return
		}

		if studio.OwnerID != userID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "You don't own this resource"},
			})
			return
		}

		c.Next()
	}
}

// JWTAuth requires a valid JWT Bearer token and puts user_id (int64) into context
func JWTAuth(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "AUTH_HEADER_MISSING", "Authorization header is required")
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, http.StatusUnauthorized, "INVALID_AUTH_FORMAT", "Authorization header must be 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := jwtService.ValidateToken(tokenString)

		if err != nil {
			response.Error(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
			c.Abort()
			return
		}

		// Everything is OK → store user_id in context for downstream handlers
		// We store it as int64 because Gin’s c.GetInt64 is convenient and safe
		c.Set("user_id", claims.UserID)
		// Optional: you can also store role if you need it later
		c.Set("role", claims.Role)

		c.Next()
	}
}
