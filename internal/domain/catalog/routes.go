package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// OwnershipMiddleware for chi — provides chi-native middleware for studio/room ownership checks.
type OwnershipMiddleware interface {
	CheckStudioOwnership() func(http.Handler) http.Handler
	CheckRoomOwnership() func(http.Handler) http.Handler
}

// RegisterRoutes registers public catalog routes (no auth required).
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public studio routes
	r.Route("/studios", func(r chi.Router) {
		r.Get("/", h.GetStudios)
		r.Get("/{id}", h.GetStudioByID)
		r.Get("/{id}/working-hours", h.GetStudioWorkingHours)
		r.Get("/{id}/working-hours/v2", h.GetStudioWorkingHoursV2)
	})

	r.Get("/room-types", h.GetRoomTypes)
	r.Get("/rooms", h.GetRooms)
	r.Get("/rooms/{id}", h.GetRoomByID)
}

// RegisterProtectedRoutes registers authenticated catalog routes (JWT required).
func (h *Handler) RegisterProtectedRoutes(r chi.Router, ownershipChecker OwnershipMiddleware) {
	// Studio management (Owner only)
	r.Route("/studios", func(r chi.Router) {
		r.Post("/", h.CreateStudio)

		r.Group(func(r chi.Router) {
			r.Use(ownershipChecker.CheckStudioOwnership())
			r.Put("/{id}", h.UpdateStudio)
			r.Put("/{id}/working-hours", h.UpdateStudioWorkingHours)
			r.Post("/{id}/rooms", h.CreateRoom)
			r.Post("/{id}/photos", h.UploadStudioPhotos)
		})
	})

	// Direct room management
	r.Group(func(r chi.Router) {
		r.Use(ownershipChecker.CheckRoomOwnership())
		r.Put("/rooms/{id}", h.UpdateRoom)
		r.Delete("/rooms/{id}", h.DeleteRoom)
		r.Post("/rooms/{id}/equipment", h.AddEquipment)
	})

	// User's studios
	r.Get("/studios/my", h.GetMyStudios)
}
