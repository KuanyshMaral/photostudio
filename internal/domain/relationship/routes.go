package relationship

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	rel := r.Group("/relationships")
	{
		rel.POST("/block", h.Block)
		rel.DELETE("/block/:user_id", h.Unblock)
		rel.GET("/blocked", h.ListBlocked)
	}
}
