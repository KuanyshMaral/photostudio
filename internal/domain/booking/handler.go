package booking

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ──────────────────────────────────────────────────
// Swagger Response Wrappers
// ──────────────────────────────────────────────────

type swaggerBookingData struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type swaggerCreateBookingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Booking swaggerBookingData `json:"booking"`
	} `json:"data"`
}

type swaggerGetBusySlotsResponse struct {
	Success bool              `json:"success"`
	Data    BusySlotsResponse `json:"data"`
}

type swaggerGetAvailabilityResponse struct {
	Success bool       `json:"success"`
	Data    []TimeSlot `json:"data"` // Assuming returns slice of TimeSlot
}

type swaggerGetMyBookingsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items  []Booking `json:"items"`
		Limit  int       `json:"limit"`
		Offset int       `json:"offset"`
	} `json:"data"`
}

type swaggerGetStudioBookingsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Bookings []Booking `json:"bookings"`
	} `json:"data"`
}

type swaggerUpdatePaymentResponse struct {
	Success bool    `json:"success"`
	Data    Booking `json:"data"`
}

type swaggerBookingResponseWrapper struct {
	Success bool            `json:"success"`
	Data    BookingResponse `json:"data"`
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

// CreateBooking creates a new booking.
//
//	@Summary		Создать бронирование
//	@Description	Создает новое бронирование на указанный период времени.
//	@Tags			Booking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateBookingRequest			true	"Данные для бронирования"
//	@Success		201		{object}	swaggerCreateBookingResponse	"Создано"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		409		{object}	response.ErrorResponse			"Конфликт времени бронирования"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/booking [post]
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RoomID    int64  `json:"room_id"`
		StudioID  int64  `json:"studio_id"`
		UserID    int64  `json:"user_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Notes     string `json:"notes,omitempty"`
	}

	if err := response.BindJSON(r, &payload); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid request body"},
		})
		return
	}

	startTime, err := parseBookingDateTime(payload.StartTime)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid start_time format"},
		})
		return
	}

	endTime, err := parseBookingDateTime(payload.EndTime)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid end_time format"},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "Missing auth"},
		})
		return
	}

	req := CreateBookingRequest{
		RoomID:    payload.RoomID,
		StudioID:  payload.StudioID,
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
		Notes:     payload.Notes,
	}

	b, err := h.service.CreateBooking(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			response.JSON(w, http.StatusBadRequest, response.H{
				"success": false,
				"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid booking time range"},
			})
		case errors.Is(err, ErrNotAvailable), errors.Is(err, ErrOverbooking):
			response.JSON(w, http.StatusConflict, response.H{
				"success": false,
				"error":   response.H{"code": "BOOKING_CONFLICT", "message": "Room is not available for the selected time"},
			})
		case errors.Is(err, ErrStudioClosed), errors.Is(err, ErrOutsideWorkingHours):
			response.JSON(w, http.StatusBadRequest, response.H{
				"success": false,
				"error":   response.H{"code": "VALIDATION_ERROR", "message": err.Error()},
			})
		default:
			response.JSON(w, http.StatusInternalServerError, response.H{
				"success": false,
				"error":   response.H{"code": "INTERNAL_ERROR", "message": "Failed to create booking"},
			})
		}
		return
	}

	response.JSON(w, http.StatusCreated, response.H{
		"success": true,
		"data":    response.H{"booking": response.H{"id": b.ID, "status": b.Status}},
	})
}

func parseBookingDateTime(raw string) (time.Time, error) {
	return parseBookingDateTimeAt(raw, time.Now())
}

func parseBookingDateTimeAt(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("empty datetime")
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"January 2, 2006 3:04 PM",
		"Jan 2, 2006 15:04",
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, raw); err == nil {
			return parsed, nil
		}
	}

	if parsed, err := time.Parse("15:04", raw); err == nil {
		candidate := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
		if !candidate.After(now) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate, nil
	}

	return time.Time{}, errors.New("unsupported datetime format")
}

type BusySlotDTO struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type BusySlotsResponse struct {
	Date      string        `json:"date"`
	RoomID    int64         `json:"room_id"`
	BusySlots []BusySlotDTO `json:"busy_slots"`
	OpenTime  string        `json:"open_time"`
	CloseTime string        `json:"close_time"`
}

// GetBusySlots returns busy slots for a specific room and date.
//
//	@Summary		Получить занятые слоты
//	@Description	Возвращает список занятых временных интервалов для комнаты на указанную дату.
//	@Tags			Booking
//	@Produce		json
//	@Param			id		path		int							true	"ID комнаты"
//	@Param			date	query		string						true	"Дата (YYYY-MM-DD)"
//	@Success		200		{object}	swaggerGetBusySlotsResponse	"Занятые слоты"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка валидации/формата"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/booking/room/{id}/busy-slots [get]
func (h *Handler) GetBusySlots(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": "invalid room id"})
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": "date is required"})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := h.service.GetBusySlots(r.Context(), roomID, startOfDay, endOfDay)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": "failed to get busy slots"})
		return
	}

	busy := make([]BusySlotDTO, 0, len(rows))
	for _, s := range rows {
		busy = append(busy, BusySlotDTO{Start: s.Start.Format("15:04"), End: s.End.Format("15:04")})
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": BusySlotsResponse{
		Date: dateStr, RoomID: roomID, BusySlots: busy, OpenTime: "09:00", CloseTime: "21:00",
	}})
}

// GetRoomAvailability gets available times for a room.
//
//	@Summary		Доступность комнаты
//	@Description	Получает доступные временные слоты для бронирования комнаты на выбранную дату.
//	@Tags			Booking
//	@Produce		json
//	@Param			id		path		int								true	"ID комнаты"
//	@Param			date	query		string							true	"Дата (YYYY-MM-DD)"
//	@Success		200		{object}	swaggerGetAvailabilityResponse	"Доступные слоты"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации/формата"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/booking/room/{id}/availability [get]
func (h *Handler) GetRoomAvailability(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || roomID <= 0 {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid room id"},
		})
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "date is required (YYYY-MM-DD)"},
		})
		return
	}

	availability, err := h.service.GetAvailability(r.Context(), roomID, date)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			response.JSON(w, http.StatusBadRequest, response.H{"success": false, "error": response.H{"code": "VALIDATION_ERROR", "message": "Invalid date format (YYYY-MM-DD)"}})
			return
		}
		response.JSON(w, http.StatusInternalServerError, response.H{"success": false, "error": response.H{"code": "INTERNAL_ERROR", "message": "Failed to get availability"}})
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": availability})
}

// GetMyBookings lists bookings for the authenticated user.
//
//	@Summary		Мои бронирования
//	@Description	Возвращает список бронирований текущего авторизованного пользователя.
//	@Tags			Booking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int								false	"Кол-во элементов (макс 100)"
//	@Param			offset	query		int								false	"Смещение"
//	@Success		200		{object}	swaggerGetMyBookingsResponse	"Список бронирований"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/booking/my [get]
func (h *Handler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "Missing auth"},
		})
		return
	}

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	items, err := h.service.GetMyBookings(r.Context(), userID, limit, offset)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{
			"success": false,
			"error":   response.H{"code": "INTERNAL_ERROR", "message": "Failed to get bookings"},
		})
		return
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"items": items, "limit": limit, "offset": offset},
	})
}

type UpdateBookingStatusRequest struct {
	Status string `json:"status"`
}

// GetStudioBookings lists bookings for a specific studio (Owner/Admin).
//
//	@Summary		Бронирования студии (владелец/админ)
//	@Description	Возвращает список всех бронирований для конкретной студии. Требует права владельца студии или админа.
//	@Tags			Booking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int									true	"ID студии"
//	@Success		200	{object}	swaggerGetStudioBookingsResponse	"Спиок бронирований студии"
//	@Failure		400	{object}	response.ErrorResponse				"Invalid studio ID"
//	@Failure		401	{object}	response.ErrorResponse				"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse				"Ошибка сервера"
//	@Router			/booking/studio/{id} [get]
func (h *Handler) GetStudioBookings(w http.ResponseWriter, r *http.Request) {
	studioID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	bookings, err := h.service.GetBookingsByStudio(r.Context(), studioID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get bookings")
		return
	}

	response.Success(w, http.StatusOK, response.H{"bookings": bookings})
}

// UpdatePaymentStatus updates the payment status of a booking.
//
//	@Summary		Обновить статус оплаты
//	@Description	Изменяет статус оплаты бронирования (Требуются права владельца/админа).
//	@Tags			Booking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID бронирования"
//	@Param			request	body		UpdatePaymentStatusRequest	true	"Новый статус оплаты"
//	@Success		200		{object}	swaggerUpdatePaymentResponse"Обновленный статус"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Нет прав для действия"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/booking/{id}/payment-status [put]
func (h *Handler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdatePaymentStatusRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	b, err := h.service.UpdatePaymentStatus(r.Context(), bookingID, userID, req.PaymentStatus)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "You cannot update this booking")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update payment status")
		return
	}

	response.Success(w, http.StatusOK, b)
}

// UpdateBookingStatus modifies the status of an existing booking.
//
//	@Summary		Обновить статус брони
//	@Description	Позволяет изменить статус бронирования (включая owner's override).
//	@Tags			Booking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID бронирования"
//	@Param			request	body		UpdateBookingStatusRequest	true	"Новый статус"
//	@Success		200		{object}	swaggerCreateBookingResponse"Обновлено"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse	"Нет прав"
//	@Failure		404		{object}	response.ErrorResponse	"Бронь не найдена"
//	@Failure		500		{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/booking/{id}/status [patch]
func (h *Handler) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || bookingID <= 0 {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid booking id"},
		})
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.JSON(w, http.StatusUnauthorized, response.H{
			"success": false,
			"error":   response.H{"code": "UNAUTHORIZED", "message": "Missing auth"},
		})
		return
	}

	role := chicontext.RoleFromCtx(r.Context())

	var req UpdateBookingStatusRequest
	if err := response.BindJSON(r, &req); err != nil || req.Status == "" {
		response.JSON(w, http.StatusBadRequest, response.H{
			"success": false,
			"error":   response.H{"code": "VALIDATION_ERROR", "message": "Invalid request body"},
		})
		return
	}

	updated, err := h.service.UpdateBookingStatus(r.Context(), bookingID, userID, role, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			response.JSON(w, http.StatusForbidden, response.H{
				"success": false,
				"error":   response.H{"code": "FORBIDDEN", "message": "Only studio owner can change status"},
			})
		case errors.Is(err, ErrInvalidStatusTransition):
			response.JSON(w, http.StatusBadRequest, response.H{
				"success": false,
				"error":   response.H{"code": "INVALID_STATUS_TRANSITION", "message": "Invalid status transition"},
			})
		case errors.Is(err, ErrNotFound):
			response.JSON(w, http.StatusNotFound, response.H{
				"success": false,
				"error":   response.H{"code": "NOT_FOUND", "message": "Booking not found"},
			})
		default:
			response.JSON(w, http.StatusInternalServerError, response.H{
				"success": false,
				"error":   response.H{"code": "INTERNAL_ERROR", "message": "Failed to update status"},
			})
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"data":    response.H{"booking": response.H{"id": updated.ID, "status": updated.Status}},
	})
}

// ConfirmBooking confirms a booking (Owner only).
//
//	@Summary		Подтвердить бронь
//	@Description	Подтверждает бронирование (доступно только владельцу студии).
//	@Tags			Booking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID бронирования"
//	@Success		200	{object}	response.SuccessResponse	"Подтверждено"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка обновления"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse		"Нет прав"
//	@Router			/booking/{id}/confirm [post]
func (h *Handler) ConfirmBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := chicontext.UserIDFromCtx(r.Context())

	isOwner, err := h.service.IsBookingStudioOwner(r.Context(), userID, bookingID)
	if err != nil || !isOwner {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Only studio owner can confirm")
		return
	}

	if err := h.service.UpdateStatus(r.Context(), bookingID, "confirmed"); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "UPDATE_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Booking confirmed"})
}

// CancelBooking cancels a booking and records a reason.
//
//	@Summary		Отменить бронирование
//	@Description	Отменяет активное бронирование с обязательным указанием причины.
//	@Tags			Booking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int								true	"ID бронирования"
//	@Param			request	body		CancelBookingRequest			true	"Причина отмены"
//	@Success		200		{object}	swaggerBookingResponseWrapper	"Отменено"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации/состояния"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		403		{object}	response.ErrorResponse			"Нет прав"
//	@Failure		404		{object}	response.ErrorResponse			"Бронь не найдена"
//	@Router			/booking/{id}/cancel [post]
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := chicontext.UserIDFromCtx(r.Context())
	userRole := chicontext.RoleFromCtx(r.Context())

	var req CancelBookingRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Причина отмены обязательна (минимум 10 символов)")
		return
	}

	booking, err := h.service.GetByID(r.Context(), bookingID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Booking not found")
		return
	}

	canCancel := booking.UserID == userID || userRole == "admin" || userRole == "studio_owner"
	if !canCancel {
		isOwner, _ := h.service.IsBookingStudioOwner(r.Context(), userID, bookingID)
		if !isOwner {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Cannot cancel this booking")
			return
		}
	}

	if booking.Status == BookingCompleted {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Cannot cancel completed booking")
		return
	}
	if booking.Status == BookingCancelled {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Booking is already cancelled")
		return
	}

	updatedBooking, err := h.service.CancelBooking(r.Context(), bookingID, req.Reason)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "CANCEL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, ToBookingResponse(updatedBooking, false))
}

// CompleteBooking marks a booking as completed.
//
//	@Summary		Завершить бронь
//	@Description	Отмечает подтвержденное бронирование как завершенное (Доступно владельцу).
//	@Tags			Booking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID бронирования"
//	@Success		200	{object}	response.SuccessResponse	"Успех"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка состояния"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Бронь не найдена"
//	@Router			/booking/{id}/complete [post]
func (h *Handler) CompleteBooking(w http.ResponseWriter, r *http.Request) {
	bookingID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := chicontext.UserIDFromCtx(r.Context())

	isOwner, err := h.service.IsBookingStudioOwner(r.Context(), userID, bookingID)
	if err != nil || !isOwner {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Only studio owner can complete")
		return
	}

	booking, err := h.service.GetByID(r.Context(), bookingID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Booking not found")
		return
	}

	if booking.Status != "confirmed" {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Can only complete confirmed bookings")
		return
	}

	if err := h.service.UpdateStatus(r.Context(), bookingID, "completed"); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "UPDATE_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Booking completed"})
}

// MarkBookingPaid marks a booking as fully paid.
//
//	@Summary		Отметить бронь оплаченной
//	@Description	Устанавливает статус оплаты в "paid".
//	@Tags			Booking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID бронирования"
//	@Success		200	{object}	swaggerUpdatePaymentResponse"Обновлено"
//	@Failure		400	{object}	response.ErrorResponse	"Неверный ID"
//	@Failure		401	{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		403	{object}	response.ErrorResponse	"Нет прав"
//	@Failure		500	{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/booking/{id}/pay [post]
func (h *Handler) MarkBookingPaid(w http.ResponseWriter, r *http.Request) {
	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing auth")
		return
	}

	b, err := h.service.UpdatePaymentStatus(r.Context(), bookingID, userID, PaymentPaid)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "You cannot update this booking")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update payment status")
		return
	}

	response.Success(w, http.StatusOK, b)
}

// UpdateDeposit updates the advanced deposit of a booking.
//
//	@Summary		Обновить депозит брони
//	@Description	Устанавливает сумму внесенного депозита за бронирование (Доступно админам/владельцам).
//	@Tags			Booking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int								true	"ID бронирования"
//	@Param			request	body		UpdateDepositRequest			true	"Сумма депозита"
//	@Success		200		{object}	swaggerBookingResponseWrapper	"Успех"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации/состояния"
//	@Failure		403		{object}	response.ErrorResponse			"Нет прав"
//	@Router			/booking/{id}/deposit [patch]
func (h *Handler) UpdateDeposit(w http.ResponseWriter, r *http.Request) {
	userRole := chicontext.RoleFromCtx(r.Context())
	if userRole != "admin" && userRole != "studio_owner" {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateDepositRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	booking, err := h.service.UpdateDeposit(r.Context(), bookingID, req.DepositAmount)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "UPDATE_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, ToBookingResponse(booking, true))
}

type swaggerPreBookingResponse struct {
	Success bool       `json:"success"`
	Data    PreBooking `json:"data"`
}

type swaggerPreBookingListResponse struct {
	Success bool         `json:"success"`
	Data    []PreBooking `json:"data"`
}

// CreatePreBooking creates a preliminary booking without payment.
//
//	@Summary		Создать pre-booking
//	@Description	Создает предварительную бронь без оплаты. Блокирует слот до ручного подтверждения владельцем.
//	@Tags			PreBooking
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateBookingRequest	true	"Данные pre-booking"
//	@Success		201		{object}	swaggerPreBookingResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		409		{object}	response.ErrorResponse
//	@Router			/pre-bookings [post]
func (h *Handler) CreatePreBooking(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		StudioID  int64  `json:"studio_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := response.BindJSON(r, &payload); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	startTime, err := parseBookingDateTime(payload.StartTime)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid start_time")
		return
	}
	endTime, err := parseBookingDateTime(payload.EndTime)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid end_time")
		return
	}
	pb, err := h.service.CreatePreBooking(r.Context(), userID, payload.StudioID, startTime, endTime)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, ErrActivePreBookingExists):
			response.CustomError(w, r, http.StatusConflict, "ACTIVE_PRE_BOOKING_EXISTS", "User already has active pre-booking")
		case errors.Is(err, ErrPreBookingConflict):
			response.CustomError(w, r, http.StatusConflict, "PRE_BOOKING_CONFLICT", "Time slot conflict with booking/pre-booking")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create pre-booking")
		}
		return
	}
	response.Success(w, http.StatusCreated, pb)
}

