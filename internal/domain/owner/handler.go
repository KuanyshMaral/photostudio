package owner

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	repo *OwnerCRMRepository
}

func NewHandler(repo *OwnerCRMRepository) *Handler {
	return &Handler{repo: repo}
}

// swaggerHasPINResponse is a wrapper strictly for generating Swagger documentation.
type swaggerHasPINResponse struct {
	Success bool `json:"success"`
	Data    struct {
		HasPIN bool `json:"has_pin"`
	} `json:"data"`
}

// swaggerProcurementListResponse is a wrapper strictly for generating Swagger documentation.
type swaggerProcurementListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items []any `json:"items"` // ProcurementItem
		Count int   `json:"count"`
	} `json:"data"`
}

// swaggerProcurementItemResponse is a wrapper strictly for generating Swagger documentation.
type swaggerProcurementItemResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Item any `json:"item"` // ProcurementItem
	} `json:"data"`
}

// swaggerMaintenanceListResponse is a wrapper strictly for generating Swagger documentation.
type swaggerMaintenanceListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items []any `json:"items"` // MaintenanceItem
		Count int   `json:"count"`
	} `json:"data"`
}

// swaggerMaintenanceItemResponse is a wrapper strictly for generating Swagger documentation.
type swaggerMaintenanceItemResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Item any `json:"item"` // MaintenanceItem
	} `json:"data"`
}

// swaggerCompanyProfileResponse is a wrapper strictly for generating Swagger documentation.
type swaggerCompanyProfileResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Profile any `json:"profile"` // CompanyProfile
	} `json:"data"`
}

// swaggerPortfolioListResponse is a wrapper strictly for generating Swagger documentation.
type swaggerPortfolioListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Projects []any `json:"projects"` // PortfolioProject
		Count    int   `json:"count"`
	} `json:"data"`
}

// swaggerPortfolioItemResponse is a wrapper strictly for generating Swagger documentation.
type swaggerPortfolioItemResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Project any `json:"project"` // PortfolioProject
	} `json:"data"`
}

// swaggerAnalyticsResponse is a wrapper strictly for generating Swagger documentation.
type swaggerAnalyticsResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Analytics any `json:"analytics"` // OwnerAnalytics
	} `json:"data"`
}

// ==================== PIN Handlers ====================

type SetPINRequest struct {
	PIN string `json:"pin"`
}

func (h *Handler) SetPIN(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var req SetPINRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	if err := h.repo.SetPIN(r.Context(), ownerID, req.PIN); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "SET_PIN_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "PIN set successfully"})
}

type VerifyPINRequest struct {
	PIN string `json:"pin"`
}

// VerifyPIN проверить PIN-код владельца.
//
//	@Summary		Проверить PIN
//	@Description	Разблокировать доступ на основе PIN-кода.
//	@Tags			Owner
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		VerifyPINRequest			true	"Пин"
//	@Success		200		{object}	response.SuccessResponse	"Проверено"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован или неверный PIN"
//	@Failure		404		{object}	response.ErrorResponse		"PIN не установлен"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/pin/verify [post]
func (h *Handler) VerifyPIN(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var req VerifyPINRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	err := h.repo.VerifyPIN(r.Context(), ownerID, req.PIN)
	if err != nil {
		if errors.Is(err, ErrInvalidPIN) {
			response.CustomError(w, r, http.StatusUnauthorized, "INVALID_PIN", "PIN is incorrect")
			return
		}
		if errors.Is(err, ErrPINNotSet) {
			response.CustomError(w, r, http.StatusNotFound, "PIN_NOT_SET", "PIN has not been set")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "VERIFY_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"verified": true, "message": "PIN verified successfully"})
}

// HasPIN проверить, установлен ли PIN-код.
//
//	@Summary		PIN установлен?
//	@Description	Возвращает boolean, установлен ли PIN у владельца.
//	@Tags			Owner
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerHasPINResponse	"Результат"
//	@Failure		401	{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/owner/pin [get]
func (h *Handler) HasPIN(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	hasPIN, err := h.repo.HasPIN(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CHECK_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"has_pin": hasPIN})
}

// ==================== Procurement Handlers ====================

// GetProcurement получить закупки.
//
//	@Summary		Список закупок
//	@Description	CRM: Список элементов оборудования/реквизита для планируемых закупок.
//	@Tags			Owner CRM - Procurement
//	@Produce		json
//	@Security		BearerAuth
//	@Param			show_completed	query		bool							false	"Показывать ли завершенные"
//	@Success		200				{object}	swaggerProcurementListResponse	"Список закупок"
//	@Failure		401				{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500				{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/procurement [get]
func (h *Handler) GetProcurement(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	showCompleted := r.URL.Query().Get("show_completed") == "true"
	items, err := h.repo.GetProcurementItems(r.Context(), ownerID, showCompleted)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"items": items, "count": len(items)})
}

type CreateProcurementRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	EstCost     float64 `json:"est_cost"`
	Priority    string  `json:"priority"`
	DueDate     string  `json:"due_date"`
}

