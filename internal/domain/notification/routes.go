package notification

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/notifications")
	{
		g.GET("", h.GetNotifications)
		g.PATCH("/:id/read", h.MarkAsRead)
		g.PATCH("/read-all", h.MarkAllAsRead)
	}
}

