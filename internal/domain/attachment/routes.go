package attachment

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the attachment domain routes.
// All routes require authentication (handled by caller middleware).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("", h.Attach)
	rg.GET("", h.ListByTarget)
	rg.DELETE("/:id", h.Delete)
	rg.PATCH("/reorder", h.Reorder)
}
