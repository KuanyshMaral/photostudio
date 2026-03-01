package upload

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers upload routes under the protected group.
// All routes require authentication (any role — client, owner, admin).
func RegisterRoutes(r chi.Router, h *Handler) {

	r.Post("/", h.Upload)
	r.Get("/", h.ListMy)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)

}
