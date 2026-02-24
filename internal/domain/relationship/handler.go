package relationship

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type blockRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// Block godoc
// @Summary Block a user
// @Tags Relationships
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body blockRequest true "User to block"
// @Success 200 {object} map[string]interface{}
// @Router /relationships/block [post]
func (h *Handler) Block(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	var req blockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if err := h.service.Block(c.Request.Context(), userID, req.UserID); err != nil {
		switch err {
		case ErrCannotBlockSelf:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		case ErrAlreadyBlocked:
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to block user"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user blocked"})
}

// Unblock godoc
// @Summary Unblock a user
// @Tags Relationships
// @Security BearerAuth
// @Param user_id path int true "User ID to unblock"
// @Success 200 {object} map[string]interface{}
// @Router /relationships/block/{user_id} [delete]
func (h *Handler) Unblock(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	targetID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid user_id"})
		return
	}
	if err := h.service.Unblock(c.Request.Context(), userID, targetID); err != nil {
		if err == ErrNotBlocked {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to unblock user"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user unblocked"})
}

// ListBlocked godoc
// @Summary List blocked users
// @Tags Relationships
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /relationships/blocked [get]
func (h *Handler) ListBlocked(c *gin.Context) {
	userID := mustUserID(c)
	if userID == 0 {
		return
	}
	blocked, err := h.service.ListBlocked(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to list blocked users"})
		return
	}
	items := make([]gin.H, 0, len(blocked))
	for _, b := range blocked {
		items = append(items, gin.H{"user_id": b.BlockedID, "blocked_at": b.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func mustUserID(c *gin.Context) int64 {
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return 0
	}
	switch v := id.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user id"})
	return 0
}
