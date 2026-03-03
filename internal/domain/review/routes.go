package review

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers review routes.
// public can be nil if only protected routes are needed, and vice versa.
func (h *Handler) RegisterRoutes(public, protected chi.Router) {
	// Public routes (no auth required)
	if public != nil {
		public.Get("/reviews", h.GetByTarget)
		// Legacy alias for graceful compatibility with old frontend
		public.Get("/studios/{id}/reviews", h.GetByStudioLegacy)
	}

	// Protected routes (auth required)
	if protected != nil {
		protected.Post("/reviews", h.Create)
		protected.Post("/reviews/{id}/response", h.AddOwnerResponse)
		protected.Get("/studios/{id}/can-review", h.CanReview)
	}
}
