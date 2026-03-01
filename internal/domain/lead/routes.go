package lead

import "github.com/go-chi/chi/v5"

// RegisterPublicRoutes registers public lead routes
func RegisterPublicRoutes(r chi.Router, handler *Handler) {
	r.Post("/submit", handler.SubmitLead)
}

// RegisterAdminRoutes registers admin lead routes
func RegisterAdminRoutes(r chi.Router, handler *Handler) {

	r.Get("/", handler.ListLeads)
	r.Get("/stats", handler.GetStats)
	r.Get("/{id}", handler.GetLead)
	r.Patch("/{id}/status", handler.UpdateStatus)
	r.Patch("/{id}/assign", handler.AssignLead)
	r.Post("/{id}/reject", handler.RejectLead)
	r.Post("/{id}/contacted", handler.MarkContacted)
	r.Post("/{id}/convert", handler.ConvertLead)

}
