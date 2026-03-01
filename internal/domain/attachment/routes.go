package attachment

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers the attachment domain routes.
// All routes require authentication (handled by caller middleware).
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Post("", h.Attach)
	r.Get("", h.ListByTarget)
	r.Delete("/{id}", h.Delete)
	r.Patch("/reorder", h.Reorder)
}
