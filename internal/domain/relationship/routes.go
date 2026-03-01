package relationship

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *Handler) {

	r.Post("/block", h.Block)
	r.Delete("/block/{user_id}", h.Unblock)
	r.Get("/blocked", h.ListBlocked)

}
