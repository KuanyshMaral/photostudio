package upload

import "github.com/gin-gonic/gin"

// RegisterRoutes registers upload routes under the protected group.
// All routes require authentication (any role — client, owner, admin).
func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	uploads := r.Group("/uploads")
	{
		uploads.POST("", h.Upload)
		uploads.GET("", h.ListMy)
		uploads.GET("/:id", h.GetByID)
		uploads.DELETE("/:id", h.Delete)
	}
}
