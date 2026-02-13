package booking

import (
	"errors"
	"net/http"
	"photostudio/internal/domain"
	"photostudio/internal/pkg/response"
	"photostudio/internal/repository"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service          *Service
	workingHoursRepo repository.StudioWorkingHoursRepository // добавить
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

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

// CreateBooking создаёт новое бронирование на указанное время и комнату
// @Summary		Создать новое бронирование
// @Description	Создаёт новое бронирование на указанную дату и время в выбранной комнате. Проверяет доступность времени, наличие конфликтов с существующими бронированиями и валидность переданных данных. Пользователь идентифицируется по токену аутентификации. При успешном создании возвращается ID и статус новой брони.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		body body CreateBookingRequest true "Данные для создания бронирования (room_id, user_id, start_time, end_time)"
// @Success		201 {object} map[string]interface{} "Бронирование успешно создано"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации или некорректные данные запроса"
// @Failure		401 {object} map[string]interface{} "Ошибка аутентификации - отсутствует или невалидный токен"
// @Failure		409 {object} map[string]interface{} "Конфликт времени - комната занята в выбранный период"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router		/bookings [post]
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

	// Extract user_id from context (set by JWT or MWork middleware)
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

	// Override user_id from context (prevents user from impersonating)
	req.UserID = userID

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

type BusySlot struct {
	Start string `json:"start"` // "10:00"
	End   string `json:"end"`   // "12:00"
}

type BusySlotsResponse struct {
	Date      string     `json:"date"`
	RoomID    int64      `json:"room_id"`
	BusySlots []BusySlot `json:"busy_slots"`
	OpenTime  string     `json:"open_time"`
	CloseTime string     `json:"close_time"`
}

// GetBusySlots возвращает список занятых временных слотов для конкретной комнаты на указанную дату
// @Summary		Получить занятые временные слоты
// @Description	Возвращает список всех занятых временных промежутков (слотов) для указанной комнаты на определённую дату. Также включает информацию о времени открытия и закрытия студии. Используется при выборе времени для новой брони. Каждый слот содержит время начала и окончания в формате HH:MM.
// @Tags		Бронирования
// @Param		id path integer true "ID комнаты"
// @Param		date query string true "Дата в формате YYYY-MM-DD (обязательный параметр)"
// @Success		200 {object} BusySlotsResponse "Список занятых временных слотов на указанную дату"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации - неверный ID комнаты или формат даты"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при получении занятых слотов"
// @Router		/rooms/{id}/busy-slots [get]
func (h *Handler) GetBusySlots(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid room id"})
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "date is required"})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Берём занятые слоты через уже существующую логику репозитория
	rows, err := h.service.GetBusySlots(c.Request.Context(), roomID, startOfDay, endOfDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to get busy slots"})
		return
	}

	busy := make([]BusySlot, 0, len(rows))
	for _, s := range rows {
		busy = append(busy, BusySlot{
			Start: s.Start.Format("15:04"),
			End:   s.End.Format("15:04"),
		})
	}

	resp := BusySlotsResponse{
		Date:      dateStr,
		RoomID:    roomID,
		BusySlots: busy,
		OpenTime:  "09:00",
		CloseTime: "21:00",
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// GetRoomAvailability возвращает информацию о доступности комнаты и забронированных слотах на конкретную дату
// @Summary		Проверить доступность комнаты на дату
// @Description	Возвращает детальную информацию о доступности указанной комнаты на определённую дату, включая забронированные временные слоты. Помогает пользователю выбрать свободное время для создания новой брони. Ответ включает информацию о всех занятых и свободных промежутках времени.
// @Tags		Бронирования
// @Param		id path integer true "ID комнаты"
// @Param		date query string true "Дата в формате YYYY-MM-DD (обязательный параметр)"
// @Success		200 {object} map[string]interface{} "Информация о доступности комнаты, включая забронированные слоты"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации - неверный ID, отсутствует дата или неправильный формат"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при получении информации о доступности"
// @Router		/rooms/{id}/availability [get]
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

// GetMyBookings возвращает список всех бронирований текущего пользователя с поддержкой пагинации
// @Summary		Получить мои бронирования
// @Description	Возвращает полный список бронирований, сделанных текущим пользователем, отфильтрованный по ID из токена аутентификации. Поддерживает пагинацию через параметры limit и offset для управления объёмом данных. Максимальное значение limit составляет 100 записей.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		limit query integer false "Максимальное количество записей (по умолчанию 20, максимум 100)"
// @Param		offset query integer false "Смещение для пагинации, начиная с 0 (по умолчанию 0)"
// @Success		200 {object} map[string]interface{} "Список бронирований текущего пользователя с информацией о пагинации"
// @Failure		401 {object} map[string]interface{} "Ошибка аутентификации - отсутствует или невалидный токен"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при получении бронирований"
// @Router		/users/me/bookings [get]
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

// GetStudioBookings возвращает список всех бронирований для конкретной студии
// @Summary		Получить бронирования студии
// @Description	Возвращает полный список всех бронирований, связанных с указанной студией. Используется владельцами студий или администраторами для управления и мониторинга бронирований. Включает информацию о клиентах, времени, статусах и платежах.
// @Tags		Бронирования
// @Param		id path integer true "ID студии"
// @Success		200 {object} map[string]interface{} "Список всех бронирований студии"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации - неверный ID студии"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при получении бронирований"
// @Router		/studios/{id}/bookings [get]
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

// UpdatePaymentStatus изменяет статус платежа для конкретного бронирования
// @Summary		Обновить статус платежа
// @Description	Обновляет информацию о статусе платежа для указанного бронирования. Только автор бронирования или администратор может обновлять статус платежа для бронирования. Статус может быть изменён с 'pending' на другие валидные значения в соответствии с логикой системы.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Param		body body UpdatePaymentStatusRequest true "Новый статус платежа (payment_status)"
// @Success		200 {object} map[string]interface{} "Статус платежа успешно обновлен"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации запроса"
// @Failure		401 {object} map[string]interface{} "Ошибка аутентификации - отсутствует токен"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - недостаточно прав для обновления этой брони"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при обновлении статуса"
// @Router		/bookings/{id}/payment-status [patch]
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

// UpdateBookingStatus изменяет статус бронирования с проверкой прав доступа и корректности перехода статуса
// @Summary		Обновить статус бронирования
// @Description	Обновляет статус указанного бронирования. Только владелец студии, администратор или автор бронирования могут изменять статусы в соответствии с правилами переходов. Поддерживает валидацию переходов статусов для обеспечения целостности бизнес-логики. Дополнительно проверяется текущий статус для предотвращения невалидных переходов.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Param		body body UpdateBookingStatusRequest true "Новый статус бронирования (status)"
// @Success		200 {object} map[string]interface{} "Статус бронирования успешно обновлен"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации или невалидный переход статуса"
// @Failure		401 {object} map[string]interface{} "Ошибка аутентификации - отсутствует или невалидный токен"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - только владелец студии может менять статус"
// @Failure		404 {object} map[string]interface{} "Бронирование не найдено"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при обновлении статуса"
// @Router		/bookings/{id}/status [patch]
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

// ConfirmBooking подтверждает бронирование, меняя его статус на 'confirmed'
// @Summary		Подтвердить бронирование
// @Description	Подтверждает указанное бронирование и устанавливает его статус в 'confirmed'. Эта операция доступна только владельцу студии, к которой относится бронирование. Подтверждение означает, что владелец студии согласился с условиями бронирования и готов к проведению сессии.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Success		200 {object} map[string]interface{} "Бронирование успешно подтверждено"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - только владелец студии может подтвердить бронирование"
// @Failure		404 {object} map[string]interface{} "Бронирование не найдено"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при подтверждении брони"
// @Router		/bookings/{id}/confirm [patch]
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

// CancelBooking отменяет существующее бронирование с обязательным указанием причины
// @Summary		Отменить бронирование
// @Description	Отменяет указанное бронирование и устанавливает его статус в 'cancelled'. Требует обязательное указание причины отмены (минимум 10 символов). Бронирование может быть отменено клиентом (автором) или владельцем/менеджером студии. Невозможно отменить уже завершённое или уже отменённое бронирование. Причина отмены сохраняется в системе для аналитики.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Param		body body CancelBookingRequest true "Причина отмены бронирования (минимум 10 символов)"
// @Success		200 {object} map[string]interface{} "Бронирование успешно отменено с указанной причиной"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации или невозможно отменить статус (уже завершено или отменено)"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - невозможно отменить это бронирование"
// @Failure		404 {object} map[string]interface{} "Бронирование не найдено"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при отмене брони"
// @Router		/bookings/{id}/cancel [patch]
func (h *Handler) CancelBooking(c *gin.Context) {
	bookingID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")
	userRole, _ := c.Get("role")

	// Block 9: Причина обязательна
	var req CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Причина отмены обязательна (минимум 10 символов)")
		return
	}

	booking, err := h.service.GetByID(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found")
		return
	}

	// Проверка прав: только владелец или менеджер студии
	canCancel := booking.UserID == userID ||
		userRole == "admin" ||
		userRole == "studio_owner"

	if !canCancel {
		// Дополнительная проверка: владелец студии
		isOwner, _ := h.service.IsBookingStudioOwner(c.Request.Context(), userID, bookingID)
		if !isOwner {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "Cannot cancel this booking")
			return
		}
	}

	// Нельзя отменить уже завершённую бронь или уже отменённую
	if booking.Status == domain.BookingCompleted {
		response.Error(c, http.StatusBadRequest, "INVALID_STATUS", "Cannot cancel completed booking")
		return
	}

	if booking.Status == domain.BookingCancelled {
		response.Error(c, http.StatusBadRequest, "INVALID_STATUS", "Booking is already cancelled")
		return
	}

	// Выполняем отмену с причиной
	updatedBooking, err := h.service.CancelBooking(c.Request.Context(), bookingID, req.Reason)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "CANCEL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, ToBookingResponse(updatedBooking, false))
}

