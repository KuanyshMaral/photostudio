package manager

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/manager", func(r chi.Router) {
		r.Get("/bookings", h.GetBookings)
		r.Get("/bookings/{id}", h.GetBooking)
		r.Patch("/bookings/{id}/deposit", h.UpdateDeposit)
		r.Patch("/bookings/{id}/status", h.UpdateBookingStatus)
		r.Get("/clients", h.GetClients)
	})
}
