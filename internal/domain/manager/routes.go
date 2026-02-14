package manager

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mgr := rg.Group("/manager")
	{
		mgr.GET("/bookings", h.GetBookings)
		mgr.GET("/bookings/:id", h.GetBooking)
		mgr.PATCH("/bookings/:id/deposit", h.UpdateDeposit)
		mgr.PATCH("/bookings/:id/status", h.UpdateBookingStatus)
		mgr.GET("/clients", h.GetClients)
	}
}