// CompleteBooking завершает подтверждённое бронирование, меняя его статус на 'completed'
// @Summary		Завершить бронирование
// @Description	Завершает указанное бронирование и устанавливает его статус в 'completed'. Доступно только владельцу студии. Бронирование можно завершить только если оно находится в статусе 'confirmed'. Завершение бронирования означает, что сессия в студии была проведена и завершена.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Success		200 {object} map[string]interface{} "Бронирование успешно завершено"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации - невозможно завершить бронирование с текущим статусом (должно быть 'confirmed')"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - только владелец студии может завершить бронирование"
// @Failure		404 {object} map[string]interface{} "Бронирование не найдено"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при завершении брони"
// @Router		/bookings/{id}/complete [patch]
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

// MarkBookingPaid отмечает бронирование как оплаченное, обновляя статус платежа на 'paid'
// @Summary		Отметить бронирование как оплаченное
// @Description	Отмечает указанное бронирование как оплаченное, изменяя статус платежа на 'paid'. Эта операция доступна только владельцу студии. Используется для подтверждения получения платежа за бронирование. После отметки как оплаченного бронирование учитывается в доходах студии.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Success		200 {object} map[string]interface{} "Статус платежа успешно обновлен на 'paid'"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации - неверный ID бронирования"
// @Failure		401 {object} map[string]interface{} "Ошибка аутентификации - отсутствует токен"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - недостаточно прав для обновления статуса платежа"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при обновлении статуса платежа"
// @Router		/bookings/{id}/mark-paid [patch]
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

