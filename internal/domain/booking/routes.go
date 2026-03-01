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
	r.Post("/", h.CreateBooking)

	// Availability endpoints (public info but under auth group)
	r.Get("/room/{id}/availability", h.GetRoomAvailability)
	r.Get("/room/{id}/busy-slots", h.GetBusySlots)

	// User booking history
	r.Get("/my", h.GetMyBookings)

	// Booking lifecycle management
	r.Patch("/{id}/status", h.UpdateBookingStatus)
	r.Post("/{id}/confirm", h.ConfirmBooking)
	r.Post("/{id}/cancel", h.CancelBooking)
	r.Post("/{id}/complete", h.CompleteBooking)
	r.Post("/{id}/pay", h.MarkBookingPaid)
	r.Put("/{id}/payment-status", h.UpdatePaymentStatus) // Replaced UpdateBookingStatus if needed

	// Deposit management
	r.Patch("/{id}/deposit", h.UpdateDeposit)
}

// RegisterStudioRoutes registers owner-specific studio booking routes.
func (h *Handler) RegisterStudioRoutes(r chi.Router, ownershipChecker OwnershipMiddleware) {
	r.Use(ownershipChecker.CheckStudioOwnership())
	r.Get("/studio/{id}", h.GetStudioBookings)
}