// GetMyPreBookings returns pre-bookings of current user.
//
//	@Summary		Мои pre-bookings
//	@Tags			PreBooking
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerPreBookingListResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Router			/pre-bookings/my [get]
func (h *Handler) GetMyPreBookings(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	items, err := h.service.GetMyPreBookings(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch pre-bookings")
		return
	}
	response.Success(w, http.StatusOK, items)
}

// CancelPreBooking cancels own pre-booking.
//
//	@Summary		Отменить pre-booking
//	@Tags			PreBooking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID pre-booking"
//	@Success		200	{object}	swaggerPreBookingResponse
//	@Failure		403	{object}	response.ErrorResponse
//	@Router			/pre-bookings/{id}/cancel [post]
func (h *Handler) CancelPreBooking(w http.ResponseWriter, r *http.Request) {
	preBookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || preBookingID <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid pre-booking ID")
		return
	}
	userID := chicontext.UserIDFromCtx(r.Context())
	pb, err := h.service.CancelPreBooking(r.Context(), preBookingID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Cannot cancel this pre-booking")
		case errors.Is(err, ErrInvalidPreBookingStatus):
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Invalid pre-booking status")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cancel pre-booking")
		}
		return
	}
	response.Success(w, http.StatusOK, pb)
}

// OwnerAcceptPreBooking accepts pre-booking by studio owner.
//
//	@Summary		Владелец принимает pre-booking
//	@Tags			PreBooking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID pre-booking"
//	@Success		200	{object}	swaggerPreBookingResponse
//	@Router			/pre-bookings/{id}/owner/accept [post]
func (h *Handler) OwnerAcceptPreBooking(w http.ResponseWriter, r *http.Request) {
	h.ownerStatusUpdate(w, r, "accept")
}

