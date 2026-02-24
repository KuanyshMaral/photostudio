package subscription

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes registers routes that don't require authentication
// (e.g., listing available plans for the pricing page)
func RegisterPublicRoutes(r *gin.RouterGroup, h *Handler) {
	r.GET("/subscriptions/plans", h.GetPlans)
}

// RegisterOwnerRoutes registers routes that require role='owner'.
// Clients CANNOT access these routes.
func RegisterOwnerRoutes(r *gin.RouterGroup, h *Handler) {
	sub := r.Group("/owner/subscription")
	{
		sub.GET("", h.GetMySubscription)
		sub.POST("", h.Subscribe)
		sub.POST("/cancel", h.Cancel)
		sub.GET("/usage", h.GetUsage)
	}
}
