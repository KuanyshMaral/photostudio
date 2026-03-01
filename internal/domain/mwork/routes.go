package mwork

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/mwork", func(r chi.Router) {
		r.Post("/users/sync", h.SyncUser)
	})
}
