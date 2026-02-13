package manager

import (
	"net/http"
	"strconv"
	"time"

	"photostudio/internal/pkg/response"
	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	bookingRepo *repository.BookingRepository
	ownerRepo   *repository.OwnerCRMRepository
}

func NewHandler(bookingRepo *repository.BookingRepository, ownerRepo *repository.OwnerCRMRepository) *Handler {
	return &Handler{
		bookingRepo: bookingRepo,
		ownerRepo:   ownerRepo,
	}
}

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

// GetBookings получает список бронирований по студиям владельца.
// @Summary		Получить отфильтрованные бронирования
// @Description	Выводит список бронирований для всех студий владельца с деталями с возможностью фильтрирования по статусу, дате, клиенту.
// @Tags		Менеджер - Управление бронированиями
// @Security	BearerAuth
// @Param		status		query	string	false	"Фильтр по статусу (all, pending, confirmed, cancelled, completed)"
// @Param		client		query	string	false	"Фильтр по имени клиента"
// @Param		studio_id	query	int	false	"Фильтр по ID студии"
// @Param		room_id		query	int	false	"Фильтр по ID комнаты"
// @Param		date_from	query	string	false	"Начальная дата (YYYY-MM-DD)"
// @Param		date_to		query	string	false	"Выконачая дата (YYYY-MM-DD)"
// @Param		page		query	int	false	"Номер страницы"
// @Param		per_page	query	int	false	"Количество бронирований на странице (макс 100)"
// @Success		200	{object}		map[string]interface{} "Список бронирований с пагинацией"
// @Failure		400	{object}		map[string]interface{} "Ошибка в параметрах"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/manager/bookings [GET]
func (h *Handler) GetBookings(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	filters := repository.ManagerBookingFilters{
		Status:     c.DefaultQuery("status", "all"),
		ClientName: c.Query("client"),
	}

	if studioID := c.Query("studio_id"); studioID != "" {
		if id, err := strconv.ParseInt(studioID, 10, 64); err == nil {
			filters.StudioID = id
		}
	}
	if roomID := c.Query("room_id"); roomID != "" {
		if id, err := strconv.ParseInt(roomID, 10, 64); err == nil {
			filters.RoomID = id
		}
	}

	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filters.DateFrom = t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filters.DateTo = t.Add(24*time.Hour - time.Second)
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filters.Page = p
		}
	}
	if perPage := c.Query("per_page"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil && pp <= 100 {
			filters.PerPage = pp
		}
	}

	bookings, total, err := h.bookingRepo.GetManagerBookings(c.Request.Context(), ownerID, filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"bookings": bookings,
		"total":    total,
		"page":     filters.Page,
		"per_page": filters.PerPage,
	})
}

// GetBooking получает детали одного бронирования.
// @Summary		Открыть детали бронирования
// @Description	Получает подробную информацию о бронировании (клиент, студия, настройки, цену, статус).
// @Tags		Менеджер - Управление бронированиями
// @Security	BearerAuth
// @Param		id	path	int	true	"ID бронирования"
// @Success		200	{object}		map[string]interface{} "Подробная информация о бронировании"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		404	{object}		map[string]interface{} "Бронирование не найдено или доступ запрещён"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/manager/bookings/:id [GET]
func (h *Handler) GetBooking(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	booking, err := h.bookingRepo.GetBookingForManager(c.Request.Context(), ownerID, bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"booking": booking})
}

type UpdateDepositRequest struct {
	DepositAmount float64 `json:"deposit_amount" binding:"required,min=0"`
}

// UpdateDeposit обновляет внесённою стоимость на бронировании.
// @Summary		Обновить рассчётные средства
// @Description	Обновляет внесённые средства (залог или авансовые платежи) для бронирования.
// @Tags		Менеджер - Управление бронированиями
// @Security	BearerAuth
// @Param		id		path	int						true	"ID бронирования"
// @Param		request	body	UpdateDepositRequest		true	"Новая сумма залога"
// @Success		200	{object}		map[string]interface{} "Намавка депозита обновлена"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверные данные"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		404	{object}		map[string]interface{} "Бронирование не найдено"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/manager/bookings/:id/deposit [PATCH]
func (h *Handler) UpdateDeposit(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// ownership check
	_, err = h.bookingRepo.GetBookingForManager(c.Request.Context(), ownerID, bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	if err := h.bookingRepo.UpdateDeposit(c.Request.Context(), bookingID, req.DepositAmount); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Deposit updated"})
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending confirmed cancelled completed"`
}

// UpdateBookingStatus обновляет статус бронирования.
// @Summary		Обновить статус бронирования
// @Description	Меняет статус бронирования: pending -> confirmed -> completed или cancelled.
// @Tags		Менеджер - Управление бронированиями
// @Security	BearerAuth
// @Param		id		path	int						true	"ID бронирования"
// @Param		request	body	UpdateStatusRequest		true	"Новый статус (pending, confirmed, cancelled, completed)"
// @Success		200	{object}		map[string]interface{} "Статус бронирования обновлен"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверные данные"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		404	{object}		map[string]interface{} "Бронирование не найдено"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/manager/bookings/:id/status [PATCH]
func (h *Handler) UpdateBookingStatus(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid booking ID")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// ownership check
	_, err = h.bookingRepo.GetBookingForManager(c.Request.Context(), ownerID, bookingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Booking not found or access denied")
		return
	}

	if err := h.bookingRepo.UpdateStatus(c.Request.Context(), bookingID, req.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Status updated"})
}

// GetClients получает список клиентов студий владельца.
// @Summary		Получить список клиентов
// @Description	Выводит список всех клиентов, которые бронировали студии владельца, с возможностью поиска по имени.
// @Tags		Менеджер - Отношения с клиентами
// @Security	BearerAuth
// @Param		search	query	string	false	"Поиск по имени клиента"
// @Param		page	query	int	false	"Номер страницы"
// @Param		per_page	query	int	false	"Количество клиентов на странице (макс 100)"
// @Success		200	{object}		map[string]interface{} "Список клиентов с пагинацией"
// @Failure		400	{object}		map[string]interface{} "Ошибка в параметрах"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/manager/clients [GET]
func (h *Handler) GetClients(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	search := c.Query("search")

	page := 1
	perPage := 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v <= 100 {
			perPage = v
		}
	}

	clients, total, err := h.ownerRepo.GetClients(c.Request.Context(), ownerID, search, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"clients":  clients,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}


