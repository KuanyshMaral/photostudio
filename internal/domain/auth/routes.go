package auth

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register/client", h.RegisterClient)
		r.Post("/login", h.Login)
		r.Post("/verify/request", h.RequestEmailVerification)
		r.Post("/verify/confirm", h.ConfirmEmailVerification)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/me", h.GetMe)
		r.Put("/me", h.UpdateProfile)
		r.Post("/verification/documents", h.UploadVerificationDocuments)
	})
}
