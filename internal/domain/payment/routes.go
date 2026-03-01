package payment

import "github.com/go-chi/chi/v5"

// RegisterPublicWebhookRoutes registers public Robokassa webhook routes directly on the root router.
func (h *Handler) RegisterPublicWebhookRoutes(r chi.Router) {
	r.Route("/webhooks/robokassa", func(r chi.Router) {
		r.Post("/result", h.ResultCallback)
		r.Get("/success", h.SuccessCallback)
		r.Post("/success", h.SuccessCallback)
		r.Get("/fail", h.FailCallback)
		r.Post("/fail", h.FailCallback)
	})
}

// RegisterProtectedRoutes registers authenticated payment routes.
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Route("/payments/robokassa", func(r chi.Router) {
		r.Post("/create", h.CreatePayment)
		r.Post("/init", h.InitPayment)
	})
	r.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", h.CreateSubscription)
		r.Get("/me", h.MySubscription)
		r.Post("/cancel", h.CancelSubscription)
	})
}