// UpdateDeposit обновляет размер залога (предоплаты) для бронирования, доступно только для менеджеров и владельцев
// @Summary		Обновить размер залога
// @Description	Обновляет размер залога (предоплаты) для указанного бронирования. Эта операция доступна только администраторам и владельцам студий. Залог представляет собой предварительный платёж для гарантирования брони. Сумма залога может быть изменена в зависимости от политики студии.
// @Tags		Бронирования
// @Security	BearerAuth
// @Param		id path integer true "ID бронирования"
// @Param		body body UpdateDepositRequest true "Новая сумма залога (deposit_amount)"
// @Success		200 {object} map[string]interface{} "Размер залога успешно обновлен"
// @Failure		400 {object} map[string]interface{} "Ошибка валидации запроса или некорректная сумма залога"
// @Failure		403 {object} map[string]interface{} "Доступ запрещен - только администраторы и владельцы студий могут обновлять залог"
// @Failure		500 {object} map[string]interface{} "Внутренняя ошибка сервера при обновлении залога"
// @Router		/bookings/{id}/deposit [patch]
func (h *Handler) UpdateDeposit(c *gin.Context) {
	userRole, _ := c.Get("role")

	// Только менеджеры и владельцы
	if userRole != "admin" && userRole != "studio_owner" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Access denied")
		return
	}

	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	booking, err := h.service.UpdateDeposit(c.Request.Context(), bookingID, req.DepositAmount)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "UPDATE_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, ToBookingResponse(booking, true))
}
