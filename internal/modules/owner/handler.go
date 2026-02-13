package owner

import (
	"errors"
	"net/http"
	"strconv"

	"photostudio/internal/domain"
	"photostudio/internal/pkg/response"
	"photostudio/internal/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *repository.OwnerCRMRepository
}

func NewHandler(repo *repository.OwnerCRMRepository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes регистрирует маршруты Owner CRM
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	owner := rg.Group("/owner")
	{
		// PIN
		owner.POST("/set-pin", h.SetPIN)
		owner.POST("/verify-pin", h.VerifyPIN)
		owner.GET("/has-pin", h.HasPIN)

		// Procurement
		owner.GET("/procurement", h.GetProcurement)
		owner.POST("/procurement", h.CreateProcurement)
		owner.PATCH("/procurement/:id", h.UpdateProcurement)
		owner.DELETE("/procurement/:id", h.DeleteProcurement)

		// Maintenance
		owner.GET("/maintenance", h.GetMaintenance)
		owner.POST("/maintenance", h.CreateMaintenance)
		owner.PATCH("/maintenance/:id", h.UpdateMaintenance)
		owner.DELETE("/maintenance/:id", h.DeleteMaintenance)

		// Analytics
		owner.GET("/analytics", h.GetAnalytics)
	}
}

// RegisterCompanyRoutes регистрирует маршруты Company Profile
func (h *Handler) RegisterCompanyRoutes(rg *gin.RouterGroup) {
	company := rg.Group("/company")
	{
		company.GET("/profile", h.GetCompanyProfile)
		company.PUT("/profile", h.UpdateCompanyProfile)
		company.GET("/portfolio", h.GetPortfolio)
		company.POST("/portfolio", h.AddPortfolioProject)
		company.DELETE("/portfolio/:id", h.DeletePortfolioProject)
		company.PUT("/portfolio/reorder", h.ReorderPortfolio)
	}
}

// ==================== PIN Handlers ====================

type SetPINRequest struct {
	PIN string `json:"pin" binding:"required,min=4,max=6,numeric"`
}

// @Summary Установка PIN кода
// @Description Устанавливает PIN код из 4-6 цифр для защиты CRM функций владельца студии
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param request body SetPINRequest true "PIN request body"
// @Success 200 {object} map[string]interface{} "PIN успешно установлен"
// @Failure 400 {object} map[string]interface{} "Некорректный формат PIN (должен быть 4-6 цифр)"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/set-pin [post]
// @Security Bearer
func (h *Handler) SetPIN(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	if ownerID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req SetPINRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.repo.SetPIN(c.Request.Context(), ownerID, req.PIN); err != nil {
		response.Error(c, http.StatusInternalServerError, "SET_PIN_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "PIN set successfully"})
}

type VerifyPINRequest struct {
	PIN string `json:"pin" binding:"required"`
}

// @Summary Проверка PIN кода
// @Description Проверяет корректность введённого PIN кода владельцем студии
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param request body VerifyPINRequest true "PIN request body"
// @Success 200 {object} map[string]interface{} "PIN успешно проверен"
// @Failure 400 {object} map[string]interface{} "Некорректный формат запроса"
// @Failure 401 {object} map[string]interface{} "PIN неверен или требуется аутентификация"
// @Failure 404 {object} map[string]interface{} "PIN не установлен"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/verify-pin [post]
// @Security Bearer
func (h *Handler) VerifyPIN(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	if ownerID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req VerifyPINRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	err := h.repo.VerifyPIN(c.Request.Context(), ownerID, req.PIN)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidPIN) {
			response.Error(c, http.StatusUnauthorized, "INVALID_PIN", "PIN is incorrect")
			return
		}
		if errors.Is(err, repository.ErrPINNotSet) {
			response.Error(c, http.StatusNotFound, "PIN_NOT_SET", "PIN has not been set")
			return
		}
		response.Error(c, http.StatusInternalServerError, "VERIFY_FAILED", err.Error())
		return
	}

	// Можно вернуть временный токен для CRM сессии
	response.Success(c, http.StatusOK, gin.H{
		"verified": true,
		"message":  "PIN verified successfully",
	})
}

