package owner

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	owner := rg.Group("/owner")
	{
		// PIN
		owner.POST("/set-pin", h.SetPIN)
		owner.POST("/verify-pin", h.VerifyPIN)
		owner.GET("/has-pin", h.HasPIN)

		// Procurement
		owner.GET("/procurement", h.GetProcurement)
		owner.POST("/procurement", h.CreateProcurement)
		owner.PATCH("/procurement/:id", h.UpdateProcurement)
		owner.DELETE("/procurement/:id", h.DeleteProcurement)

		// Maintenance
		owner.GET("/maintenance", h.GetMaintenance)
		owner.POST("/maintenance", h.CreateMaintenance)
		owner.PATCH("/maintenance/:id", h.UpdateMaintenance)
		owner.DELETE("/maintenance/:id", h.DeleteMaintenance)

		// Analytics
		owner.GET("/analytics", h.GetAnalytics)
	}
}

func (h *Handler) RegisterCompanyRoutes(rg *gin.RouterGroup) {
	company := rg.Group("/company")
	{
		company.GET("/profile", h.GetCompanyProfile)
		company.PUT("/profile", h.UpdateCompanyProfile)
		company.GET("/portfolio", h.GetPortfolio)
		company.POST("/portfolio", h.AddPortfolioProject)
		company.DELETE("/portfolio/:id", h.DeletePortfolioProject)
		company.PUT("/portfolio/reorder", h.ReorderPortfolio)
	}
}