// CreateProcurement создать заявку на закупку.
//
//	@Summary		Добавить закупку
//	@Description	CRM: Создание новой заявки на закупку (например, новая камера).
//	@Tags			Owner CRM - Procurement
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateProcurementRequest		true	"Данные закупки"
//	@Success		201		{object}	swaggerProcurementItemResponse	"Закупка добавлена"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/procurement [post]
func (h *Handler) CreateProcurement(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	var req CreateProcurementRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	item := &ProcurementItem{
		OwnerID:     ownerID,
		Title:       req.Title,
		Description: req.Description,
		Quantity:    req.Quantity,
		EstCost:     req.EstCost,
		Priority:    req.Priority,
	}
	if item.Quantity == 0 {
		item.Quantity = 1
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if err := h.repo.CreateProcurementItem(r.Context(), item); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CREATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusCreated, response.H{"item": item})
}

// UpdateProcurement обновить закупку.
//
//	@Summary		Обновить закупку
//	@Description	CRM: Частичное обновление закупки (например, изменение статуса).
//	@Tags			Owner CRM - Procurement
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID закупки"
//	@Param			request	body		map[string]any				true	"Поля для обновления"
//	@Success		200		{object}	response.SuccessResponse	"Закупка обновлена"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Закупка не найдена"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/procurement/{id} [patch]
func (h *Handler) UpdateProcurement(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	itemID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}
	var updates map[string]interface{}
	if err := response.BindJSON(r, &updates); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	delete(updates, "id")
	delete(updates, "owner_id")
	delete(updates, "created_at")
	if err := h.repo.UpdateProcurementItem(r.Context(), ownerID, itemID, updates); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Item updated"})
}

// DeleteProcurement удалить закупку.
//
//	@Summary		Удалить закупку
//	@Description	CRM: Удаление заявки на закупку.
//	@Tags			Owner CRM - Procurement
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID закупки"
//	@Success		200	{object}	response.SuccessResponse	"Закупка удалена"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Закупка не найдена"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/procurement/{id} [delete]
func (h *Handler) DeleteProcurement(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	itemID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}
	if err := h.repo.DeleteProcurementItem(r.Context(), ownerID, itemID); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Item deleted"})
}

// ==================== Maintenance Handlers ====================

// GetMaintenance получить задачи обслуживания.
//
//	@Summary		Список обслуживания
//	@Description	CRM: Задачи обслуживания оборудования/студии (чистка матрицы и т.д.).
//	@Tags			Owner CRM - Maintenance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string							false	"Статус обслуживания"
//	@Success		200		{object}	swaggerMaintenanceListResponse	"Список задач"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/maintenance [get]
func (h *Handler) GetMaintenance(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	items, err := h.repo.GetMaintenanceItems(r.Context(), ownerID, status)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"items": items, "count": len(items)})
}

type CreateMaintenanceRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	AssignedTo  string `json:"assigned_to"`
	DueDate     string `json:"due_date"`
}

// CreateMaintenance создать задачу обслуживания.
//
//	@Summary		Добавить задачу обслуживания
//	@Description	CRM: Добавляет задачу на обслуживание.
//	@Tags			Owner CRM - Maintenance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateMaintenanceRequest		true	"Задача на обслуживание"
//	@Success		201		{object}	swaggerMaintenanceItemResponse	"Создано"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка параметров"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/maintenance [post]
func (h *Handler) CreateMaintenance(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	var req CreateMaintenanceRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	item := &MaintenanceItem{
		OwnerID:     ownerID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		AssignedTo:  req.AssignedTo,
		Status:      "pending",
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if err := h.repo.CreateMaintenanceItem(r.Context(), item); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CREATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusCreated, response.H{"item": item})
}

// UpdateMaintenance обновить задачу обслуживания.
//
//	@Summary		Обновить задачу обслуживания
//	@Description	CRM: Частичное обновление задачи (статус, ответственный).
//	@Tags			Owner CRM - Maintenance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID задачи"
//	@Param			request	body		map[string]any				true	"Поля"
//	@Success		200		{object}	response.SuccessResponse	"Обновлено"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка параметров"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Задача не найдена"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/maintenance/{id} [patch]
func (h *Handler) UpdateMaintenance(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	itemID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}
	var updates map[string]interface{}
	if err := response.BindJSON(r, &updates); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	delete(updates, "id")
	delete(updates, "owner_id")
	delete(updates, "created_at")
	if err := h.repo.UpdateMaintenanceItem(r.Context(), ownerID, itemID, updates); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Item updated"})
}

// DeleteMaintenance удалить задачу обслуживания.
//
//	@Summary		Удалить задачу
//	@Description	CRM: Удаляет задачу из базы.
//	@Tags			Owner CRM - Maintenance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID задачи"
//	@Success		200	{object}	response.SuccessResponse	"Удалено"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Задача не найдена"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/maintenance/{id} [delete]
func (h *Handler) DeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	itemID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}
	if err := h.repo.DeleteMaintenanceItem(r.Context(), ownerID, itemID); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Item deleted"})
}

// ==================== Analytics Handler ====================

