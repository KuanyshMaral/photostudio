package middleware

import (
	"net/http"
	"strconv"

	"photostudio/internal/repository"

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
