package review

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(public, protected *gin.RouterGroup) {
	// Public routes (no auth required)
	if public != nil {
		public.GET("/studios/:id/reviews", h.GetByStudio)
	}

	// Protected routes (auth required)
	if protected != nil {
		protected.POST("/reviews", h.Create)
		protected.POST("/reviews/:id/response", h.AddOwnerResponse)
	}
}

