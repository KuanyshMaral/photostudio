package attachment

import "github.com/go-chi/chi/v5"

// RegisterPublicRoutes registers the attachment endpoints that do not require authentication.
func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Get("/", h.ListByTarget)
}

// RegisterProtectedRoutes registers the attachment endpoints that require authentication.
func RegisterProtectedRoutes(r chi.Router, h *Handler) {
	r.Post("/", h.Attach)
	r.Delete("/{id}", h.Delete)
	r.Patch("/reorder", h.Reorder)
}
