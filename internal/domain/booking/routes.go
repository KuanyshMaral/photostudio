package booking

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// OwnershipMiddleware for chi — wraps the studio ownership check.
type OwnershipMiddleware interface {
	CheckStudioOwnership() func(http.Handler) http.Handler
}

// RegisterRoutes registers all booking routes on a chi.Router that already has JWT auth applied.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/bookings", h.CreateBooking)

	// Availability endpoints (public info but under auth group)
	r.Get("/rooms/{id}/availability", h.GetRoomAvailability)
	r.Get("/rooms/{id}/busy-slots", h.GetBusySlots)

	// User booking history
	r.Get("/users/me/bookings", h.GetMyBookings)

	// Booking lifecycle management
	r.Patch("/bookings/{id}/status", h.UpdateBookingStatus)
	r.Patch("/bookings/{id}/confirm", h.ConfirmBooking)
	r.Patch("/bookings/{id}/cancel", h.CancelBooking)
	r.Patch("/bookings/{id}/complete", h.CompleteBooking)
	r.Patch("/bookings/{id}/mark-paid", h.MarkBookingPaid)

	// Deposit management
	r.Patch("/bookings/{id}/deposit", h.UpdateDeposit)
}

// RegisterStudioRoutes registers owner-specific studio booking routes.
func (h *Handler) RegisterStudioRoutes(r chi.Router, ownershipChecker OwnershipMiddleware) {
	r.Route("/studios/{id}/bookings", func(r chi.Router) {
		r.Use(ownershipChecker.CheckStudioOwnership())
		r.Get("/", h.GetStudioBookings)
	})
}