// OwnerConfirmPreBookingPayment marks pre-booking as paid_confirmed by owner.
//
//	@Summary		Владелец подтверждает оплату pre-booking
//	@Tags			PreBooking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID pre-booking"
//	@Success		200	{object}	swaggerPreBookingResponse
//	@Router			/pre-bookings/{id}/owner/confirm-payment [post]
func (h *Handler) OwnerConfirmPreBookingPayment(w http.ResponseWriter, r *http.Request) {
	h.ownerStatusUpdate(w, r, "confirm")
}

// OwnerCancelPreBooking rejects/cancels pre-booking by owner.
//
//	@Summary		Владелец отклоняет pre-booking
//	@Tags			PreBooking
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID pre-booking"
//	@Success		200	{object}	swaggerPreBookingResponse
//	@Router			/pre-bookings/{id}/owner/cancel [post]
func (h *Handler) OwnerCancelPreBooking(w http.ResponseWriter, r *http.Request) {
	preBookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || preBookingID <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid pre-booking ID")
		return
	}
	ownerID := chicontext.UserIDFromCtx(r.Context())
	pb, err := h.service.RejectPreBookingByOwner(r.Context(), preBookingID, ownerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Only studio owner can perform this action")
		case errors.Is(err, ErrInvalidPreBookingStatus):
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Invalid pre-booking status transition")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update pre-booking")
		}
		return
	}
	response.Success(w, http.StatusOK, pb)
}

func (h *Handler) ownerStatusUpdate(w http.ResponseWriter, r *http.Request, action string) {
	preBookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || preBookingID <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid pre-booking ID")
		return
	}
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var pb *PreBooking
	switch action {
	case "accept":
		pb, err = h.service.AcceptPreBookingByOwner(r.Context(), preBookingID, ownerID)
	case "confirm":
		pb, err = h.service.ConfirmPreBookingPaymentByOwner(r.Context(), preBookingID, ownerID)
	default:
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ACTION", "Invalid owner action")
		return
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Only studio owner can perform this action")
		case errors.Is(err, ErrInvalidPreBookingStatus):
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_STATUS", "Invalid pre-booking status transition")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update pre-booking")
		}
		return
	}
	response.Success(w, http.StatusOK, pb)
}
