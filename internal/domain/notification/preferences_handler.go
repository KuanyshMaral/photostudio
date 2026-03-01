package notification

import (
	"net/http"
	"time"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// PreferencesHandler handles notification preferences API endpoints
type PreferencesHandler struct {
	service *Service
}

// NewPreferencesHandler creates preferences handler
func NewPreferencesHandler(service *Service) *PreferencesHandler {
	return &PreferencesHandler{service: service}
}

// GetPreferences возвращает настройки уведомлений пользователя.
//
//	@Summary	Получить настройки уведомлений
//	@Tags		Уведомления - Настройки
//	@Security	BearerAuth
//	@Router		/notifications/preferences [GET]
func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	prefs, err := h.service.GetPreferences(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get preferences")
		return
	}

	response.Success(w, http.StatusOK, h.prefsToResponse(prefs))
}

// UpdatePreferences обновляет настройки уведомлений.
//
//	@Summary	Обновить настройки уведомлений
//	@Tags		Уведомления - Настройки
//	@Security	BearerAuth
//	@Router		/notifications/preferences [PATCH]
func (h *PreferencesHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req UpdatePreferencesRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.IsEmpty() {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "empty update payload")
		return
	}

	prefs, err := h.service.UpdatePreferences(r.Context(), userID, &req)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update preferences")
		return
	}

	response.Success(w, http.StatusOK, h.prefsToResponse(prefs))
}

// ResetPreferences сбрасывает настройки на значения по умолчанию.
//
//	@Summary	Сбросить настройки
//	@Tags		Уведомления - Настройки
//	@Security	BearerAuth
//	@Router		/notifications/preferences/reset [POST]
func (h *PreferencesHandler) ResetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if userID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	prefs, err := h.service.ResetPreferences(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reset preferences")
		return
	}

	response.Success(w, http.StatusOK, h.prefsToResponse(prefs))
}

func (h *PreferencesHandler) prefsToResponse(prefs *UserPreferences) *PreferencesResponse {
	return &PreferencesResponse{
		ID:              prefs.ID,
		UserID:          prefs.UserID,
		EmailEnabled:    prefs.EmailEnabled,
		PushEnabled:     prefs.PushEnabled,
		InAppEnabled:    prefs.InAppEnabled,
		DigestEnabled:   prefs.DigestEnabled,
		DigestFrequency: prefs.DigestFrequency,
		PerTypeSettings: prefs.PerTypeSettings,
		CreatedAt:       prefs.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       prefs.UpdatedAt.Format(time.RFC3339),
	}
}
