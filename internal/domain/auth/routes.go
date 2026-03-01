package auth

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/register/client", h.RegisterClient)
	r.Post("/login", h.Login)
	r.Post("/verify/request", h.RequestEmailVerification)
	r.Post("/verify/confirm", h.ConfirmEmailVerification)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	// Only returning empty block since other protected routes were moved
}
