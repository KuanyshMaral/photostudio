package lead

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/domain/profile"
	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/utils"
	"photostudio/internal/pkg/validator"
)

// Handler handles lead HTTP requests
type Handler struct {
	service        *Service
	profileService *profile.Service
}

// NewHandler creates lead handler
func NewHandler(service *Service, profileService *profile.Service) *Handler {
	return &Handler{
		service:        service,
		profileService: profileService,
	}
}

// swaggerLeadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerLeadResponse struct {
	Success bool       `json:"success"`
	Data    *OwnerLead `json:"data"`
}

// swaggerLeadListResponse is a wrapper strictly for generating Swagger documentation.
type swaggerLeadListResponse struct {
	Success bool             `json:"success"`
	Data    LeadListResponse `json:"data"`
}

// swaggerConvertLeadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerConvertLeadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  int64  `json:"user_id"`
	Email   string `json:"email"`
}

// swaggerLeadStatsResponse is a wrapper strictly for generating Swagger documentation.
type swaggerLeadStatsResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// SubmitLead подача заявки владельцем студии.
//
//	@Summary		Подать заявку лида
//	@Description	Публичный эндпоинт для отправки заявки на регистрацию студии.
//	@Tags			Leads
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SubmitLeadRequest		true	"Данные заявки"
//	@Success		201		{object}	swaggerLeadResponse		"Заявка создана"
//	@Failure		400		{object}	response.ErrorResponse	"Ошибка входных данных"
//	@Failure		409		{object}	response.ErrorResponse	"Email уже существует"
//	@Failure		422		{object}	response.ErrorResponse	"Ошибка валидации"
//	@Failure		500		{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/leads/submit [post]
func (h *Handler) SubmitLead(w http.ResponseWriter, r *http.Request) {
	var req SubmitLeadRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	ip := utils.GetClientIP(r)
	userAgent := r.UserAgent()

	lead, err := h.service.SubmitLead(r.Context(), &req, ip, userAgent)
	if err != nil {
		if err == ErrEmailExists {
			response.CustomError(w, r, http.StatusConflict, "EMAIL_EXISTS", "Email already registered")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": lead})
}

// GetLead получение заявки по ID.
//
//	@Summary		Получить лид
//	@Description	Админский эндпоинт для получения заявки по её идентификатору.
//	@Tags			Admin Leads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int						true	"ID заявки"
//	@Success		200	{object}	swaggerLeadResponse		"Данные заявки"
//	@Failure		400	{object}	response.ErrorResponse	"Некорректный ID"
//	@Failure		401	{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse	"Заявка не найдена"
//	@Failure		500	{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id} [get]
func (h *Handler) GetLead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	lead, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": lead})
}

// ListLeads список всех заявок.
//
//	@Summary		Список лидов
//	@Description	Админский эндпоинт для просмотра заявок с возможностью фильтрации по статусу.
//	@Tags			Admin Leads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string					false	"Фильтр по статусу (new, contacted, qualified, converted, rejected)"
//	@Param			limit	query		int						false	"Лимит (дефолт: 50)"
//	@Param			offset	query		int						false	"Отступ"
//	@Success		200		{object}	swaggerLeadListResponse	"Список заявок"
//	@Failure		401		{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse	"Внутренняя ошибка сервера"
//	@Router			/admin/leads [get]
func (h *Handler) ListLeads(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var status *Status
	if s := q.Get("status"); s != "" {
		sv := Status(s)
		status = &sv
	}

	limit := 50
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	leads, total, err := h.service.ListLeads(r.Context(), status, limit, offset)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": LeadListResponse{
		Leads: convertLeads(leads),
		Total: total,
	}})
}

// UpdateStatus изменение статуса заявки.
//
//	@Summary		Обновить статус лида
//	@Description	Админский эндпоинт. Позволяет вручную указать статус, примечания и причину отказа.
//	@Tags			Admin Leads
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID заявки"
//	@Param			request	body		UpdateLeadStatusRequest		true	"Новый статус"
//	@Success		200		{object}	response.SuccessResponse	"Статус обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Заявка не найдена"
//	@Failure		409		{object}	response.ErrorResponse		"Заявка уже конвертирована"
//	@Failure		422		{object}	response.ErrorResponse		"Ошибка валидации"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id}/status [patch]
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req UpdateLeadStatusRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	if err := h.service.UpdateStatus(r.Context(), id, req.Status, req.Notes, req.Reason); err != nil {
		switch err {
		case ErrLeadNotFound:
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
		case ErrAlreadyConverted:
			response.CustomError(w, r, http.StatusConflict, "ALREADY_CONVERTED", "Lead already converted")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "Status updated"})
}

