package admin

import (
	"github.com/go-chi/chi/v5"

	jwtsvc "photostudio/internal/pkg/jwt"
)

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Post("/auth/login", h.authHandler.Login)
	})
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router, jwtService *jwtsvc.Service) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(ChiAdminJWTAuth(jwtService))

		// studios moderation
		r.Get("/studios/pending", h.GetPendingStudios)
		r.Post("/studios/{id}/approve", h.ApproveStudio)
		r.Post("/studios/{id}/reject", h.RejectStudio)
		r.Post("/studios/{id}/verify", h.ApproveStudio) // alias
		r.Patch("/studios/{id}/vip", h.SetStudioVIP)
		r.Patch("/studios/{id}/gold", h.SetStudioGold)
		r.Patch("/studios/{id}/promo", h.SetStudioPromo)

		// Auth
		r.Get("/auth/me", h.authHandler.GetMe)

		// Admin Management
		r.Get("/admins", h.managementHandler.ListAdmins)
		r.Post("/admins", h.managementHandler.CreateAdmin)
		r.Patch("/admins/{id}", h.managementHandler.UpdateAdmin)

		// statistics
		r.Get("/stats", h.GetStats)
		r.Get("/statistics", h.GetStats) // alias
		r.Get("/analytics", h.GetPlatformAnalytics)

		// users moderation
		r.Get("/users", h.GetUsers)
		r.Patch("/users/{id}/ban", h.BanUser)
		r.Post("/users/{id}/ban", h.BanUser) // alias
		r.Patch("/users/{id}/unban", h.UnbanUser)
		r.Post("/users/{id}/unban", h.UnbanUser) // alias
		r.Post("/users/{id}/block", h.BanUser)
		r.Post("/users/{id}/unblock", h.UnbanUser)

		// reviews moderation
		r.Get("/reviews", h.GetReviews)
		r.Post("/reviews/{id}/hide", h.HideReview)
		r.Patch("/reviews/{id}/hide", h.HideReview)
		r.Post("/reviews/{id}/show", h.ShowReview)
		r.Delete("/reviews/{id}", h.DeleteReview)

		// ads
		r.Get("/ads", h.GetAds)
		r.Post("/ads", h.CreateAd)
		r.Patch("/ads/{id}", h.UpdateAd)
		r.Delete("/ads/{id}", h.DeleteAd)
	})
}