// @Summary Проверка наличия PIN кода
// @Description Проверяет, установлен ли PIN код для текущего владельца студии
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{} "Статус наличия PIN кода"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/has-pin [get]
// @Security Bearer
func (h *Handler) HasPIN(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	if ownerID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	hasPIN, err := h.repo.HasPIN(c.Request.Context(), ownerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CHECK_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"has_pin": hasPIN})
}

// ==================== Procurement Handlers ====================

// @Summary Получение списка закупок
// @Description Получает список закупок для студии владельца с опциональной фильтрацией по статусу завершённости
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param show_completed query bool false "Показывать ли завершённые закупки (по умолчанию false)"
// @Success 200 {object} map[string]interface{} "Список закупок и их количество"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/procurement [get]
// @Security Bearer
func (h *Handler) GetProcurement(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	showCompleted := c.Query("show_completed") == "true"

	items, err := h.repo.GetProcurementItems(c.Request.Context(), ownerID, showCompleted)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"items": items, "count": len(items)})
}

type CreateProcurementRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	EstCost     float64 `json:"est_cost,omitempty"`
	Priority    string  `json:"priority,omitempty"` // low, medium, high
	DueDate     string  `json:"due_date,omitempty"` // RFC3339
}

// @Summary Создание новой закупки
// @Description Создаёт новую запись о закупке оборудования или материалов для студии. Приоритет по умолчанию - medium, количество - 1
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param request body CreateProcurementRequest true "Данные для создания закупки"
// @Success 201 {object} map[string]interface{} "Закупка успешно создана"
// @Failure 400 {object} map[string]interface{} "Некорректные данные запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/procurement [post]
// @Security Bearer
func (h *Handler) CreateProcurement(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	var req CreateProcurementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	item := &domain.ProcurementItem{
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

	if err := h.repo.CreateProcurementItem(c.Request.Context(), item); err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"item": item})
}

// @Summary Обновление закупки
// @Description Обновляет данные существующей закупки. Нельзя изменять ID, владельца и дату создания
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param id path int64 true "ID закупки"
// @Param request body map[string]interface{} true "Поля для обновления"
// @Success 200 {object} map[string]interface{} "Закупка успешно обновлена"
// @Failure 400 {object} map[string]interface{} "Некорректный ID закупки"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 404 {object} map[string]interface{} "Закупка не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/procurement/{id} [patch]
// @Security Bearer
func (h *Handler) UpdateProcurement(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Удаляем поля, которые нельзя обновлять
	delete(updates, "id")
	delete(updates, "owner_id")
	delete(updates, "created_at")

	if err := h.repo.UpdateProcurementItem(c.Request.Context(), ownerID, itemID, updates); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Item updated"})
}

// @Summary Удаление закупки
// @Description Удаляет закупку по ID. Может удалить только владелец студии
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param id path int64 true "ID закупки"
// @Success 200 {object} map[string]interface{} "Закупка успешно удалена"
// @Failure 400 {object} map[string]interface{} "Некорректный ID закупки"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 404 {object} map[string]interface{} "Закупка не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/procurement/{id} [delete]
// @Security Bearer
func (h *Handler) DeleteProcurement(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}

	if err := h.repo.DeleteProcurementItem(c.Request.Context(), ownerID, itemID); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Item deleted"})
}

// ==================== Maintenance Handlers ====================

// @Summary Получение списка обслуживания
// @Description Получает список записей об обслуживании оборудования и помещений студии. Можно отфильтровать по статусу: all, pending, in_progress, completed
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param status query string false "Фильтр по статусу (all, pending, in_progress, completed) - по умолчанию all"
// @Success 200 {object} map[string]interface{} "Список записей об обслуживании и их количество"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/maintenance [get]
// @Security Bearer
func (h *Handler) GetMaintenance(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	status := c.DefaultQuery("status", "all") // all, pending, in_progress, completed

	items, err := h.repo.GetMaintenanceItems(c.Request.Context(), ownerID, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"items": items, "count": len(items)})
}

type CreateMaintenanceRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssignedTo  string `json:"assigned_to,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

// @Summary Создание новой записи об обслуживании
// @Description Создаёт новую запись об обслуживании оборудования, помещений или инженерных систем студии. При отсутствии приоритета устанавливается значение medium. Статус по умолчанию - pending
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param request body CreateMaintenanceRequest true "Данные для создания записи об обслуживании"
// @Success 201 {object} map[string]interface{} "Запись об обслуживании успешно создана"
// @Failure 400 {object} map[string]interface{} "Некорректные данные запроса"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/maintenance [post]
// @Security Bearer
func (h *Handler) CreateMaintenance(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	var req CreateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	item := &domain.MaintenanceItem{
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

	if err := h.repo.CreateMaintenanceItem(c.Request.Context(), item); err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"item": item})
}

// @Summary Обновление записи об обслуживании
// @Description Обновляет данные существующей записи об обслуживании. Нельзя изменять ID, владельца и дату создания
// @Tags Owner
// @Accept json
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param id path int64 true "ID записи об обслуживании"
// @Param request body map[string]interface{} true "Поля для обновления"
// @Success 200 {object} map[string]interface{} "Запись об обслуживании успешно обновлена"
// @Failure 400 {object} map[string]interface{} "Некорректный ID записи об обслуживании"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 404 {object} map[string]interface{} "Запись об обслуживании не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/maintenance/{id} [patch]
// @Security Bearer
func (h *Handler) UpdateMaintenance(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	delete(updates, "id")
	delete(updates, "owner_id")
	delete(updates, "created_at")

	if err := h.repo.UpdateMaintenanceItem(c.Request.Context(), ownerID, itemID, updates); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Item updated"})
}

// @Summary Удаление записи об обслуживании
// @Description Удаляет запись об обслуживании по ID. Может удалить только владелец студии
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Param id path int64 true "ID записи об обслуживании"
// @Success 200 {object} map[string]interface{} "Запись об обслуживании успешно удалена"
// @Failure 400 {object} map[string]interface{} "Некорректный ID записи об обслуживании"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 404 {object} map[string]interface{} "Запись об обслуживании не найдена"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/maintenance/{id} [delete]
// @Security Bearer
func (h *Handler) DeleteMaintenance(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid item ID")
		return
	}

	if err := h.repo.DeleteMaintenanceItem(c.Request.Context(), ownerID, itemID); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Item deleted"})
}

// ==================== Analytics Handler ====================

// @Summary Получение аналитики студии
// @Description Получает аналитические данные о работе студии владельца, включая статистику бронирований, доходы и другие метрики
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{} "Аналитические данные студии"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /owner/analytics [get]
// @Security Bearer
func (h *Handler) GetAnalytics(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	analytics, err := h.repo.GetOwnerAnalytics(c.Request.Context(), ownerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ANALYTICS_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"analytics": analytics})
}

// ==================== Company Profile Handlers ====================

// @Summary Получение профиля компании
// @Description Получает полные данные профиля компании/студии владельца, включая логотип, контактные данные, описание и специализацию
// @Tags Owner
// @Produce json
// @Param authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{} "Профиль компании/студии"
// @Failure 401 {object} map[string]interface{} "Требуется аутентификация"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /company/profile [get]
// @Security Bearer
func (h *Handler) GetCompanyProfile(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	profile, err := h.repo.GetCompanyProfile(c.Request.Context(), ownerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"profile": profile})
}

type UpdateCompanyProfileRequest struct {
	Logo            string            `json:"logo,omitempty"`
	CompanyName     string            `json:"company_name,omitempty"`
	ContactPerson   string            `json:"contact_person,omitempty"`
	Email           string            `json:"email,omitempty"`
	Phone           string            `json:"phone,omitempty"`
	Website         string            `json:"website,omitempty"`
	City            string            `json:"city,omitempty"`
	CompanyType     string            `json:"company_type,omitempty"`
	Description     string            `json:"description,omitempty"`
	Specialization  string            `json:"specialization,omitempty"`
	YearsExperience int               `json:"years_experience,omitempty"`
	TeamSize        int               `json:"team_size,omitempty"`
	WorkHours       string            `json:"work_hours,omitempty"`
	Services        []string          `json:"services,omitempty"`
	Socials         map[string]string `json:"socials,omitempty"`
}

// UpdateCompanyProfile обновляет профиль компании владельца.
// @Summary		Обновить профиль компании
// @Description	Обновляет все данные профиля компании/студии: логотип, контакты, описание, специализацию, социальные сети и другую информацию.
// @Tags		Owner - Профиль компании
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		request	body	UpdateCompanyProfileRequest	true	"Данные для обновления профиля компании"
// @Success		200	{object}	gin.H{message=string} "Профиль компании успешно обновлён"
// @Failure		400	{object}	gin.H "Ошибка валидации: неверные данные"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при обновлении профиля"
// @Router		/company/profile [PUT]
func (h *Handler) UpdateCompanyProfile(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	var req UpdateCompanyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	profile := &domain.CompanyProfile{
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

	if err := h.repo.UpdateCompanyProfile(c.Request.Context(), ownerID, profile); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Profile updated"})
}

// ==================== Portfolio Handlers ====================

// GetPortfolio получает портфолио студии владельца.
// @Summary		Получить портфолио
// @Description	Возвращает полный список проектов портфолио студии с информацией о каждом проекте, включая изображения и описание.
// @Tags		Owner - Портфолио
// @Security	BearerAuth
// @Produce	json
// @Success		200	{object}	gin.H{projects=[]interface{},count=int} "Список проектов портфолио"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при получении портфолио"
// @Router		/company/portfolio [GET]
func (h *Handler) GetPortfolio(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	projects, err := h.repo.GetPortfolio(c.Request.Context(), ownerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"projects": projects, "count": len(projects)})
}

type AddPortfolioRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
}

// AddPortfolioProject добавляет новый проект в портфолио студии.
// @Summary		Добавить проект в портфолио
// @Description	Добавляет новый проект в портфолио с изображением, названием и категорией. Проект автоматически добавляется в конец портфолио. Проект можно переупорядочить в любой момент.
// @Tags		Owner - Портфолио
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		request	body	AddPortfolioRequest	true	"Данные проекта (image_url, title, category)"
// @Success		201	{object}	gin.H{project=interface{}} "Проект успешно добавлен в портфолио"
// @Failure		400	{object}	gin.H "Ошибка валидации: отсутствует URL изображения"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при добавлении проекта"
// @Router		/company/portfolio [POST]
func (h *Handler) AddPortfolioProject(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	var req AddPortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	project := &domain.PortfolioProject{
		OwnerID:  ownerID,
		ImageURL: req.ImageURL,
		Title:    req.Title,
		Category: req.Category,
	}

	if err := h.repo.AddPortfolioProject(c.Request.Context(), project); err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"project": project})
}

// DeletePortfolioProject удаляет проект из портфолио.
// @Summary		Удалить проект из портфолио
// @Description	Удаляет проект из портфолио по ID. После удаления проект больше не будет виден в портфолио студии. Остальные проекты сохраняют свой порядок.
// @Tags		Owner - Портфолио
// @Security	BearerAuth
// @Produce	json
// @Param		id	path	int	true	"ID проекта портфолио"
// @Success		200	{object}	gin.H{message=string} "Проект успешно удалён из портфолио"
// @Failure		400	{object}	gin.H "Ошибка: неверный ID проекта"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		404	{object}	gin.H "Проект не найден"
// @Failure		500	{object}	gin.H "Ошибка сервера при удалении проекта"
// @Router		/company/portfolio/:id [DELETE]
func (h *Handler) DeletePortfolioProject(c *gin.Context) {
	ownerID := c.GetInt64("user_id")
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid project ID")
		return
	}

	if err := h.repo.DeletePortfolioProject(c.Request.Context(), ownerID, projectID); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Project not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Project deleted"})
}

type ReorderPortfolioRequest struct {
	ProjectIDs []int64 `json:"project_ids" binding:"required"`
}

// ReorderPortfolio переупорядочивает проекты в портфолио.
// @Summary		Переупорядочить портфолио
// @Description	Переупорядочивает проекты портфолио согласно переданному порядку ID проектов. Новый порядок заменяет старый полностью.
// @Tags		Owner - Портфолио
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		request	body	ReorderPortfolioRequest	true	"Новый порядок проектов (массив ID)"
// @Success		200	{object}	gin.H{message=string} "Портфолио успешно переупорядочено"
// @Failure		400	{object}	gin.H "Ошибка влидации: отсутствует или пустой массив ID проектов"
// @Failure		401	{object}	gin.H "Ошибка аутентификации: требуется токен"
// @Failure		500	{object}	gin.H "Ошибка сервера при переупорядочении портфолио"
// @Router		/company/portfolio/reorder [PUT]
func (h *Handler) ReorderPortfolio(c *gin.Context) {
	ownerID := c.GetInt64("user_id")

	var req ReorderPortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.repo.ReorderPortfolio(c.Request.Context(), ownerID, req.ProjectIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, "REORDER_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Portfolio reordered"})
}
