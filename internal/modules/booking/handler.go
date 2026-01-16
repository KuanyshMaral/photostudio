package booking

import (
	"errors"
	"net/http"
	"photostudio/internal/domain"
	"photostudio/internal/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/bookings", h.CreateBooking)

	// Task 3.1
	rg.GET("/rooms/:id/availability", h.GetRoomAvailability)

	// Task 3.2 (requires auth middleware that sets user_id in context)
	rg.GET("/users/me/bookings", h.GetMyBookings)

	// Task 3.3 (requires auth middleware that sets user_id and role in context)
	rg.PATCH("/bookings/:id/status", h.UpdateBookingStatus)

	// Task 3.1: Booking Status Workflow endpoints
	rg.PATCH("/bookings/:id/confirm", h.ConfirmBooking)
	rg.PATCH("/bookings/:id/cancel", h.CancelBooking)
	rg.PATCH("/bookings/:id/complete", h.CompleteBooking)
	rg.PATCH("/bookings/:id/mark-paid", h.MarkBookingPaid)
}

func (h *Handler) CreateBooking(c *gin.Context) {
	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request body",
			},
		})
		return
	}

	b, err := h.service.CreateBooking(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Invalid booking time range",
				},
			})
			return
		case errors.Is(err, ErrNotAvailable), errors.Is(err, ErrOverbooking):
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "BOOKING_CONFLICT",
					"message": "Room is not available for the selected time",
				},
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to create booking",
				},
			})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"booking": gin.H{
				"id":     b.ID,
				"status": b.Status,
			},
		},
	})
}

// GetRoomAvailability Task 3.1: GET /rooms/:id/availability?date=YYYY-MM-DD
// Now returns booked slots in addition to available slots
func (h *Handler) GetRoomAvailability(c *gin.Context) {
	roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || roomID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "VALIDATION_ERROR", "message": "Invalid room id"},
		})
		return
	}

	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "VALIDATION_ERROR", "message": "date is required (YYYY-MM-DD)"},
		})
		return
	}

	availability, err := h.service.GetAvailability(c.Request.Context(), roomID, date)
	if err != nil {
		code := "INTERNAL_ERROR"
		msg := "Failed to get availability"
		if errors.Is(err, ErrValidation) {
			code = "VALIDATION_ERROR"
			msg = "Invalid date format (YYYY-MM-DD)"
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": code, "message": msg}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": code, "message": msg}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    availability,
	})
}

// GetMyBookings Task 3.2: GET /users/me/bookings?limit=&offset=
// Requires middleware to set c.Set("user_id", int64(...))
func (h *Handler) GetMyBookings(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing auth"},
		})
		return
	}

	userID, ok := userIDAny.(int64)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid auth context"},
		})
		return
	}

	limit := 20
	offset := 0

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	items, err := h.service.GetMyBookings(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get bookings"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":  items,
			"limit":  limit,
			"offset": offset,
		},
	})
}

type UpdateBookingStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) GetStudioBookings(c *gin.Context) {
	studioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	bookings, err := h.service.GetBookingsByStudio(c.Request.Context(), studioID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get bookings")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"bookings": bookings})
}
func (h *Handler) UpdatePaymentStatus(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdatePaymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	b, err := h.service.UpdatePaymentStatus(c.Request.Context(), bookingID, userID, req.PaymentStatus)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You cannot update this booking")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update payment status")
		return
	}

	response.Success(c, http.StatusOK, b)
}

// UpdateBookingStatus Task 3.3: PATCH /bookings/:id/status
// Requires middleware to set c.Set("user_id", int64(...)) and c.Set("role", string(...))
func (h *Handler) UpdateBookingStatus(c *gin.Context) {
	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || bookingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "VALIDATION_ERROR", "message": "Invalid booking id"},
		})
		return
	}

	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing auth"},
		})
		return
	}
	userID, ok := userIDAny.(int64)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid auth context"},
		})
		return
	}

	roleAny, ok := c.Get("role")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing role"},
		})
		return
	}
	role, _ := roleAny.(string)

	var req UpdateBookingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "VALIDATION_ERROR", "message": "Invalid request body"},
		})
		return
	}

	updated, err := h.service.UpdateBookingStatus(c.Request.Context(), bookingID, userID, role, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Only studio owner can change status"},
			})
			return
		case errors.Is(err, ErrInvalidStatusTransition):
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   gin.H{"code": "INVALID_STATUS_TRANSITION", "message": "Invalid status transition"},
			})
			return
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "NOT_FOUND", "message": "Booking not found"},
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update status"},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"booking": gin.H{
				"id":     updated.ID,
				"status": updated.Status,
			},
		},
	})
}

// ConfirmBooking PATCH /api/v1/bookings/:id/confirm (only studio owner)
func (h *Handler) ConfirmBooking(c *gin.Context) {
	bookingID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	// Проверяем что пользователь — владелец студии
	isOwner, err := h.service.IsBookingStudioOwner(c.Request.Context(), userID, bookingID)
	if err != nil || !isOwner {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only studio owner can confirm")
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), bookingID, "confirmed"); err != nil {
		response.Error(c, http.StatusBadRequest, "UPDATE_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Booking confirmed"})
}

// CancelBooking PATCH /api/v1/bookings/:id/cancel (client or owner)
func (h *Handler) CancelBooking(c *gin.Context) {
	bookingID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	booking, err := h.service.GetByID(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found")
		return
	}

	// Проверяем права: либо владелец брони, либо владелец студии
	if booking.UserID != userID {
		isOwner, _ := h.service.IsBookingStudioOwner(c.Request.Context(), userID, bookingID)
		if !isOwner {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Cannot cancel this booking")
			return
		}
	}

	// Нельзя отменить уже завершённую бронь
	if booking.Status == "completed" {
		response.Error(c, http.StatusBadRequest, "INVALID_STATUS", "Cannot cancel completed booking")
		return
	}

	h.service.UpdateStatus(c.Request.Context(), bookingID, "cancelled")
	response.Success(c, http.StatusOK, gin.H{"message": "Booking cancelled"})
}

// CompleteBooking PATCH /api/v1/bookings/:id/complete (only owner)
// Аналогично confirm, но меняем на "completed"
// Дополнительная проверка: статус должен быть "confirmed"
func (h *Handler) CompleteBooking(c *gin.Context) {
	bookingID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	// Проверяем что пользователь — владелец студии
	isOwner, err := h.service.IsBookingStudioOwner(c.Request.Context(), userID, bookingID)
	if err != nil || !isOwner {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only studio owner can complete")
		return
	}

	// Дополнительная проверка: статус должен быть "confirmed"
	booking, err := h.service.GetByID(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found")
		return
	}

	if booking.Status != "confirmed" {
		response.Error(c, http.StatusBadRequest, "INVALID_STATUS", "Can only complete confirmed bookings")
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), bookingID, "completed"); err != nil {
		response.Error(c, http.StatusBadRequest, "UPDATE_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Booking completed"})
}

// MarkBookingPaid PATCH /api/v1/bookings/:id/mark-paid (only owner)
func (h *Handler) MarkBookingPaid(c *gin.Context) {
	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing auth")
		return
	}

	b, err := h.service.UpdatePaymentStatus(c.Request.Context(), bookingID, userID, domain.PaymentPaid)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "You cannot update this booking")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update payment status")
		return
	}

	response.Success(c, http.StatusOK, b)
}
