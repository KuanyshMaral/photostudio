package booking

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/bookings", h.CreateBooking)

	// Task 3.1
	rg.GET("/rooms/:id/availability", h.GetRoomAvailability)

	rg.GET("/rooms/:id/busy-slots", h.GetBusySlots)

	// Task 3.2 (requires auth middleware that sets user_id in context)
	rg.GET("/users/me/bookings", h.GetMyBookings)

	// Task 3.3 (requires auth middleware that sets user_id and role in context)
	rg.PATCH("/bookings/:id/status", h.UpdateBookingStatus)

	// Task 3.1: Booking Status Workflow endpoints
	rg.PATCH("/bookings/:id/confirm", h.ConfirmBooking)
	rg.PATCH("/bookings/:id/cancel", h.CancelBooking)
	rg.PATCH("/bookings/:id/complete", h.CompleteBooking)
	rg.PATCH("/bookings/:id/mark-paid", h.MarkBookingPaid)

	// Block 10: Deposit management (только для менеджеров)
	rg.PATCH("/bookings/:id/deposit", h.UpdateDeposit)
}

