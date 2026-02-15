package lead

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes registers public lead routes
func RegisterPublicRoutes(r *gin.RouterGroup, handler *Handler) {
	r.POST("/leads/submit", handler.SubmitLead)
}

// RegisterAdminRoutes registers admin lead routes
func RegisterAdminRoutes(r *gin.RouterGroup, handler *Handler) {
	leads := r.Group("/leads")
	{
		leads.GET("", handler.ListLeads)
		leads.GET("/stats", handler.GetStats)
		leads.GET("/:id", handler.GetLead)
		leads.PATCH("/:id/status", handler.UpdateStatus)
		leads.PATCH("/:id/assign", handler.AssignLead)
		leads.POST("/:id/reject", handler.RejectLead)
		leads.POST("/:id/contacted", handler.MarkContacted)
		leads.POST("/:id/convert", handler.ConvertLead)
	}
}