// GetAnalytics статистика владельца студии.
//
//	@Summary		Аналитика
//	@Description	Возвращает графики и метрики для дэшборда.
//	@Tags			Owner
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerAnalyticsResponse	"Аналитика"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/analytics [get]
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	analytics, err := h.repo.GetOwnerAnalytics(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "ANALYTICS_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"analytics": analytics})
}

// ==================== Company Profile Handlers ====================

// GetCompanyProfile профиль компании владельца.
//
//	@Summary		Профиль компании
//	@Description	Возвращает информацию о компании (юр. лицо) для B2B профиля.
//	@Tags			Owner Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerCompanyProfileResponse	"Профиль"
//	@Failure		401	{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/profile [get]
func (h *Handler) GetCompanyProfile(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	profile, err := h.repo.GetCompanyProfile(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"profile": profile})
}

type UpdateCompanyProfileRequest struct {
	Logo            string            `json:"logo"`
	CompanyName     string            `json:"company_name"`
	ContactPerson   string            `json:"contact_person"`
	Email           string            `json:"email"`
	Phone           string            `json:"phone"`
	Website         string            `json:"website"`
	City            string            `json:"city"`
	CompanyType     string            `json:"company_type"`
	Description     string            `json:"description"`
	Specialization  string            `json:"specialization"`
	YearsExperience int               `json:"years_experience"`
	TeamSize        int               `json:"team_size"`
	WorkHours       string            `json:"work_hours"`
	Services        []string          `json:"services"`
	Socials         map[string]string `json:"socials"`
}

// UpdateCompanyProfile обновить профиль компании.
//
//	@Summary		Обновить профиль
//	@Description	Сохраняет обновленные данные B2B профиля.
//	@Tags			Owner Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateCompanyProfileRequest	true	"Профиль"
//	@Success		200		{object}	response.SuccessResponse	"Обновлено"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибки параметров"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/profile [patch]
func (h *Handler) UpdateCompanyProfile(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	var req UpdateCompanyProfileRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	profile := &CompanyProfile{
		Logo:            req.Logo,
		CompanyName:     req.CompanyName,
		ContactPerson:   req.ContactPerson,
		Email:           req.Email,
		Phone:           req.Phone,
		Website:         req.Website,
		City:            req.City,
		CompanyType:     req.CompanyType,
		Description:     req.Description,
		Specialization:  req.Specialization,
		YearsExperience: req.YearsExperience,
		TeamSize:        req.TeamSize,
		WorkHours:       req.WorkHours,
		Services:        req.Services,
		Socials:         req.Socials,
	}
	if err := h.repo.UpdateCompanyProfile(r.Context(), ownerID, profile); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Profile updated"})
}

// ==================== Portfolio Handlers ====================

// GetPortfolio портфолио владельца.
//
//	@Summary		Скачать портфолио
//	@Description	Возвращает проекты/фотографии портфолио владельца.
//	@Tags			Owner Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerPortfolioListResponse	"Портфолио"
//	@Failure		401	{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/portfolio [get]
func (h *Handler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	projects, err := h.repo.GetPortfolio(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"projects": projects, "count": len(projects)})
}

type AddPortfolioRequest struct {
	ImageURL string `json:"image_url"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

// AddPortfolioProject добавить работу в портфолио.
//
//	@Summary		Добавить работу в портфолио
//	@Description	Добавляет проект в портфолио владельца (используя загруженное фото).
//	@Tags			Owner Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		AddPortfolioRequest				true	"Работа"
//	@Success		201		{object}	swaggerPortfolioItemResponse	"Успешно добавлено"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка параметров"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/owner/portfolio [post]
func (h *Handler) AddPortfolioProject(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	var req AddPortfolioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	project := &PortfolioProject{
		OwnerID:  ownerID,
		ImageURL: req.ImageURL,
		Title:    req.Title,
		Category: req.Category,
	}
	if err := h.repo.AddPortfolioProject(r.Context(), project); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CREATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusCreated, response.H{"project": project})
}

// DeletePortfolioProject удалить работу из портфолио.
//
//	@Summary		Удалить из портфолио
//	@Description	Удаляет проект/фото из портфолио.
//	@Tags			Owner Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID проекта"
//	@Success		200	{object}	response.SuccessResponse	"Удалено"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Проект не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/portfolio/{id} [delete]
func (h *Handler) DeletePortfolioProject(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	projectID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid project ID")
		return
	}
	if err := h.repo.DeletePortfolioProject(r.Context(), ownerID, projectID); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", "Project not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Project deleted"})
}

type ReorderPortfolioRequest struct {
	ProjectIDs []int64 `json:"project_ids"`
}

// ReorderPortfolio сохранить новый порядок портфолио.
//
//	@Summary		Порядок портфолио
//	@Description	Переупорядочить проекты в портфолио.
//	@Tags			Owner Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ReorderPortfolioRequest		true	"IDs в новом порядке"
//	@Success		200		{object}	response.SuccessResponse	"Обновлено"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/owner/portfolio/reorder [patch]
func (h *Handler) ReorderPortfolio(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	var req ReorderPortfolioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	if err := h.repo.ReorderPortfolio(r.Context(), ownerID, req.ProjectIDs); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "REORDER_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"message": "Portfolio reordered"})
}
