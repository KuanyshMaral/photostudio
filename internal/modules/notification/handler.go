package notification

import (
	"errors"
	"net/http"
	"strconv"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/notifications")
	{
		g.GET("", h.GetNotifications)
		g.PATCH("/:id/read", h.MarkAsRead)
		g.PATCH("/read-all", h.MarkAllAsRead)
	}
}

func (h *Handler) GetNotifications(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	limit := 20
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
			if limit > 100 {
				limit = 100
			}
		}
	}

	list, unread, err := h.service.GetUserNotifications(c.Request.Context(), userID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"notifications": list,
		"unread_count":  unread,
	})
}

func (h *Handler) MarkAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Notification not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "read"})
}

func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to mark as read")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "all_read"})
}
