package notification

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"photostudio/internal/pkg/response"
	"time"
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
// @Summary		Получить настройки уведомлений
// @Description	Возвращает текущие настройки уведомлений для пользователя.
// @Tags		Уведомления - Настройки
// @Security	BearerAuth
// @Success		200	{object}		PreferencesResponse "Настройки уведомлений"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/preferences [GET]
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	prefs, err := h.service.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get preferences")
		return
	}

	respPrefs := h.prefsToResponse(prefs)
	response.Success(c, http.StatusOK, respPrefs)
}

// UpdatePreferences обновляет настройки уведомлений.
// @Summary		Обновить настройки уведомлений
// @Description	Обновляет настройки уведомлений для текущего пользователя.
// @Tags		Уведомления - Настройки
// @Security	BearerAuth
// @Param		body	body		UpdatePreferencesRequest	true	"Данные для обновления"
// @Success		200	{object}		PreferencesResponse "Обновленные настройки"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/preferences [PATCH]
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Create update object
	updates := &UserPreferences{
		UserID: userID,
	}

	if req.EmailEnabled != nil {
		updates.EmailEnabled = *req.EmailEnabled
	}
	if req.PushEnabled != nil {
		updates.PushEnabled = *req.PushEnabled
	}
	if req.InAppEnabled != nil {
		updates.InAppEnabled = *req.InAppEnabled
	}
	if req.DigestEnabled != nil {
		updates.DigestEnabled = *req.DigestEnabled
	}
	if req.DigestFrequency != nil {
		updates.DigestFrequency = *req.DigestFrequency
	}
	if req.PerTypeSettings != nil {
		updates.PerTypeSettings = req.PerTypeSettings
	}

	prefs, err := h.service.UpdatePreferences(c.Request.Context(), userID, updates)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update preferences")
		return
	}

	respPrefs := h.prefsToResponse(prefs)
	response.Success(c, http.StatusOK, respPrefs)
}

// ResetPreferences сбрасывает настройки на значения по умолчанию.
// @Summary		Сбросить настройки
// @Description	Сбрасывает все настройки уведомлений на значения по умолчанию.
// @Tags		Уведомления - Настройки
// @Security	BearerAuth
// @Success		200	{object}		PreferencesResponse "Восстановленные настройки"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера"
// @Router		/notifications/preferences/reset [POST]
func (h *PreferencesHandler) ResetPreferences(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.CustomError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	prefs, err := h.service.ResetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "RESET_FAILED", "Failed to reset preferences")
		return
	}

	respPrefs := h.prefsToResponse(prefs)
	response.Success(c, http.StatusOK, respPrefs)
}

// prefsToResponse converts UserPreferences to PreferencesResponse
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
