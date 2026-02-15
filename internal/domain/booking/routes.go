package booking

import (
	"github.com/gin-gonic/gin"
)

// OwnershipMiddleware для проверки прав владения студией
type OwnershipMiddleware interface {
	CheckStudioOwnership() gin.HandlerFunc
}

// RegisterRoutes регистрирует все маршруты для бронирований
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/bookings", h.CreateBooking)

	// Availability endpoints
	rg.GET("/rooms/:id/availability", h.GetRoomAvailability)
	rg.GET("/rooms/:id/busy-slots", h.GetBusySlots)

	// User booking history
	rg.GET("/users/me/bookings", h.GetMyBookings)

	// Booking lifecycle management
	rg.PATCH("/bookings/:id/status", h.UpdateBookingStatus)
	rg.PATCH("/bookings/:id/confirm", h.ConfirmBooking)
	rg.PATCH("/bookings/:id/cancel", h.CancelBooking)
	rg.PATCH("/bookings/:id/complete", h.CompleteBooking)
	rg.PATCH("/bookings/:id/mark-paid", h.MarkBookingPaid)

	// Deposit management
	rg.PATCH("/bookings/:id/deposit", h.UpdateDeposit)
}

// RegisterStudioRoutes регистрирует маршруты для владельцев студий
func (h *Handler) RegisterStudioRoutes(r *gin.RouterGroup, ownershipChecker OwnershipMiddleware) {
	// Owner-specific routes для управления бронированиями студий
	studios := r.Group("/studios")
	{
		studios.GET("/:id/bookings", ownershipChecker.CheckStudioOwnership(), h.GetStudioBookings)
	}
}
