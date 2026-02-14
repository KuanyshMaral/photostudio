package admin

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	// studios moderation
	admin.GET("/studios/pending", h.GetPendingStudios)
	admin.POST("/studios/:id/approve", h.ApproveStudio)
	admin.POST("/studios/:id/reject", h.RejectStudio)

	// statistics
	admin.GET("/stats", h.GetStats)

	// users moderation
	admin.GET("/users", h.GetUsers)
	admin.PATCH("/users/:id/ban", h.BanUser)
	admin.PATCH("/users/:id/unban", h.UnbanUser)

	// reviews moderation
	admin.GET("/reviews", h.GetReviews)
	admin.POST("/reviews/:id/hide", h.HideReview)
	admin.POST("/reviews/:id/show", h.ShowReview)

	// Aliases для обратной совместимости
	admin.POST("/studios/:id/verify", h.ApproveStudio)
	admin.GET("/statistics", h.GetStats)
	admin.POST("/users/:id/block", h.BanUser)
	admin.POST("/users/:id/unblock", h.UnbanUser)

	// analytics
	admin.GET("/analytics", h.GetPlatformAnalytics)

	// vip/gold/promo
	admin.PATCH("/studios/:id/vip", h.SetStudioVIP)
	admin.PATCH("/studios/:id/gold", h.SetStudioGold)
	admin.PATCH("/studios/:id/promo", h.SetStudioPromo)

	// ads
	admin.GET("/ads", h.GetAds)
	admin.POST("/ads", h.CreateAd)
	admin.PATCH("/ads/:id", h.UpdateAd)
	admin.DELETE("/ads/:id", h.DeleteAd)

	// reviews new style (keep old POST routes too)
	admin.PATCH("/reviews/:id/hide", h.HideReview)
	admin.DELETE("/reviews/:id", h.DeleteReview)

}

