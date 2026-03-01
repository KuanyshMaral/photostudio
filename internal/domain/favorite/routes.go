package favorite

import "github.com/go-chi/chi/v5"

// RegisterRoutes регистрирует маршруты избранного в chi-роутере.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/favorites", func(r chi.Router) {
		r.Get("/", h.GetFavorites)
		r.Post("/", h.AddFavorite)
		r.Delete("/{type}/{id}", h.RemoveFavorite)
		r.Get("/{type}/{id}/check", h.CheckFavorite)
	})
}
