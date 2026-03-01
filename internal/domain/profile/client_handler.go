package profile

import (
	"net/http"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// ClientHandler handles client profile HTTP requests
type ClientHandler struct {
	service *Service
}

// NewClientHandler creates client profile handler
func NewClientHandler(service *Service) *ClientHandler {
	return &ClientHandler{service: service}
}

// swaggerClientProfileResponse wrapper
type swaggerClientProfileResponse struct {
	Success bool           `json:"success"`
	Data    *ClientProfile `json:"data"`
}

// GetProfile возвращает профиль клиента.
//
//	@Summary		Профиль клиента
//	@Description	Возвращает данные профиля текущего клиента. Если не существует, создает пустой.
//	@Tags			Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerClientProfileResponse	"Профиль"
//	@Failure		401	{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/profile/client [get]
func (h *ClientHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

	profile, err := h.service.GetClientProfile(r.Context(), userID)
	if err != nil {
		if err == ErrProfileNotFound {
			profile, err = h.service.EnsureClientProfile(r.Context(), userID)
			if err != nil {
				response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
				return
			}
		} else {
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}
	}

	response.Success(w, http.StatusOK, profile)
}

// GetMe returns the current user's profile based on their role (unified endpoint).
//
//	@Summary		Мой профиль
//	@Description	Возвращает профиль текущего пользователя в зависимости от его роли.
//	@Tags			Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}	"Профиль"
//	@Failure		401	{object}	response.ErrorResponse	"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse	"Ошибка сервера"
//	@Router			/profiles/me [get]
func (h *ClientHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	role := chicontext.RoleFromCtx(r.Context())

	switch role {
	case "client":
		profile, err := h.service.EnsureClientProfile(r.Context(), userID)
		if err != nil {
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}
		response.Success(w, http.StatusOK, profile)
	case "studio_owner":
		profile, err := h.service.GetOwnerProfile(r.Context(), userID)
		if err != nil {
			if err == ErrProfileNotFound {
				response.CustomError(w, r, http.StatusNotFound, "PROFILE_NOT_FOUND", "Owner profile not found")
				return
			}
			response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
			return
		}
		response.Success(w, http.StatusOK, profile)
	default:
		response.CustomError(w, r, http.StatusBadRequest, "UNSUPPORTED_ROLE", "Use role-specific profile endpoint")
	}
}

// UpdateProfile редактирует профиль клиента.
//
//	@Summary		Обновить профиль
//	@Description	Обновляет профиль клиента.
//	@Tags			Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateClientProfileRequest		true	"Данные"
//	@Success		200		{object}	swaggerClientProfileResponse	"Профиль обновлен"
//	@Failure		400		{object}	response.ErrorResponse			"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse			"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse			"Профиль не найден"
//	@Failure		422		{object}	response.ErrorResponse			"Unprocessable entity"
//	@Failure		500		{object}	response.ErrorResponse			"Ошибка сервера"
//	@Router			/profile/client [patch]
func (h *ClientHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

	var req UpdateClientProfileRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	profile, err := h.service.UpdateClientProfile(r.Context(), userID, &req)
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