// AssignLead назначить менеджера (админа) на заявку.
//
//	@Summary		Назначить лида
//	@Description	Админ назначает заявку на определенного менеджера и задает приоритет.
//	@Tags			Admin Leads
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID заявки"
//	@Param			request	body		AssignLeadRequest			true	"Данные для назначения"
//	@Success		200		{object}	response.SuccessResponse	"Менеджер назначен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка входных данных"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Заявка не найдена"
//	@Failure		422		{object}	response.ErrorResponse		"Ошибка валидации"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id}/assign [patch]
func (h *Handler) AssignLead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req AssignLeadRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	if err := h.service.Assign(r.Context(), id, req.AdminID, req.Priority); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "Lead assigned"})
}

// RejectLead отклонить заявку.
//
//	@Summary		Отклонить лида
//	@Description	Админ отклоняет заявку (статус rejected) с указанием причины.
//	@Tags			Admin Leads
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID заявки"
//	@Param			request	body		UpdateLeadStatusRequest		true	"Укажите reason (причину отказа)"
//	@Success		200		{object}	response.SuccessResponse	"Заявка отклонена"
//	@Failure		400		{object}	response.ErrorResponse		"Заявка уже конвертирована или неверный ID"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Заявка не найдена"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id}/reject [post]
func (h *Handler) RejectLead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req UpdateLeadStatusRequest
	_ = response.BindJSON(r, &req)

	if err := h.service.RejectLead(r.Context(), id, req.Reason); err != nil {
		switch err {
		case ErrLeadNotFound:
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
		case ErrAlreadyConverted:
			response.CustomError(w, r, http.StatusBadRequest, "ALREADY_CONVERTED", "Lead already converted")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "Lead rejected"})
}

// MarkContacted отметить заявку как "Contacted".
//
//	@Summary		Отметить лида как Contacted
//	@Description	Меняет статус заявки на contacted.
//	@Tags			Admin Leads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int							true	"ID заявки"
//	@Success		200	{object}	response.SuccessResponse	"Заявка отмечена как Contacted"
//	@Failure		400	{object}	response.ErrorResponse		"Некорректный ID"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Заявка не найдена"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id}/contacted [post]
func (h *Handler) MarkContacted(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	if err := h.service.MarkContacted(r.Context(), id); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "Lead marked as contacted"})
}

// ConvertLead конвертировать заявку в аккаунт владельца.
//
//	@Summary		Конвертировать лида
//	@Description	Создает пользователя-владельца (owner) из заявки (статус converted) и профиль компании.
//	@Tags			Admin Leads
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"ID заявки"
//	@Param			request	body		ConvertLeadRequest			true	"Данные конвертации (Юр. адрес и тд)"
//	@Success		200		{object}	swaggerConvertLeadResponse	"Лид успешно конвертирован в пользователя"
//	@Failure		400		{object}	response.ErrorResponse		"Нельзя конвертировать (например, некорректный статус) или неверный запрос"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Заявка не найдена"
//	@Failure		409		{object}	response.ErrorResponse		"Email уже зарегистрирован"
//	@Failure		422		{object}	response.ErrorResponse		"Ошибка валидации"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/{id}/convert [post]
func (h *Handler) ConvertLead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req ConvertLeadRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	user, err := h.service.ConvertLead(r.Context(), id, &req)
	if err != nil {
		switch err {
		case ErrLeadNotFound:
			response.CustomError(w, r, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
		case ErrAlreadyConverted:
			response.CustomError(w, r, http.StatusBadRequest, "ALREADY_CONVERTED", "Lead already converted")
		case ErrCannotConvert:
			response.CustomError(w, r, http.StatusBadRequest, "CANNOT_CONVERT", "Lead must be qualified or contacted")
		case ErrEmailExists:
			response.CustomError(w, r, http.StatusConflict, "EMAIL_EXISTS", "User with this email already exists")
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		}
		return
	}

	// Create owner profile from lead data
	leadDetails, _ := h.service.GetByID(r.Context(), id)
	if leadDetails != nil {
		ownerProfileReq := &profile.CreateOwnerProfileRequest{
			CompanyName:     leadDetails.CompanyName,
			Bin:             getValue(leadDetails.Bin),
			LegalAddress:    req.LegalAddress,
			ContactPerson:   leadDetails.ContactName,
			ContactPosition: getValue(leadDetails.ContactPosition),
			Phone:           leadDetails.ContactPhone,
			Email:           leadDetails.ContactEmail,
			Website:         getValue(leadDetails.Website),
		}
		_, _ = h.profileService.EnsureOwnerProfile(r.Context(), user.ID, ownerProfileReq)
	}

	response.JSON(w, http.StatusOK, response.H{
		"success": true,
		"message": "Lead converted successfully and profile created",
		"user_id": user.ID,
		"email":   user.Email,
	})
}

// GetStats получение статистики по лидам.
//
//	@Summary		Статистика по лидам
//	@Description	Админский эндпоинт для просмотра воронки заявок (счетчики по статусам).
//	@Tags			Admin Leads
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerLeadStatsResponse	"Успех"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/admin/leads/stats [get]
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": stats})
}

func getValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func convertLeads(leads []*OwnerLead) []OwnerLead {
	result := make([]OwnerLead, len(leads))
	for i, lead := range leads {
		result[i] = *lead
	}
	return result
}
