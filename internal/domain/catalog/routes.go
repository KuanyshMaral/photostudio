package catalog

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// OwnershipMiddleware for chi.
type OwnershipMiddleware interface {
	CheckStudioOwnership() func(http.Handler) http.Handler
	CheckRoomOwnership() func(http.Handler) http.Handler
}

// ---------------- Public Routes ----------------

// RegisterPublicStudioRoutes mounts GET /studios
func (h *Handler) RegisterPublicStudioRoutes(r chi.Router) {
	r.Get("/", h.GetStudios)
	r.Get("/{id}", h.GetStudioByID)
	r.Get("/{id}/working-hours", h.GetStudioWorkingHours)
	r.Get("/{id}/working-hours/v2", h.GetStudioWorkingHoursV2)
}

// RegisterPublicRoomRoutes mounts GET /rooms
func (h *Handler) RegisterPublicRoomRoutes(r chi.Router) {
	r.Get("/", h.GetRooms)
	r.Get("/{id}", h.GetRoomByID)
}

// RegisterPublicRoomTypes mounts GET /room-types
func (h *Handler) RegisterPublicRoomTypes(r chi.Router) {
	r.Get("/", h.GetRoomTypes)
}

// ---------------- Protected Routes ----------------

// RegisterProtectedStudioRoutes mounts POST/PUT /studios
func (h *Handler) RegisterProtectedStudioRoutes(r chi.Router, ownershipChecker OwnershipMiddleware) {
	r.Post("/", h.CreateStudio)
	r.Get("/my", h.GetMyStudios)

	r.Group(func(r chi.Router) {
		r.Use(ownershipChecker.CheckStudioOwnership())
		r.Put("/{id}", h.UpdateStudio)
		r.Put("/{id}/working-hours", h.UpdateStudioWorkingHours)
		r.Post("/{id}/rooms", h.CreateRoom)
		r.Post("/{id}/photos", h.UploadStudioPhotos)
	})
}

// RegisterProtectedRoomRoutes mounts PUT/DELETE /rooms
func (h *Handler) RegisterProtectedRoomRoutes(r chi.Router, ownershipChecker OwnershipMiddleware) {
	r.Group(func(r chi.Router) {
		r.Use(ownershipChecker.CheckRoomOwnership())
		r.Put("/{id}", h.UpdateRoom)
		r.Delete("/{id}", h.DeleteRoom)
		r.Post("/{id}/equipment", h.AddEquipment)
	})
}
