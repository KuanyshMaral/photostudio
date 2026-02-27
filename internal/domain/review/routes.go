package review

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(public, protected *gin.RouterGroup) {
	// Public routes (no auth required)
	if public != nil {
		public.GET("/reviews", h.GetByTarget)
		// Legacy alias for graceful compatibility
		public.GET("/studios/:id/reviews", h.GetByStudioLegacy)
	}

	// Protected routes (auth required)
	if protected != nil {
		protected.POST("/reviews", h.Create)
		protected.POST("/reviews/:id/response", h.AddOwnerResponse)
	}
}
