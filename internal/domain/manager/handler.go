package manager

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/domain/booking"
	"photostudio/internal/domain/owner"
	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	bookingRepo booking.BookingRepository
	ownerRepo   *owner.OwnerCRMRepository
}

func NewHandler(bookingRepo booking.BookingRepository, ownerRepo *owner.OwnerCRMRepository) *Handler {
	return &Handler{
		bookingRepo: bookingRepo,
		ownerRepo:   ownerRepo,
	}
}

// swaggerManagerBookingsResponse is a wrapper strictly for generating Swagger documentation.
type swaggerManagerBookingsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Bookings []any `json:"bookings"` // booking.Booking type
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PerPage  int   `json:"per_page"`
	} `json:"data"`
}

// swaggerManagerBookingResponse is a wrapper strictly for generating Swagger documentation.
type swaggerManagerBookingResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Booking any `json:"booking"`
	} `json:"data"`
}

// swaggerManagerClientsResponse is a wrapper strictly for generating Swagger documentation.
type swaggerManagerClientsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Clients []any `json:"clients"` // owner.Client type
		Total   int64 `json:"total"`
		Page    int   `json:"page"`
		PerPage int   `json:"per_page"`
	} `json:"data"`
}

// GetBookings получение забронированных студий для менеджера.
//
//	@Summary		Получить бронирования (для менеджера)
//	@Description	Возвращает список бронирований студий владельца по фильтрам.
//	@Tags			Manager
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string							false	"Фильтр по статусу (например, completed, cancelled, all)"
//	@Param			client		query		string							false	"Фильтр по имени клиента"
//	@Param			studio_id	query		int								false	"Фильтр по ID студии"
//	@Param			room_id		query		int								false	"Фильтр по ID комнаты"
//	@Param			date_from	query		string							false	"Начальная дата (YYYY-MM-DD)"
//	@Param			date_to		query		string							false	"Конечная дата (YYYY-MM-DD)"
//	@Param			page		query		int								false	"Номер страницы"
//	@Param			per_page	query		int								false	"Количество элементов (макс 100)"
//	@Success		200			{object}	swaggerManagerBookingsResponse	"Список бронирований"
//	@Failure		401			{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500			{object}	response.ErrorResponse			"Внутренняя ошибка сервера"
//	@Router			/manager/bookings [get]
func (h *Handler) GetBookings(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	q := r.URL.Query()

	statusVal := q.Get("status")
	if statusVal == "" {
		statusVal = "all"
	}
	filters := booking.ManagerBookingFilters{
		Status:     statusVal,
		ClientName: q.Get("client"),
	}

	if v := q.Get("studio_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filters.StudioID = id
		}
	}
	if v := q.Get("room_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filters.RoomID = id
		}
	}
	if v := q.Get("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filters.DateFrom = t
		}
	}
	if v := q.Get("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filters.DateTo = t.Add(24*time.Hour - time.Second)
		}
	}
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filters.Page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp <= 100 {
			filters.PerPage = pp
		}
	}

	bookings, total, err := h.bookingRepo.GetManagerBookings(r.Context(), ownerID, filters)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"bookings": bookings,
		"total":    total,
		"page":     filters.Page,
		"per_page": filters.PerPage,
	})
}

// GetBooking детали бронирования.
//
//	@Summary		Детали бронирования
//	@Description	Возвращает подробности конкретного бронирования.
//	@Tags			Manager
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int								true	"ID бронирования"
//	@Success		200	{object}	swaggerManagerBookingResponse	"Данные бронирования"
//	@Failure		400	{object}	response.ErrorResponse			"Некорректный ID"
//	@Failure		401	{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse			"Бронирование не найдено или нет прав"
//	@Router			/manager/bookings/{id} [get]
func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())

	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	b, err := h.bookingRepo.GetBookingForManager(r.Context(), ownerID, bookingID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	response.Success(w, http.StatusOK, response.H{"booking": b})
}

type UpdateDepositRequest struct {
	DepositAmount float64 `json:"deposit_amount"`
}

// UpdateDeposit изменение суммы залога бронирования.
//
//	@Summary		Обновить залог
//	@Description	Менеджер обновляет расчет суммы залога за бронирование.
//	@Tags			Manager
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID бронирования"
//	@Param			request	body		UpdateDepositRequest		true	"Новая сумма"
//	@Success		200		{object}	response.SuccessResponse	"Залог обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Бронирование не найдено"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/manager/bookings/{id}/deposit [patch]
func (h *Handler) UpdateDeposit(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())

	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateDepositRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if _, err = h.bookingRepo.GetBookingForManager(r.Context(), ownerID, bookingID); err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	if err := h.bookingRepo.UpdateDeposit(r.Context(), bookingID, req.DepositAmount); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Deposit updated"})
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateBookingStatus обновление статуса бронирования.
//
//	@Summary		Обновить статус брони
//	@Description	Меняет статус бронирования (например, подтверждено, отменено).
//	@Tags			Manager
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID бронирования"
//	@Param			request	body		UpdateStatusRequest			true	"Новый статус"
//	@Success		200		{object}	response.SuccessResponse	"Статус обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Бронирование не найдено"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка на сервере"
//	@Router			/manager/bookings/{id}/status [patch]
func (h *Handler) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())

	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateStatusRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if _, err = h.bookingRepo.GetBookingForManager(r.Context(), ownerID, bookingID); err != nil {
		response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	if err := h.bookingRepo.UpdateStatus(r.Context(), bookingID, req.Status); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Status updated"})
}

// GetClients получение списка клиентов менеджера (владельца).
//
//	@Summary		Список клиентов
//	@Description	Возвращает список клиентов с пагинацией и поиском по имени.
//	@Tags			Manager
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search		query		string							false	"Поиск по имени"
//	@Param			page		query		int								false	"Номер страницы"
//	@Param			per_page	query		int								false	"Клиентов на странице (макс 100)"
//	@Success		200			{object}	swaggerManagerClientsResponse	"Список клиентов"
//	@Failure		401			{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500			{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/manager/clients [get]
func (h *Handler) GetClients(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	q := r.URL.Query()
	search := q.Get("search")

	page := 1
	perPage := 20

	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp <= 100 {
			perPage = pp
		}
	}

	clients, total, err := h.ownerRepo.GetClients(r.Context(), ownerID, search, page, perPage)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{
		"clients":  clients,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
