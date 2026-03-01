package profile

import (
	"net/http"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// OwnerHandler handles owner profile HTTP requests
type OwnerHandler struct {
	service *Service
}

func NewOwnerHandler(service *Service) *OwnerHandler {
	return &OwnerHandler{service: service}
}

// swaggerOwnerProfileResponse wrapper for documentation
type swaggerOwnerProfileResponse struct {
	Success bool          `json:"success"`
	Data    *OwnerProfile `json:"data"`
}

// GetProfile handles GET /api/v1/profile/owner
//
//	@Summary		Профиль владельца
//	@Description	Возвращает данные профиля текущего владельца фотостудии.
//	@Tags			Profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerOwnerProfileResponse	"Профиль"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse		"Профиль не найден"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/profile/owner [get]
func (h *OwnerHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

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
}

// UpdateProfile handles PUT /api/v1/profile/owner
//
//	@Summary		Обновить профиль
//	@Description	Обновляет профиль владельца студии.
//	@Tags			Profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateOwnerProfileRequest	true	"Данные"
//	@Success		200		{object}	swaggerOwnerProfileResponse	"Профиль обновлен"
//	@Failure		400		{object}	response.ErrorResponse		"Ошибка валидации/формата"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Профиль не найден"
//	@Failure		422		{object}	response.ErrorResponse		"Unprocessable entity"
//	@Failure		500		{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/profile/owner [put]
func (h *OwnerHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())

	var req UpdateOwnerProfileRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.CustomError(w, r, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errs)
		return
	}

	profile, err := h.service.UpdateOwnerProfile(r.Context(), userID, &req)
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
