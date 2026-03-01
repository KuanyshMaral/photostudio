package subscription

import "github.com/go-chi/chi/v5"

// RegisterPublicRoutes registers routes that don't require authentication
// (e.g., listing available plans for the pricing page)
func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Get("/subscriptions/plans", h.GetPlans)
}

// RegisterOwnerRoutes registers routes that require role='owner'.
// Clients CANNOT access these routes.
func RegisterOwnerRoutes(r chi.Router, h *Handler) {
	r.Route("/owner/subscription", func(r chi.Router) {
		r.Get("/", h.GetMySubscription)
		r.Post("/", h.Subscribe)
		r.Post("/cancel", h.Cancel)
		r.Get("/usage", h.GetUsage)
	})
}
