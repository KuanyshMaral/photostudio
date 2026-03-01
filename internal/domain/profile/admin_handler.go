package profile

import (
	"net/http"

	"github.com/google/uuid"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// AdminHandler handles admin profile HTTP requests
type AdminHandler struct {
	service *Service
}

// NewAdminHandler creates admin profile handler
func NewAdminHandler(service *Service) *AdminHandler {
	return &AdminHandler{service: service}
}

// swaggerAdminProfileResponse wrapper
type swaggerAdminProfileResponse struct {
	Success bool          `json:"success"`
	Data    *AdminProfile `json:"data"`
}

// GetProfile возвращает профиль администратора.
//
//	@Summary		Профиль админа
//	@Description	Возвращает данные профиля текущего администратора.
//	@Tags			Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerAdminProfileResponse	"Профиль"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Профиль не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/profile/admin [get]
func (h *AdminHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	adminIDStr := chicontext.AdminIDFromCtx(r.Context())
	userID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.CustomError(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid admin ID")
		return
	}

	profile, err := h.service.GetAdminProfile(r.Context(), userID)
	if err != nil {
		if err == ErrProfileNotFound {
			response.CustomError(w, r, http.StatusNotFound, "PROFILE_NOT_FOUND", "Admin profile not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, profile)
}

// UpdateProfile редактирует профиль администратора.
//
//	@Summary		Обновить профиль
//	@Description	Обновляет профиль администратора.
//	@Tags			Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateAdminProfileRequest	true	"Данные"
//	@Success		200		{object}	swaggerAdminProfileResponse	"Профиль обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Профиль не найден"
//	@Failure		422		{object}	response.ErrorResponse		"Unprocessable entity"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/profile/admin [patch]
func (h *AdminHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	adminIDStr := chicontext.AdminIDFromCtx(r.Context())
	userID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.CustomError(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid admin ID")
		return
	}

	var req UpdateAdminProfileRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	profile, err := h.service.UpdateAdminProfile(r.Context(), userID, &req)
	if err != nil {
		if err == ErrProfileNotFound {
			response.CustomError(w, r, http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
			return
		}
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, profile)
}
