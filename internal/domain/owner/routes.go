package owner

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {

	// PIN
	r.Post("/pin/verify", h.VerifyPIN) // POST /owner/pin/verify
	r.Post("/pin/set", h.SetPIN)
	r.Get("/pin", h.HasPIN) // GET /owner/pin

	// Procurement
	r.Get("/procurement", h.GetProcurement)
	r.Post("/procurement", h.CreateProcurement)
	r.Patch("/procurement/{id}", h.UpdateProcurement)
	r.Delete("/procurement/{id}", h.DeleteProcurement)

	// Maintenance
	r.Get("/maintenance", h.GetMaintenance)
	r.Post("/maintenance", h.CreateMaintenance)
	r.Patch("/maintenance/{id}", h.UpdateMaintenance)
	r.Delete("/maintenance/{id}", h.DeleteMaintenance)

	// Analytics
	r.Get("/analytics", h.GetAnalytics)
}

func (h *Handler) RegisterCompanyRoutes(r chi.Router) {

	r.Get("/profile", h.GetCompanyProfile)
	r.Put("/profile", h.UpdateCompanyProfile)
	r.Get("/portfolio", h.GetPortfolio)
	r.Post("/portfolio", h.AddPortfolioProject)
	r.Delete("/portfolio/{id}", h.DeletePortfolioProject)
	r.Put("/portfolio/reorder", h.ReorderPortfolio)

}
